package main

import (
	"database/sql"
)

// ///////////////////////////////////////
// ///////// DATABASE STUFF? ///////////
// ///////////////////////////////////////
type DBConnection struct {
	DB   *sql.DB
	path string
}

var sqlOpen = sql.Open

var pragmaExec = func(db *sql.DB, query string) (sql.Result, error) {
	return db.Exec(query)
}

// ///////////////////////////////////////
// ///////// MAIN ? ///////////
// ///////////////////////////////////////
func main() {
	dbPath := "./mock.db"

	db, _ := sqlOpen("sqlite3", dbPath)

	pragmaExec(db, "PRAGMA foreign_keys = ON")

	database := &DBConnection{DB: db, path: dbPath}

	repos := NewRepositories(database.DB)

}
