package server

import (
	"database/sql"
	"html/template"
	"log"
	"net/http"
	"time"
)

// Server holds the application dependencies, primarily the SQL database connection pool.
type Server struct {
	DB *sql.DB
}

// PackageView represents the flattened structure of a shipment for UI rendering.
type PackageView struct {
	ID                   int
	TrackingNumber       string
	Carrier              string
	CreatedAt            string
	TrackingURL          string
	ExpectedDeliveryDate string
	PackageState         int
	UrgencyClass         string // Computed: "urgency-today", "urgency-stale", or "urgency-normal"
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

	// Fetch standard tracking records filtered by trajectory urgency priority, hiding archived items (state 3)
	query := `
		SELECT 
			id, 
			tracking_number, 
			carrier, 
			datetime(updated_at, 'localtime'), 
			COALESCE(expected_delivery_date, ''), 
			package_state 
		FROM packages 
		WHERE package_state != 3 
		ORDER BY 
			CASE 
				WHEN expected_delivery_date = date('now', 'localtime') THEN 0 
				WHEN expected_delivery_date < date('now', 'localtime') AND expected_delivery_date != '' THEN 1 
				ELSE 2 
			END, 
			id DESC`

	rows, err := s.DB.Query(query)
	if err != nil {
		log.Printf("[ERROR] Failed to query packages: %v", err)
		http.Error(w, "Internal Server Database Error", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	nowStr := time.Now().Format("2006-01-02")

	for rows.Next() {
		var p PackageView
		err := rows.Scan(&p.ID, &p.TrackingNumber, &p.Carrier, &p.CreatedAt, &p.ExpectedDeliveryDate, &p.PackageState)
		if err != nil {
			log.Printf("[ERROR] Row scanning failure: %v", err)
			continue
		}

		p.TrackingURL = resolveSmartLink(p.Carrier, p.TrackingNumber)
		p.UrgencyClass = "urgency-normal"

		// Evaluate dynamic visual priority thresholds
		if p.ExpectedDeliveryDate != "" {
			if p.ExpectedDeliveryDate == nowStr {
				p.UrgencyClass = "urgency-today"
			} else if p.ExpectedDeliveryDate < nowStr {
				p.UrgencyClass = "urgency-stale"
			}
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
	http.HandleFunc("/locker/clear", s.LockerClearHandler)
	http.HandleFunc("/package/archive", s.ArchivePackageHandler)
	http.HandleFunc("/package/to-locker", s.MoveToLockerHandler)
	http.HandleFunc("/package/delay", s.DelayPackageHandler)
	http.HandleFunc("/package/convert-usps", s.ConvertToUSPSHandler)

	fs := http.FileServer(http.Dir("web/static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	log.Printf("[SERVER] High-Contrast Delivery Dashboard launching on http://localhost:%s", port)
	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("[CRITICAL] Dashboard server crashed: %v", err)
	}
}

// resolveSmartLink pairs a tracking number with a carrier's native deep-link syntax.
func resolveSmartLink(carrier, trackingNum string) string {
	switch carrier {
	case "USPS":
		return "https://tools.usps.com/go/TrackConfirmAction?tLabels=" + trackingNum
	case "FedEx":
		return "https://www.fedex.com/fedextrack/?trknbr=" + trackingNum
	case "UPS":
		return "https://www.ups.com/track?tracknum=" + trackingNum
	case "DHL":
		return "https://www.dhl.com/en/express/tracking.html?AWB=" + trackingNum
	case "OSM":
		return "https://www.osmworldwide.com/tracking/?tracking-number=" + trackingNum
	default:
		return ""
	}
}

// LockerClearHandler handles POST requests to clear the active master locker PIN
// and cascade-archives any unverified building locker packages.
func (s *Server) LockerClearHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	// Begin an atomic database transaction to guarantee data integrity
	tx, err := s.DB.Begin()
	if err != nil {
		log.Printf("[ERROR] Failed to start locker clearance transaction: %v", err)
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	// Defer a rollback defensive check. If Commit() is called successfully, this becomes a no-op.
	defer tx.Rollback()

	// 1. Deactivate the active locker code for Account 1
	lockerQuery := "UPDATE locker_status SET is_active = 0, updated_at = CURRENT_TIMESTAMP WHERE account_id = 1 AND is_active = 1"
	_, err = tx.Exec(lockerQuery)
	if err != nil {
		log.Printf("[ERROR] Transaction failed during locker status update: %v", err)
		http.Error(w, "Database Update Failure", http.StatusInternalServerError)
		return
	}

	// 2. Cascade archive any packages currently marked as Left at Locker (state 2 -> state 3)
	packageQuery := "UPDATE packages SET package_state = 3, updated_at = CURRENT_TIMESTAMP WHERE package_state = 2"
	res, err := tx.Exec(packageQuery)
	if err != nil {
		log.Printf("[ERROR] Transaction failed during cascade package archive: %v", err)
		http.Error(w, "Database Update Failure", http.StatusInternalServerError)
		return
	}

	// Log out how many unverified locker packages were caught in the sweep
	rowsAffected, _ := res.RowsAffected()

	// Commit the unified database transaction atomic layer
	if err := tx.Commit(); err != nil {
		log.Printf("[ERROR] Failed to commit locker clearance transaction: %v", err)
		http.Error(w, "Database Commit Failure", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Account 1 master locker code manually deactivated. Cascade-archived %d unverified locker packages.", rowsAffected)

	// Redirect back to the main dashboard page smoothly
	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

// ArchivePackageHandler handles POST requests to transition packages into the historical archive state.
func (s *Server) ArchivePackageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	_, err := s.DB.Exec("UPDATE packages SET package_state = 3, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		log.Printf("[ERROR] Failed to archive package %s: %v", id, err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

// MoveToLockerHandler processes fallback actions for carrier package locker delivery shortfalls.
func (s *Server) MoveToLockerHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	_, err := s.DB.Exec("UPDATE packages SET package_state = 2, updated_at = CURRENT_TIMESTAMP WHERE id = ?", id)
	if err != nil {
		log.Printf("[ERROR] Failed to shift package %s to locker state: %v", id, err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

// DelayPackageHandler appends a +1 day increment to the dynamic expected arrival trajectory row.
func (s *Server) DelayPackageHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")
	query := "UPDATE packages SET expected_delivery_date = date(COALESCE(NULLIF(expected_delivery_date, ''), 'now', 'localtime'), '+1 day'), updated_at = CURRENT_TIMESTAMP WHERE id = ?"
	_, err := s.DB.Exec(query, id)
	if err != nil {
		log.Printf("[ERROR] Failed to delay package %s target date: %v", id, err)
		http.Error(w, "Database Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}

// ConvertToUSPSHandler upgrades a shipping partner package card directly to standard USPS tracking.
func (s *Server) ConvertToUSPSHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method Not Allowed", http.StatusMethodNotAllowed)
		return
	}

	id := r.FormValue("id")

	// Update both the carrier string and refresh the tracking state context
	query := `
		UPDATE packages 
		SET carrier = 'USPS', 
		    updated_at = CURRENT_TIMESTAMP 
		WHERE id = ?`

	_, err := s.DB.Exec(query, id)
	if err != nil {
		log.Printf("[ERROR] Failed to convert package %s to USPS: %v", id, err)
		http.Error(w, "Database Modification Error", http.StatusInternalServerError)
		return
	}

	log.Printf("[OK] Package ID %s carrier manually converted to USPS.", id)

	w.Header().Set("Location", "/")
	w.WriteHeader(http.StatusSeeOther)
}
