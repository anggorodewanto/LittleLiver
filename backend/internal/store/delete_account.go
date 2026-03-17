package store

import (
	"database/sql"
	"fmt"
)

// DeleteAccount deletes a user account with full cascade behavior per spec §2.2.
//
// Deletion order:
//  1. Identify babies where the user is the last remaining parent.
//  2. Delete those babies (ON DELETE CASCADE cleans associated data).
//  3. Delete all invites created by the user.
//  4. Anonymize logged_by/updated_by to 'deleted_user' across all tables in
//     anonymizeTables, and anonymize invites.used_by to 'deleted_user'.
//  5. Delete the user record (ON DELETE CASCADE cleans sessions, baby_parents,
//     push_subscriptions).
//
// anonymizeTables is a configurable list of table names that have logged_by and
// updated_by columns. Future phases (e.g., medications, med_logs) add their
// tables to this list when implemented.
func DeleteAccount(db *sql.DB, userID string, anonymizeTables []string) error {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("delete account: begin tx: %w", err)
	}

	// Step 1: Identify babies where this user is the last remaining parent.
	rows, err := tx.Query(
		`SELECT bp.baby_id FROM baby_parents bp
		 WHERE bp.user_id = ?
		 AND (SELECT COUNT(*) FROM baby_parents bp2 WHERE bp2.baby_id = bp.baby_id) = 1`,
		userID,
	)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("delete account: find last-parent babies: %w", err)
	}
	var lastParentBabyIDs []string
	for rows.Next() {
		var babyID string
		if err := rows.Scan(&babyID); err != nil {
			rows.Close()
			tx.Rollback()
			return fmt.Errorf("delete account: scan baby_id: %w", err)
		}
		lastParentBabyIDs = append(lastParentBabyIDs, babyID)
	}
	rows.Close()
	if err := rows.Err(); err != nil {
		tx.Rollback()
		return fmt.Errorf("delete account: iterate last-parent babies: %w", err)
	}

	// Step 2: Delete those babies (CASCADE handles associated data).
	for _, babyID := range lastParentBabyIDs {
		_, err = tx.Exec("DELETE FROM babies WHERE id = ?", babyID)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("delete account: delete baby %s: %w", babyID, err)
		}
	}

	// Step 3: Delete all invites created by the user.
	_, err = tx.Exec("DELETE FROM invites WHERE created_by = ?", userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("delete account: delete invites: %w", err)
	}

	// Step 4: Anonymize logged_by/updated_by across configured tables.
	for _, table := range anonymizeTables {
		_, err = tx.Exec(
			fmt.Sprintf("UPDATE %s SET logged_by = 'deleted_user' WHERE logged_by = ?", table),
			userID,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("delete account: anonymize %s.logged_by: %w", table, err)
		}
		_, err = tx.Exec(
			fmt.Sprintf("UPDATE %s SET updated_by = 'deleted_user' WHERE updated_by = ?", table),
			userID,
		)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("delete account: anonymize %s.updated_by: %w", table, err)
		}
	}

	// Anonymize invites.used_by to 'deleted_user'.
	_, err = tx.Exec("UPDATE invites SET used_by = 'deleted_user' WHERE used_by = ?", userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("delete account: anonymize invites.used_by: %w", err)
	}

	// Step 5: Delete the user record (CASCADE cleans sessions, baby_parents, push_subscriptions).
	_, err = tx.Exec("DELETE FROM users WHERE id = ?", userID)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("delete account: delete user: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("delete account: commit: %w", err)
	}

	return nil
}
