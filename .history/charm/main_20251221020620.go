package main

import (
	"database/sql"
	"embed"
)

// ///////////////////////////////////////
// ///////// MIGRATION RUNNER? ///////////
// ///////////////////////////////////////
type DBConnection struct {
	DB   *sql.DB
	path string
}

var sqlOpen = sql.Open

var pragmaExec = func(db *sql.DB, query string) (sql.Result, error) {
	return db.Exec(query)
}

/////////////////////////////////////////
/////////// MIGRATION RUNNER? ///////////
/////////////////////////////////////////

//go:embed sql/migrations
var migrationFiles embed.FS

type MigrationRunner struct {
}

var createMigrationRunner = CreateMigrationRunner

func CreateMigrationRunner() *MigrationRunner {
	migrationRunner := &MigrationRunner{}
	return migrationRunner
}

/////////////////////////////////////////
/////////////////////////////////////////
/////////////////////////////////////////

func main() {
	dbPath := "./mock.db"

	db, _ := sqlOpen("sqlite3", dbPath)

	pragmaExec(db, "PRAGMA foreign_keys = ON")

	database := &DBConnection{DB: db, path: dbPath}

	runner := createMigrationRunner(db, migrationFiles)

}
