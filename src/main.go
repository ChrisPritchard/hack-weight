package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
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

	openingMessage := fmt.Sprintf("Application started! Listening locally at port %s", config.ListenURL)
	if config.IsDevelopment {
		openingMessage += " and running in DEVELOPMENT mode"
	}
	log.Println(openingMessage)
	log.Println(http.ListenAndServe(config.ListenURL, http.DefaultServeMux))
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

	http.HandleFunc("/today/weight", weightHandler)
	http.HandleFunc("/today/calories", caloriesHandler)
	http.HandleFunc("/today", todayHandler)
	http.HandleFunc("/goals", goalsHandler)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "hello world")
}

func weightHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	formValue := r.FormValue("weight")
	if formValue == "" {
		http.Error(w, "bad request", 400)
		return
	}

	val, err := strconv.ParseFloat(formValue, 32)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	err = addWeightEntry(float32(val))
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func caloriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "hello world")
}

func todayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "hello world")
}

func goalsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}
	fmt.Fprintln(w, "hello world")
}
