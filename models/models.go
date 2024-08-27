package models

import (
	"encoding/json"
	"fmt"
	"golang.org/x/crypto/bcrypt"
)

type Storable interface {
	SetId(int)
}

type Chirp struct {
	Id   int    `json:"id"`
	Body string `json:"body"`
}

func (c *Chirp) SetId(id int) {
	c.Id = id
}

type User struct {
	Id               int    `json:"id"`
	Email            string `json:"email"`
	Password         string `json:"password"`
	ExpiresInSeconds int    `json:"expires_in_seconds"`
}

type UserResponse struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
}

type APIUserResponse struct {
	Id    int    `json:"id"`
	Email string `json:"email"`
	Token string `json:"token"`
}

func (u *User) SetId(id int) {
	u.Id = id
}

func (u *User) SetHashPass(pass string) {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	if err != nil {
		fmt.Println(err)
	}
	u.Password = string(hashedPassword)
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
