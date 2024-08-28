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

	loadedDB, err := db.LoadDB()
	if err != nil {
		return nil, nil, err
	}

	switch v := item.(type) {
	case *models.Chirp:
		newID = db.generateID(len(loadedDB.Chirps), "chirp")
	case *models.User:
		newID = db.generateID(len(loadedDB.Users), "user")
		db.mux.Lock()
		v.SetHashPass(v.Password)
		v.GenerateRefreshToken()

		if v.ExpiresInSeconds == 0 {
			v.ExpiresInSeconds = 5184000
		}
		db.mux.Unlock()

		userResponse = models.UserResponse{
			Id:    newID,
			Email: v.Email,
		}
	}

	db.mux.Lock()
	item.SetId(newID)
	db.mux.Unlock()

	return item, &userResponse, nil
}

func (db *DB) UpdateItem(body string, typeItem string, id int) (models.Storable, *models.UserResponse, error) {
	unmarshalFunc, ok := models.UnmarshalFunc[typeItem]
	if !ok {
		return nil, nil, errors.New("invalid type item")
	}

	newItem, err := unmarshalFunc([]byte(body))
	if err != nil {
		return nil, nil, err
	}

	newItemWithType := newItem.(*models.User)
	userResponse := models.UserResponse{}

	items, err := db.GetItems(typeItem)
	if err != nil {
		return nil, nil, err
	}

	for _, item := range items {
		if item.GetId() == id {
			switch v := item.(type) {
			case *models.Chirp:
				v.Id = id
			case *models.User:
				db.mux.Lock()
				v.SetHashPass(v.Password)
				v.Email = newItemWithType.Email

				if v.ExpiresInSeconds == 0 {
					v.ExpiresInSeconds = 5184000
				} else {
					v.ExpiresInSeconds = newItemWithType.ExpiresInSeconds
				}
				db.mux.Unlock()

				userResponse = models.UserResponse{
					Id:    id,
					Email: newItemWithType.Email,
				}
			}

			db.mux.Lock()
			item.SetId(id)
			db.mux.Unlock()

			return item, &userResponse, nil
		}
	}
	return nil, nil, errors.New("item not found")
}

func (db *DB) GetItems(typeItem string) ([]models.Storable, error) {
	var result []models.Storable

	loadDB, err := db.LoadDB()
	if err != nil {
		return nil, err
	}

	switch typeItem {
	case "chirp":
		for _, v := range loadDB.Chirps {
			result = append(result, &v)
		}
	case "user":
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
		dbStructure.Chirps[db.generateID(len(dbStructure.Chirps), "chirp")] = *item
	case *models.User:
		dbStructure.Users[db.generateID(len(dbStructure.Users), "user")] = *item
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

func (db *DB) generateID(lenItems int, typeId string) int {
	if typeId == "chirp" {
		return lenItems + 1
	} else if typeId == "user" {
		return lenItems + 1
	}

	return 0
}

func (db *DB) EmailValidator(email string) bool {
	allUsers, err := db.GetItems("user")
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

func (db *DB) DeleteItem(id int) DBStructure {
	items, err := db.LoadDB()
	if err != nil {
		fmt.Println(err)
	}

	for key, _ := range items.Users {
		if key == id {
			delete(items.Users, key)
		}
	}
	return items
}

func (db *DB) RevokeRefreshToken(id int) DBStructure {
	items, err := db.LoadDB()
	if err != nil {
		fmt.Println(err)
	}

	for key, val := range items.Users {
		if key == id {
			val.RefreshToken = ""
			items.Users[key] = val
		}
	}

	return items
}
