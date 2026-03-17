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

// --- POST /api/babies/:id/invite ---

func TestCreateInviteHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/invite")
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		Code      string `json:"code"`
		ExpiresAt string `json:"expires_at"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}

	if len(resp.Code) != 6 {
		t.Errorf("expected 6-digit code, got %q", resp.Code)
	}
	if resp.ExpiresAt == "" {
		t.Error("expected non-empty expires_at")
	}
}

func TestCreateInviteHandler_NotLinked_Returns403(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user1 := testutil.CreateTestUser(t, db)
	user2 := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user1.ID)

	// user2 tries to generate invite for user1's baby
	req := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/invite")
	req.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Fatalf("expected 403, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateInviteHandler_BabyNotFound_Returns404(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/nonexistent/invite")
	req.SetPathValue("id", "nonexistent")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusNotFound {
		t.Fatalf("expected 404, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateInviteHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.CreateInviteHandler(db))
	req := httptest.NewRequest(http.MethodPost, "/api/babies/someid/invite", nil)
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

func TestCreateInviteHandler_MissingBabyID(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies//invite")
	// Don't set path value — empty ID

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestCreateInviteHandler_DeletesPriorCodes(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))

	// Generate first invite
	req1 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/invite")
	req1.SetPathValue("id", baby.ID)
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("first invite: expected 201, got %d", rec1.Code)
	}

	var resp1 struct{ Code string `json:"code"` }
	json.Unmarshal(rec1.Body.Bytes(), &resp1)

	// Generate second invite — should delete first
	req2 := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/invite")
	req2.SetPathValue("id", baby.ID)
	rec2 := httptest.NewRecorder()
	h.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusCreated {
		t.Fatalf("second invite: expected 201, got %d", rec2.Code)
	}

	var resp2 struct{ Code string `json:"code"` }
	json.Unmarshal(rec2.Body.Bytes(), &resp2)

	if resp1.Code == resp2.Code {
		t.Error("expected different codes")
	}

	// First code should be gone
	var count int
	db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = ?", resp1.Code).Scan(&count)
	if count != 0 {
		t.Errorf("expected prior code to be deleted, got count=%d", count)
	}
}

func TestJoinBabyHandler_ExpiredCode(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	creator := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, creator.ID)

	invite, err := store.CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	// Manually expire the code
	_, err = db.Exec("UPDATE invites SET expires_at = '2020-01-01 00:00:00' WHERE code = ?", invite.Code)
	if err != nil {
		t.Fatalf("expire invite failed: %v", err)
	}

	joiner := testutil.CreateTestUser(t, db)
	body := `{"code":"` + invite.Code + `"}`
	req := testutil.AuthenticatedRequest(t, db, joiner.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

// --- POST /api/babies/join ---

func TestJoinBabyHandler_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	creator := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, creator.ID)

	invite, err := store.CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	joiner := testutil.CreateTestUser(t, db)

	body := `{"code":"` + invite.Code + `"}`
	req := testutil.AuthenticatedRequest(t, db, joiner.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		BabyID  string `json:"baby_id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, resp.BabyID)
	}

	// Verify joiner is linked
	linked, err := store.IsParentOfBaby(db, joiner.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if !linked {
		t.Error("expected joiner to be linked to baby")
	}
}

