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

func (db *DB) CreateChirp(body string, authorId int) (models.Storable, error) {
	unmarshalFunc, ok := models.UnmarshalFunc["chirp"]
	if !ok {
		return nil, errors.New("invalid type item")
	}

	loadedDB, err := db.LoadDB()
	if err != nil {
		return nil, err
	}

	newId := db.generateID(len(loadedDB.Chirps), "chirp")

	db.mux.Lock()
	chirp, err := unmarshalFunc([]byte(body))
	if err != nil {
		return nil, err
	}

	chirp.(*models.Chirp).AuthorId = authorId
	chirp.SetId(newId)
	db.mux.Unlock()

	return chirp, nil
}

func (db *DB) CreateUser(body string) (models.Storable, *models.UserResponse, error) {
	unmarshalFunc, ok := models.UnmarshalFunc["user"]
	if !ok {
		return nil, nil, errors.New("invalid type item")
	}

	newID := 0
	userResponse := models.UserResponse{}

	user, err := unmarshalFunc([]byte(body))
	if err != nil {
		return nil, nil, err
	}

	loadedDB, err := db.LoadDB()
	if err != nil {
		return nil, nil, err
	}

	newID = db.generateID(len(loadedDB.Users), "user")
	typedUser := user.(*models.User)
	db.mux.Lock()
	typedUser.SetHashPass(typedUser.Password)
	typedUser.GenerateRefreshToken()

	if typedUser.ExpiresInSeconds == 0 {
		typedUser.ExpiresInSeconds = 5184000
	}
	db.mux.Unlock()

	userResponse = models.UserResponse{
		Id:          newID,
		Email:       typedUser.Email,
		IsChirpyRed: typedUser.IsChirpyRed,
	}

	db.mux.Lock()
	user.SetId(newID)
	db.mux.Unlock()

	return user, &userResponse, nil
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
				v.IsChirpyRed = newItemWithType.IsChirpyRed

				if v.ExpiresInSeconds == 0 {
					v.ExpiresInSeconds = 5184000
				} else {
					v.ExpiresInSeconds = newItemWithType.ExpiresInSeconds
				}
				db.mux.Unlock()

				userResponse = models.UserResponse{
					Id:          id,
					Email:       newItemWithType.Email,
					IsChirpyRed: newItemWithType.IsChirpyRed,
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
	allItems, err := db.LoadDB()
	if err != nil {
		fmt.Errorf("Problem with downloading database %e", err)
	}

	newID := 1

	if typeId == "chirp" {
		allChirps := allItems.Chirps

		for chirp := range allChirps {
			if newID == chirp {
				newID += 1
			}
		}
	} else if typeId == "user" {
		allChirps := allItems.Users

		for user := range allChirps {
			if newID == user {
				newID += 1
			}
		}
	}

	return newID
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

func (db *DB) DeleteItem(id int, itemType string) DBStructure {
	items, err := db.LoadDB()
	if err != nil {
		fmt.Println(err)
	}

	if itemType == "users" {
		for key := range items.Users {
			if key == id {
				delete(items.Users, key)
			}
		}
	} else if itemType == "chirps" {
		for key := range items.Chirps {
			if key == id {
				delete(items.Chirps, key)
			}
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
