package model_test

import (
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func TestUserFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	u := model.User{
		ID:        "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		GoogleID:  "google-123",
		Email:     "test@example.com",
		Name:      "Test User",
		Timezone:  ptrStr("America/New_York"),
		CreatedAt: now,
	}
	if u.ID != "01ARZ3NDEKTSV4RRFFQ69G5FAV" {
		t.Errorf("User.ID = %q", u.ID)
	}
	if u.GoogleID != "google-123" {
		t.Errorf("User.GoogleID = %q", u.GoogleID)
	}
	if u.Email != "test@example.com" {
		t.Errorf("User.Email = %q", u.Email)
	}
	if u.Name != "Test User" {
		t.Errorf("User.Name = %q", u.Name)
	}
	if u.Timezone == nil || *u.Timezone != "America/New_York" {
		t.Errorf("User.Timezone = %v", u.Timezone)
	}
	if u.CreatedAt != now {
		t.Errorf("User.CreatedAt = %v", u.CreatedAt)
	}
}

func TestBabyFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	dob := time.Date(2026, 1, 15, 0, 0, 0, 0, time.UTC)
	diag := time.Date(2026, 1, 20, 0, 0, 0, 0, time.UTC)
	kasai := time.Date(2026, 1, 25, 0, 0, 0, 0, time.UTC)
	b := model.Baby{
		ID:                "01ARZ3NDEKTSV4RRFFQ69G5FAV",
		Name:              "Baby",
		Sex:               "female",
		DateOfBirth:       dob,
		DiagnosisDate:     &diag,
		KasaiDate:         &kasai,
		DefaultCalPerFeed: 67.0,
		Notes:             ptrStr("some notes"),
		CreatedAt:         now,
	}
	if b.Sex != "female" {
		t.Errorf("Baby.Sex = %q", b.Sex)
	}
	if b.DateOfBirth != dob {
		t.Errorf("Baby.DateOfBirth = %v", b.DateOfBirth)
	}
	if b.DiagnosisDate == nil || *b.DiagnosisDate != diag {
		t.Errorf("Baby.DiagnosisDate = %v", b.DiagnosisDate)
	}
	if b.KasaiDate == nil || *b.KasaiDate != kasai {
		t.Errorf("Baby.KasaiDate = %v", b.KasaiDate)
	}
	if b.DefaultCalPerFeed != 67.0 {
		t.Errorf("Baby.DefaultCalPerFeed = %f", b.DefaultCalPerFeed)
	}
}

func TestBabyParentFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	bp := model.BabyParent{
		BabyID:   "baby-1",
		UserID:   "user-1",
		Role:     "parent",
		JoinedAt: now,
	}
	if bp.BabyID != "baby-1" {
		t.Errorf("BabyParent.BabyID = %q", bp.BabyID)
	}
	if bp.Role != "parent" {
		t.Errorf("BabyParent.Role = %q", bp.Role)
	}
}

func TestSessionFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	exp := now.Add(30 * 24 * time.Hour)
	s := model.Session{
		ID:        "session-ulid",
		UserID:    "user-1",
		Token:     "random-token",
		ExpiresAt: exp,
		CreatedAt: now,
	}
	if s.ID != "session-ulid" {
		t.Errorf("Session.ID = %q", s.ID)
	}
	if s.Token != "random-token" {
		t.Errorf("Session.Token = %q", s.Token)
	}
	if s.ExpiresAt != exp {
		t.Errorf("Session.ExpiresAt = %v", s.ExpiresAt)
	}
}

func TestInviteFields(t *testing.T) {
	t.Parallel()
	now := time.Now()
	exp := now.Add(24 * time.Hour)
	inv := model.Invite{
		Code:      "483921",
		BabyID:    "baby-1",
		CreatedBy: "user-1",
		UsedBy:    nil,
		UsedAt:    nil,
		ExpiresAt: exp,
		CreatedAt: now,
	}
	if inv.Code != "483921" {
		t.Errorf("Invite.Code = %q", inv.Code)
	}
	if inv.UsedBy != nil {
		t.Errorf("Invite.UsedBy = %v, want nil", inv.UsedBy)
	}
	if inv.ExpiresAt != exp {
		t.Errorf("Invite.ExpiresAt = %v", inv.ExpiresAt)
	}
}

func ptrStr(s string) *string { return &s }
