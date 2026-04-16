package query

import (
	"testing"
)

// issueFields mirrors the field configs matching the examples from the issue.
var issueFields = []FieldConfig{
	{Name: "state", Type: TypeText, AllowedOps: TextOps},
	{Name: "name", Type: TypeText, AllowedOps: TextOps},
	{Name: "make", Type: TypeText, AllowedOps: TextOps},
	{Name: "description", Type: TypeText, AllowedOps: TextOps},
	{Name: "cluster", Type: TypeText, AllowedOps: TextOps},
	{Name: "customer_id", Type: TypeText, AllowedOps: TextOps},
	{Name: "year", Type: TypeInteger, AllowedOps: NumericOps},
	{Name: "total", Type: TypeDecimal, AllowedOps: NumericOps},
	{Name: "created_at", Type: TypeDate, AllowedOps: DateOps},
	{Name: "tire_size", Type: TypeText, AllowedOps: append(TextOps, OpPresence)},
	{Name: "labels", Type: TypeText, AllowedOps: TextOps, Nested: true},
	{Name: "ttl", Type: TypeDuration, AllowedOps: DurationOps, Nested: true},
}

func TestParseAndValidate_IssueExamples(t *testing.T) {
	// Every example from the issue body should parse and validate successfully.
	examples := []string{
		"state=draft",
		"year>2020",
		"total>=50000",
		"created_at<=2026-03-31",
		"name=John*",
		"make=*yota",
		"description=*testing*",
		"state!=cancelled",
		"tire_size",
		"state=draft AND customer_id=customer_john-doe",
		"(state=draft OR state=issued) AND total>50000",
		"NOT state=cancelled",
		"(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo",
		"created_at>=2026-01-01 AND created_at<=2026-03-31",
		"ttl.duration>1d",
		"created_at:2026-01-01..2026-03-31",
	}

	for _, q := range examples {
		t.Run(q, func(t *testing.T) {
			expr, err := ParseAndValidate(q, issueFields)
			if err != nil {
				t.Fatalf("ParseAndValidate(%q) error: %v", q, err)
			}
			if expr == nil {
				t.Fatal("expected non-nil expression")
			}
		})
	}
}

func TestParse_DefaultMaxLength(t *testing.T) {
	// Queries up to 256 chars should work
	q := "state=draft"
	_, err := Parse(q)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Query exceeding 256 chars should fail
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	_, err = Parse(string(long))
	if err == nil {
		t.Fatal("expected error for query exceeding default max length")
	}
}

func TestParse_WithMaxLength(t *testing.T) {
	q := "state=draft" // 11 chars

	// Should fail with max length 5
	_, err := Parse(q, WithMaxLength(5))
	if err == nil {
		t.Fatal("expected error for query exceeding custom max length")
	}

	// Should succeed with max length 0 (disabled)
	_, err = Parse(q, WithMaxLength(0))
	if err != nil {
		t.Fatalf("unexpected error with disabled max length: %v", err)
	}

	// Should succeed with max length 100
	_, err = Parse(q, WithMaxLength(100))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestParseAndValidate_ParseError(t *testing.T) {
	_, err := ParseAndValidate("", issueFields)
	if err == nil {
		t.Fatal("expected parse error for empty query")
	}
}

func TestParseAndValidate_ValidationError(t *testing.T) {
	_, err := ParseAndValidate("nonexistent=value", issueFields)
	if err == nil {
		t.Fatal("expected validation error for unknown field")
	}
}

func TestParseAndValidate_RoundTrip(t *testing.T) {
	examples := []string{
		"state=draft",
		"state!=cancelled",
		"year>2020",
		"total>=50000",
		"tire_size",
		"name=John*",
		"labels.dev=jane",
		"NOT state=cancelled",
		"state=draft AND total>50000",
		"state=draft OR state=issued",
		"(state=draft OR state=issued) AND total>50000",
		"(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo",
		"created_at:2026-01-01..2026-03-31",
	}

	for _, q := range examples {
		t.Run(q, func(t *testing.T) {
			expr, err := Parse(q)
			if err != nil {
				t.Fatalf("Parse(%q) error: %v", q, err)
			}
			got := String(expr)
			if got != q {
				t.Errorf("round-trip failed:\n  input: %q\n  got:   %q", q, got)
			}
		})
	}
}

func TestErrors_Integration(t *testing.T) {
	// Parse error should be extractable via Errors()
	_, err := Parse("=invalid")
	if err == nil {
		t.Fatal("expected error")
	}
	if !IsQueryError(err) {
		t.Errorf("expected IsQueryError=true for %T", err)
	}
	errs := Errors(err)
	if len(errs) == 0 {
		t.Error("expected at least one QueryError")
	}
}

func TestValidate_Standalone(t *testing.T) {
	expr, err := Parse("state=draft")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Validate(expr, issueFields); err != nil {
		t.Errorf("unexpected validation error: %v", err)
	}
}

func TestValidate_StandaloneError(t *testing.T) {
	expr, err := Parse("nonexistent=value")
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if err := Validate(expr, issueFields); err == nil {
		t.Error("expected validation error")
	}
}

func TestString_Nil(t *testing.T) {
	got := String(nil)
	if got != "" {
		t.Errorf("String(nil): got %q, want empty", got)
	}
}

func TestWalk_Nil(t *testing.T) {
	// Should not panic
	Walk(nil, func(e Expression) bool { return true })
}
