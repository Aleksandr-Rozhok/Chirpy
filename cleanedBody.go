package main

import (
	"strings"
)

type CleanedBody struct {
	CleanedBody string `json:"cleaned_body"`
}

func (b *Body) cleanBody() CleanedBody {
	profane := []string{
		"kerfuffle",
		"sharbert",
		"fornax",
	}

	splitBody := strings.Split(b.Body, " ")

	for i, str := range splitBody {
		for _, profane := range profane {
			if strings.ToLower(str) == profane {
				splitBody[i] = "****"
			}
		}
	}

	return CleanedBody{
		CleanedBody: strings.Join(splitBody, " "),
	}
}
