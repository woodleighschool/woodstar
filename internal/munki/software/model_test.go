package software

import (
	"errors"
	"testing"

	"github.com/woodleighschool/woodstar/internal/dbutil"
)

func TestCreateMutationRejectsAmbiguousMunkiNames(t *testing.T) {
	for _, name := range []string{"App/Installer", "App-1", "App--1.2.3"} {
		t.Run(name, func(t *testing.T) {
			mutation := CreateMutation{Name: name}
			mutation.normalize()
			if err := mutation.validate(); !errors.Is(err, dbutil.ErrInvalidInput) {
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

	mutation.normalize()

	if mutation.Name != "Caf\u00e9" {
		t.Fatalf("name = %q, want NFC-normalized name", mutation.Name)
	}
	if mutation.DisplayName != "" {
		t.Fatalf("display name = %q, want redundant value removed", mutation.DisplayName)
	}
}
