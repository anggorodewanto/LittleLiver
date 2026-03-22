package store

import (
	"database/sql"
	"strings"
	"testing"
	"time"
)

func TestCreateInvite_Returns6DigitCode(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	invite, err := CreateInvite(db, baby.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	if len(invite.Code) != 6 {
		t.Errorf("expected 6-digit code, got %q (len=%d)", invite.Code, len(invite.Code))
	}
	for _, c := range invite.Code {
		if c < '0' || c > '9' {
			t.Errorf("expected numeric code, got %q", invite.Code)
			break
		}
	}
	if invite.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, invite.BabyID)
	}
	if invite.CreatedBy != user.ID {
		t.Errorf("expected created_by=%q, got %q", user.ID, invite.CreatedBy)
	}
	if invite.UsedBy != nil {
		t.Errorf("expected used_by=nil, got %v", invite.UsedBy)
	}
	if invite.UsedAt != nil {
		t.Errorf("expected used_at=nil, got %v", invite.UsedAt)
	}
	// Expiry should be ~24h from now
	diff := time.Until(invite.ExpiresAt)
	if diff < 23*time.Hour || diff > 25*time.Hour {
		t.Errorf("expected expiry ~24h from now, got %v", diff)
	}
}

func TestCreateInvite_DeletesPriorCodes(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	// Create first invite
	first, err := CreateInvite(db, baby.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInvite (first) failed: %v", err)
	}

	// Create second invite — should delete the first
	second, err := CreateInvite(db, baby.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInvite (second) failed: %v", err)
	}

	if first.Code == second.Code {
		t.Error("expected different codes for first and second invite")
	}

	// First code should no longer exist
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = ?", first.Code).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected prior code to be deleted, got count=%d", count)
	}

	// Second code should exist
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = ?", second.Code).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected new code to exist, got count=%d", count)
	}
}

func TestRedeemInvite_ValidCode(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	creator, err := UpsertUser(db, "google1", "a@b.com", "Creator")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, creator.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	invite, err := CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	joiner, err := UpsertUser(db, "google2", "b@b.com", "Joiner")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	babyID, err := RedeemInvite(db, invite.Code, joiner.ID)
	if err != nil {
		t.Fatalf("RedeemInvite failed: %v", err)
	}
	if babyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, babyID)
	}

	// Joiner should now be linked to baby
	linked, err := IsParentOfBaby(db, joiner.ID, baby.ID)
	if err != nil {
		t.Fatalf("IsParentOfBaby failed: %v", err)
	}
	if !linked {
		t.Error("expected joiner to be linked to baby")
	}

	// Code should be marked as used
	var usedBy sql.NullString
	err = db.QueryRow("SELECT used_by FROM invites WHERE code = ?", invite.Code).Scan(&usedBy)
	if err != nil {
		t.Fatalf("query used_by failed: %v", err)
	}
	if !usedBy.Valid || usedBy.String != joiner.ID {
		t.Errorf("expected used_by=%q, got %v", joiner.ID, usedBy)
	}
}

func TestRedeemInvite_ExpiredCode(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	creator, err := UpsertUser(db, "google1", "a@b.com", "Creator")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, creator.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	invite, err := CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	// Manually expire the invite
	_, err = db.Exec("UPDATE invites SET expires_at = ? WHERE code = ?",
		time.Now().Add(-1*time.Hour).UTC().Format(time.DateTime), invite.Code)
	if err != nil {
		t.Fatalf("expire invite failed: %v", err)
	}

	joiner, err := UpsertUser(db, "google2", "b@b.com", "Joiner")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	_, err = RedeemInvite(db, invite.Code, joiner.ID)
	if err == nil {
		t.Fatal("expected error for expired code, got nil")
	}
	if err != ErrInvalidInvite {
		t.Errorf("expected ErrInvalidInvite, got %v", err)
	}
}

func TestRedeemInvite_UsedCode(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	creator, err := UpsertUser(db, "google1", "a@b.com", "Creator")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, creator.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	invite, err := CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	joiner1, err := UpsertUser(db, "google2", "b@b.com", "Joiner1")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	joiner2, err := UpsertUser(db, "google3", "c@b.com", "Joiner2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	// First redemption succeeds
	_, err = RedeemInvite(db, invite.Code, joiner1.ID)
	if err != nil {
		t.Fatalf("first RedeemInvite failed: %v", err)
	}

	// Second redemption fails
	_, err = RedeemInvite(db, invite.Code, joiner2.ID)
	if err == nil {
		t.Fatal("expected error for used code, got nil")
	}
	if err != ErrInvalidInvite {
		t.Errorf("expected ErrInvalidInvite, got %v", err)
	}
}

func TestRedeemInvite_InvalidCode(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	creator, err := UpsertUser(db, "google1", "a@b.com", "Creator")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	_, err = RedeemInvite(db, "000000", creator.ID)
	if err == nil {
		t.Fatal("expected error for invalid code, got nil")
	}
	if err != ErrInvalidInvite {
		t.Errorf("expected ErrInvalidInvite, got %v", err)
	}
}

func TestRedeemInvite_AlreadyLinked(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	creator, err := UpsertUser(db, "google1", "a@b.com", "Creator")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, creator.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	invite, err := CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	// Creator is already linked — should get ErrAlreadyLinked
	babyID, err := RedeemInvite(db, invite.Code, creator.ID)
	if err != ErrAlreadyLinked {
		t.Fatalf("expected ErrAlreadyLinked, got %v", err)
	}
	if babyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, babyID)
	}

	// Verify the invite is NOT marked as used (already-linked is a no-op)
	var usedBy sql.NullString
	err = db.QueryRow("SELECT used_by FROM invites WHERE code = ?", invite.Code).Scan(&usedBy)
	if err != nil {
		t.Fatalf("query used_by failed: %v", err)
	}
	if usedBy.Valid {
		t.Errorf("expected used_by to remain NULL for already-linked parent, got %v", usedBy.String)
	}
}

