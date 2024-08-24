package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

type Error struct {
	Error string `json:"error"`
}

func respondWithError(w http.ResponseWriter, code int, msg string) {
	resBody := Error{
		Error: msg,
	}

	dat, err := json.Marshal(resBody)
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
