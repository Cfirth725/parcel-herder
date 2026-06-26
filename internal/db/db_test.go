package db

import (
	"testing"
)

// TestInsertOrUpdatePackage verifies that the transaction engine handles multi-table writes smoothly.
func TestInsertOrUpdatePackage(t *testing.T) {
	// Create an in-memory SQLite database for lightning-fast testing isolation
	database := InitDB(":memory:")
	defer database.Close()

	id, err := CreateAccount(database, "test@example.com")
	if err != nil {
		t.Fatalf("[SECURE] Failed to provision test account: %v", err)
	}

	// Verify our transaction engine handles writes smoothly without key violations
	err = InsertOrUpdatePackage(database, id, "9400111899563214785210", "USPS", "", 0)
	if err != nil {
		t.Errorf("[ERROR] InsertOrUpdatePackage transaction failed: %v", err)
	}
}

func TestCheckDriverShortfalls(t *testing.T) {
	database := InitDB(":memory:")
	defer database.Close()

	id, _ := CreateAccount(database, "shortfall@example.com")

	// 1. Simulate a lazy driver delivery (Marked locker delivery, but locker_status table stays empty)
	packageQuery := `
		INSERT INTO packages (account_id, tracking_number, box_sequence, carrier, last_status, location_state, is_active)
		VALUES (?, '1Z1234567890', 1, 'UPS', 'Delivered', 'Dropped at Locker Terminal', 1);
	`
	_, err := database.Exec(packageQuery, id)
	if err != nil {
		t.Fatalf("[ERROR] Failed to set up mock package: %v", err)
	}

	// 2. Evaluate the anomaly detection rule
	hasShortfall, err := CheckDriverShortfalls(database, id)
	if err != nil {
		t.Fatalf("[ERROR] Exception check broken: %v", err)
	}

	if !hasShortfall {
		t.Errorf("[ERROR] Exception handler failed to catch the missing locker code shortfall!")
	}
}
