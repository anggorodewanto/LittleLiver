package store

import (
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// InviteExpiry is how long an invite code is valid.
const InviteExpiry = 24 * time.Hour

// maxRetries is the number of times to retry on code collision.
const maxRetries = 5

// ErrInvalidInvite is returned when a code is expired, used, or does not exist.
var ErrInvalidInvite = errors.New("invalid or expired code")

// ErrAlreadyLinked is returned when the user is already a parent of the baby.
var ErrAlreadyLinked = errors.New("already linked to this baby")

// generateCode returns a cryptographically random 6-digit numeric string.
func generateCode() (string, error) {
	n, err := rand.Int(rand.Reader, big.NewInt(1000000))
	if err != nil {
		return "", fmt.Errorf("generate invite code: %w", err)
	}
	return fmt.Sprintf("%06d", n.Int64()), nil
}

// CreateInvite generates a new 6-digit invite code for the given baby,
// hard-deleting all prior codes for that baby first.
// On UNIQUE constraint collision, it retries up to 5 times.
func CreateInvite(db *sql.DB, babyID, createdBy string) (*model.Invite, error) {
	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("create invite: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Hard-delete all prior codes for this baby
	_, err = tx.Exec("DELETE FROM invites WHERE baby_id = ?", babyID)
	if err != nil {
		return nil, fmt.Errorf("create invite: delete prior: %w", err)
	}

	now := time.Now().UTC()
	expiresAt := now.Add(InviteExpiry)

	var code string
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		code, err = generateCode()
		if err != nil {
			return nil, err
		}

		_, lastErr = tx.Exec(
			"INSERT INTO invites (code, baby_id, created_by, expires_at) VALUES (?, ?, ?, ?)",
			code, babyID, createdBy, expiresAt.Format(time.DateTime),
		)
		if lastErr == nil {
			break
		}
		// Only retry on UNIQUE constraint violations; other errors are fatal
		if !strings.Contains(lastErr.Error(), "UNIQUE constraint") {
			return nil, fmt.Errorf("create invite: insert: %w", lastErr)
		}
	}
	if lastErr != nil {
		return nil, fmt.Errorf("create invite: max retries exceeded: %w", lastErr)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("create invite: commit: %w", err)
	}

	return &model.Invite{
		Code:      code,
		BabyID:    babyID,
		CreatedBy: createdBy,
		ExpiresAt: expiresAt,
		CreatedAt: now,
	}, nil
}

// RedeemInvite redeems an invite code, linking the user to the baby.
// Returns the baby ID on success.
// Returns ErrInvalidInvite if the code is invalid, expired, or already used.
// If the user is already linked, returns the baby ID without error (no-op).
func RedeemInvite(db *sql.DB, code, userID string) (string, error) {
	tx, err := db.Begin()
	if err != nil {
		return "", fmt.Errorf("redeem invite: begin tx: %w", err)
	}
	defer tx.Rollback()

	var babyID string
	var usedAt sql.NullString
	var expiresStr string
	err = tx.QueryRow(
		"SELECT baby_id, used_at, expires_at FROM invites WHERE code = ?", code,
	).Scan(&babyID, &usedAt, &expiresStr)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrInvalidInvite
		}
		return "", fmt.Errorf("redeem invite: query: %w", err)
	}

	// Check if already used
	if usedAt.Valid {
		return "", ErrInvalidInvite
	}

	// Check if expired
	expiresAtTime, err := ParseTime(expiresStr)
	if err != nil {
		return "", fmt.Errorf("redeem invite: parse expires_at: %w", err)
	}
	if time.Now().UTC().After(expiresAtTime) {
		return "", ErrInvalidInvite
	}

	// Check if user is already linked to this baby
	var linkCount int
	err = tx.QueryRow(
		"SELECT COUNT(*) FROM baby_parents WHERE baby_id = ? AND user_id = ?",
		babyID, userID,
	).Scan(&linkCount)
	if err != nil {
		return "", fmt.Errorf("redeem invite: check link: %w", err)
	}
	if linkCount > 0 {
		// Already linked — return baby ID with ErrAlreadyLinked, do not mark code as used
		return babyID, ErrAlreadyLinked
	}

	// Mark invite as used
	_, err = tx.Exec(
		"UPDATE invites SET used_by = ?, used_at = ? WHERE code = ?",
		userID, time.Now().UTC().Format(time.DateTime), code,
	)
	if err != nil {
		return "", fmt.Errorf("redeem invite: mark used: %w", err)
	}

	// Link user to baby
	_, err = tx.Exec(
		"INSERT INTO baby_parents (baby_id, user_id) VALUES (?, ?)",
		babyID, userID,
	)
	if err != nil {
		return "", fmt.Errorf("redeem invite: link parent: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return "", fmt.Errorf("redeem invite: commit: %w", err)
	}

	return babyID, nil
}
