package db

import (
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"log"
	"os"
	"strings"

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

	executeSchema(db, "internal/db/schema.sql")
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

// hashEmail normalizes and hashes the email address so raw text never hits the disk.
func hashEmail(email string) string {
	cleanEmail := strings.TrimSpace(strings.ToLower(email))
	hasher := sha256.New()
	hasher.Write([]byte(cleanEmail))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CreateAccount inserts a cryptographic hash of the email into the database layer.
func CreateAccount(db *sql.DB, email string) (int64, error) {
	hashedEmail := hashEmail(email)

	query := `INSERT INTO accounts (email) VALUES (?) ON CONFLICT(email) DO UPDATE SET email=email;`

	result, err := db.Exec(query, hashedEmail)
	if err != nil {
		return 0, err
	}

	id, err := result.LastInsertId()
	if err != nil || id == 0 {
		return GetAccountID(db, email)
	}

	return id, nil
}

// GetAccountID retrieves the unique internal ID using the hashed email identifier.
func GetAccountID(db *sql.DB, email string) (int64, error) {
	hashedEmail := hashEmail(email)
	var id int64
	query := `SELECT id FROM accounts WHERE email = ?;`

	err := db.QueryRow(query, hashedEmail).Scan(&id)
	if err != nil {
		return 0, err
	}

	return id, nil
}
