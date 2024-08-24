package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Body struct {
	Body string `json:"body"`
}

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
	mux.HandleFunc("POST /api/validate_chirp", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")

		decoder := json.NewDecoder(r.Body)
		params := Body{}
		err := decoder.Decode(&params)

		if err != nil {
			respondWithError(w, 400, "Chirp is too long")
		} else if len(params.Body) > 140 {
			respondWithError(w, 400, "Something went wrong")
		} else {
			cleanedBody := params.cleanBody()
			respondWithJSON(w, 200, cleanedBody)
		}
	})

	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileServerHandler))

	err := http.ListenAndServe(server.Addr, server.Handler)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
