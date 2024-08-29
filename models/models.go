package models

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type Storable interface {
	SetId(int)
	GetId() int
}

type Chirp struct {
	Id       int    `json:"id"`
	Body     string `json:"body"`
	AuthorId int    `json:"author_id"`
}

func (c *Chirp) SetId(id int) {
	c.Id = id
}

func (c *Chirp) GetId() int {
	return c.Id
}

type User struct {
	Id               int    `json:"id"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
	RefreshToken     string `json:"refresh_token"`
	IsChirpyRed      bool   `json:"is_chirpy_red"`
}

type UserResponse struct {
	Id          int    `json:"id"`
	Email       string `json:"email"`
	IsChirpyRed bool   `json:"is_chirpy_red"`
}

type APIUserResponse struct {
	Id           int    `json:"id"`
	Email        string `json:"email"`
	Token        string `json:"token"`
	RefreshToken string `json:"refresh_token"`
	IsChirpyRed  bool   `json:"is_chirpy_red"`
}

type TokenResponse struct {
	Token string `json:"token"`
}

func (u *User) SetId(id int) {
	u.Id = id
}

func (u *User) GetId() int {
	return u.Id
}

func (u *User) SetHashPass(pass string) {
	if len(pass) == 60 {
		u.Password = pass
	} else {
		hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
		if err != nil {
			fmt.Println(err)
		}
		u.Password = string(hashedPassword)
	}
}

func (u *User) GenerateRefreshToken() {
	randomBytes := make([]byte, 32)

	_, err := rand.Read(randomBytes)
	if err != nil {
		fmt.Println(err)
	}

	u.RefreshToken = hex.EncodeToString(randomBytes)
}

type Webhooks struct {
	Event string `json:"event"`
	Data  Data   `json:"data"`
}
type Data struct {
	UserID int `json:"user_id"`
}

var UnmarshalFunc = map[string]func([]byte) (Storable, error){
	"chirp": func(data []byte) (Storable, error) {
		var chirp Chirp
		if err := json.Unmarshal(data, &chirp); err != nil {
			return nil, err
		}
		return &chirp, nil
	},
	"user": func(data []byte) (Storable, error) {
		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			return nil, err
		}
		return &user, nil
	},
}
