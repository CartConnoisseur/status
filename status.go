package main

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

type response struct {
	Msg       string `json:"msg"`
	Timestamp int64  `json:"timestamp"`
}

func main() {
	var msg string = ""
	var timestamp int64 = 0
	var adminHash []byte
	var userHash []byte

	user, err := user.Current()
	if err != nil {
		log.Fatal(err.Error())
	}

	var path string = user.HomeDir + "/.status/auth"
	if len(os.Args) > 2 {
		path = os.Args[2]
	}

	file, err := os.Open(path)
	if err != nil {
		log.Fatal(err.Error())
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()

		if after, found := strings.CutPrefix(line, "admin:"); found {
			adminHash = []byte(after)
		}

		if after, found := strings.CutPrefix(line, "user:"); found {
			userHash = []byte(after)
		}
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			err := bcrypt.CompareHashAndPassword(userHash, []byte(username+password))
			if err != nil {
				deny(w)
				return
			}

			res := response{
				Msg:       msg,
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

	http.HandleFunc("/update", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			err := bcrypt.CompareHashAndPassword(adminHash, []byte(username+password))
			if err != nil {
				deny(w)
				return
			}

			if r.Method == "POST" {
				buf, err := io.ReadAll(r.Body)
				if err != nil {
					http.Error(w, "Failed to read request body", http.StatusInternalServerError)
					return
				}

				msg = string(buf)
				timestamp = time.Now().Unix()

				log.Print(string(buf))
				w.Write([]byte("success"))
				return
			}
		}

		deny(w)
	})

	http.HandleFunc("/generate-hash", func(w http.ResponseWriter, r *http.Request) {
		username, password, ok := r.BasicAuth()
		if ok {
			hash, err := bcrypt.GenerateFromPassword([]byte(username+password), 0)
			if err != nil {
				http.Error(w, "Failed to generate hash", http.StatusInternalServerError)
				return
			}

			log.Print(string(hash))
			w.Write([]byte("Hash successfully generated (output to server log for some semblance of security)"))
			return
		}

		deny(w)
	})

	log.Fatal(http.ListenAndServe(":"+os.Args[1], nil))
}

func deny(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
