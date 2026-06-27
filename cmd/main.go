package main

import (
	"log"
	"os"

	"github.com/Cfirth725/parcel-herder/internal/db"
	"github.com/Cfirth725/parcel-herder/internal/scraper"
	"github.com/Cfirth725/parcel-herder/internal/server"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment setup safely
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] No .env file found, defaulting to system environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "parcel_herder.db"
	}

	// 1. Boot Database Plane
	database := db.InitDB(dbPath)
	defer database.Close()

	// 2. Fetch network and target configuration keys
	imapServer := os.Getenv("YAHOO_IMAP_SERVER")
	envEmail1 := os.Getenv("USER_1_EMAIL")
	envEmail2 := os.Getenv("USER_2_EMAIL")
	imapPassword1 := os.Getenv("YAHOO_PASSWORD_1")
	imapPassword2 := os.Getenv("YAHOO_PASSWORD_2")

	if imapServer == "" || envEmail1 == "" || envEmail2 == "" || imapPassword1 == "" || imapPassword2 == "" {
		log.Fatalf("[ERROR] Security Failure: Required configurations missing from your local .env file")
	}

	// 3. Provision Isolated Multi-Tenant Accounts
	id1, err := db.CreateAccount(database, envEmail1)
	if err != nil {
		log.Fatalf("[ERROR] Failed to provision User 1: %v", err)
	}
	log.Printf("[SECURE] Secure Blind Index Account mapped successfully [Internal ID: %d]", id1)

	id2, err := db.CreateAccount(database, envEmail2)
	if err != nil {
		log.Fatalf("[ERROR] Failed to provision User 2: %v", err)
	}
	log.Printf("[SECURE] Secure Blind Index Account mapped successfully [Internal ID: %d]", id2)

	log.Println("[INIT] Parcel Herder database architecture is live, secure, and cleanly mapped!")

	// 4. Define our synchronization targets dynamically
	targets := []struct {
		email    string
		password string
	}{
		{email: envEmail1, password: imapPassword1},
		{email: envEmail2, password: imapPassword2},
	}

	// 5. Execute the scraper stream loops sequentially
	log.Println("[SYNC] Kicking off network synchronization sequence...")
	for _, target := range targets {
		err = scraper.FetchAndProcessMailboxes(imapServer, target.email, target.password, database)
		if err != nil {
			log.Printf("[WARN] IMAP Stream Notification for %s: %v", target.email, err)
		}
	}

	log.Println("All synchronization routines finalized!")

	// STRICT PORT CHECKING: Fail fast if the environment isn't fully configured
	appPort := os.Getenv("PORT")
	if appPort == "" {
		log.Fatalf("[CRITICAL] Configuration Failure: PORT variable is missing or blank in your local .env file")
	}
	srv := server.NewServer(database)
	srv.Start(appPort)
}
