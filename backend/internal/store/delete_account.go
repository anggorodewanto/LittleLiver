package store

import (
	"database/sql"
	"fmt"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

// DeleteAccountResult holds side-effect data from account deletion.
type DeleteAccountResult struct {
	// OrphanedR2Keys contains R2 object keys (originals + thumbnails) from
	// photos belonging to babies that were cascade-deleted. The caller should
	// delete these from R2 after the transaction commits.
	OrphanedR2Keys []string
}

// DeleteAccount deletes a user account with full cascade behavior per spec §2.2.
//
// Deletion order:
//  1. Identify babies where the user is the last remaining parent.
//  1b. Collect R2 photo keys for those babies (before CASCADE deletes the rows).
//  2. Delete those babies (ON DELETE CASCADE cleans associated data).
//  3. Delete all invites created by the user.
//  4. Anonymize logged_by/updated_by to 'deleted_user' across all tables in
//     anonymizeTables, and anonymize invites.used_by to 'deleted_user'.
//  5. Delete the user record (ON DELETE CASCADE cleans sessions, baby_parents,
//     push_subscriptions).
//
// anonymizeTables is a configurable list of table names that have logged_by and
// updated_by columns.
func DeleteAccount(db *sql.DB, userID string, anonymizeTables []string) (*DeleteAccountResult, error) {
	result := &DeleteAccountResult{}

	tx, err := db.Begin()
	if err != nil {
		return nil, fmt.Errorf("delete account: begin tx: %w", err)
	}
	defer tx.Rollback()

	// Step 1: Identify babies where this user is the last remaining parent.
	rows, err := tx.Query(
		`SELECT bp.baby_id FROM baby_parents bp
		 WHERE bp.user_id = ?
		 AND (SELECT COUNT(*) FROM baby_parents bp2 WHERE bp2.baby_id = bp.baby_id) = 1`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("delete account: find last-parent babies: %w", err)
	}
	var lastParentBabyIDs []string
	for rows.Next() {
		var babyID string
		if err := rows.Scan(&babyID); err != nil {
			rows.Close()
			return nil, fmt.Errorf("delete account: scan baby_id: %w", err)
		}
		lastParentBabyIDs = append(lastParentBabyIDs, babyID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("delete account: iterate last-parent babies: %w", err)
	}

	// Step 1b: Collect R2 photo keys for babies about to be cascade-deleted.
	// After CASCADE, the photo_uploads rows are gone so R2 keys would be lost.
	for _, babyID := range lastParentBabyIDs {
		photoRows, err := tx.Query(
			`SELECT r2_key, thumbnail_key FROM photo_uploads WHERE baby_id = ?`,
			babyID,
		)
		if err != nil {
			return nil, fmt.Errorf("delete account: query photos for baby %s: %w", babyID, err)
		}
		for photoRows.Next() {
			var r2Key string
			var thumbKey sql.NullString
			if err := photoRows.Scan(&r2Key, &thumbKey); err != nil {
				photoRows.Close()
				return nil, fmt.Errorf("delete account: scan photo key: %w", err)
			}
			result.OrphanedR2Keys = append(result.OrphanedR2Keys, r2Key)
			if thumbKey.Valid && thumbKey.String != "" {
				result.OrphanedR2Keys = append(result.OrphanedR2Keys, thumbKey.String)
			}
		}
		photoRows.Close()
		if err := photoRows.Err(); err != nil {
			return nil, fmt.Errorf("delete account: iterate photo keys: %w", err)
		}
	}

	// Step 2: Delete those babies (CASCADE handles associated data).
	for _, babyID := range lastParentBabyIDs {
		_, err = tx.Exec("DELETE FROM babies WHERE id = ?", babyID)
		if err != nil {
			return nil, fmt.Errorf("delete account: delete baby %s: %w", babyID, err)
		}
	}

	// Step 3: Delete all invites created by the user.
	_, err = tx.Exec("DELETE FROM invites WHERE created_by = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("delete account: delete invites: %w", err)
	}

	// Step 4: Anonymize logged_by/updated_by across configured tables.
	for _, table := range anonymizeTables {
		_, err = tx.Exec(
			fmt.Sprintf("UPDATE %s SET logged_by = ? WHERE logged_by = ?", table),
			model.DeletedUserSentinel, userID,
		)
		if err != nil {
			return nil, fmt.Errorf("delete account: anonymize %s.logged_by: %w", table, err)
		}
		_, err = tx.Exec(
			fmt.Sprintf("UPDATE %s SET updated_by = ? WHERE updated_by = ?", table),
			model.DeletedUserSentinel, userID,
		)
		if err != nil {
			return nil, fmt.Errorf("delete account: anonymize %s.updated_by: %w", table, err)
		}
	}

	// Anonymize invites.used_by.
	_, err = tx.Exec("UPDATE invites SET used_by = ? WHERE used_by = ?", model.DeletedUserSentinel, userID)
	if err != nil {
		return nil, fmt.Errorf("delete account: anonymize invites.used_by: %w", err)
	}

	// Step 5: Delete the user record (CASCADE cleans sessions, baby_parents, push_subscriptions).
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		return nil, fmt.Errorf("delete account: delete user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("delete account: commit: %w", err)
	}

	return result, nil
}
