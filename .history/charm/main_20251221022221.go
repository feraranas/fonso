package main

import (
	"database/sql"
	"embed"
	"io/fs"
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

/////////////////////////////////////////
/////////// MIGRATION RUNNER? ///////////
/////////////////////////////////////////

//go:embed sql/migrations
var migrationFiles embed.FS

type FileSystem interface {
	ReadDir(name string) ([]fs.DirEntry, error)
	ReadFile(name string) ([]byte, error)
}

type MigrationRunner struct {
	db             *sql.DB
	migrationFiles FileSystem
	runFn          func() error
}

func CreateMigrationRunner() *MigrationRunner {
	migrationRunner := &MigrationRunner{}
	return migrationRunner
}

var createMigrationRunner = CreateMigrationRunner

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
