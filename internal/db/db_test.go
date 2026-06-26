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
