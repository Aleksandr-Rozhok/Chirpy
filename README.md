# 🦤 Chirpy

This is a training web server for the fictional social network Chirps. Here are presented and implemented basic aspects of creating a web server and API

---

## 🎯 Goal

Learn to write a web server using best practices. Also improve your GoLang language skills

---

## ⚙️ Installation

First move the repository to a local directory. You can do this using the command:
```
git clone https://github.com/Aleksandr-Rozhok/Chirpy
```

---

## 🚀 Quick Start Consumer

Inside the local repository you can use the following command to start a web server
```
go run . --debug
```

Next, by clicking on the link http://localhost:8080/app/ you can make sure that the server is working and the data is displayed

## API For Chirps 🔨

### Users resource 🧍

```json
{
  "id": 1,
  "email": "user@example.com",
  "password": "12345abc",
  "is_chirpy_red": false
}
```

#### POST /api/users

Add new user to database

##### Response body

```json
[
  {
  "email": "walt@breakingbad.com",
  "id": 1,
  "is_chirpy_red": false
 },
  {
    "email": "jessy@breakingbad.com",
    "id": 2,
    "is_chirpy_red": true
  }
]
```

#### PUT /api/users/

Change user information into database

##### Response body

```json
 {
  "email": "alex@breakingbad.com",
  "id": 1,
  "is_chirpy_red": false
 }
```

#### POST /api/login
Check user's email, password and jwt token

##### Response body

```json
{
  "email": "walt@breakingbad.com",
  "id": 1,
  "is_chirpy_red": false,
  "refresh_token": "cabae460b93f33805c11da44ea6d24a931058ea9bd717c489bd35232a0493f30",
  "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIxIiwiZXhwIjoxNzI1MDE4MzIxfQ.TYjtpS0rYZJaIYuF-wMKtyn0oURTM76X8QLcF3o48LU"
}
```

### Info resource 📄

#### GET /admin/metrics

Give information about visitor of website

#### GET /api/reset

Reset information about visitors

### Chirps resource 🦤


```json
{
  "id": 1,
  "body": "Hello, this is my first chirp!",
  "author_id": 1
}
```

#### GET /api/chirps

Return slice of chirps

##### Response body

```json
 [
  {
    "id": 1,
    "body": "Hello, this is my first chirp!",
    "author_id": 1
  },
  {
    "id": 2,
    "body": "This is my second chirp!",
    "author_id": 1
  },
  {
    "id": 3,
    "body": "Hello world!",
    "author_id": 2
  }
]
```

#### POST /api/chirps

Add new chirp into database

##### Response body

```json
{
    "id": 1,
    "body": "Hello world!",
    "author_id": 1
  }
```

#### GET /api/chirps/{chirpID}

Return chirp by id

#### DELETE /api/chirps/{chirpID}

Delete chirp from database by id

### Token Resource

### POST /api/refresh

Refresh JWT token for user

##### Response body

```json
{
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJpc3MiOiJjaGlycHkiLCJzdWIiOiIxIiwiZXhwIjoxNzI1MDE4MzIxfQ.TYjtpS0rYZJaIYuF-wMKtyn0oURTM76X8QLcF3o48LU"
  }
```

#### "POST /api/revoke"

Refresh long-term token for user

