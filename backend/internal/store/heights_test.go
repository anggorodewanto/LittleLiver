package store

import (
	"database/sql"
	"testing"
)

func TestCreateHeight_StoresFieldsCorrectly(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH1", "ah@b.com", "Parent")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	ts := "2025-07-01T10:30:00Z"
	source := "home_scale"
	notes := "lying flat"

	h, err := CreateHeight(db, baby.ID, user.ID, ts, 54.2, &source, &notes)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}

	if h.ID == "" {
		t.Error("expected non-empty height ID")
	}
	if len(h.ID) != 26 {
		t.Errorf("expected 26-char ULID, got %d chars", len(h.ID))
	}
	if h.BabyID != baby.ID {
		t.Errorf("expected baby_id=%q, got %q", baby.ID, h.BabyID)
	}
	if h.LoggedBy != user.ID {
		t.Errorf("expected logged_by=%q, got %q", user.ID, h.LoggedBy)
	}
	if h.UpdatedBy != nil {
		t.Errorf("expected nil updated_by, got %v", h.UpdatedBy)
	}
	if h.HeightCm != 54.2 {
		t.Errorf("expected height_cm=54.2, got %f", h.HeightCm)
	}
	if h.MeasurementSource == nil || *h.MeasurementSource != "home_scale" {
		t.Errorf("expected measurement_source=home_scale, got %v", h.MeasurementSource)
	}
	if h.Notes == nil || *h.Notes != "lying flat" {
		t.Errorf("expected notes=lying flat, got %v", h.Notes)
	}
	if h.CreatedAt.IsZero() {
		t.Error("expected non-zero created_at")
	}
	if h.UpdatedAt.IsZero() {
		t.Error("expected non-zero updated_at")
	}
}

func TestCreateHeight_NilOptionalFields(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH2", "bh@b.com", "Parent2")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	h, err := CreateHeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}
	if h.MeasurementSource != nil {
		t.Errorf("expected nil measurement_source, got %v", h.MeasurementSource)
	}
	if h.Notes != nil {
		t.Errorf("expected nil notes, got %v", h.Notes)
	}
}

func TestGetHeightByID_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := GetHeightByID(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent height")
	}
}

func TestListHeights_Empty(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH3", "ch@b.com", "Parent3")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	page, err := ListHeights(db, baby.ID, nil, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListHeights failed: %v", err)
	}
	if len(page.Data) != 0 {
		t.Errorf("expected 0 heights, got %d", len(page.Data))
	}
	if page.NextCursor != nil {
		t.Error("expected nil next_cursor")
	}
}

func TestListHeights_Pagination(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH4", "dh@b.com", "Parent4")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	for i := 0; i < 3; i++ {
		_, err := CreateHeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
		if err != nil {
			t.Fatalf("CreateHeight failed: %v", err)
		}
	}

	page, err := ListHeights(db, baby.ID, nil, nil, nil, 2)
	if err != nil {
		t.Fatalf("ListHeights failed: %v", err)
	}
	if len(page.Data) != 2 {
		t.Errorf("expected 2 heights, got %d", len(page.Data))
	}
	if page.NextCursor == nil {
		t.Error("expected non-nil next_cursor")
	}
}

func TestUpdateHeight_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH5", "eh@b.com", "Parent5")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	h, err := CreateHeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}

	src := "clinic"
	notes := "updated"
	updated, err := UpdateHeight(db, baby.ID, h.ID, user.ID, "2025-07-01T11:00:00Z", 51.5, &src, &notes)
	if err != nil {
		t.Fatalf("UpdateHeight failed: %v", err)
	}
	if updated.HeightCm != 51.5 {
		t.Errorf("expected height_cm=51.5, got %f", updated.HeightCm)
	}
	if updated.UpdatedBy == nil || *updated.UpdatedBy != user.ID {
		t.Errorf("expected updated_by=%q, got %v", user.ID, updated.UpdatedBy)
	}
}

func TestUpdateHeight_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	_, err := UpdateHeight(db, "nonexistent-baby", "nonexistent-id", "user1", "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err == nil {
		t.Error("expected error for nonexistent height")
	}
}

