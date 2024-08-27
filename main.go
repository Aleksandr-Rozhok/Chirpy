package main

import (
	"Chirpy/database"
	"Chirpy/models"
	"flag"
	"fmt"
	"github.com/golang-jwt/jwt/v5"
	"github.com/joho/godotenv"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	err := godotenv.Load()
	if err != nil {
		fmt.Println("Error loading .env file")
	}
	jwtSecret := []byte(os.Getenv("JWT_SECRET"))

	debug := flag.Bool("debug", false, "Run server in debug mode")
	flag.Parse()

	if *debug {
		err := os.Remove("database.json")
		if err != nil && !os.IsNotExist(err) {
			fmt.Printf("Failed to delete database: %v\n", err)
		} else {
			fmt.Println("Database deleted successfully")
		}
	}

	mux := http.NewServeMux()

	server := &http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	cfg := apiConfig{
		fileserverHits: 0,
		jwtSecret:      jwtSecret,
	}

	mux.HandleFunc("GET /api/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.WriteHeader(http.StatusOK)

		write, err := w.Write([]byte("OK"))
		if err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}

		fmt.Printf("Response written to: %d bytes\n", write)
	})
	mux.HandleFunc("GET /admin/metrics", cfg.checkMainPageVisit)
	mux.HandleFunc("GET /api/reset", cfg.resetVisitCounter)
	mux.HandleFunc("/api/chirps", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

		switch r.Method {
		case "GET":
			chirps, err := db.GetItems("chirp")
			if err != nil {
				return
			}

			respondWithJSON(w, http.StatusOK, chirps)
		case "POST":
			db, err := database.NewDB("database.json")

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}

			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {
					fmt.Printf("Error closing body: %v\n", err)
				}
			}(r.Body)

			chirp, _, err := db.CreateItem(string(bodyBytes), "chirp")
			if err != nil {
				fmt.Printf("Error creating chirp: %v\n", err)
			}
			fmt.Printf("Created chirp %v\n", chirp)

			loadDB, err := db.LoadDB()
			if err != nil {
				fmt.Printf("Error loading DB: %v\n", err)
			}

			err = db.WriteDB(loadDB, chirp)
			if err != nil {
				fmt.Printf("Error writing database: %v\n", err)
			}

			respondWithJSON(w, http.StatusCreated, chirp)
		}
	})
	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")

		path := r.URL.Path
		sliceOfPath := strings.Split(path, "/")
		idOfChirp, err := strconv.Atoi(sliceOfPath[len(sliceOfPath)-1])
		if err != nil {
			fmt.Printf("Error converting chirp ID to int: %v\n", err)
		}

		chirps, err := db.GetItems("chirp")
		if err != nil {
			fmt.Printf("Error getting chirp %v\n", chirps)
		}

		if len(chirps) < idOfChirp {
			respondWithError(w, http.StatusNotFound, "chirp not found")
		} else {
			chirp := chirps[idOfChirp-1]
			respondWithJSON(w, http.StatusOK, chirp)
		}
	})
	mux.HandleFunc("POST /api/users", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("Error writing response: %v\n", err)
			}
		}(r.Body)

		user, userResponse, err := db.CreateItem(string(bodyBytes), "user")
		if err != nil {
			fmt.Printf("Error creating chirp: %v\n", err)
		}
		fmt.Printf("Created user %v\n", user)

		userA, _ := user.(*models.User)

		if !db.EmailValidator(userA.Email) {
			respondWithError(w, http.StatusConflict, "This email address already exists")
		}

		loadDB, err := db.LoadDB()
		if err != nil {
			fmt.Printf("Error loading DB: %v\n", err)
		}

		err = db.WriteDB(loadDB, user)
		if err != nil {
			fmt.Printf("Error writing database: %v\n", err)
		}

		respondWithJSON(w, http.StatusCreated, userResponse)
	})
	mux.HandleFunc("PUT /api/users", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

		headerAuth := r.Header.Get("Authorization")
		tokenWithoutPrefix := strings.TrimPrefix(headerAuth, "Bearer ")

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("Error writing response: %v\n", err)
			}
		}(r.Body)

		claims := &jwt.RegisteredClaims{}
		token, err := jwt.ParseWithClaims(tokenWithoutPrefix, claims, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
			}
			return jwtSecret, nil
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

		userID, err := strconv.Atoi(claims.Subject)
		if err != nil {
			fmt.Printf("Error converting user ID to int: %v\n", err)
		}

		updatedUser, updatedUserResponse, err := db.UpdateItem(string(bodyBytes), "user", userID)

		loadDB, err := db.LoadDB()
		if err != nil {
			fmt.Printf("Error loading DB: %v\n", err)
		}

		err = db.WriteDB(loadDB, updatedUser)
		if err != nil {
			fmt.Printf("Error writing database: %v\n", err)
		}

		respondWithJSON(w, http.StatusOK, updatedUserResponse)
	})
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")

		bodyBytes, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, "Error reading request body", http.StatusInternalServerError)
			return
		}

		defer func(Body io.ReadCloser) {
			err := Body.Close()
			if err != nil {
				fmt.Printf("Error writing response: %v\n", err)
			}
		}(r.Body)

		unmarshalFunc, ok := models.UnmarshalFunc["user"]
		if !ok {
			fmt.Printf("Error unmarshalling user: %v\n", err)
		}

		item, err := unmarshalFunc(bodyBytes)
		if err != nil {
			fmt.Printf("Error unmarshalling user: %v\n", err)
		}

		users, err := db.GetItems("users")
		if err != nil {
			respondWithError(w, http.StatusNotFound, "users not found")
		}

		userA, _ := item.(*models.User)

		for _, user := range users {
			userB, _ := user.(*models.User)

			equalPass := bcrypt.CompareHashAndPassword([]byte(userB.Password), []byte(userA.Password))
			if equalPass != nil {
				respondWithError(w, http.StatusUnauthorized, "Wrong username or password")
			} else if userA.Email != userB.Email {
				respondWithError(w, http.StatusUnauthorized, "Wrong username or password")
			} else {
				claims := &jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Second + time.Duration(userA.ExpiresInSeconds))),
					Issuer:    "chirpy",
					Subject:   strconv.Itoa(userB.Id),
				}

				token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

				tokenString, err := token.SignedString(jwtSecret)
				if err != nil {
					fmt.Println("Error signing token:", err)
					return
				}

				userResponse := models.APIUserResponse{
					Id:    userB.Id,
					Email: userB.Email,
					Token: tokenString,
				}

				respondWithJSON(w, http.StatusOK, userResponse)
			}
		}

	})

	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileServerHandler))

	err = http.ListenAndServe(server.Addr, server.Handler)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
