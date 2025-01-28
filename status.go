package main

import (
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math/big"
	"net/http"
	"os"
	"os/user"
	"time"

	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

type response struct {
	Msg       string `json:"msg"`
	Timestamp int64  `json:"timestamp"`
}

type user_info struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Status   string `json:"status"`
}

const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*()_+"

func generateSecurePassword(length int32) (string, error) {
	password := make([]byte, length)
	charsetLength := big.NewInt(int64(len(charset)))
	for i := range password {
		index, err := rand.Int(rand.Reader, charsetLength)
		if err != nil {
			return "", fmt.Errorf("error generating random index: %v", err)
		}
		password[i] = charset[index.Int64()]
	}

	return string(password), nil
}

func main() {
	var msg string = ""
	var timestamp int64 = time.Now().Unix()
	var userHash []byte

	user, err := user.Current()
	if err != nil {
		log.Fatal(err.Error())
	}

	db, err := sql.Open("sqlite3", user.HomeDir+"/.status/auth.db")
	if err != nil {
		log.Fatal(err.Error())
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS users (username VARCHAR(255) NOT NULL PRIMARY KEY, hash TEXT, status TEXT, timestamp INTEGER)")
	if err != nil {
		log.Fatal(err.Error())
	}

	stmt, _ := db.Prepare("INSERT INTO users(username, hash, status, timestamp) values(?, ?, '', 0)")
	defer stmt.Close()

	update, _ := db.Prepare("UPDATE users SET status = ?,timestamp = ? WHERE username = ?")

	rows, _ := db.Query("SELECT * FROM users WHERE username='admin'")
	defer rows.Close()

	var username string
	var hash string
	var status string
	if !rows.Next() {
		log.Println("Could not find admin user, generating...")

		password, _ := generateSecurePassword(16)
		hash, _ := bcrypt.GenerateFromPassword([]byte(password), 10)
		stmt.Exec("admin", hash)

		//      															bold escape sequence 																	reset escape sequence
		log.Println("Admin user generated. \033[1mSAVE THE INFO BELOW! IT CAN NOT BE RECOVERED!!\033[0m")
		log.Print("username: admin\n")
		log.Printf("password: %s", password)

		rows, _ = db.Query("SELECT * FROM users WHERE username='admin'")
		rows.Next()
	}
	err = rows.Scan(&username, &hash, &status, &timestamp)
	rows.Next()
	if err != nil {
		log.Fatal("Error while finding admin user\n", err.Error())
	}

	userStmt, _ := db.Prepare("SELECT * FROM users WHERE username = ?")
	defer userStmt.Close()

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			err := userStmt.QueryRow(username).Scan(&username, &hash, &status, &timestamp)
			if err != nil {
				deny(w)
				log.Fatal(err.Error())
				return
			}

			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
			if err != nil {
				deny(w)
				return
			}

			res := response{
				Msg:       status,
				Timestamp: timestamp,
			}

			json, err := json.Marshal(res)
			if err != nil {
				http.Error(w, "Failed create JSON body", http.StatusInternalServerError)
				return
			}

			w.Write(json)
			return
		}

		deny(w)
	})

	http.HandleFunc("/raw", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			err := bcrypt.CompareHashAndPassword(userHash, []byte(username+password))
			if err != nil {
				deny(w)
				return
			}

			w.Write([]byte(msg))
			return
		}

		deny(w)
	})

	http.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok && r.Method == "POST" {
			err := userStmt.QueryRow(username).Scan(&username, &hash, &status, &timestamp)
			if err != nil {
				deny(w)
				return
			}

			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
			if err != nil {
				deny(w)
				return
			}

			buf, err := io.ReadAll(r.Body)
			if err != nil {
				http.Error(w, "Failed to read request body", http.StatusInternalServerError)
				return
			}

			timestamp = time.Now().Unix()
			update.Exec(string(buf), timestamp, username)

			w.Write([]byte("success"))
			return
		}

		deny(w)
	})

	http.HandleFunc("/users", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()

		if ok && r.Method == "POST" {
			decoder := json.NewDecoder(r.Body)
			var userInfo user_info
			err := decoder.Decode(&userInfo)

			if err != nil {
				http.Error(w, "Bad request", http.StatusBadRequest)
				return
			}

			var hash string = ""
			err = userStmt.QueryRow("admin").Scan(&username, &hash, &status, &timestamp)
			if err != nil {
				http.Error(w, "Failed to retrieve admin user", http.StatusInternalServerError)
				return
			}

			err = bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
			if err != nil {
				http.Error(w, "Invalid admin credentials", http.StatusForbidden)
				return
			}

			passHash, err := bcrypt.GenerateFromPassword([]byte(userInfo.Password), 10)
			if err != nil {
				http.Error(w, "Unexpected server error", http.StatusInternalServerError)
				return
			}

			_, err = stmt.Exec(userInfo.Username, passHash)
			if err != nil {
				http.Error(w, "User already exists", http.StatusBadRequest)
				return
			}

			w.Write([]byte("User successfully created"))
			return
		}

		deny(w)
	})

	if len(os.Args) != 2 {
		log.Fatal("Usage: status <port>")
		return
	}
	log.Fatal(http.ListenAndServe(":"+os.Args[1], nil))
}

func deny(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
