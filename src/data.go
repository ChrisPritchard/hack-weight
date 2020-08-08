package main

import (
	"database/sql"
	"strconv"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

var passwordConfig = &argon2Config{
	time:    1,
	memory:  64 * 1024,
	threads: 4,
	keyLen:  32,
}

func insertOrUpdateUser(user, pass string) error {
	passwordHash, err := generateArgonHash(passwordConfig, pass)
	if err != nil {
		return err
	}

	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return err
	}

	res, err := database.Exec("UPDATE users SET password = ? WHERE username = ?", passwordHash, user)

	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 1 {
		return err
	}

	_, err = database.Exec("INSERT INTO users (username, password) VALUES (?, ?)", user, passwordHash)
	return err
}

func testAuthAgainstDB(user, pass string) (bool, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return false, err
	}

	var passwordHash string

	row := database.QueryRow("SELECT password FROM users WHERE username = ?", user)
	err = row.Scan(&passwordHash)

	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return compareWithArgonHash(pass, passwordHash)
	}
}

func getSettings() (map[string]string, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return nil, err
	}

	rows, err := database.Query("SELECT setting_key, setting_value FROM settings WHERE username = ?", currentUser)
	if err != nil {
		return nil, err
	}

	result := make(map[string]string)
	for rows.Next() {
		var key, val string
		err = rows.Scan(&key, &val)
		if err != nil {
			return nil, err
		}
		result[key] = val
	}

	return result, nil
}

func setSetting(key, val string) error {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return err
	}

	res, err := database.Exec("UPDATE settings SET setting_value = ? WHERE setting_key = ? AND username = ?", val, key, currentUser)

	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 1 {
		return err
	}

	_, err = database.Exec("INSERT INTO settings (setting_key, setting_value, username) VALUES (?, ?, ?)", key, val, currentUser)
	return err
}

type goals struct {
	TargetWeight float64
	TargetDate   string
	BurnRate     int
}

func getGoals() (*goals, error) {
	settings, err := getSettings()
	if err != nil {
		return nil, err
	}

	var targetWeight float64
	weightVal, exists := settings["target_weight"]
	if exists {
		targetWeight, err = strconv.ParseFloat(weightVal, 32)
		if err != nil {
			return nil, err
		}
	}

	date, _ := settings["target_date"]

	burnRate := 0
	burnRateVal, exists := settings["daily_burn_rate"]
	if exists {
		burnRate, err = strconv.Atoi(burnRateVal)
		if err != nil {
			return nil, err
		}
	}

	return &goals{targetWeight, date, burnRate}, nil
}

func addWeightEntry(val float64) error {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return err
	}

	date := time.Now().Format(time.RFC3339)

	_, err = database.Exec("INSERT INTO weight_entry (date, weight, username) VALUES (?, ?, ?)", date, val, currentUser)
	return err
}

func getCalorieCategories() ([]string, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return nil, err
	}

	rows, err := database.Query("SELECT DISTINCT category FROM calorie_entry WHERE username = ?", currentUser)
	if err != nil {
		return nil, err
	}

	result := make([]string, 0)
	for rows.Next() {
		var val string
		err = rows.Scan(&val)
		if err != nil {
			return nil, err
		}
		result = append(result, val)
	}

	return result, nil
}

func addCalorieEntry(amount int, category string) error {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return err
	}

	date := time.Now().Format(time.RFC3339)

	_, err = database.Exec("INSERT INTO calorie_entry (date, amount, category, username) VALUES (?, ?, ?, ?)", date, amount, category, currentUser)
	return err
}

func getDayStartAndEnd(day time.Time) (string, string) {
	start := day.Truncate(24 * time.Hour)
	endParam := start.Add(24 * time.Hour).Format(time.RFC3339)
	return start.Format(time.RFC3339), endParam
}

func getDayWeight(day time.Time) (float64, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return 0, err
	}

	start, end := getDayStartAndEnd(day)
	var todaysWeight float64

	row := database.QueryRow("SELECT weight	FROM weight_entry WHERE	date >= ? AND date <= ?	AND username = ? ORDER BY date DESC	LIMIT 1", start, end, currentUser)
	err = row.Scan(&todaysWeight)

	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		return todaysWeight, nil
	}
}

func getLatestWeight() (float64, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return 0, err
	}

	var lastWeight float64

	row := database.QueryRow("SELECT weight	FROM weight_entry WHERE username = ? ORDER BY date DESC LIMIT 1", currentUser)
	err = row.Scan(&lastWeight)

	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		return lastWeight, nil
	}
}

type calorieEntry struct {
	Amount   int
	Category string
}

func getDayCalories(day time.Time) ([]calorieEntry, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return nil, err
	}

	start, end := getDayStartAndEnd(day)

	rows, err := database.Query("SELECT amount, category FROM calorie_entry WHERE date >= ? AND date <= ? AND username = ?", start, end, currentUser)
	if err != nil {
		return nil, err
	}

	result := make([]calorieEntry, 0)
	for rows.Next() {
		var row calorieEntry
		err = rows.Scan(&row.Amount, &row.Category)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, nil
}
