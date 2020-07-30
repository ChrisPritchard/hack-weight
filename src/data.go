package main

import (
	"database/sql"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

func getSettings() (map[string]string, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return nil, err
	}

	rows, err := database.Query("SELECT setting_key, setting_value FROM settings")
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

	res, err := database.Exec(`
		UPDATE 
			settings
		SET
			settings_value = ?
		WHERE
			settings_key = ?`, val, key)

	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 1 {
		return err
	}

	_, err = database.Exec(`
		INSERT INTO 
			settings (setting_key, setting_value) 
		VALUES 
			(?, ?)`, key, val)
	return err
}

func addWeightEntry(val float32) error {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return err
	}

	date := time.Now().Format(time.RFC3339)

	_, err = database.Exec(`
		INSERT INTO 
			weight_entry (date, weight) 
		VALUES 
			(?, ?)`, date, val)
	return err
}

func getCalorieCategories() ([]string, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return nil, err
	}

	rows, err := database.Query("SELECT DISTINCT category FROM calorie_entry")
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

	_, err = database.Exec(`
		INSERT INTO 
			calorie_entry (date, amount, category) 
		VALUES 
			(?, ?, ?)`, date, amount, category)
	return err
}

func getDayStartAndEnd(day time.Time) (string, string) {
	start := day.Truncate(24 * time.Hour)
	endParam := start.Add(24 * time.Hour).Format(time.RFC3339)
	return start.Format(time.RFC3339), endParam
}

func getDayWeight(day time.Time) (float32, error) {
	database, err := sql.Open("sqlite3", config.DatabasePath)
	defer database.Close()
	if err != nil {
		return 0, err
	}

	start, end := getDayStartAndEnd(day)
	var todaysWeight float32

	row := database.QueryRow(`
		SELECT 
			weight
		FROM 
			weight_entry 
		WHERE 
			date >= ? 
		AND 
			date <= ?
		ORDER BY
			date DESC
		LIMIT 1`, start, end)
	err = row.Scan(&todaysWeight)

	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		return todaysWeight, nil
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

	rows, err := database.Query(`
		SELECT 
			amount, category 
		FROM 
			calorie_entry 
		WHERE 
			date >= ? 
		AND 
			date <= ?`, start, end)
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
