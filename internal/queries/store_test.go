package queries

import (
	"strings"
	"testing"
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

func assertStringPtr(t *testing.T, name string, got *string, want *string) {
	t.Helper()
	switch {
	case got == nil && want == nil:
		return
	case got == nil || want == nil:
		t.Fatalf("%s = %v, want %v", name, got, want)
	case *got != *want:
		t.Fatalf("%s = %q, want %q", name, *got, *want)
	}
}
