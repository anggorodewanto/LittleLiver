package handler_test

import (
	"database/sql"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestDeleteAccountHandler_Returns204(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteAccountHandler(db))))

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/users/me")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAccountHandler_UserDeleted(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteAccountHandler(db))))

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/users/me")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Verify user is deleted
	_, err := store.GetUserByID(db, user.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected user to be deleted (sql.ErrNoRows), got err=%v", err)
	}
}

func TestDeleteAccountHandler_NoAuth_Returns401(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.DeleteAccountHandler(db))
	req := httptest.NewRequest(http.MethodDelete, "/api/users/me", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestDeleteAccountHandler_LastParentBabyDeleted(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteAccountHandler(db))))

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/users/me")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	// Baby should be deleted since user was last parent
	_, err := store.GetBabyByID(db, baby.ID)
	if err != sql.ErrNoRows {
		t.Errorf("expected baby to be deleted (sql.ErrNoRows), got err=%v", err)
	}
}

func TestDeleteAccountHandler_InvitesCreatedByUserDeleted(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	// Link user2 to baby so baby is not deleted
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, user2.ID)
	if err != nil {
		t.Fatalf("link user2 failed: %v", err)
	}

	// Create invite by user1
	expiresAt := time.Now().Add(24 * time.Hour).UTC().Format(time.DateTime)
	_, err = db.Exec("INSERT INTO invites (code, baby_id, created_by, expires_at) VALUES ('123456', ?, ?, ?)", baby.ID, user1.ID, expiresAt)
	if err != nil {
		t.Fatalf("insert invite failed: %v", err)
	}

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteAccountHandler(db))))

	req := testutil.AuthenticatedRequest(t, db, user1.ID, testCookieName, testSecret, http.MethodDelete, "/api/users/me")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = '123456'").Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected invite created by user1 to be deleted, found %d", count)
	}
}

func TestDeleteAccountHandler_StoreError_Returns500(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)

	user := testutil.CreateTestUser(t, db)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)

	// Create the authenticated request before closing the DB
	// so auth middleware can validate the session
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/users/me")

	// Use a separate DB for the delete handler that is closed
	db2 := testutil.SetupTestDB(t)
	db2.Close()

	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteAccountHandler(db2))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("expected 500, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestDeleteAccountHandler_InvitesUsedByAnonymized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user2.ID)

	// Invite created by user2, used by user1
	expiresAt := time.Now().Add(24 * time.Hour).UTC().Format(time.DateTime)
	usedAt := time.Now().UTC().Format(time.DateTime)
	_, err := db.Exec(
		"INSERT INTO invites (code, baby_id, created_by, used_by, used_at, expires_at) VALUES ('654321', ?, ?, ?, ?, ?)",
		baby.ID, user2.ID, user1.ID, usedAt, expiresAt,
	)
	if err != nil {
		t.Fatalf("insert invite failed: %v", err)
	}

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteAccountHandler(db))))

	req := testutil.AuthenticatedRequest(t, db, user1.ID, testCookieName, testSecret, http.MethodDelete, "/api/users/me")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d", rec.Code)
	}

	var usedBy sql.NullString
	err = db.QueryRow("SELECT used_by FROM invites WHERE code = '654321'").Scan(&usedBy)
	if err != nil {
		t.Fatalf("query invite failed: %v", err)
	}
	if !usedBy.Valid || usedBy.String != "deleted_user" {
		t.Errorf("expected used_by='deleted_user', got %v", usedBy)
	}
}
