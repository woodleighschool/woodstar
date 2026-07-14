package validation

import "testing"

func TestStructValidatesFieldsAndHTTPSOrigins(t *testing.T) {
	t.Parallel()

	type input struct {
		Name   string `json:"name"   validate:"required"`
		Origin string `json:"origin" validate:"https_origin"`
	}

	if err := Struct(input{Name: "woodstar", Origin: "https://example.com/"}); err != nil {
		t.Fatalf("Struct() = %v, want nil", err)
	}

	tests := []struct {
		name string
		in   input
		want string
	}{
		{name: "required", in: input{Origin: "https://example.com"}, want: "Name is required"},
		{
			name: "https",
			in:   input{Name: "woodstar", Origin: "http://example.com"},
			want: "Origin must be an HTTPS origin",
		},
		{
			name: "path",
			in:   input{Name: "woodstar", Origin: "https://example.com/path"},
			want: "Origin must be an HTTPS origin",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			if err := Struct(test.in); err == nil || err.Error() != test.want {
				t.Fatalf("Struct() = %v, want %q", err, test.want)
			}
		})
	}
}
