package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func respondWithJSON(w http.ResponseWriter, code int, payload interface{}) {
	dat, err := json.Marshal(payload)
	if err != nil {
		fmt.Printf("Error writing response: %v\n", err)
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	write, err := w.Write(dat)
	if err != nil {
		fmt.Printf("Error writing respones: %s", err)
	}

	fmt.Printf("Response written to: %d bytes\n", write)
}
