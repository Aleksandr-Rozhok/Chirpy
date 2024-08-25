package database

import (
	"Chirpy/models"
	"encoding/json"
	"fmt"
	"os"
	"sync"
)

type DB struct {
	path string
	mux  *sync.RWMutex
}

type DBStructure struct {
	Chirps map[int]models.Chirp `json:"chirps"`
}

func NewDB(path string) (*DB, error) {
	db := DB{
		path: path,
		mux:  new(sync.RWMutex),
	}

	err := db.ensureDB()
	if err != nil {
		return nil, err
	}

	return &db, nil
}

func (db *DB) CreateChirp(body string) (models.Chirp, error) {
	var chirp models.Chirp
	err := json.Unmarshal([]byte(body), &chirp)
	if err != nil {
		fmt.Println(err)
	}

	newID := db.generateID()
	db.mux.Lock()
	chirp.Id = newID
	db.mux.Unlock()

	return chirp, nil
}

func (db *DB) GetChirps() ([]models.Chirp, error) {
	loadDB, err := db.LoadDB()
	if err != nil {
		return nil, err
	}

	result := make([]models.Chirp, len(loadDB.Chirps)+1)

	for k, v := range loadDB.Chirps {
		fmt.Println(k, v)
		result[k] = v
	}

	result = result[1:]

	return result, nil
}

func (db *DB) ensureDB() error {
	filename := "database.json"
	emptyDB := DBStructure{
		Chirps: make(map[int]models.Chirp),
	}

	data, err := json.Marshal(&emptyDB)
	if err != nil {
		return err
	}

	if _, err := os.Stat(filename); os.IsNotExist(err) {
		err = os.WriteFile(filename, data, 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

func (db *DB) LoadDB() (DBStructure, error) {
	file, err := os.ReadFile(db.path)
	if err != nil {
		return DBStructure{}, err
	}

	var dbStructure DBStructure
	err = json.Unmarshal(file, &dbStructure)
	if err != nil {
		return DBStructure{}, err
	}
	return dbStructure, nil
}

func (db *DB) WriteDB(dbStructure DBStructure, newChirp models.Chirp) error {
	db.mux.Lock()
	defer db.mux.Unlock()
	dbStructure.Chirps[db.generateID()] = newChirp

	data, err := json.Marshal(dbStructure)
	if err != nil {
		return err
	}

	err = os.WriteFile(db.path, data, 0644)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) generateID() int {
	allChirps, _ := db.GetChirps()
	return len(allChirps) + 1
}
