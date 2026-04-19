package store

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	_ "modernc.org/sqlite"
)

// OpenDB opens a SQLite database at the given path with WAL mode and foreign keys enabled.
// Use ":memory:" for an in-memory database.
func OpenDB(path string) (*sql.DB, error) {
	dsn := "file::memory:?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)"
	if path != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_pragma=journal_mode(wal)&_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)", path)
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := db.Ping(); err != nil {
		db.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return db, nil
}

// RunMigrations applies all .sql files from the given directory in lexicographic order.
// It tracks applied migrations in a _migrations table and skips already-applied ones.
func RunMigrations(db *sql.DB, dir string) error {
	// Create migrations tracking table
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS _migrations (
		filename TEXT PRIMARY KEY,
		applied_at DATETIME DEFAULT CURRENT_TIMESTAMP
	)`)
	if err != nil {
		return fmt.Errorf("create _migrations table: %w", err)
	}

	// Read migration files
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read migrations dir: %w", err)
	}

	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".sql") {
			files = append(files, e.Name())
		}
	}
	sort.Strings(files)

	for _, fname := range files {
		// Check if already applied
		var count int
		err := db.QueryRow("SELECT COUNT(*) FROM _migrations WHERE filename = ?", fname).Scan(&count)
		if err != nil {
			return fmt.Errorf("check migration %s: %w", fname, err)
		}
		if count > 0 {
			continue
		}

		// Read migration file
		content, err := os.ReadFile(filepath.Join(dir, fname))
		if err != nil {
			return fmt.Errorf("read migration %s: %w", fname, err)
		}

		// Apply atomically in a transaction
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("begin tx for %s: %w", fname, err)
		}

		if _, err = tx.Exec(string(content)); err != nil {
			tx.Rollback()
			return fmt.Errorf("apply migration %s: %w", fname, err)
		}

		if _, err = tx.Exec("INSERT INTO _migrations (filename) VALUES (?)", fname); err != nil {
			tx.Rollback()
			return fmt.Errorf("record migration %s: %w", fname, err)
		}

		if err := tx.Commit(); err != nil {
			return fmt.Errorf("commit migration %s: %w", fname, err)
		}
	}

	return nil
}
