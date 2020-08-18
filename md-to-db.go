// This is not part of the main program, but rather a tool to convert
// a prior manual tracking format I was using into database queries for insertion.

// I used entries in a markdown table, like below:
//
//	|day|weight|breakfast|lunch|dinner|snacks|drinks|total|
//	|---|------|---------|-----|------|------|----------|-----|
//	|01/08/2020|98.4|200|100|600|0|0|900|
//	|02/08/2020|98.6|200|400|600|0|0|1200|
//
// This script, run with go run md-to-db-go <username> <raw markdown>, emits a series of insert queries
// that can then be run via the sqlite3 cli

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"strings"
)

func main() {
	log.SetFlags(0)
	log.SetOutput(os.Stdout)

	if len(os.Args) != 3 {
		log.Println("two args required: [username] [path to md file]")
		return
	}

	username := os.Args[1]
	file, err := os.Open(os.Args[2])
	if err != nil {
		log.Fatal(err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		if len(line) == 0 || line[0] != '|' {
			continue
		}
		cells := strings.FieldsFunc(line, func(c rune) bool { return c == '|' })
		if len(cells) != 8 {
			continue
		}
		dateParts := strings.Split(cells[0], "/")
		if len(dateParts) != 3 {
			continue // header or spacing row
		}

		date := fmt.Sprintf("%s-%s-%sT00:00:00+12:00", dateParts[2], dateParts[1], dateParts[0])
		log.Printf("INSERT INTO weight_entry (date, weight, username) VALUES (\"%s\", %s, \"%s\");\n", date, cells[1], username)

		addEntry := func(amount, category string) {
			if amount != "0" {
				log.Printf("INSERT INTO calorie_entry (date, amount, category, username) VALUES (\"%s\", %s, \"%s\", \"%s\");\n", date, amount, category, username)
			}
		}

		addEntry(cells[2], "Breakfast")
		addEntry(cells[3], "Lunch")
		addEntry(cells[4], "Dinner")
		addEntry(cells[5], "Snacks")
		addEntry(cells[6], "Drinks")
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}
}
