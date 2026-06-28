package db

import (
	"crypto/sha256"
	"database/sql"
	"embed"
	"encoding/hex"
	"log"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

//go:embed schema.sql
var schemaFS embed.FS

// InitDB initializes the database file connection and configures performance pragmas.
func InitDB(filepath string) *sql.DB {
	db, err := sql.Open("sqlite3", filepath)
	if err != nil {
		log.Fatalf("[ERROR] Failed to connect to database: %v", err)
	}

	_, err = db.Exec(`PRAGMA foreign_keys = ON; PRAGMA journal_mode = WAL;`)
	if err != nil {
		log.Fatalf("[ERROR] Failed to set SQLite pragmas: %v", err)
	}

	executeSchema(db)
	return db
}

// executeSchema reads the embedded schema DDL script and applies it to the database instance.
func executeSchema(db *sql.DB) {
	statements, err := schemaFS.ReadFile("schema.sql")
	if err != nil {
		log.Fatalf("[ERROR] Failed to read embedded database schema file: %v", err)
	}

	_, err = db.Exec(string(statements))
	if err != nil {
		log.Fatalf("[ERROR] Failed to execute schema initialization script: %v", err)
	}

	log.Println("[INIT] Database architecture initialized smoothly via embedded schema.")
}

// hashEmail normalizes an email address string and returns its SHA-256 hex digest.
func hashEmail(email string) string {
	cleanEmail := strings.TrimSpace(strings.ToLower(email))
	hasher := sha256.New()
	hasher.Write([]byte(cleanEmail))
	return hex.EncodeToString(hasher.Sum(nil))
}

// CreateAccount inserts a blind cryptographic hash of an email into the accounts mapping.
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

// GetAccountID fetches the unique internal row ID corresponding to a given email address string.
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

// InsertOrUpdatePackage writes parsed tracking telemetry and smart locker pins into the data layer defensively.
func InsertOrUpdatePackage(db *sql.DB, accountID int64, trackingNum, carrier, lockerCode string, isLockerInt int) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if trackingNum != "" {
		if trackingNum == "MANUAL_ACTION_REQUIRED" {
			var maxSeq int
			err := tx.QueryRow(`SELECT COALESCE(MAX(box_sequence), 0) FROM packages WHERE tracking_number = 'MANUAL_ACTION_REQUIRED'`).Scan(&maxSeq)
			if err != nil {
				return err
			}
			nextSeq := maxSeq + 1

			insertQuery := `
				INSERT INTO packages (account_id, tracking_number, box_sequence, carrier, last_status, location_state, is_active, updated_at)
				VALUES (?, ?, ?, ?, 'In Transit', 'Action Required', 1, CURRENT_TIMESTAMP);
			`
			_, err = tx.Exec(insertQuery, accountID, trackingNum, nextSeq, carrier)
			if err != nil {
				return err
			}
		} else {
			upsertQuery := `
				INSERT INTO packages (account_id, tracking_number, box_sequence, carrier, last_status, location_state, is_active, updated_at)
				VALUES (?, ?, 1, ?, 'In Transit', 'Sorting Facility', 1, CURRENT_TIMESTAMP)
				ON CONFLICT(tracking_number, box_sequence) DO UPDATE SET
					last_status = 'In Transit',
					updated_at = CURRENT_TIMESTAMP;
			`
			_, err = tx.Exec(upsertQuery, accountID, trackingNum, carrier)
			if err != nil {
				return err
			}
		}
	}

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

// CheckDriverShortfalls verifies if an account holds packages marked delivered to a locker station
// that lack a corresponding active collection PIN inside locker_status.
func CheckDriverShortfalls(db *sql.DB, accountID int64) (bool, error) {
	var count int
	query := `
		SELECT COUNT(*) 
		FROM packages p
		LEFT JOIN locker_status l ON p.account_id = l.account_id AND l.is_active = 1
		WHERE p.account_id = ? 
		  AND p.last_status = 'Delivered' 
		  AND (p.location_state LIKE '%Locker%' OR p.location_state LIKE '%Hub%')
		  AND (l.latest_code IS NULL OR l.latest_code = '');
	`

	err := db.QueryRow(query, accountID).Scan(&count)
	if err != nil {
		return false, err
	}

	return count > 0, nil
}
