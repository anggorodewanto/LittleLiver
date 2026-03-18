package handler

import (
	"bytes"
	"database/sql"
	"log"
	"net/http"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
	"github.com/ablankz/LittleLiver/backend/internal/report"
	"github.com/ablankz/LittleLiver/backend/internal/storage"
)

// ReportHandler handles GET /api/babies/{id}/report?from=&to=.
// Generates a PDF report for the given baby and date range.
func ReportHandler(db *sql.DB, objStore storage.ObjectStore) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := requireUser(w, r)
		if !ok {
			return
		}

		baby, ok := requireBabyAccess(w, r, db, user.ID)
		if !ok {
			return
		}

		from := r.URL.Query().Get("from")
		to := r.URL.Query().Get("to")
		if from == "" || to == "" {
			http.Error(w, "from and to query parameters are required", http.StatusBadRequest)
			return
		}

		if _, err := time.Parse(model.DateFormat, from); err != nil {
			http.Error(w, "invalid from date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}
		if _, err := time.Parse(model.DateFormat, to); err != nil {
			http.Error(w, "invalid to date format, expected YYYY-MM-DD", http.StatusBadRequest)
			return
		}

		var buf bytes.Buffer
		if err := report.Generate(db, objStore, baby, from, to, &buf); err != nil {
			log.Printf("generate report: %v", err)
			http.Error(w, "internal error", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/pdf")
		w.Header().Set("Content-Disposition", "inline; filename=\"report.pdf\"")
		w.Write(buf.Bytes())
	}
}
