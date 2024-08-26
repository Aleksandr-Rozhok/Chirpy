package models

import "encoding/json"

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
	Id    int    `json:"id"`
	Email string `json:"email"`
}

func (u *User) SetId(id int) {
	u.Id = id
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

//var MarshalFunc = map[string]func([]byte) (Storable, error){
//	"chirp": func(data []byte) (Storable, error) {
//		var chirp Chirp
//		data, err := json.Marshal(&user)
//		if err != nil {
//			return err
//		}
//		return &chirp, nil
//	},
//	"user": func(data []byte) (Storable, error) {
//		var user User
//		if err := json.Marshal(data, &user); err != nil {
//			return nil, err
//		}
//		return &user, nil
//	},
//}
