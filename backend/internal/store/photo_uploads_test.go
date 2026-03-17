package store_test

import (
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestCreatePhotoUpload(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	photo, err := store.CreatePhotoUpload(db, baby.ID, "photos/abc123.jpg", "photos/thumb_abc123.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload failed: %v", err)
	}

	if photo.ID == "" {
		t.Error("expected non-empty ID")
	}
	if photo.R2Key != "photos/abc123.jpg" {
		t.Errorf("expected r2_key=photos/abc123.jpg, got %s", photo.R2Key)
	}
	if photo.ThumbnailKey == nil || *photo.ThumbnailKey != "photos/thumb_abc123.jpg" {
		t.Errorf("expected thumbnail_key=photos/thumb_abc123.jpg, got %v", photo.ThumbnailKey)
	}
	if photo.BabyID == nil || *photo.BabyID != baby.ID {
		t.Error("expected baby_id to match")
	}
}

func TestGetPhotoUploadByR2Key(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	_, err := store.CreatePhotoUpload(db, baby.ID, "photos/lookup.jpg", "photos/thumb_lookup.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload failed: %v", err)
	}

	photo, err := store.GetPhotoUploadByR2Key(db, "photos/lookup.jpg")
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key failed: %v", err)
	}
	if photo.R2Key != "photos/lookup.jpg" {
		t.Errorf("expected r2_key=photos/lookup.jpg, got %s", photo.R2Key)
	}
}
