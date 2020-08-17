package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"math"
	"net/http"
	"strconv"
	"time"
)

func currentUser(r *http.Request) string {
	return r.Context().Value(authenticatedUser).(string)
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

func todayHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	day := time.Now()
	currentUser := currentUser(r)

	weight, err := getDayWeight(day, currentUser)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	var lastWeight float64
	if weight == 0 {
		lastWeight, err = getLatestWeight(currentUser)
		if err != nil {
			log.Println("ERROR: " + err.Error())
			http.Error(w, "server error", 500)
			return
		}
	}

	calories, err := getDayCalories(day, currentUser)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	goals, err := getGoals(currentUser)
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	var todayMax *int
	if weight != 0 {
		todayMax = calcTodayMax(*goals, weight)
	} else if lastWeight != 0 {
		todayMax = calcTodayMax(*goals, lastWeight)
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		result := struct {
			Weight     float64
			LastWeight float64
			Calories   []calorieEntry
			TodayMax   *int
		}{weight, lastWeight, calories, todayMax}
		json.NewEncoder(w).Encode(result)
	} else {
		fmt.Fprintln(w, weight)
		for _, entry := range calories {
			fmt.Fprintf(w, "%d %s\n", entry.Amount, entry.Category)
		}
	}
}

func calcTodayMax(goals goals, currentWeight float64) *int {
	if goals.TargetDate == "" || goals.TargetWeight == 0 || goals.TargetWeight >= currentWeight || goals.BurnRate == 0 {
		return nil
	}
	date, err := time.Parse("2006-01-02", goals.TargetDate)
	if err != nil {
		return nil
	}
	days := date.Sub(time.Now()).Hours() / 24
	amount := (currentWeight - goals.TargetWeight) * 7700 // 7700 is cals per kg, roughly
	result := int(float64(goals.BurnRate) - (amount / days))
	if result < 0 {
		return nil
	}
	return &result
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

	rounded := math.Round(val*100) / 100

	err = addWeightEntry(rounded, currentUser(r))
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

	err = addCalorieEntry(int(calories), category, currentUser(r))
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func deleteEntryHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	formValue := r.FormValue("id")
	if formValue == "" {
		http.Error(w, "bad request", 400)
		return
	}

	id, err := strconv.Atoi(formValue)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	err = deleteCalorieEntry(id, currentUser(r))
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func categoriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	categories, err := getCalorieCategories(currentUser(r))
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
		setGoalsHandler(w, r)
	} else if r.Method == "GET" {
		getGoalsHandler(w, r)
	} else {
		http.NotFound(w, r)
	}
}

func setGoalsHandler(w http.ResponseWriter, r *http.Request) {
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

	_, err = time.Parse("2006-01-02", date)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	burnRate := r.FormValue("daily_burn_rate")
	if burnRate == "" {
		http.Error(w, "bad request", 400)
		return
	}

	_, err = strconv.Atoi(burnRate)
	if err != nil {
		http.Error(w, "bad request", 400)
		return
	}

	currentUser := currentUser(r)
	err = setSetting("target_weight", weight, currentUser)
	if err == nil {
		err = setSetting("target_date", date, currentUser)
		if err == nil {
			err = setSetting("daily_burn_rate", burnRate, currentUser)
		}
	}
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}

func getGoalsHandler(w http.ResponseWriter, r *http.Request) {
	goals, err := getGoals(currentUser(r))
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(goals)
	} else {
		fmt.Fprintln(w, goals.TargetWeight)
		fmt.Fprintln(w, goals.TargetDate)
		fmt.Fprintln(w, goals.BurnRate)
	}
}

func historyHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "GET" {
		http.NotFound(w, r)
		return
	}

	result, err := allDaysForUser(currentUser(r))
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	contentType := r.Header.Get("Content-type")
	if contentType == "application/json" {
		w.Header().Set("Content-Type", contentType)
		json.NewEncoder(w).Encode(result)
	} else {
		for _, day := range result {
			fmt.Fprintf(w, "%s %f\n", day.date, day.weight)
			for _, entry := range day.entries {
				fmt.Fprintf(w, "%d\t%s\n", entry.Amount, entry.Category)
			}
		}
	}
}

func clearAllEntriesHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.NotFound(w, r)
		return
	}

	err := clearAllEntries(currentUser(r))
	if err != nil {
		log.Println("ERROR: " + err.Error())
		http.Error(w, "server error", 500)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
