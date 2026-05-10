package models

import (
	"encoding/json"
	"strings"
	"testing"
	"time"
)

func TestCleanQueryCreate(t *testing.T) {
	tests := []struct {
		name    string
		in      QueryCreate
		want    QueryCreate
		wantErr string
	}{
		{
			name: "saved query defaults to unscheduled snapshot",
			in: QueryCreate{
				Name:        " Local admins ",
				Description: " Users with admin rights ",
				Query:       " select * from users; ",
				Platform:    new(" darwin "),
			},
			want: QueryCreate{
				Name:        "Local admins",
				Description: "Users with admin rights",
				Query:       "select * from users;",
				Platform:    new("darwin"),
				LoggingType: QueryLoggingSnapshot,
			},
		},
		{
			name: "scheduled report keeps interval",
			in: QueryCreate{
				Name:             "Battery health",
				Query:            "select * from battery;",
				ScheduleInterval: 3600,
			},
			want: QueryCreate{
				Name:             "Battery health",
				Query:            "select * from battery;",
				ScheduleInterval: 3600,
				LoggingType:      QueryLoggingSnapshot,
			},
		},
		{
			name:    "missing name is invalid",
			in:      QueryCreate{Query: "select 1;"},
			wantErr: "name is required",
		},
		{
			name:    "missing sql is invalid",
			in:      QueryCreate{Name: "No SQL"},
			wantErr: "query is required",
		},
		{
			name: "negative schedule is invalid",
			in: QueryCreate{
				Name:             "Bad schedule",
				Query:            "select 1;",
				ScheduleInterval: -1,
			},
			wantErr: "schedule interval cannot be negative",
		},
		{
			name: "unsupported logging type is invalid",
			in: QueryCreate{
				Name:        "Differential",
				Query:       "select 1;",
				LoggingType: "differential",
			},
			wantErr: "logging type must be snapshot",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got, err := cleanQueryCreate(tt.in)
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("cleanQueryCreate error = %v, want containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("cleanQueryCreate returned error: %v", err)
			}
			assertQueryCreate(t, got, tt.want)
		})
	}
}

func TestCleanCheckCreate(t *testing.T) {
	got, err := cleanCheckCreate(CheckCreate{
		Name:        " Gatekeeper enabled ",
		Description: " Security check ",
		Query:       " select 1 from gatekeeper where assessments_enabled = 1; ",
		Platform:    new(" darwin "),
	})
	if err != nil {
		t.Fatalf("cleanCheckCreate returned error: %v", err)
	}
	if got.Name != "Gatekeeper enabled" {
		t.Fatalf("Name = %q, want Gatekeeper enabled", got.Name)
	}
	if got.Query != "select 1 from gatekeeper where assessments_enabled = 1;" {
		t.Fatalf("Query = %q, want trimmed SQL", got.Query)
	}
	assertStringPtr(t, "Platform", got.Platform, new("darwin"))
}

func TestNormalizeLabelScope(t *testing.T) {
	scope := NormalizeLabelScope(LabelScope{
		Mode:     ScopeExcludeAny,
		LabelIDs: []int64{5, 2, 5, 0, -1},
	})
	if scope.Mode != ScopeExcludeAny {
		t.Fatalf("Mode = %q, want %q", scope.Mode, ScopeExcludeAny)
	}
	assertInt64s(t, "LabelIDs", scope.LabelIDs, []int64{2, 5})

	empty := NormalizeLabelScope(LabelScope{Mode: ScopeIncludeAll})
	if empty.Mode != ScopeNone {
		t.Fatalf("empty Mode = %q, want %q", empty.Mode, ScopeNone)
	}
}

func TestSnapshotResultRowsStoreEachOsqueryRowSeparately(t *testing.T) {
	fetchedAt := time.Date(2026, 5, 9, 10, 30, 0, 0, time.UTC)
	rows, err := snapshotResultRows([]map[string]string{
		{"name": "alpha", "version": "1"},
		{"name": "bravo", "version": "2"},
	}, fetchedAt)
	if err != nil {
		t.Fatalf("snapshotResultRows returned error: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("len(rows) = %d, want 2", len(rows))
	}
	for i, row := range rows {
		if row.data == nil {
			t.Fatalf("rows[%d].data is nil, want JSON object", i)
		}
		var got map[string]string
		if err := json.Unmarshal(*row.data, &got); err != nil {
			t.Fatalf("unmarshal rows[%d]: %v", i, err)
		}
		if got["name"] == "" || got["version"] == "" {
			t.Fatalf("rows[%d] data = %#v, want osquery columns", i, got)
		}
		if !row.lastFetched.Equal(fetchedAt) {
			t.Fatalf("rows[%d].lastFetched = %s, want %s", i, row.lastFetched, fetchedAt)
		}
	}
}

func TestSnapshotResultRowsPreserveEmptyFetchWithNullData(t *testing.T) {
	fetchedAt := time.Date(2026, 5, 9, 10, 30, 0, 0, time.UTC)
	rows, err := snapshotResultRows(nil, fetchedAt)
	if err != nil {
		t.Fatalf("snapshotResultRows returned error: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("len(rows) = %d, want 1", len(rows))
	}
	if rows[0].data != nil {
		t.Fatalf("rows[0].data = %s, want nil sentinel", string(*rows[0].data))
	}
	if !rows[0].lastFetched.Equal(fetchedAt) {
		t.Fatalf("rows[0].lastFetched = %s, want %s", rows[0].lastFetched, fetchedAt)
	}
}

func assertQueryCreate(t *testing.T, got QueryCreate, want QueryCreate) {
	t.Helper()
	if got.Name != want.Name {
		t.Fatalf("Name = %q, want %q", got.Name, want.Name)
	}
	if got.Description != want.Description {
		t.Fatalf("Description = %q, want %q", got.Description, want.Description)
	}
	if got.Query != want.Query {
		t.Fatalf("Query = %q, want %q", got.Query, want.Query)
	}
	if got.ScheduleInterval != want.ScheduleInterval {
		t.Fatalf("ScheduleInterval = %d, want %d", got.ScheduleInterval, want.ScheduleInterval)
	}
	if got.LoggingType != want.LoggingType {
		t.Fatalf("LoggingType = %q, want %q", got.LoggingType, want.LoggingType)
	}
	assertStringPtr(t, "Platform", got.Platform, want.Platform)
}

func assertInt64s(t *testing.T, name string, got []int64, want []int64) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("%s = %#v, want %#v", name, got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("%s = %#v, want %#v", name, got, want)
		}
	}
}
