package storage_test

import (
	"strings"
	"testing"
	"time"

	"github.com/woodleighschool/woodstar/internal/storage"
	"github.com/woodleighschool/woodstar/internal/testutil/storagecontract"
)

func TestFileBackendContract(t *testing.T) {
	t.Parallel()

	backend, err := storage.New(t.Context(), storage.Config{
		Kind:        storage.KindFile,
		TransferTTL: time.Minute,
		File: storage.FileConfig{
			Root:             t.TempDir(),
			BaseURL:          "http://woodstar.test",
			CapabilityKeyHex: strings.Repeat("42", 32),
		},
	})
	if err != nil {
		t.Fatalf("create file storage: %v", err)
	}
	storagecontract.Run(t, backend)
}
