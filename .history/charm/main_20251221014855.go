package main

import "fmt"

func sqlOpen(database string, dbPath string) {}

func main() {
	dbPath := "some"
	// sqlOpen?
	db, err := sqlOpen("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}
}
