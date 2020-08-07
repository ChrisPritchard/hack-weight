package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"time"
)

type siteConfig struct {
	DatabasePath string
	ListenURL    string
}

var config = siteConfig{}

var currentUser = ""

func main() {

	loadConfig() // load settings from ./config.json and setup oauth config

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

		currentUser = user

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

		h.ServeHTTP(w, r)
	})
}

func setupRoutes() {
	http.HandleFunc("/", indexHandler) // note: this will catch any request not caught by the others
	http.Handle("/static/", runtimeStaticHandler())

	http.HandleFunc("/today/weight", weightHandler)
	http.HandleFunc("/today/calories", caloriesHandler)
	http.HandleFunc("/today", todayHandler)
	http.HandleFunc("/categories", categoriesHandler)
	http.HandleFunc("/goals", goalsHandler)
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

	var lastWeight float32
	if weight == 0 {
		lastWeight, err = getLatestWeight()
		if err != nil {
			log.Println("ERROR: " + err.Error())
			http.Error(w, "server error", 500)
			return
		}
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
			Weight     float32
			LastWeight float32
			Calories   []calorieEntry
		}{weight, lastWeight, calories}
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

	var targetWeight float64
	weightVal, exists := settings["target_weight"]
	if exists {
		targetWeight, _ = strconv.ParseFloat(weightVal, 32)
	}

	date, _ := settings["target_date"]

	burnRate := 0
	burnRateVal, exists := settings["burn_rate"]
	if exists {
		burnRate, _ = strconv.Atoi(burnRateVal)
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		result := struct {
			TargetWeight float64
			TargetDate   string
			BurnRate     int
		}{targetWeight, date, burnRate}
		json.NewEncoder(w).Encode(result)
	} else {
		fmt.Fprintln(w, targetWeight)
		fmt.Fprintln(w, date)
		fmt.Fprintln(w, burnRate)
	}
}
