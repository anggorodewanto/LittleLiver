package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/auth"
	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
	"github.com/ablankz/LittleLiver/backend/internal/who"
)

func setupWHOMux(t *testing.T) (*http.ServeMux, *testutil.TestFixture) {
	t.Helper()
	db := testutil.SetupTestDB(t)
	user := testutil.CreateTestUser(t, db)

	mux := handler.NewMux(
		handler.WithDB(db),
		handler.WithAuthConfig(auth.Config{
			ClientID:      "test-client-id",
			ClientSecret:  "test-client-secret",
			RedirectURL:   "http://localhost:8080/auth/google/callback",
			SessionSecret: testSecret,
		}),
	)

	return mux, &testutil.TestFixture{DB: db, User: user}
}

// percentileCurvesResponse matches the JSON structure returned by the endpoint.
type percentileCurvesResponse struct {
	Curves []struct {
		Percentile float64 `json:"percentile"`
		Points     []struct {
			AgeDays  int     `json:"age_days"`
			WeightKg float64 `json:"weight_kg"`
		} `json:"points"`
	} `json:"curves"`
}

func TestWHOPercentiles_MaleCurvesReturnExpectedValues(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&from_days=0&to_days=10")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp percentileCurvesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	if len(resp.Curves) != 5 {
		t.Fatalf("expected 5 curves, got %d", len(resp.Curves))
	}

	// Verify the expected percentiles are returned
	expectedPercentiles := []float64{3, 15, 50, 85, 97}
	for i, curve := range resp.Curves {
		if curve.Percentile != expectedPercentiles[i] {
			t.Errorf("curve %d: expected percentile %v, got %v", i, expectedPercentiles[i], curve.Percentile)
		}
	}

	// Verify the 50th percentile at day 0 matches the WHO package
	whoCurves, err := who.PercentileCurves("male", 0, 10)
	if err != nil {
		t.Fatalf("who.PercentileCurves: %v", err)
	}

	// Find the 50th percentile curve (index 2)
	p50 := whoCurves[2]
	if resp.Curves[2].Points[0].WeightKg != p50.Points[0].WeightKg {
		t.Errorf("50th percentile day 0 weight: expected %v, got %v",
			p50.Points[0].WeightKg, resp.Curves[2].Points[0].WeightKg)
	}
}

func TestWHOPercentiles_FemaleCurvesDifferFromMale(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	// Request male curves
	reqM := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&from_days=0&to_days=30")
	rrM := httptest.NewRecorder()
	mux.ServeHTTP(rrM, reqM)

	if rrM.Code != http.StatusOK {
		t.Fatalf("male request: expected 200, got %d", rrM.Code)
	}

	var maleResp percentileCurvesResponse
	if err := json.NewDecoder(rrM.Body).Decode(&maleResp); err != nil {
		t.Fatalf("decode male response: %v", err)
	}

	// Request female curves
	reqF := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=female&from_days=0&to_days=30")
	rrF := httptest.NewRecorder()
	mux.ServeHTTP(rrF, reqF)

	if rrF.Code != http.StatusOK {
		t.Fatalf("female request: expected 200, got %d", rrF.Code)
	}

	var femaleResp percentileCurvesResponse
	if err := json.NewDecoder(rrF.Body).Decode(&femaleResp); err != nil {
		t.Fatalf("decode female response: %v", err)
	}

	// At least one point in the 50th percentile should differ
	malePts := maleResp.Curves[2].Points
	femalePts := femaleResp.Curves[2].Points
	anyDifferent := false
	for i := range malePts {
		if malePts[i].WeightKg != femalePts[i].WeightKg {
			anyDifferent = true
			break
		}
	}
	if !anyDifferent {
		t.Error("expected male and female 50th percentile curves to differ")
	}
}

func TestWHOPercentiles_InvalidSexReturns400(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=unknown&from_days=0&to_days=30")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for invalid sex, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWHOPercentiles_MissingSexReturns400(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?from_days=0&to_days=30")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing sex, got %d: %s", rr.Code, rr.Body.String())
	}
}

func TestWHOPercentiles_CurvesSpanRequestedDayRange(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	fromDays := 10
	toDays := 50

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&from_days=10&to_days=50")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rr.Code, rr.Body.String())
	}

	var resp percentileCurvesResponse
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}

	expectedPoints := toDays - fromDays + 1
	for i, curve := range resp.Curves {
		if len(curve.Points) != expectedPoints {
			t.Errorf("curve %d: expected %d points, got %d", i, expectedPoints, len(curve.Points))
		}
		// First point should be at fromDays
		if curve.Points[0].AgeDays != fromDays {
			t.Errorf("curve %d: first point age_days expected %d, got %d", i, fromDays, curve.Points[0].AgeDays)
		}
		// Last point should be at toDays
		last := curve.Points[len(curve.Points)-1]
		if last.AgeDays != toDays {
			t.Errorf("curve %d: last point age_days expected %d, got %d", i, toDays, last.AgeDays)
		}
	}
}

func TestWHOPercentiles_MissingFromDaysReturns400(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&to_days=30")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing from_days, got %d", rr.Code)
	}
}

func TestWHOPercentiles_MissingToDaysReturns400(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&from_days=0")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for missing to_days, got %d", rr.Code)
	}
}

func TestWHOPercentiles_NonIntegerFromDaysReturns400(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&from_days=abc&to_days=30")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-integer from_days, got %d", rr.Code)
	}
}

func TestWHOPercentiles_NonIntegerToDaysReturns400(t *testing.T) {
	t.Parallel()
	mux, fix := setupWHOMux(t)

	req := testutil.AuthenticatedRequest(t, fix.DB, fix.User.ID, testCookieName, testSecret, http.MethodGet,
		"/api/who/percentiles?sex=male&from_days=0&to_days=xyz")
	rr := httptest.NewRecorder()
	mux.ServeHTTP(rr, req)

	if rr.Code != http.StatusBadRequest {
		t.Errorf("expected 400 for non-integer to_days, got %d", rr.Code)
	}
}
