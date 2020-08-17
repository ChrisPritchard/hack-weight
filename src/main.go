package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

type siteConfig struct {
	DatabasePath string
	ListenURL    string
}

var config = siteConfig{}
var database *sql.DB

var authenticatedUser = struct{}{}

func main() {

	loadConfig() // load settings from ./config.json and setup oauth config

	db, err := sql.Open("sqlite3", config.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	database = db

	if len(os.Args) == 4 && os.Args[1] == "--create-user" {
		err := insertOrUpdateUser(os.Args[2], os.Args[3])
		if err != nil {
			log.Fatal(err)
		}
		fmt.Println("user created or updated successfully")
		return
	}

	setupRoutes() // configure handlers for url fragments

	openingMessage := fmt.Sprintf("Application started! Listening locally at port %s", config.ListenURL)
	log.Println(openingMessage)
	log.Println(http.ListenAndServe(config.ListenURL, globalHandler(http.DefaultServeMux)))
}

func loadConfig() {
	configJSON, err := ioutil.ReadFile("./config.json")
	if err != nil {
		log.Fatal(err)
	}
	err = json.Unmarshal(configJSON, &config)
	if err != nil {
		log.Fatal(err)
	}

	verificationErrors := ""
	if _, err := os.Stat(config.DatabasePath); os.IsNotExist(err) {
		verificationErrors += fmt.Sprintf("database file not found at path '%s'", config.DatabasePath)
	}

	if verificationErrors != "" {
		log.Fatal(verificationErrors)
	}
}

func globalHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		user, pass, ok := r.BasicAuth()
		valid, err := testAuthAgainstDB(user, pass)
		if err != nil {
			log.Fatal(err)
		}

		if !ok || !valid {
			w.Header().Set("WWW-Authenticate", `Basic realm="Hack Weight Authentication"`)
			w.WriteHeader(401)
			w.Write([]byte("Unauthorised.\n"))
			return
		}

		userCtx := context.WithValue(r.Context(), authenticatedUser, user)

		headers := w.Header()
		headers.Set("X-Frame-Options", "SAMEORIGIN")
		headers.Set("X-XSS-Protection", "1; mode=block")
		headers.Set("X-Content-Type-Options", "nosniff")

		csp := "default-src 'none';"
		csp += "script-src 'self' https://use.fontawesome.com;"
		csp += "style-src 'self' https://use.fontawesome.com;"
		csp += "font-src 'self' https://use.fontawesome.com;"
		csp += "connect-src 'self';"
		csp += "img-src 'self';"
		csp += "frame-src 'self';"
		headers.Set("Content-Security-Policy", csp)

		h.ServeHTTP(w, r.WithContext(userCtx))
	})
}

func setupRoutes() {
	http.HandleFunc("/", indexHandler) // note: this will catch any request not caught by the others
	http.Handle("/static/", runtimeStaticHandler())

	http.HandleFunc("/today/weight", weightHandler)
	http.HandleFunc("/today/calories", caloriesHandler)
	http.HandleFunc("/calories/delete", deleteEntryHandler)
	http.HandleFunc("/today", todayHandler)
	http.HandleFunc("/categories", categoriesHandler)
	http.HandleFunc("/goals", goalsHandler)
	http.HandleFunc("/history/clear", clearAllEntriesHandler)
}

func runtimeStaticHandler() http.Handler {
	server := http.FileServer(http.Dir("static"))
	fileHandler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		setMimeType(w, r)
		server.ServeHTTP(w, r)
	})

	return http.StripPrefix("/static/", fileHandler)
}

func setMimeType(w http.ResponseWriter, r *http.Request) {
	headers := w.Header()
	ext := filepath.Ext(r.URL.Path)

	switch ext {
	case ".css":
		headers.Set("Content-Type", "text/css")
	case ".js":
		headers.Set("Content-Type", "application/javascript")
	case ".png":
		headers.Set("Content-Type", "image/png")
	default:
		return
	}
}
