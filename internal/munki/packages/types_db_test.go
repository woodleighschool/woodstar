package packages

import (
	"encoding/json"
	"testing"
)

func TestScanJSONSlice(t *testing.T) {
	t.Run("nil yields empty non-nil slice", func(t *testing.T) {
		var v packageInstallItems
		if err := v.Scan(nil); err != nil {
			t.Fatalf("Scan(nil): %v", err)
		}
		if v == nil {
			t.Fatal("expected non-nil slice after Scan(nil)")
		}
		if len(v) != 0 {
			t.Fatalf("expected empty slice, got len %d", len(v))
		}
	})

	t.Run("empty JSON array yields empty non-nil slice", func(t *testing.T) {
		var v packageInstallItems
		if err := v.Scan([]byte("[]")); err != nil {
			t.Fatalf("Scan([]byte): %v", err)
		}
		if v == nil {
			t.Fatal("expected non-nil slice after Scan([]byte(\"[]\"))")
		}
		if len(v) != 0 {
			t.Fatalf("expected empty slice, got len %d", len(v))
		}
	})

	t.Run("populated array round-trips", func(t *testing.T) {
		input := packageInstallItems{
			{
				Path:             "/Applications/Foo.app",
				Type:             "application",
				BundleIdentifier: "com.example.foo",
				BundleVersion:    "1.0",
			},
		}
		data, err := json.Marshal(input)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		var got packageInstallItems
		if err := got.Scan(data); err != nil {
			t.Fatalf("Scan: %v", err)
		}
		if len(got) != 1 {
			t.Fatalf("expected 1 item, got %d", len(got))
		}
		if got[0].Path != input[0].Path {
			t.Errorf("Path: got %q, want %q", got[0].Path, input[0].Path)
		}
		if got[0].BundleIdentifier != input[0].BundleIdentifier {
			t.Errorf("BundleIdentifier: got %q, want %q", got[0].BundleIdentifier, input[0].BundleIdentifier)
		}
	})
}

func TestJSONSliceValue(t *testing.T) {
	t.Run("nil slice yields empty JSON array not NULL", func(t *testing.T) {
		var v packageInstallItems
		val, err := v.Value()
		if err != nil {
			t.Fatalf("Value(): %v", err)
		}
		b, ok := val.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", val)
		}
		if string(b) != "[]" {
			t.Errorf("expected %q, got %q", "[]", string(b))
		}
	})

	t.Run("populated slice yields correct JSON", func(t *testing.T) {
		v := packageReceipts{
			{PackageID: "com.example.pkg", Version: "2.0", InstalledSize: 1024},
		}
		val, err := v.Value()
		if err != nil {
			t.Fatalf("Value(): %v", err)
		}
		b, ok := val.([]byte)
		if !ok {
			t.Fatalf("expected []byte, got %T", val)
		}
		var out packageReceipts
		if err := json.Unmarshal(b, &out); err != nil {
			t.Fatalf("unmarshal: %v", err)
		}
		if len(out) != 1 {
			t.Fatalf("expected 1 item, got %d", len(out))
		}
		if out[0].PackageID != v[0].PackageID {
			t.Errorf("PackageID: got %q, want %q", out[0].PackageID, v[0].PackageID)
		}
		if out[0].InstalledSize != v[0].InstalledSize {
			t.Errorf("InstalledSize: got %d, want %d", out[0].InstalledSize, v[0].InstalledSize)
		}
	})
}
