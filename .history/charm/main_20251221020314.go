package main

import (
	"database/sql"
)

type DBConnection struct {
	DB   *sql.DB
	path string
}

var sqlOpen = sql.Open

var pragmaExec = func(db *sql.DB, query string) (sql.Result, error) {
	return db.Exec(query)
}

var createMigrationRunner = CreateMigrationRunner

func CreateMigrationRunner() {}

func main() {
	dbPath := "./mock.db"

	db, _ := sqlOpen("sqlite3", dbPath)

	pragmaExec(db, "PRAGMA foreign_keys = ON")

	database := &DBConnection{DB: db, path: dbPath}

	createMigrationRunner(db, migrationFiles)
}
