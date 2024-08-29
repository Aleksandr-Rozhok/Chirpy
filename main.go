package main

import (
	"Chirpy/database"
	"Chirpy/models"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
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
	mux.HandleFunc("GET /api/chirps", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

		chirps, err := db.GetItems("chirp")
		if err != nil {
			return
		}

		respondWithJSON(w, http.StatusOK, chirps)
	})
	mux.HandleFunc("POST /api/chirps", cfg.checkJWTToken(func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value("claims").(*jwt.RegisteredClaims)
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

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

		authorIDStr, err := claims.GetSubject()
		if err != nil {
			http.Error(w, "Error extracting subject claims", http.StatusInternalServerError)
		}

		authorID, err := strconv.Atoi(authorIDStr)
		if err != nil {
			http.Error(w, "Error extracting subject claims", http.StatusInternalServerError)
		}

		chirp, err := db.CreateChirp(string(bodyBytes), authorID)
		if err != nil {
			fmt.Printf("Error creating chirp: %v\n", err)
		}

		loadDB, err := db.LoadDB()
		if err != nil {
			fmt.Printf("Error loading DB: %v\n", err)
		}

		err = db.WriteDB(loadDB, chirp)
		if err != nil {
			fmt.Printf("Error writing database: %v\n", err)
		}

		respondWithJSON(w, http.StatusCreated, chirp)
	}))
	mux.HandleFunc("GET /api/chirps/{chirpID}", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

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
	mux.HandleFunc("DELETE /api/chirps/{chirpID}", cfg.checkJWTToken(func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value("claims").(*jwt.RegisteredClaims)

		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

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

		authorIDStr, err := claims.GetSubject()
		if err != nil {
			http.Error(w, "Error extracting subject claims", http.StatusInternalServerError)
		}

		authorID, err := strconv.Atoi(authorIDStr)
		if err != nil {
			http.Error(w, "Error extracting subject claims", http.StatusInternalServerError)
		}

		if len(chirps) < idOfChirp {
			respondWithError(w, http.StatusNotFound, "chirp not found")
		} else {
			chirp := chirps[idOfChirp-1]
			typedChirp := chirp.(*models.Chirp)

			if typedChirp.AuthorId == authorID {
				newDB := db.DeleteItem(idOfChirp, "chirps")
				var emptyItem models.Storable
				err = db.WriteDB(newDB, emptyItem)
				if err != nil {
					fmt.Printf("Error writing database: %v\n", err)
				}

				respondWithJSON(w, http.StatusNoContent, chirp)
			} else {
				respondWithError(w, http.StatusForbidden, "You don't have access to deleting this chirp")
			}

		}
	}))
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

		loadDB, err := db.LoadDB()
		if err != nil {
			fmt.Printf("Error loading DB: %v\n", err)
		}

		user, userResponse, err := db.CreateUser(string(bodyBytes))
		if err != nil {
			fmt.Printf("Error creating chirp: %v\n", err)
		}

		userA, _ := user.(*models.User)

		if !db.EmailValidator(userA.Email) {
			respondWithError(w, http.StatusConflict, "This email address already exists")
			return
		}

		err = db.WriteDB(loadDB, user)
		if err != nil {
			fmt.Printf("Error writing database: %v\n", err)
		}

		respondWithJSON(w, http.StatusCreated, userResponse)
	})
	mux.HandleFunc("PUT /api/users", cfg.checkJWTToken(func(w http.ResponseWriter, r *http.Request) {
		claims := r.Context().Value("claims").(*jwt.RegisteredClaims)
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

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

		userID, err := strconv.Atoi(claims.Subject)
		if err != nil {
			fmt.Printf("Error converting user ID to int: %v\n", err)
		}

		newDB := db.DeleteItem(userID, "users")
		updatedUser, updatedUserResponse, err := db.UpdateItem(string(bodyBytes), "user", userID)

		err = db.WriteDB(newDB, updatedUser)
		if err != nil {
			fmt.Printf("Error writing database: %v\n", err)
		}

		respondWithJSON(w, http.StatusOK, updatedUserResponse)
	}))
	mux.HandleFunc("POST /api/login", func(w http.ResponseWriter, r *http.Request) {
		checkFlag := false
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

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

		users, err := db.GetItems("user")
		if err != nil {
			respondWithError(w, http.StatusNotFound, "users not found")
		}

		userA, _ := item.(*models.User)

		for _, user := range users {
			userB, _ := user.(*models.User)

			equalPass := bcrypt.CompareHashAndPassword([]byte(userB.Password), []byte(userA.Password))
			if userA.Email == userB.Email && equalPass == nil {
				checkFlag = true

				claims := &jwt.RegisteredClaims{
					ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
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
					Id:           userB.Id,
					Email:        userB.Email,
					Token:        tokenString,
					RefreshToken: userB.RefreshToken,
					IsChirpyRed:  userB.IsChirpyRed,
				}

				respondWithJSON(w, http.StatusOK, userResponse)
			}
		}

		if !checkFlag {
			respondWithError(w, http.StatusUnauthorized, "Wrong username or password")
		}
	})
	mux.HandleFunc("POST /api/refresh", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error opening database: %v\n", err)
		}

		equal := false
		userId := 0

		headerAuth := r.Header.Get("Authorization")
		refreshTokenWithoutPrefix := strings.TrimPrefix(headerAuth, "Bearer ")

		users, err := db.GetItems("user")
		if err != nil {
			respondWithError(w, http.StatusNotFound, "users not found")
		}

		bytes1, err := hex.DecodeString(refreshTokenWithoutPrefix)
		if err != nil {
			fmt.Println(err)
		}

		for _, user := range users {
			userB, _ := user.(*models.User)

			bytes2, err := hex.DecodeString(userB.RefreshToken)
			if err != nil {
				fmt.Println(err)
			}

			if subtle.ConstantTimeCompare(bytes1, bytes2) == 1 {
				equal = true
				userId = userB.Id
			}
		}

		if equal {
			claims := &jwt.RegisteredClaims{
				ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour * 1)),
				Issuer:    "chirpy",
				Subject:   strconv.Itoa(userId),
			}

			token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)

			tokenString, err := token.SignedString(jwtSecret)
			if err != nil {
				fmt.Println("Error signing token:", err)
				return
			}

			tokenResponse := models.TokenResponse{
				Token: tokenString,
			}
			respondWithJSON(w, http.StatusOK, tokenResponse)
		} else {
			respondWithError(w, http.StatusUnauthorized, "Wrong refresh token")
		}
	})
	mux.HandleFunc("POST /api/revoke", func(w http.ResponseWriter, r *http.Request) {
		db, err := database.NewDB("database.json")
		if err != nil {
			fmt.Printf("Error loading DB: %v\n", err)
		}

		allUsers, err := db.GetItems("user")
		if err != nil {
			respondWithError(w, http.StatusNotFound, "users not found")
		}

		headerAuth := r.Header.Get("Authorization")
		refreshTokenWithoutPrefix := strings.TrimPrefix(headerAuth, "Bearer ")
		flagCheck := false

		for _, user := range allUsers {
			userB, _ := user.(*models.User)

			if userB.RefreshToken == refreshTokenWithoutPrefix {
				flagCheck = true
				newDB := db.RevokeRefreshToken(userB.Id)

				data, err := json.Marshal(newDB)
				if err != nil {
					fmt.Printf("Error marshalling DB: %v\n", err)
				}

				err = os.WriteFile("database.json", data, 0644)
				if err != nil {
					fmt.Printf("Error writing DB: %v\n", err)
				}

				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
				w.WriteHeader(http.StatusNoContent)
			}
		}

		if !flagCheck {
			respondWithError(w, http.StatusUnauthorized, "There is not yours token")
		}
	})
	mux.HandleFunc("POST /api/polka/webhooks", func(w http.ResponseWriter, r *http.Request) {
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

		var data models.Webhooks
		if err := json.Unmarshal(bodyBytes, &data); err != nil {
			fmt.Printf("Error unmarshalling user: %v\n", err)
		}

		if data.Event != "user.upgraded" {
			w.WriteHeader(http.StatusNoContent)
			return
		} else {
			db, err := database.NewDB("database.json")
			if err != nil {
				fmt.Printf("Error opening database: %v\n", err)
			}

			allUsers, err := db.GetItems("user")
			if err != nil {
				respondWithError(w, http.StatusNotFound, "users not found")
			}

			if len(allUsers) < data.Data.UserID {
				respondWithError(w, http.StatusNotFound, "users not found")
			} else {
				ourUser := allUsers[data.Data.UserID-1]
				ourTypedUser := ourUser.(*models.User)

				user := fmt.Sprintf(`{"email": "%s", 
					"password": "%s", 
					"is_chirpy_red": %v,
					"refresh_token": "%s",
					"expires_in_seconds": %d
					}`, ourTypedUser.Email,
					ourTypedUser.Password,
					true,
					ourTypedUser.RefreshToken,
					ourTypedUser.ExpiresInSeconds)

				updatedUser, _, err := db.UpdateItem(user, "user", ourTypedUser.Id)
				if err != nil {
					return
				}

				newDB := db.DeleteItem(ourTypedUser.Id, "users")
				err = db.WriteDB(newDB, updatedUser)
				if err != nil {
					fmt.Printf("Error writing database: %v\n", err)
				}

				w.WriteHeader(http.StatusNoContent)
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
