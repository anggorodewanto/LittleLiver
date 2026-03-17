package model_test

import (
	"strings"
	"testing"
	"time"

	"github.com/ablankz/LittleLiver/backend/internal/model"
)

func TestNewULID_ProducesValidFormat(t *testing.T) {
	t.Parallel()
	id := model.NewULID()
	// ULIDs are 26 characters, Crockford's Base32
	if len(id) != 26 {
		t.Fatalf("ULID length = %d, want 26", len(id))
	}
	const crockford = "0123456789ABCDEFGHJKMNPQRSTVWXYZ"
	for i, c := range id {
		if !strings.ContainsRune(crockford, c) {
			t.Fatalf("ULID char %d = %c, not in Crockford Base32", i, c)
		}
	}
}

func TestNewULID_Unique(t *testing.T) {
	t.Parallel()
	seen := make(map[string]bool)
	for i := 0; i < 1000; i++ {
		id := model.NewULID()
		if seen[id] {
			t.Fatalf("duplicate ULID: %s", id)
		}
		seen[id] = true
	}
}

func TestNewULID_Sortable(t *testing.T) {
	t.Parallel()
	// Generate two ULIDs with a time gap; the later one must be lexicographically greater.
	id1 := model.NewULID()
	time.Sleep(2 * time.Millisecond)
	id2 := model.NewULID()
	if id1 >= id2 {
		t.Fatalf("ULIDs not sortable: %s >= %s", id1, id2)
	}
}
