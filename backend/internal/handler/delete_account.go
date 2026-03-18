package handler

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// AnonymizeTables is the configurable list of table names that have
// logged_by/updated_by columns to anonymize on account deletion.
// Future phases (e.g., medications, med_logs) append their table names here.
var AnonymizeTables = []string{
	"feedings",
	"stools",
	"urine",
	"weights",
	"temperatures",
	"abdomen_observations",
	"skin_observations",
	"bruising",
	"lab_results",
	"general_notes",
	"medications",
	"med_logs",
}

// DeleteAccountHandler handles DELETE /api/users/me.
// Deletes the authenticated user's account with full cascade behavior per spec §2.2.
// Returns 204 No Content on success.
func DeleteAccountHandler(db *sql.DB) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		if err := store.DeleteAccount(db, user.ID, AnonymizeTables); err != nil {
			log.Printf("delete account: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		w.WriteHeader(http.StatusNoContent)
	}
}
