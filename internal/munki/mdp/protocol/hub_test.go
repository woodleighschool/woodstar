package protocol

import (
	"log/slog"
	"testing"
)

func TestNewServerRejectsInvalidBuildVersion(t *testing.T) {
	server, err := NewServer(t.Context(), nil, nil, " invalid ", slog.Default())
	if err == nil || server != nil {
		t.Fatalf("NewServer = (%v, %v), want nil server and error", server, err)
	}
}
