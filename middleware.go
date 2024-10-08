package main

import (
	"context"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type apiConfig struct {
	fileserverHits int
	jwtSecret      []byte
}

func (cfg *apiConfig) middlewareMetricsInc(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Println("Middleware triggered for:", r.URL.Path, r.Method)
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")

		if strings.HasPrefix(r.URL.Path, "/app/") {
			cfg.fileserverHits++
		}

		next.ServeHTTP(w, r)
	})
}

func (cfg *apiConfig) checkMainPageVisit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.WriteHeader(200)

	body := fmt.Sprintf(`<html>

		<body>
			<h1>Welcome, Chirpy Admin</h1>
			<p>Chirpy has been visited %d times!</p>
		</body>

		</html>`, cfg.fileserverHits)

	write, err := w.Write([]byte(body))
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}

	fmt.Printf("Response written to: %d bytes\n", write)
}

func (cfg *apiConfig) resetVisitCounter(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
	w.WriteHeader(200)

	cfg.fileserverHits = 0

	write, err := w.Write([]byte("Hits: " + strconv.Itoa(cfg.fileserverHits) + "\n"))
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}

	fmt.Printf("Response written to: %d bytes\n", write)
}

func (cfg *apiConfig) checkJWTToken(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		headerAuth := r.Header.Get("Authorization")
		tokenWithoutPrefix := strings.TrimPrefix(headerAuth, "Bearer ")

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenWithoutPrefix, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return cfg.jwtSecret, nil
		})

		if err != nil {
			http.Error(w, "Unauthorized: "+err.Error(), http.StatusUnauthorized)
			return
		}

		if !token.Valid {
			http.Error(w, "Invalid token", http.StatusUnauthorized)
			return
		}

		if claims.ExpiresAt.Time.Before(time.Now()) {
			http.Error(w, "Token has expired", http.StatusUnauthorized)
			return
		}

		if claims.Subject == "" {
			http.Error(w, "Token subject missing", http.StatusUnauthorized)
			return
		}

		if claims.Issuer != "chirpy" {
			http.Error(w, "Invalid issuer", http.StatusUnauthorized)
			return
		}

		ctx := context.WithValue(r.Context(), "claims", claims)
		next.ServeHTTP(w, r.WithContext(ctx))
	}
}
