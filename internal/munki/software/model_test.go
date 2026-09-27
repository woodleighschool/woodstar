package software

import (
	"errors"
	"reflect"
	"testing"

	"github.com/woodleighschool/woodstar/internal/fault"
)

func TestIconURLUsesStorageObjectIdentity(t *testing.T) {
	objectID := int64(42)
	if got := IconURL(&objectID); got != "/api/munki/icons/42/content" {
		t.Fatalf("IconURL() = %q, want object-specific content URL", got)
	}

	zero := int64(0)
	if got := IconURL(&zero); got != "" {
		t.Fatalf("IconURL() with zero object ID = %q, want empty URL", got)
	}
	if got := IconURL(nil); got != "" {
		t.Fatalf("IconURL() without object ID = %q, want empty URL", got)
	}
}

func TestCreateMutationRejectsAmbiguousMunkiNames(t *testing.T) {
	for _, name := range []string{"App/Installer", "App-1", "App--1.2.3"} {
		t.Run(name, func(t *testing.T) {
			mutation := CreateMutation{Name: name}
			mutation.Normalize()
			if err := mutation.Validate(); !errors.Is(err, fault.ErrInvalidInput) {
				t.Fatalf("validate() error = %v, want invalid input", err)
			}
		})
	}
}

func TestCreateMutationNormalizesMunkiMetadata(t *testing.T) {
	mutation := CreateMutation{
		Name:        " Cafe\u0301 ",
		DisplayName: "Caf\u00e9",
	}

	mutation.Normalize()

	if mutation.Name != "Caf\u00e9" {
		t.Fatalf("name = %q, want NFC-normalized name", mutation.Name)
	}
	if mutation.DisplayName != "" {
		t.Fatalf("display name = %q, want redundant value removed", mutation.DisplayName)
	}
}

// PATCH merges into this projection, so an unprojected field would be reset.
func TestSoftwareMutationProjectsEveryEditableField(t *testing.T) {
	displayName := "Example"
	iconObjectID := int64(1)
	title := Software{
		DisplayName:  &displayName,
		Description:  "Description",
		Category:     "Category",
		Developer:    "Developer",
		IconObjectID: &iconObjectID,
	}
	mutation := reflect.ValueOf(title.Mutation(Targets{Include: []Include{{LabelID: 1}}}))
	for i := range mutation.NumField() {
		if mutation.Field(i).IsZero() {
			t.Errorf("Mutation() does not project %s", mutation.Type().Field(i).Name)
		}
	}
}
