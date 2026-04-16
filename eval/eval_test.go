package eval

import (
	"testing"
	"time"

	"github.com/trazo-lat/query/validate"
)

var testFields = []validate.FieldConfig{
	{Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
	{Name: "name", Type: validate.TypeText, AllowedOps: validate.TextOps},
	{Name: "year", Type: validate.TypeInteger, AllowedOps: validate.NumericOps},
	{Name: "total", Type: validate.TypeDecimal, AllowedOps: validate.NumericOps},
	{Name: "active", Type: validate.TypeBoolean, AllowedOps: validate.BoolOps},
	{Name: "created_at", Type: validate.TypeDate, AllowedOps: validate.DateOps},
	{Name: "ttl", Type: validate.TypeDuration, AllowedOps: validate.DurationOps, Nested: true},
	{Name: "labels", Type: validate.TypeText, AllowedOps: validate.TextOps, Nested: true},
	{Name: "cluster", Type: validate.TypeText, AllowedOps: validate.TextOps},
	{Name: "tire_size", Type: validate.TypeText, AllowedOps: append(validate.TextOps, validate.OpPresence)},
}

func TestCompile_Match(t *testing.T) {
	tests := []struct {
		query string
		data  map[string]any
		want  bool
	}{
		// Equality
		{"state=draft", map[string]any{"state": "draft"}, true},
		{"state=draft", map[string]any{"state": "published"}, false},
		{"state=draft", map[string]any{}, false},

		// Not equal
		{"state!=cancelled", map[string]any{"state": "draft"}, true},
		{"state!=cancelled", map[string]any{"state": "cancelled"}, false},

		// Integer comparison
		{"year>2020", map[string]any{"year": 2025}, true},
		{"year>2020", map[string]any{"year": 2020}, false},
		{"year>2020", map[string]any{"year": int64(2025)}, true},
		{"year>=2020", map[string]any{"year": 2020}, true},
		{"year<2025", map[string]any{"year": 2020}, true},
		{"year<=2025", map[string]any{"year": 2025}, true},

		// Decimal
		{"total>=50000", map[string]any{"total": 60000.0}, true},
		{"total>=50000", map[string]any{"total": 100.0}, false},

		// Boolean
		{"active=true", map[string]any{"active": true}, true},
		{"active=true", map[string]any{"active": false}, false},

		// Date
		{"created_at>=2026-01-01", map[string]any{"created_at": time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC)}, true},
		{"created_at>=2026-01-01", map[string]any{"created_at": time.Date(2025, 6, 15, 0, 0, 0, 0, time.UTC)}, false},
		{"created_at>=2026-01-01", map[string]any{"created_at": "2026-06-15"}, true},

		// Wildcard
		{"name=John*", map[string]any{"name": "John Doe"}, true},
		{"name=John*", map[string]any{"name": "Jane Doe"}, false},
		{"name=*ohn", map[string]any{"name": "John"}, true},
		{"name=*testing*", map[string]any{"name": "some testing here"}, true},

		// Presence
		{"tire_size", map[string]any{"tire_size": "225/45R17"}, true},
		{"tire_size", map[string]any{}, false},

		// AND
		{"state=draft AND year>2020", map[string]any{"state": "draft", "year": 2025}, true},
		{"state=draft AND year>2020", map[string]any{"state": "draft", "year": 2019}, false},
		{"state=draft AND year>2020", map[string]any{"state": "published", "year": 2025}, false},

		// OR
		{"state=draft OR state=issued", map[string]any{"state": "draft"}, true},
		{"state=draft OR state=issued", map[string]any{"state": "issued"}, true},
		{"state=draft OR state=issued", map[string]any{"state": "cancelled"}, false},

		// NOT
		{"NOT state=cancelled", map[string]any{"state": "draft"}, true},
		{"NOT state=cancelled", map[string]any{"state": "cancelled"}, false},

		// Grouped
		{"(state=draft OR state=issued) AND total>50000",
			map[string]any{"state": "draft", "total": 60000.0}, true},
		{"(state=draft OR state=issued) AND total>50000",
			map[string]any{"state": "draft", "total": 100.0}, false},

		// Nested fields
		{"labels.dev=jane", map[string]any{"labels.dev": "jane"}, true},
		{"labels.dev=jane", map[string]any{"labels.dev": "john"}, false},

		// Case-insensitive string match
		{"state=Draft", map[string]any{"state": "draft"}, true},
	}

	for _, tt := range tests {
		t.Run(tt.query, func(t *testing.T) {
			prog, err := Compile(tt.query, testFields)
			if err != nil {
				t.Fatalf("compile: %v", err)
			}
			got := prog.Match(tt.data)
			if got != tt.want {
				t.Errorf("Match(%v) = %v, want %v", tt.data, got, tt.want)
			}
		})
	}
}

