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

// InitDB initializes the database and configures performance pragmas.
func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to database: %v", err)
	}

	// Pragmas for local home-server write optimization
	_, err = db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`)
	if err != nil {
		log.Fatalf("[ERROR] Failed to set SQLite pragmas: %v", err)
	}

	executeSchema(db, "internal/db/schema.sql")
	return db
}

// executeSchema reads and executes the raw SQL schema initialization script.
func executeSchema(db *sql.DB, schemaFile string) {
	statements, err := os.ReadFile(schemaFile)
	if err != nil {
		log.Fatalf("[ERROR] Failed to read database schema file: %v", err)
	}

	_, err = db.Exec(string(statements))
	if err != nil {
		log.Fatalf("[ERROR] Failed to execute schema initialization script: %v", err)
	}

	log.Println("[INIT] Database architecture initialized smoothly via external schema file.")
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

// InsertOrUpdatePackage writes parsed tracking telemetry into the database plane using an atomic transaction.
func InsertOrUpdatePackage(db *sql.DB, accountID int64, trackingNum, carrier, lockerCode string, isLockerInt int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 1. If we have a tracking number, write it to the packages table
	if trackingNum != "" {
		packageQuery := `
			INSERT INTO packages (account_id, tracking_number, box_sequence, carrier, last_status, location_state, is_active, updated_at)
			VALUES (?, ?, 1, ?, 'In Transit', 'Sorting Facility', 1, CURRENT_TIMESTAMP)
			ON CONFLICT(tracking_number, box_sequence) DO UPDATE SET
				last_status = 'In Transit',
				updated_at = CURRENT_TIMESTAMP;
		`
		_, err = tx.Exec(packageQuery, accountID, trackingNum, carrier)
		if err != nil {
			return err
		}
	}

	// 2. If an Amazon Hub locker token is detected, write it to the locker_status table
	if lockerCode != "" {
		lockerQuery := `
			INSERT INTO locker_status (account_id, latest_code, is_active, updated_at)
			VALUES (?, ?, ?, CURRENT_TIMESTAMP)
			ON CONFLICT(account_id) DO UPDATE SET
				latest_code = ?,
				is_active = ?,
				updated_at = CURRENT_TIMESTAMP;
		`
		_, err = tx.Exec(lockerQuery, accountID, lockerCode, isLockerInt, lockerCode, isLockerInt)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}
