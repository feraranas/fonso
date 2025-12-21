package main

import "fmt"

type db struct {
	connection string
}

func (db) sqlOpen(database string, dbPath string) {}

func main() {
	dbPath := "some"
	// sqlOpen?
	db, err := sqlOpen("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// what do with db? someMethod?
	db.someMethod
}