func TestCompile_Range(t *testing.T) {
	prog, err := Compile("created_at:2026-01-01..2026-03-31", testFields)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	tests := []struct {
		date string
		want bool
	}{
		{"2026-02-15", true},
		{"2026-01-01", true},
		{"2026-03-31", true},
		{"2025-12-31", false},
		{"2026-04-01", false},
	}

	for _, tt := range tests {
		t.Run(tt.date, func(t *testing.T) {
			d, _ := time.Parse("2006-01-02", tt.date)
			got := prog.Match(map[string]any{"created_at": d})
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestCompile_MatchFunc(t *testing.T) {
	prog, err := Compile("state=draft", testFields)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}

	data := map[string]string{"state": "draft"}
	got := prog.MatchFunc(func(field string) (any, bool) {
		v, ok := data[field]
		return v, ok
	})
	if !got {
		t.Error("expected match")
	}
}

func TestCompile_Fields(t *testing.T) {
	prog, err := Compile("state=draft AND total>50000", testFields)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	fields := prog.Fields()
	if len(fields) != 2 {
		t.Errorf("got %d fields, want 2", len(fields))
	}
}

func TestCompile_StringRoundTrip(t *testing.T) {
	q := "state=draft AND total>50000"
	prog, err := Compile(q, testFields)
	if err != nil {
		t.Fatalf("compile: %v", err)
	}
	if prog.String() != q {
		t.Errorf("String: got %q, want %q", prog.String(), q)
	}
	if prog.Stringify() != q {
		t.Errorf("Stringify: got %q, want %q", prog.Stringify(), q)
	}
}

func TestCompile_WithAllowedFields(t *testing.T) {
	_, err := Compile("state=draft AND total>50000", testFields,
		WithAllowedFields("state"))
	if err == nil {
		t.Fatal("expected error: total should not be allowed")
	}
}

func TestCompile_WithAllowedOps(t *testing.T) {
	_, err := Compile("year>2020", testFields,
		WithAllowedOps(validate.OpEq, validate.OpNeq))
	if err == nil {
		t.Fatal("expected error: > should not be allowed")
	}
}

func TestCompile_WithMaxDepth(t *testing.T) {
	_, err := Compile("(a=1 OR b=2) AND c=3", []validate.FieldConfig{
		{Name: "a", Type: validate.TypeText, AllowedOps: validate.TextOps},
		{Name: "b", Type: validate.TypeText, AllowedOps: validate.TextOps},
		{Name: "c", Type: validate.TypeText, AllowedOps: validate.TextOps},
	}, WithMaxDepth(2))
	if err == nil {
		t.Fatal("expected error: depth exceeds limit")
	}
}

func TestCompile_ParseError(t *testing.T) {
	_, err := Compile("=invalid", testFields)
	if err == nil {
		t.Fatal("expected parse error")
	}
}

func TestCompile_ValidationError(t *testing.T) {
	_, err := Compile("nonexistent=value", testFields)
	if err == nil {
		t.Fatal("expected validation error")
	}
}
