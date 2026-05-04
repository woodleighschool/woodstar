package osquery

import "testing"

func TestDetailQueryRegistryIsComplete(t *testing.T) {
	queries := DetailQueries()
	if len(queries) != 4 {
		t.Fatalf("len(DetailQueries()) = %d, want 4", len(queries))
	}

	for name, query := range queries {
		if name == "" {
			t.Fatal("query name is empty")
		}
		if query.SQL == "" {
			t.Fatalf("%s SQL is empty", name)
		}
		if query.Ingest == nil {
			t.Fatalf("%s ingest func is nil", name)
		}
	}
}

func TestDetailQueriesDue(t *testing.T) {
	if got := detailQueriesDue(nil); len(got) != 4 {
		t.Fatalf("nil timestamp returned %d queries, want 4", len(got))
	}
}
