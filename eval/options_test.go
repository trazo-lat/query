package eval

import (
	"testing"

	"github.com/trazo-lat/query/validate"
)

func TestWithMaxLength(t *testing.T) {
	fields := []validate.FieldConfig{
		{Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
	}

	tests := []struct {
		name    string
		maxLen  int
		wantErr bool
	}{
		{"exceeds limit", 5, true},
		{"within limit", 100, false},
		{"disabled", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile("state=draft", fields, WithMaxLength(tt.maxLen))
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
		})
	}
}

func TestWithMaxDepth(t *testing.T) {
	fields := []validate.FieldConfig{
		{Name: "a", Type: validate.TypeInteger, AllowedOps: validate.NumericOps},
		{Name: "b", Type: validate.TypeInteger, AllowedOps: validate.NumericOps},
		{Name: "c", Type: validate.TypeInteger, AllowedOps: validate.NumericOps},
	}
	tests := []struct {
		name    string
		query   string
		max     int
		wantErr bool
	}{
		{"depth 1 allowed", "a=1", 1, false},
		{"depth exceeds limit", "(a=1 OR b=2) AND c=3", 2, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := Compile(tt.query, fields, WithMaxDepth(tt.max))
			if (err != nil) != tt.wantErr {
				t.Errorf("wantErr=%v, got err=%v", tt.wantErr, err)
			}
		})
	}
}

func TestCompileArgResolvers_NestedCall(t *testing.T) {
	fields := []validate.FieldConfig{
		{Name: "name", Type: validate.TypeText, AllowedOps: validate.TextOps},
	}
	prog, err := Compile("lower(upper(name))=john", fields)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	tests := []struct {
		name  string
		input string
	}{
		{"uppercase input", "JOHN"},
		{"mixed case", "John"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if !prog.Match(map[string]any{"name": tt.input}) {
				t.Error("expected match")
			}
		})
	}
}

func TestCompileArgResolvers_MissingFunc(t *testing.T) {
	fields := []validate.FieldConfig{
		{Name: "name", Type: validate.TypeText, AllowedOps: validate.TextOps},
	}
	_, err := Compile("lower(nonexistent(name))=john", fields,
		WithNoBuiltins(),
		WithFunctions(Func{
			Name: "lower",
			Call: func(args ...any) (any, error) {
				if args[0] == nil {
					return "", nil
				}
				return args[0], nil
			},
		}),
	)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
}