func TestDeleteHeight_Success(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH6", "fh@b.com", "Parent6")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	h, err := CreateHeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}

	err = DeleteHeight(db, baby.ID, h.ID)
	if err != nil {
		t.Fatalf("DeleteHeight failed: %v", err)
	}

	_, err = GetHeightByID(db, baby.ID, h.ID)
	if err == nil {
		t.Error("expected error after deletion")
	}
}

func TestDeleteHeight_NotFound(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	err := DeleteHeight(db, "nonexistent-baby", "nonexistent-id")
	if err == nil {
		t.Error("expected error for nonexistent height")
	}
}

func TestListHeights_DateFilters(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH7", "gh@b.com", "Parent7")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateHeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}
	_, err = CreateHeight(db, baby.ID, user.ID, "2025-07-03T10:30:00Z", 50.5, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}

	from := "2025-07-02"
	page, err := ListHeights(db, baby.ID, &from, nil, nil, 50)
	if err != nil {
		t.Fatalf("ListHeights failed: %v", err)
	}
	if len(page.Data) != 1 {
		t.Errorf("expected 1 height with from filter, got %d", len(page.Data))
	}
}

func TestCreateHeight_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := CreateHeight(db, "baby1", "user1", "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestGetHeightByID_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := GetHeightByID(db, "baby1", "h1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListHeights_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := ListHeights(db, "baby1", nil, nil, nil, 50)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestDeleteHeight_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	err := DeleteHeight(db, "baby1", "h1")
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestUpdateHeight_ClosedDB(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	db.Close()

	_, err := UpdateHeight(db, "baby1", "h1", "user1", "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err == nil {
		t.Error("expected error with closed DB")
	}
}

func TestListHeights_InvalidFromDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListHeights(db, "baby1", &badDate, nil, nil, 50)
	if err == nil {
		t.Error("expected error for invalid from date")
	}
}

func TestListHeights_InvalidToDate(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	badDate := "not-a-date"
	_, err := ListHeights(db, "baby1", nil, &badDate, nil, 50)
	if err == nil {
		t.Error("expected error for invalid to date")
	}
}

func TestDeleteHeight_CascadeOnBabyDelete(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH8", "hh@b.com", "Parent8")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}

	_, err = CreateHeight(db, baby.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}

	_, err = db.Exec("DELETE FROM baby_parents WHERE baby_id = ?", baby.ID)
	if err != nil {
		t.Fatalf("delete baby_parents failed: %v", err)
	}
	_, err = db.Exec("DELETE FROM babies WHERE id = ?", baby.ID)
	if err != nil {
		t.Fatalf("delete baby failed: %v", err)
	}

	var count int
	err = db.QueryRow("SELECT COUNT(*) FROM heights WHERE baby_id = ?", baby.ID).Scan(&count)
	if err != nil {
		t.Fatalf("count query failed: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0 heights after cascade, got %d", count)
	}
}

func TestGetHeightByID_WrongBaby(t *testing.T) {
	t.Parallel()
	db := setupTestDB(t)
	defer db.Close()

	user, err := UpsertUser(db, "googleH9", "ih@b.com", "Parent9")
	if err != nil {
		t.Fatalf("UpsertUser failed: %v", err)
	}
	baby1, err := CreateBaby(db, user.ID, "Luna", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby failed: %v", err)
	}
	baby2, err := CreateBaby(db, user.ID, "Stella", "female", "2025-06-15", nil, nil, nil, nil)
	if err != nil {
		t.Fatalf("CreateBaby 2 failed: %v", err)
	}

	h, err := CreateHeight(db, baby1.ID, user.ID, "2025-07-01T10:30:00Z", 50.0, nil, nil)
	if err != nil {
		t.Fatalf("CreateHeight failed: %v", err)
	}

	_, err = GetHeightByID(db, baby2.ID, h.ID)
	if err == nil {
		t.Error("expected error when accessing height from wrong baby")
	}
	if err != nil && err != sql.ErrNoRows {
		// wrapping is fine
	}
}
