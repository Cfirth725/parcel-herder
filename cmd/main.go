package main

import (
	"log"
	"os"

	"github.com/Cfirth725/parcel-herder/internal/db"
	"github.com/Cfirth725/parcel-herder/internal/scraper"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment setup safely
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, defaulting to system environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "parcel_herder.db"
	}

	// 1. Boot Database Plane
	database := db.InitDB(dbPath)
	defer database.Close()

	// 2. Fetch raw target emails from the secured environment layer
	envEmail1 := os.Getenv("USER_1_EMAIL")
	envEmail2 := os.Getenv("USER_2_EMAIL")

	if envEmail1 == "" || envEmail2 == "" {
		log.Fatalf("Security Failure: Target emails missing from your local .env configuration")
	}

	// 3. Provision Isolated Multi-Tenant Accounts (Hashing occurs instantly inside internal/db)
	id1, err := db.CreateAccount(database, envEmail1)
	if err != nil {
		log.Fatalf("Failed to provision User 1: %v", err)
	}
	// Note: logged the IDs to prove isolation works, but no leaks of plain-text strings to the console logs
	log.Printf("Secure Blind Index Account mapped successfully [Internal ID: %d]", id1)

	id2, err := db.CreateAccount(database, envEmail2)
	if err != nil {
		log.Fatalf("Failed to provision User 2: %v", err)
	}
	log.Printf("Secure Blind Index Account mapped successfully [Internal ID: %d]", id2)

	log.Println("Parcel Herder database architecture is live, secure, and cleanly mapped!")

	// TEST PACKAGE
	// _ = scraper.FetchAndProcessMailboxes("imap.example.com:993", "test@example.com", "pass", database)
}
