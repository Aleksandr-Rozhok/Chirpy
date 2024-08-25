package main

import (
	"Chirpy/models"
	"strings"
)

type CleanedBody struct {
	CleanedBody string `json:"cleaned_body"`
}

type LocalChirp struct {
	*models.Chirp
}

func (b *LocalChirp) cleanBody() CleanedBody {
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
