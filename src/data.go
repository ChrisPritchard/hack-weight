package main

import (
	"database/sql"
	"sort"
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
	var passwordHash string

	row := database.QueryRow("SELECT password FROM users WHERE username = ?", user)
	err := row.Scan(&passwordHash)

	if err == sql.ErrNoRows {
		return false, nil
	} else if err != nil {
		return false, err
	} else {
		return compareWithArgonHash(pass, passwordHash)
	}
}

func getSettings(username string) (map[string]string, error) {
	rows, err := database.Query("SELECT setting_key, setting_value FROM settings WHERE username = ?", username)
	defer rows.Close()
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

func setSetting(key, val, username string) error {
	res, err := database.Exec("UPDATE settings SET setting_value = ? WHERE setting_key = ? AND username = ?", val, key, username)
	if err != nil {
		return err
	}
	rows, err := res.RowsAffected()
	if err != nil || rows == 1 {
		return err
	}
	_, err = database.Exec("INSERT INTO settings (setting_key, setting_value, username) VALUES (?, ?, ?)", key, val, username)
	return err
}

type goals struct {
	TargetWeight float64
	TargetDate   string
	BurnRate     int
}

func getGoals(username string) (*goals, error) {
	settings, err := getSettings(username)
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

func addWeightEntry(val float64, username string) error {
	date := time.Now().Format(time.RFC3339)
	_, err := database.Exec("INSERT INTO weight_entry (date, weight, username) VALUES (?, ?, ?)", date, val, username)
	return err
}

func getCalorieCategories(username string) ([]string, error) {
	rows, err := database.Query("SELECT DISTINCT category FROM calorie_entry WHERE username = ?", username)
	defer rows.Close()
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

func addCalorieEntry(amount int, category, username string) error {
	date := time.Now().Format(time.RFC3339)
	_, err := database.Exec("INSERT INTO calorie_entry (date, amount, category, username) VALUES (?, ?, ?, ?)", date, amount, category, username)
	return err
}

func deleteCalorieEntry(id int, username string) error {
	_, err := database.Exec("DELETE FROM calorie_entry WHERE Id = ? AND username = ?", id, username)
	return err
}

func getDayStartAndEnd(day time.Time) (string, string) {
	y, m, d := day.Date()
	start := time.Date(y, m, d, 0, 0, 0, 0, day.Location())
	endParam := start.Add(24 * time.Hour).Format(time.RFC3339)
	return start.Format(time.RFC3339), endParam
}

func getDayWeight(day time.Time, username string) (float64, error) {
	start, end := getDayStartAndEnd(day)
	var todaysWeight float64

	row := database.QueryRow("SELECT weight	FROM weight_entry WHERE	date >= ? AND date <= ?	AND username = ? ORDER BY date DESC	LIMIT 1", start, end, username)
	err := row.Scan(&todaysWeight)

	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		return todaysWeight, nil
	}
}

func getLatestWeight(username string) (float64, error) {
	var lastWeight float64

	row := database.QueryRow("SELECT weight	FROM weight_entry WHERE username = ? ORDER BY date DESC LIMIT 1", username)
	err := row.Scan(&lastWeight)

	if err == sql.ErrNoRows {
		return 0, nil
	} else if err != nil {
		return 0, err
	} else {
		return lastWeight, nil
	}
}

type calorieEntry struct {
	ID       int
	Amount   int
	Category string
}

func getDayCalories(day time.Time, username string) ([]calorieEntry, error) {
	start, end := getDayStartAndEnd(day)

	rows, err := database.Query("SELECT id, amount, category FROM calorie_entry WHERE date >= ? AND date <= ? AND username = ?", start, end, username)
	defer rows.Close()
	if err != nil {
		return nil, err
	}

	result := make([]calorieEntry, 0)
	for rows.Next() {
		var row calorieEntry
		err = rows.Scan(&row.ID, &row.Amount, &row.Category)
		if err != nil {
			return nil, err
		}
		result = append(result, row)
	}

	return result, nil
}

func clearAllEntries(username string) error {
	_, err := database.Exec("delete from settings WHERE username = ?", username)
	if err != nil {
		return err
	}
	_, err = database.Exec("delete from weight_entry WHERE username = ?", username)
	if err != nil {
		return err
	}
	_, err = database.Exec("delete from calorie_entry WHERE username = ?", username)
	if err != nil {
		return err
	}
	return nil
}

type recordedDay struct {
	date    string
	weight  float64
	entries []calorieEntry
	total   int
}

func allDaysForUser(username string) ([]recordedDay, error) {
	weightRows, err := database.Query("SELECT weight, date FROM weight_entry WHERE username = ? ORDER BY date", username)
	defer weightRows.Close()
	if err != nil {
		return nil, err
	}

	days := make(map[string]recordedDay)
	for weightRows.Next() {
		var weight float64
		var date string
		err = weightRows.Scan(&weight, &date)
		if err != nil {
			return nil, err
		}

		dateVal, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, err
		}

		start, _ := getDayStartAndEnd(dateVal)
		days[start] = recordedDay{start, weight, []calorieEntry{}, 0}
	}

	caloryRows, err := database.Query("SELECT id, amount, category, date FROM calorie_entry WHERE username = ? ORDER BY date", username)
	defer caloryRows.Close()
	if err != nil {
		return nil, err
	}

	for caloryRows.Next() {
		var id int
		var amount int
		var category, date string
		err = caloryRows.Scan(&id, &amount, &category, &date)
		if err != nil {
			return nil, err
		}

		dateVal, err := time.Parse(time.RFC3339, date)
		if err != nil {
			return nil, err
		}

		start, _ := getDayStartAndEnd(dateVal)
		entry, exists := days[start]
		if !exists {
			continue
		}
		entry.entries = append(entry.entries, calorieEntry{id, amount, category})
		days[start] = entry
	}

	return sortMap(days), nil
}

func sortMap(m map[string]recordedDay) []recordedDay {
	keys := make([]string, len(m))
	i := 0
	for k := range m {
		keys[i] = k
		i++
	}
	sort.Strings(keys)
	result := make([]recordedDay, len(m))
	j := 0
	for _, k := range keys {
		result[j] = m[k]
		j++
	}
	return result
}
