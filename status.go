package main

import (
	"bufio"
	"io"
	"log"
	"net/http"
	"os"
	"os/user"
	"strings"

	"golang.org/x/crypto/bcrypt"
)

func main() {
	var msg string = ""
	var adminHash []byte
	var userHash []byte

	user, err := user.Current()
	if err != nil {
		log.Fatal(err.Error())
	}

	file, err := os.Open(user.HomeDir + "/.status/auth")
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

				log.Print(string(buf))
				msg = string(buf)
				w.Write([]byte("success"))
				return
			}
		}

		deny(w)
	})

	log.Fatal(http.ListenAndServe(":8080", nil))
}

func deny(w http.ResponseWriter) {
	w.Header().Set("WWW-Authenticate", `Basic realm="restricted", charset="UTF-8"`)
	http.Error(w, "Unauthorized", http.StatusUnauthorized)
}
