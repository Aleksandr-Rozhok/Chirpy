package main

import (
	"Chirpy/database"
	"Chirpy/models"
	"flag"
	"fmt"
	"golang.org/x/crypto/bcrypt"
	"io"
	"net/http"
	"os"
	"strconv"
	"strings"
)

func main() {
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
		fmt.Printf("Created chirp %v\n", user)

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

		for _, user := range users {
			userA, _ := item.(*models.User)
			userB, _ := user.(*models.User)

			equalPass := bcrypt.CompareHashAndPassword([]byte(userB.Password), []byte(userA.Password))
			if equalPass != nil {
				respondWithError(w, http.StatusUnauthorized, "Wrong username or password")
			} else if userA.Email != userB.Email {
				respondWithError(w, http.StatusUnauthorized, "Wrong username or password")
			} else {
				userResponse := models.UserResponse{
					Id:    userB.Id,
					Email: userB.Email,
				}

				respondWithJSON(w, http.StatusOK, userResponse)
			}
		}

	})

	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileServerHandler))

	err := http.ListenAndServe(server.Addr, server.Handler)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
