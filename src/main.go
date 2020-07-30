package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
)

type siteConfig struct {
	DatabasePath  string
	ListenURL     string
	VerboseErrors bool
	IsDevelopment bool
}

var config = siteConfig{}

func main() {
	loadConfig()  // load settings from ./config.json and setup oauth config
	setupRoutes() // configure handlers for url fragments

	server := globalHandler(http.DefaultServeMux)

	openingMessage := fmt.Sprintf("Application started! Listening locally at port %s", config.ListenURL)
	if config.IsDevelopment {
		openingMessage += " and running in DEVELOPMENT mode"
	}
	log.Println(openingMessage)
	log.Println(http.ListenAndServe(config.ListenURL, server))
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

func setupRoutes() {
	http.HandleFunc("/", indexHandler) // note: this will catch any request not caught by the others
}

func globalHandler(h http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

		// set security headers
		headers := w.Header()
		headers.Set("X-Frame-Options", "SAMEORIGIN")
		headers.Set("X-XSS-Protection", "1; mode=block")
		headers.Set("X-Content-Type-Options", "nosniff")

		csp := "default-src 'none';"
		headers.Set("Content-Security-Policy", csp)

		h.ServeHTTP(w, r)
	})
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "hello world")
}
