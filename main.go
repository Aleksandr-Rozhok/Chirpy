package main

import (
	"Chirpy/database"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func main() {
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
		w.WriteHeader(200)

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
			w.Header().Set("Content-Type", "json; charset=utf-8")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.WriteHeader(200)

			chirps, err := db.GetChirps()
			if err != nil {
				return
			}

			data, err := json.MarshalIndent(chirps, "", "  ")
			write, err := w.Write(data)
			if err != nil {
				fmt.Printf("Error writing response: %v\n", err)
			}
			fmt.Printf("Response written to: %d bytes\n", write)
		case "POST":
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
			w.WriteHeader(http.StatusCreated)

			bodyBytes, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Error reading request body", http.StatusInternalServerError)
				return
			}

			defer func(Body io.ReadCloser) {
				err := Body.Close()
				if err != nil {

				}
			}(r.Body)

			chirp, err := db.CreateChirp(string(bodyBytes))
			if err != nil {
				return
			}
			fmt.Printf("Created chirp %v\n", chirp)

			loadDB, err := db.LoadDB()
			if err != nil {
				return
			}

			err = db.WriteDB(loadDB, chirp)
			if err != nil {
				fmt.Printf("Error writing database: %v\n", err)
			}

			responseData, err := json.Marshal(chirp)
			write, err := w.Write(responseData)
			if err != nil {
				return
			}
			fmt.Printf("Response written to: %d bytes\n", write)
		}
	})

	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileServerHandler))

	err := http.ListenAndServe(server.Addr, server.Handler)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
