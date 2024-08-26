package database

import (
	"Chirpy/models"
	"encoding/json"
	"errors"
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
	Users  map[int]models.User  `json:"users"`
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

func (db *DB) CreateItem(body string, typeItem string) (models.Storable, *models.UserResponse, error) {
	unmarshalFunc, ok := models.UnmarshalFunc[typeItem]
	if !ok {
		return nil, nil, errors.New("invalid type item")
	}

	newID := 0
	userResponse := models.UserResponse{}

	item, err := unmarshalFunc([]byte(body))
	if err != nil {
		return nil, nil, err
	}

	switch v := item.(type) {
	case *models.Chirp:
		newID = db.generateID("chirp")
	case *models.User:
		newID = db.generateID("user")
		db.mux.Lock()
		v.SetHashPass(v.Password)
		db.mux.Unlock()

		userResponse = models.UserResponse{
			Id:    newID,
			Email: v.Email,
		}

		if !db.emailValidator(v.Email) {
			return nil, nil, fmt.Errorf("This email address is already in use")
		}
	}

	db.mux.Lock()
	item.SetId(newID)
	db.mux.Unlock()

	return item, &userResponse, nil
}

func (db *DB) GetItems(typeItem string) ([]models.Storable, error) {
	var result []models.Storable

	loadDB, err := db.LoadDB()
	if err != nil {
		return nil, err
	}

	switch typeItem {
	case "chirps":
		for _, v := range loadDB.Chirps {
			result = append(result, &v)
		}
	case "users":
		for _, v := range loadDB.Users {
			result = append(result, &v)
		}
	}

	return result, nil
}

func (db *DB) ensureDB() error {
	filename := "database.json"
	emptyDB := DBStructure{
		Chirps: make(map[int]models.Chirp),
		Users:  make(map[int]models.User),
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

func (db *DB) WriteDB(dbStructure DBStructure, newItem models.Storable) error {
	db.mux.Lock()
	defer db.mux.Unlock()

	switch item := newItem.(type) {
	case *models.Chirp:
		dbStructure.Chirps[db.generateID("chirp")] = *item
	case *models.User:
		dbStructure.Users[db.generateID("user")] = *item
	}

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

func (db *DB) generateID(typeId string) int {
	allItems, _ := db.GetItems(typeId)
	return len(allItems) + 1
}

func (db *DB) emailValidator(email string) bool {
	allUsers, err := db.GetItems("users")
	if err != nil {
		fmt.Println(err)
	}

	for _, user := range allUsers {
		switch v := user.(type) {
		case *models.User:
			if v.Email == email {
				return false
			}
		}
	}

	return true
}
