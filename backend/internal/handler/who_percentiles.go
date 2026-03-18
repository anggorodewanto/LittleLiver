package handler

import (
	"net/http"
	"strconv"

	"github.com/ablankz/LittleLiver/backend/internal/who"
)

// WHOPercentilesHandler handles GET /api/who/percentiles.
// Returns 5 percentile curves (3rd, 15th, 50th, 85th, 97th) for the given sex and day range.
func WHOPercentilesHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()

		sex := q.Get("sex")
		if sex == "" {
			http.Error(w, "sex parameter is required", http.StatusBadRequest)
			return
		}

		fromStr := q.Get("from_days")
		if fromStr == "" {
			http.Error(w, "from_days parameter is required", http.StatusBadRequest)
			return
		}
		fromDays, err := strconv.Atoi(fromStr)
		if err != nil {
			http.Error(w, "from_days must be an integer", http.StatusBadRequest)
			return
		}

		toStr := q.Get("to_days")
		if toStr == "" {
			http.Error(w, "to_days parameter is required", http.StatusBadRequest)
			return
		}
		toDays, err := strconv.Atoi(toStr)
		if err != nil {
			http.Error(w, "to_days must be an integer", http.StatusBadRequest)
			return
		}

		curves, err := who.PercentileCurves(sex, fromDays, toDays)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		writeJSON(w, http.StatusOK, map[string]any{
			"curves": curves,
		})
	}
}
