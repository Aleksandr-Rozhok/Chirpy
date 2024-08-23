package main

import (
	"fmt"
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

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store, no-cache, must-revalidate, max-age=0")
		w.WriteHeader(200)

		write, err := w.Write([]byte("OK"))
		if err != nil {
			fmt.Printf("Error writing response: %v\n", err)
		}

		fmt.Printf("Response written to: %d bytes\n", write)
	})
	mux.HandleFunc("/metrics", cfg.checkMainPageVisit)
	mux.HandleFunc("/reset", cfg.resetVisitCounter)

	fileServerHandler := http.StripPrefix("/app", http.FileServer(http.Dir(".")))
	mux.Handle("/app/", cfg.middlewareMetricsInc(fileServerHandler))

	err := http.ListenAndServe(server.Addr, server.Handler)
	if err != nil {
		fmt.Println("Error:", err)
	}
}
