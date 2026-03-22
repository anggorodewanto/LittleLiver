package handler

import (
	"context"
	"database/sql"
	"log"
	"net/http"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
	"github.com/ablankz/LittleLiver/backend/internal/store"
)

// AnonymizeTables is the configurable list of table names that have
// logged_by/updated_by columns to anonymize on account deletion.
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
func DeleteAccountHandler(db *sql.DB, objStore storage.ObjectStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		result, err := store.DeleteAccount(db, user.ID, AnonymizeTables)
		if err != nil {
			log.Printf("delete account: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}

		// Clean up orphaned R2 objects asynchronously (best-effort)
		if objStore != nil && len(result.OrphanedR2Keys) > 0 {
			go func() {
				for _, key := range result.OrphanedR2Keys {
					if err := objStore.Delete(context.Background(), key); err != nil {
						log.Printf("delete orphaned R2 object %s: %v", key, err)
					}
				}
			}()
		}

		// Clear session cookie since the account (and all sessions) are now deleted
		http.SetCookie(w, &http.Cookie{
			Name:     auth.CookieName,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			Secure:   true,
			SameSite: http.SameSiteLaxMode,
			MaxAge:   -1,
		})

		w.WriteHeader(http.StatusNoContent)
	}
}
