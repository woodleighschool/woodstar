//go:build postgres

package inventory

import (
	"testing"

	"github.com/woodleighschool/woodstar/internal/testutil/testdb"
)

func TestGetTitleLoadsVersionCollection(t *testing.T) {
	db, ctx := testdb.Open(t)
	store := NewStore(db)

	var titleID int64
	if err := db.Pool().QueryRow(ctx, `
		INSERT INTO software_titles (name, source, bundle_identifier)
		VALUES ('Versioned App', 'apps', 'com.example.versioned')
		RETURNING id
	`).Scan(&titleID); err != nil {
		t.Fatalf("insert software title: %v", err)
	}
	if _, err := db.Pool().Exec(ctx, `
		INSERT INTO software (title_id, name, version, source, bundle_identifier)
		VALUES
			($1, 'Versioned App', '2.0', 'apps', 'com.example.versioned'),
			($1, 'Versioned App', '1.0', 'apps', 'com.example.versioned')
	`, titleID); err != nil {
		t.Fatalf("insert software versions: %v", err)
	}

	title, err := store.GetTitle(ctx, titleID)
	if err != nil {
		t.Fatalf("GetTitle: %v", err)
	}
	if title.Versions.Count != 2 {
		t.Fatalf("version count = %d, want 2", title.Versions.Count)
	}
	if len(title.Versions.Items) != 2 {
		t.Fatalf("version items = %d, want 2", len(title.Versions.Items))
	}
	if got := title.Versions.Items[0].Version; got != "1.0" {
		t.Fatalf("first version = %q, want 1.0", got)
	}
	if got := title.Versions.Items[1].Version; got != "2.0" {
		t.Fatalf("second version = %q, want 2.0", got)
	}
}
