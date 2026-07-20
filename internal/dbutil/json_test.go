package dbutil

import (
	"encoding/json"
	"testing"
)

type jsonSliceItem struct {
	Name string `json:"name"`
}

func TestJSONSliceScan(t *testing.T) {
	emptyValues := []struct {
		name string
		src  any
	}{
		{name: "SQL null", src: nil},
		{name: "empty bytes", src: []byte{}},
		{name: "empty string", src: ""},
		{name: "JSON null bytes", src: []byte("null")},
		{name: "JSON null string", src: "null"},
		{name: "empty array bytes", src: []byte("[]")},
		{name: "empty array string", src: "[]"},
	}
	for _, tt := range emptyValues {
		t.Run(tt.name, func(t *testing.T) {
			values := JSONSlice[jsonSliceItem]{{Name: "stale"}}
			if err := values.Scan(tt.src); err != nil {
				t.Fatalf("Scan(%#v): %v", tt.src, err)
			}
			if values == nil {
				t.Fatal("Scan returned a nil slice")
			}
			if len(values) != 0 {
				t.Fatalf("Scan returned %#v, want an empty slice", values)
			}
		})
	}

	populatedValues := []struct {
		name string
		src  any
	}{
		{name: "bytes", src: []byte(`[{"name":"one"},{"name":"two"}]`)},
		{name: "string", src: `[{"name":"one"},{"name":"two"}]`},
	}
	for _, tt := range populatedValues {
		t.Run("populated "+tt.name, func(t *testing.T) {
			var values JSONSlice[jsonSliceItem]
			if err := values.Scan(tt.src); err != nil {
				t.Fatalf("Scan: %v", err)
			}
			if len(values) != 2 || values[0].Name != "one" || values[1].Name != "two" {
				t.Fatalf("Scan returned %#v", values)
			}
		})
	}

	t.Run("invalid JSON", func(t *testing.T) {
		var values JSONSlice[jsonSliceItem]
		if err := values.Scan(`[invalid]`); err == nil {
			t.Fatal("Scan returned nil error")
		}
	})

	t.Run("non-array JSON", func(t *testing.T) {
		var values JSONSlice[jsonSliceItem]
		if err := values.Scan(`{"name":"one"}`); err == nil {
			t.Fatal("Scan returned nil error")
		}
	})

	t.Run("unsupported source", func(t *testing.T) {
		var values JSONSlice[jsonSliceItem]
		if err := values.Scan(42); err == nil {
			t.Fatal("Scan returned nil error")
		}
	})
}

func TestJSONSliceValue(t *testing.T) {
	t.Run("nil slice", func(t *testing.T) {
		var values JSONSlice[jsonSliceItem]
		value, err := values.Value()
		if err != nil {
			t.Fatalf("Value: %v", err)
		}
		data, ok := value.([]byte)
		if !ok {
			t.Fatalf("Value returned %T, want []byte", value)
		}
		if string(data) != "[]" {
			t.Fatalf("Value returned %q, want []", data)
		}
	})

	t.Run("populated slice", func(t *testing.T) {
		values := JSONSlice[jsonSliceItem]{{Name: "one"}, {Name: "two"}}
		value, err := values.Value()
		if err != nil {
			t.Fatalf("Value: %v", err)
		}
		data, ok := value.([]byte)
		if !ok {
			t.Fatalf("Value returned %T, want []byte", value)
		}
		var decoded []jsonSliceItem
		if err := json.Unmarshal(data, &decoded); err != nil {
			t.Fatalf("decode Value result: %v", err)
		}
		if len(decoded) != 2 || decoded[0].Name != "one" || decoded[1].Name != "two" {
			t.Fatalf("Value decoded to %#v", decoded)
		}
	})
}
