package handler_test

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/handler"
	"github.com/ablankz/LittleLiver/backend/internal/middleware"
	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

// --- POST /api/babies/{id}/notes ---

func TestCreateGeneralNoteHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","content":"Baby seemed fussy after feeding","category":"behavior"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] == nil || resp["id"] == "" {
		t.Error("expected non-empty note ID")
	}
	if resp["content"] != "Baby seemed fussy after feeding" {
		t.Errorf("expected content='Baby seemed fussy after feeding', got %v", resp["content"])
	}
	if resp["category"] != "behavior" {
		t.Errorf("expected category=behavior, got %v", resp["category"])
	}
	if resp["logged_by"] != user.ID {
		t.Errorf("expected logged_by=%s, got %v", user.ID, resp["logged_by"])
	}
}

func TestCreateGeneralNoteHandler_MissingContent(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","category":"behavior"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGeneralNoteHandler_InvalidCategory(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","content":"test note","category":"invalid_cat"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGeneralNoteHandler_NilCategory(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T10:30:00Z","content":"note without category"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGeneralNoteHandler_MissingTimestamp(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"content":"no timestamp"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateGeneralNoteHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/notes")
	req.Body = io.NopCloser(bytes.NewBufferString("not json"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("POST /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

// --- GET /api/babies/{id}/notes ---

func TestListGeneralNotesHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	for i := 0; i < 3; i++ {
		_, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "note content", nil, nil)
		if err != nil {
			t.Fatalf("CreateGeneralNote failed: %v", err)
		}
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.ListGeneralNotesHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Data       []map[string]any `json:"data"`
		NextCursor *string          `json:"next_cursor"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if len(resp.Data) != 3 {
		t.Errorf("expected 3 notes, got %d", len(resp.Data))
	}
}

// --- GET /api/babies/{id}/notes/{entryId} ---

func TestGetGeneralNoteHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	note, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "note content", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes/"+note.ID)
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetGeneralNoteHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["id"] != note.ID {
		t.Errorf("expected id=%q, got %v", note.ID, resp["id"])
	}
}

func TestGetGeneralNoteHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodGet, "/api/babies/"+baby.ID+"/notes/nonexistent")
	authMw := middleware.Auth(db, testCookieName)
	h := authMw(http.HandlerFunc(handler.GetGeneralNoteHandler(db)))

	mux := http.NewServeMux()
	mux.Handle("GET /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- PUT /api/babies/{id}/notes/{entryId} ---

func TestUpdateGeneralNoteHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	note, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "original content", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	body := `{"timestamp":"2025-07-01T11:00:00Z","content":"updated content","category":"sleep"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/"+note.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp["content"] != "updated content" {
		t.Errorf("expected content=updated content, got %v", resp["content"])
	}
	if resp["updated_by"] == nil || resp["updated_by"] == "" {
		t.Error("expected non-empty updated_by after update")
	}
}

func TestUpdateGeneralNoteHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	body := `{"timestamp":"2025-07-01T11:00:00Z","content":"updated"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/nonexistent")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- DELETE /api/babies/{id}/notes/{entryId} ---

func TestDeleteGeneralNoteHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	note, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "to be deleted", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/notes/"+note.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	_, err = store.GetGeneralNoteByID(db, baby.ID, note.ID)
	if err == nil {
		t.Error("expected note to be deleted")
	}
}

func TestDeleteGeneralNoteHandler_NotFound(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/notes/nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d", rec.Code)
	}
}

// --- Cross-cutting: logged_by immutability, updated_by, cross-parent auth ---

func TestGeneralNote_LoggedByImmutableAfterEditByDifferentParent(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	parent1 := testutil.CreateTestUser(t, db)
	parent2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, parent1.ID)

	// Link parent2 to the same baby
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, parent2.ID)
	if err != nil {
		t.Fatalf("link parent2: %v", err)
	}

	// Parent1 creates a note
	note, err := store.CreateGeneralNote(db, baby.ID, parent1.ID, "2025-07-01T10:30:00Z", "original", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	// Parent2 updates the note
	body := `{"timestamp":"2025-07-01T11:00:00Z","content":"edited by parent2"}`
	req := testutil.AuthenticatedRequest(t, db, parent2.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/"+note.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	// logged_by must still be parent1
	if resp["logged_by"] != parent1.ID {
		t.Errorf("logged_by should be immutable: expected %s, got %v", parent1.ID, resp["logged_by"])
	}

	// updated_by must be parent2
	if resp["updated_by"] != parent2.ID {
		t.Errorf("updated_by should be parent2: expected %s, got %v", parent2.ID, resp["updated_by"])
	}
}

func TestGeneralNote_CrossParentEditAuthorization(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	parent1 := testutil.CreateTestUser(t, db)
	parent2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, parent1.ID)

	// Link parent2
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, parent2.ID)
	if err != nil {
		t.Fatalf("link parent2: %v", err)
	}

	// Parent1 creates
	note, err := store.CreateGeneralNote(db, baby.ID, parent1.ID, "2025-07-01T10:30:00Z", "created by p1", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	// Parent2 can edit (200, not 403)
	body := `{"timestamp":"2025-07-01T11:00:00Z","content":"edited by p2"}`
	req := testutil.AuthenticatedRequest(t, db, parent2.ID, testCookieName, testSecret, http.MethodPut, "/api/babies/"+baby.ID+"/notes/"+note.ID)
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.UpdateGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("PUT /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("cross-parent edit should succeed: expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGeneralNote_CrossParentDeleteAuthorization(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	parent1 := testutil.CreateTestUser(t, db)
	parent2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, parent1.ID)

	// Link parent2
	_, err := db.Exec("INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)", baby.ID, parent2.ID)
	if err != nil {
		t.Fatalf("link parent2: %v", err)
	}

	// Parent1 creates
	note, err := store.CreateGeneralNote(db, baby.ID, parent1.ID, "2025-07-01T10:30:00Z", "created by p1", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}

	// Parent2 can delete
	req := testutil.AuthenticatedRequest(t, db, parent2.ID, testCookieName, testSecret, http.MethodDelete, "/api/babies/"+baby.ID+"/notes/"+note.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.DeleteGeneralNoteHandler(db))))

	mux := http.NewServeMux()
	mux.Handle("DELETE /api/babies/{id}/notes/{entryId}", h)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("cross-parent delete should succeed: expected 204, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestGeneralNote_UpdatedBySetCorrectly(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create note — updated_by should be nil
	note, err := store.CreateGeneralNote(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", "original", nil, nil)
	if err != nil {
		t.Fatalf("CreateGeneralNote failed: %v", err)
	}
	if note.UpdatedBy != nil {
		t.Errorf("expected updated_by=nil on creation, got %v", *note.UpdatedBy)
	}

	// Update note — updated_by should be the editing user
	updated, err := store.UpdateGeneralNote(db, baby.ID, note.ID, user.ID, "2025-07-01T11:00:00Z", "updated", nil, nil)
	if err != nil {
		t.Fatalf("UpdateGeneralNote failed: %v", err)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%s after update, got %v", user.ID, updated.UpdatedBy)
	}
}
