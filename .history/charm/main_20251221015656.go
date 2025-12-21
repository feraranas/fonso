package main

import (
	"database/sql"
	"fmt"
)

type db struct {
	connection string
}

// wtf?
var sqlOpen = sql.Open
var pragmaExec = func(db *sql.DB, query string) (sql.Result, error) {
	return db.Exec(query)
}

func main() {
	dbPath := "./mock.db"
	// sqlOpen?
	db, err := sqlOpen("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	pragmaExec(db, "PRAGMA foreign_keys = ON")

	// what do with db? someMethod?
	db.someMethod
}
