package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/Cfirth725/parcel-herder/internal/db"
	"github.com/Cfirth725/parcel-herder/internal/scraper"
	"github.com/Cfirth725/parcel-herder/internal/server"
	"github.com/joho/godotenv"
)

// main initializes the application state, executes multi-tenant mailbox sync loops,
// and launches the local HTTP dashboard server.
func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("[WARN] No .env file found, defaulting to system environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "parcel_herder.db"
	}

	database := db.InitDB(dbPath)

	imapServer := os.Getenv("YAHOO_IMAP_SERVER")
	envEmail1 := os.Getenv("USER_1_EMAIL")
	envEmail2 := os.Getenv("USER_2_EMAIL")
	imapPassword1 := os.Getenv("YAHOO_PASSWORD_1")
	imapPassword2 := os.Getenv("YAHOO_PASSWORD_2")

	if imapServer == "" || envEmail1 == "" || envEmail2 == "" || imapPassword1 == "" || imapPassword2 == "" {
		log.Fatalf("[ERROR] Required configurations missing from local runtime environment")
	}

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

	targets := []struct {
		email    string
		password string
	}{
		{email: envEmail1, password: imapPassword1},
		{email: envEmail2, password: imapPassword2},
	}

	log.Println("[SYNC] Kicking off network synchronization sequence...")
	for _, target := range targets {
		err = scraper.FetchAndProcessMailboxes(imapServer, target.email, target.password, database)
		if err != nil {
			log.Printf("[WARN] IMAP Stream Notification for %s: %v", target.email, err)
		}
	}

	log.Println("All synchronization routines finalized!")

	appPort := os.Getenv("PORT")
	if appPort == "" {
		log.Fatalf("[CRITICAL] Configuration Failure: PORT variable is missing or blank in your local environment")
	}

	srv := server.NewServer(database)
	srv.StartArchivingCron(12 * time.Hour)

	// --- Graceful Shutdown Logic ---
	shutdownChan := make(chan os.Signal, 1)
	signal.Notify(shutdownChan, os.Interrupt, syscall.SIGTERM)

	// Run the HTTP server inside its own concurrent goroutine so it doesn't block main
	go func() {
		srv.Start(appPort)
	}()

	// Main execution thread blocks right here waiting for you to press Ctrl+C
	<-shutdownChan
	log.Println("\n[SHUTDOWN] Intercepted termination signal. Initiating graceful fallback...")

	// Force an SQLite checkpoint to flush the WAL and SHM journal data to the primary disk block
	log.Println("[SHUTDOWN] Executing checkpoint to flush WAL files...")
	_, err = database.Exec("PRAGMA wal_checkpoint(TRUNCATE);")
	if err != nil {
		log.Printf("[ERROR] Failed to checkpoint WAL log layers: %v", err)
	}

	// Safely close the database pool connections exclusively here
	log.Println("[SHUTDOWN] Severing thread-safe connection pools...")
	if err := database.Close(); err != nil {
		log.Printf("[ERROR] Database closure failed: %v", err)
	}

	log.Println("[OK] Parcel Herder shut down gracefully. Disk state is immaculate. Goodbye!")
}
