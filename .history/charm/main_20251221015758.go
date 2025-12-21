package main

import (
	"database/sql"
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
	db, _ := sqlOpen("sqlite3", dbPath)

	pragmaExec(db, "PRAGMA foreign_keys = ON")

	// what do with db? someMethod?
	db.someMethod
}
