package server

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
)

// Server holds the application dependencies, primarily the SQL database connection pool.
type Server struct {
	DB *sql.DB
}

// PackageView represents the flattened structure of a shipment for UI rendering.
type PackageView struct {
	ID             int
	TrackingNumber string
	Carrier        string
	CreatedAt      string
}

// DashboardData consolidates package streams and active master locker PINs into a single template payload.
type DashboardData struct {
	Packages   []PackageView
	LockerCode string
}

// NewServer initializes and returns a pointer to a new Server instance.
func NewServer(db *sql.DB) *Server {
	return &Server{DB: db}
}

// DashboardHandler pulls active logistics telemetry and executes the high-contrast dashboard layout.
func (s *Server) DashboardHandler(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	var data DashboardData

	// Query the master locker code if it exists and remains active (pinned to Account 1)
	err := s.DB.QueryRow("SELECT latest_code FROM locker_status WHERE account_id = 1 AND is_active = 1").Scan(&data.LockerCode)
	if err != nil && err != sql.ErrNoRows {
		log.Printf("[ERROR] Failed to query locker status: %v", err)
	}

	// Fetch standard tracking records ordered chronologically by newest ingestion entry
	rows, err := s.DB.Query("SELECT id, tracking_number, carrier, datetime(updated_at, 'localtime') FROM packages ORDER BY id DESC")
	if err != nil {
		log.Printf("[ERROR] Failed to query packages: %v", err)
		http.Error(w, "Internal Server Database Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	for rows.Next() {
		var p PackageView
		if err := rows.Scan(&p.ID, &p.TrackingNumber, &p.Carrier, &p.CreatedAt); err != nil {
			log.Printf("[ERROR] Row scanning failure: %v", err)
			continue
		}
		data.Packages = append(data.Packages, p)
	}

	tmpl, err := template.ParseFiles("web/templates/dashboard.html")
	if err != nil {
		log.Printf("[ERROR] Template parsing failure: %v", err)
		http.Error(w, "UI Template Render Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		log.Printf("[ERROR] Template execution failure: %v", err)
	}
}

// Start registers the endpoints, mounts the flat static assets directory, and serves the HTTP infrastructure.
func (s *Server) Start(port string) {
	http.HandleFunc("/", s.DashboardHandler)

	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("[SERVER] High-Contrast Delivery Dashboard launching on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("[CRITICAL] Dashboard server crashed: %v", err)
	}
}