func TestCreateInvite_CodeIsNumeric(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	// Generate multiple invites to increase confidence in code format
	for i := 0; i < 10; i++ {
		invite, err := CreateInvite(db, baby.ID, user.ID)
		if err != nil {
			t.Fatalf("CreateInvite failed on iteration %d: %v", i, err)
		}
		if len(invite.Code) != 6 {
			t.Errorf("iteration %d: expected 6-digit code, got %q", i, invite.Code)
		}
		for _, c := range invite.Code {
			if c < '0' || c > '9' {
				t.Errorf("iteration %d: non-numeric character in code %q", i, invite.Code)
				break
			}
		}
	}
}

func TestCreateInvite_MultipleInvites_DifferentBabies(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby1, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 1 failed: %v", err)
	}
	baby2, err := CreateBaby(db, user.ID, "Kai", "male", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 2 failed: %v", err)
	}

	// Invites for different babies should both exist
	inv1, err := CreateInvite(db, baby1.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInvite 1 failed: %v", err)
	}
	inv2, err := CreateInvite(db, baby2.ID, user.ID)
	if err != nil {
		t.Fatalf("CreateInvite 2 failed: %v", err)
	}

	// Both should still exist (different babies, no deletion)
	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = ?", inv1.Code).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected inv1 to still exist, got count=%d", count)
	}
	err = db.QueryRow("SELECT COUNT(*) FROM invites WHERE code = ?", inv2.Code).Scan(&count)
	if err != nil {
		t.Fatalf("query failed: %v", err)
	}
	if count != 1 {
		t.Errorf("expected inv2 to still exist, got count=%d", count)
	}
}

func TestRedeemInvite_LinksParentCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	creator, err := UpsertUser(db, "google1", "a@b.com", "Creator")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, creator.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	invite, err := CreateInvite(db, baby.ID, creator.ID)
	if err != nil {
		t.Fatalf("CreateInvite failed: %v", err)
	}

	joiner, err := UpsertUser(db, "google2", "b@b.com", "Joiner")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}

	babyID, err := RedeemInvite(db, invite.Code, joiner.ID)
	if err != nil {
		t.Fatalf("RedeemInvite failed: %v", err)
	}
	if babyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, babyID)
	}

	// Verify used_at is set
	var usedAt sql.NullString
	err = db.QueryRow("SELECT used_at FROM invites WHERE code = ?", invite.Code).Scan(&usedAt)
	if err != nil {
		t.Fatalf("query used_at failed: %v", err)
	}
	if !usedAt.Valid {
		t.Error("expected used_at to be set after redemption")
	}

	// Verify joiner can see the baby
	babies, err := GetBabiesByUserID(db, joiner.ID)
	if err != nil {
		t.Fatalf("GetBabiesByUserID failed: %v", err)
	}
	if len(babies) != 1 {
		t.Fatalf("expected 1 baby for joiner, got %d", len(babies))
	}
	if babies[0].ID != baby.ID {
		t.Errorf("expected baby ID=%q, got %q", baby.ID, babies[0].ID)
	}
}

// TestCreateInvite_NonUniqueErrorNotRetried tests that non-UNIQUE-constraint
// errors are returned immediately rather than retried.
func TestCreateInvite_NonUniqueErrorNotRetried(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "google1", "a@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	// Rename invites table and create a replacement with a CHECK constraint that
	// always fails on INSERT, causing a non-UNIQUE error.
	_, err = db.Exec("ALTER TABLE invites RENAME TO invites_bak")
	if err != nil {
		t.Fatalf("rename invites: %v", err)
	}
	_, err = db.Exec(`CREATE TABLE invites (
		code TEXT PRIMARY KEY,
		baby_id TEXT NOT NULL,
		created_by TEXT NOT NULL,
		expires_at TEXT NOT NULL,
		used_by TEXT,
		used_at TEXT,
		CHECK(length(code) > 100)
	)`)
	if err != nil {
		t.Fatalf("create fake invites table: %v", err)
	}

	_, err = CreateInvite(db, baby.ID, user.ID)
	if err == nil {
		t.Fatal("expected error from CHECK constraint, got nil")
	}
	// The error should be returned immediately (not retried),
	// so it should mention "insert" not "max retries exceeded"
	if strings.Contains(err.Error(), "max retries exceeded") {
		t.Error("should not retry on non-UNIQUE errors; got 'max retries exceeded'")
	}
	if !strings.Contains(err.Error(), "create invite: insert") {
		t.Errorf("expected 'create invite: insert' error, got: %v", err)
	}
}

func TestCreateInvite_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateInvite(db, "baby1", "user1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}

func TestRedeemInvite_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := RedeemInvite(db, "123456", "user1")
	if err == nil {
		t.Fatal("expected error for closed DB, got nil")
	}
}
