package main

import (
	"Chirpy/models"
)

func responseWithSort(chirps []models.Storable, sortType string) []models.Storable {
	if sortType == "desc" {
		for i, j := 0, len(chirps)-1; i < j; i, j = i+1, j-1 {
			chirps[i], chirps[j] = chirps[j], chirps[i]
		}

		return chirps
	} else {
		return chirps
	}
}
