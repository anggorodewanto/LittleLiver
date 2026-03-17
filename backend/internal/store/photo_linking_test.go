package store_test

import (
	"strings"
	"testing"

	"github.com/ablankz/LittleLiver/backend/internal/store"
	"github.com/ablankz/LittleLiver/backend/internal/testutil"
)

func TestValidateAndLinkPhotos_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create photo uploads
	p1, err := store.CreatePhotoUpload(db, baby.ID, "photos/key1.jpg", "photos/thumb_key1.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}
	p2, err := store.CreatePhotoUpload(db, baby.ID, "photos/key2.jpg", "photos/thumb_key2.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}

	// Link them
	keys := []string{p1.R2Key, p2.R2Key}
	err = store.ValidateAndLinkPhotos(db, baby.ID, keys)
	if err != nil {
		t.Fatalf("ValidateAndLinkPhotos: %v", err)
	}

	// Verify linked_at is set
	photo, err := store.GetPhotoUploadByR2Key(db, p1.R2Key)
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo.LinkedAt == nil {
		t.Error("expected linked_at to be set")
	}

	photo2, err := store.GetPhotoUploadByR2Key(db, p2.R2Key)
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo2.LinkedAt == nil {
		t.Error("expected linked_at to be set for second photo")
	}
}

func TestValidateAndLinkPhotos_WrongBaby(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby1 := testutil.CreateTestBaby(t, db, user.ID)
	baby2 := testutil.CreateTestBaby(t, db, user.ID)

	// Upload photo for baby1
	p, err := store.CreatePhotoUpload(db, baby1.ID, "photos/key1.jpg", "photos/thumb_key1.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}

	// Try to link to baby2 — should fail
	err = store.ValidateAndLinkPhotos(db, baby2.ID, []string{p.R2Key})
	if err == nil {
		t.Fatal("expected error for wrong baby photo key")
	}
}

func TestValidateAndLinkPhotos_NonexistentKey(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	err := store.ValidateAndLinkPhotos(db, baby.ID, []string{"photos/nonexistent.jpg"})
	if err == nil {
		t.Fatal("expected error for nonexistent photo key")
	}
}

func TestValidateAndLinkPhotos_ExceedsLimit(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Create 5 photo uploads
	var keys []string
	for i := 0; i < 5; i++ {
		p, err := store.CreatePhotoUpload(db, baby.ID, "photos/key"+string(rune('a'+i))+".jpg", "photos/thumb_key"+string(rune('a'+i))+".jpg")
		if err != nil {
			t.Fatalf("CreatePhotoUpload %d: %v", i, err)
		}
		keys = append(keys, p.R2Key)
	}

	// Linking 5 should exceed 4-photo limit
	err := store.ValidateAndLinkPhotos(db, baby.ID, keys)
	if err == nil {
		t.Fatal("expected error for exceeding 4-photo limit")
	}
}

func TestValidateAndLinkPhotos_AlreadyLinked(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	p, err := store.CreatePhotoUpload(db, baby.ID, "photos/already.jpg", "photos/thumb_already.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}

	// Link the photo
	err = store.ValidateAndLinkPhotos(db, baby.ID, []string{p.R2Key})
	if err != nil {
		t.Fatalf("first link: %v", err)
	}

	// Attempt to link again — should fail
	err = store.ValidateAndLinkPhotos(db, baby.ID, []string{p.R2Key})
	if err == nil {
		t.Fatal("expected error for already-linked photo")
	}
	if !strings.Contains(err.Error(), "already linked") {
		t.Errorf("expected 'already linked' error, got: %v", err)
	}
}

func TestValidateAndLinkPhotos_EmptyKeys(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	// Empty keys should be a no-op
	err := store.ValidateAndLinkPhotos(db, baby.ID, nil)
	if err != nil {
		t.Fatalf("expected no error for nil keys, got: %v", err)
	}

	err = store.ValidateAndLinkPhotos(db, baby.ID, []string{})
	if err != nil {
		t.Fatalf("expected no error for empty keys, got: %v", err)
	}
}

func TestUnlinkPhotos_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	p, err := store.CreatePhotoUpload(db, baby.ID, "photos/key1.jpg", "photos/thumb_key1.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}

	// Link first
	err = store.ValidateAndLinkPhotos(db, baby.ID, []string{p.R2Key})
	if err != nil {
		t.Fatalf("ValidateAndLinkPhotos: %v", err)
	}

	// Now unlink
	err = store.UnlinkPhotos(db, []string{p.R2Key})
	if err != nil {
		t.Fatalf("UnlinkPhotos: %v", err)
	}

	// Verify linked_at is null
	photo, err := store.GetPhotoUploadByR2Key(db, p.R2Key)
	if err != nil {
		t.Fatalf("GetPhotoUploadByR2Key: %v", err)
	}
	if photo.LinkedAt != nil {
		t.Error("expected linked_at to be nil after unlinking")
	}
}

func TestGetPhotoUploadsByR2Keys_Success(t *testing.T) {
	t.Parallel()
	db := testutil.SetupTestDB(t)
	defer db.Close()

	user := testutil.CreateTestUser(t, db)
	baby := testutil.CreateTestBaby(t, db, user.ID)

	p1, err := store.CreatePhotoUpload(db, baby.ID, "photos/key1.jpg", "photos/thumb_key1.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}
	p2, err := store.CreatePhotoUpload(db, baby.ID, "photos/key2.jpg", "photos/thumb_key2.jpg")
	if err != nil {
		t.Fatalf("CreatePhotoUpload: %v", err)
	}

	photos, err := store.GetPhotoUploadsByR2Keys(db, []string{p1.R2Key, p2.R2Key})
	if err != nil {
		t.Fatalf("GetPhotoUploadsByR2Keys: %v", err)
	}
	if len(photos) != 2 {
		t.Fatalf("expected 2 photos, got %d", len(photos))
	}
}
