package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"time"
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
	http.HandleFunc("/categories", categoriesHandler)
	http.HandleFunc("/goals", goalsHandler)
}

func indexHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}
	html, err := ioutil.ReadFile("./index.html")
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}
	w.Header().Set("Content-Type", "text/html")
	w.Write(html)
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

	formValue := r.FormValue("amount")
	if formValue == "" {
		http.Error(w, "bad request", 400)
		return
	}

	calories, err := strconv.Atoi(formValue)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	category := r.FormValue("category")

	err = addCalorieEntry(int(calories), category)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func todayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	day := time.Now()

	weight, err := getDayWeight(day)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	calories, err := getDayCalories(day)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		result := struct {
			Weight   float32
			Calories []calorieEntry
		}{weight, calories}
		json.NewEncoder(w).Encode(result)
	} else {
		fmt.Fprintln(w, weight)
		for _, entry := range calories {
			fmt.Fprintf(w, "%d %s\n", entry.Amount, entry.Category)
		}
	}
}

func categoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	categories, err := getCalorieCategories()
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(categories)
	} else {
		for _, category := range categories {
			fmt.Fprintln(w, category)
		}
	}
}

func goalsHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method == "POST" {
		setGoals(w, r)
	} else if r.Method == "GET" {
		getGoals(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func setGoals(w http.ResponseWriter, r *http.Request) {
	weight := r.FormValue("target_weight")
	if weight == "" {
		http.Error(w, "bad request", 400)
		return
	}

	_, err := strconv.ParseFloat(weight, 32)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	date := r.FormValue("target_date")
	if date == "" {
		http.Error(w, "bad request", 400)
		return
	}

	_, err = time.Parse(time.RFC3339, date)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	err = setSetting("target_weight", weight)
	if err == nil {
		err = setSetting("target_date", date)
	}
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func getGoals(w http.ResponseWriter, r *http.Request) {
	settings, err := getSettings()
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	weight, exists := settings["target_weight"]
	if !exists {
		http.NotFound(w, r)
		return
	}

	date, exists := settings["target_date"]
	if !exists {
		http.NotFound(w, r)
		return
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		result := struct{ TargetWeight, TargetDate string }{weight, date}
		json.NewEncoder(w).Encode(result)
	} else {
		fmt.Fprintln(w, weight)
		fmt.Fprintln(w, date)
	}
}
