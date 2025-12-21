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

func main() {
	dbPath := "./mock.db"
	// sqlOpen?
	db, err := sqlOpen("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// what do with db? someMethod?
	db.someMethod
}
