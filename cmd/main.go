package main

import (
	"log"
	"os"

	"github.com/Cfirth725/parcel-herder/internal/db"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("⚠️ No .env file found, defaulting to system environment variables")
	}

	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "parcel_herder.db"
	}

	database := db.InitDB(dbPath)
	defer database.Close()

	log.Println("Parcel Herder database architecture is live, secure, and cleanly mapped!")
}