func TestJoinBabyHandler_InvalidCode(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"code":"999999"}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestJoinBabyHandler_AlreadyLinked(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	creator := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, creator.ID)

	invite, err := store.CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	// Creator tries to join their own baby
	body := `{"code":"` + invite.Code + `"}`
	req := testutil.AuthenticatedRequest(t, db, creator.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d. Body: %s", rec.Code, rec.Body.String())
	}

	var resp struct {
		BabyID  string `json:"baby_id"`
		Message string `json:"message"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal failed: %v", err)
	}
	if resp.Message != "already linked to this baby" {
		t.Errorf("expected friendly already-linked message, got %q", resp.Message)
	}
}

func TestJoinBabyHandler_UsedCode(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	creator := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, creator.ID)

	invite, err := store.CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	// user1 redeems the code
	user1 := testutil.CreateTestUser(t, db)
	body1 := `{"code":"` + invite.Code + `"}`
	req1 := testutil.AuthenticatedRequest(t, db, user1.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req1.Body = io.NopCloser(bytes.NewBufferString(body1))
	req1.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec1 := httptest.NewRecorder()
	h.ServeHTTP(rec1, req1)

	if rec1.Code != http.StatusOK {
		t.Fatalf("first redeem: expected 200, got %d. Body: %s", rec1.Code, rec1.Body.String())
	}

	// user2 tries the same code — should get 400
	user2 := testutil.CreateTestUser(t, db)
	body2 := `{"code":"` + invite.Code + `"}`
	req2 := testutil.AuthenticatedRequest(t, db, user2.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req2.Body = io.NopCloser(bytes.NewBufferString(body2))
	req2.Header.Set("Content-Type", "application/json")

	rec2 := httptest.NewRecorder()
	h2 := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	h2.ServeHTTP(rec2, req2)

	if rec2.Code != http.StatusBadRequest {
		t.Fatalf("used code: expected 400, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}
}

func TestJoinBabyHandler_InvalidJSON(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req.Body = io.NopCloser(bytes.NewBufferString("{bad"))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d", rec.Code)
	}
}

func TestJoinBabyHandler_MissingCode(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)

	body := `{"code":""}`
	req := testutil.AuthenticatedRequest(t, db, user.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	req.Body = io.NopCloser(bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	h := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected 400, got %d. Body: %s", rec.Code, rec.Body.String())
	}
}

func TestJoinBabyHandler_Unauthorized(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	h := http.HandlerFunc(handler.JoinBabyHandler(db))
	req := httptest.NewRequest(http.MethodPost, "/api/babies/join", bytes.NewBufferString(`{"code":"123456"}`))
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()

	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected 401, got %d", rec.Code)
	}
}

// --- End-to-end invite flow ---

func TestInviteFlow_EndToEnd(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	creator := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, creator.ID)

	// 1. Generate invite code
	inviteReq := testutil.AuthenticatedRequest(t, db, creator.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/"+baby.ID+"/invite")
	inviteReq.SetPathValue("id", baby.ID)

	authMw := middleware.Auth(db, testCookieName)
	csrfMw := middleware.CSRF(db, testCookieName, testSecret)
	inviteH := authMw(csrfMw(http.HandlerFunc(handler.CreateInviteHandler(db))))
	rec1 := httptest.NewRecorder()
	inviteH.ServeHTTP(rec1, inviteReq)

	if rec1.Code != http.StatusCreated {
		t.Fatalf("create invite: expected 201, got %d. Body: %s", rec1.Code, rec1.Body.String())
	}

	var inviteResp struct {
		Code string `json:"code"`
	}
	if err := json.Unmarshal(rec1.Body.Bytes(), &inviteResp); err != nil {
		t.Fatalf("unmarshal invite response failed: %v", err)
	}

	// 2. Joiner redeems the code
	joiner := testutil.CreateTestUser(t, db)
	joinBody := `{"code":"` + inviteResp.Code + `"}`
	joinReq := testutil.AuthenticatedRequest(t, db, joiner.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	joinReq.Body = io.NopCloser(bytes.NewBufferString(joinBody))
	joinReq.Header.Set("Content-Type", "application/json")

	joinH := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	rec2 := httptest.NewRecorder()
	joinH.ServeHTTP(rec2, joinReq)

	if rec2.Code != http.StatusOK {
		t.Fatalf("join: expected 200, got %d. Body: %s", rec2.Code, rec2.Body.String())
	}

	// 3. Verify joiner can now access the baby
	linked, err := store.IsParentOfBaby(db, joiner.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if !linked {
		t.Error("expected joiner to be linked to baby after redeeming invite")
	}

	// 4. Try to reuse the same code — should fail
	joiner2 := testutil.CreateTestUser(t, db)
	reuseBody := `{"code":"` + inviteResp.Code + `"}`
	reuseReq := testutil.AuthenticatedRequest(t, db, joiner2.ID, testCookieName, testSecret, http.MethodPost, "/api/babies/join")
	reuseReq.Body = io.NopCloser(bytes.NewBufferString(reuseBody))
	reuseReq.Header.Set("Content-Type", "application/json")

	rec3 := httptest.NewRecorder()
	joinH2 := authMw(csrfMw(http.HandlerFunc(handler.JoinBabyHandler(db))))
	joinH2.ServeHTTP(rec3, reuseReq)

	if rec3.Code != http.StatusBadRequest {
		t.Fatalf("reuse: expected 400, got %d. Body: %s", rec3.Code, rec3.Body.String())
	}
}
