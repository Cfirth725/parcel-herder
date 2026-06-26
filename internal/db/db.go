package db

import (
	"database/sql"
	"log"
	"os"

	_ "github.com/mattn/go-sqlite3"
)

func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("FAILURE: Failed to connect to database: %v", err)
	}

	// Pragmas for local home-server write optimization
	_, err = db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`)
	if err != nil {
		log.Fatalf("FAILURE: Failed to set SQLite pragmas: %v", err)
	}

	executeSchema(db, "schema.sql")
	return db
}

func executeSchema(db *sql.DB, schemaFile string) {
	statements, err := os.ReadFile(schemaFile)
	if err != nil {
		log.Fatalf("FAILURE: Failed to read database schema file: %v", err)
	}

	_, err = db.Exec(string(statements))
	if err != nil {
		log.Fatalf("ERROR: Error executing schema initialization script: %v", err)
	}

	log.Println("Database architecture initialized smoothly via external schema file.")
}
