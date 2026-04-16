package validate

import (
	"errors"
	"testing"

	"github.com/trazo-lat/query/parser"
)

var testFields = []FieldConfig{
	{Name: "state", Type: TypeText, AllowedOps: TextOps},
	{Name: "name", Type: TypeText, AllowedOps: TextOps},
	{Name: "description", Type: TypeText, AllowedOps: TextOps},
	{Name: "year", Type: TypeInteger, AllowedOps: NumericOps},
	{Name: "total", Type: TypeDecimal, AllowedOps: NumericOps},
	{Name: "active", Type: TypeBoolean, AllowedOps: BoolOps},
	{Name: "created_at", Type: TypeDate, AllowedOps: DateOps},
	{Name: "ttl", Type: TypeDuration, AllowedOps: DurationOps, Nested: true},
	{Name: "labels", Type: TypeText, AllowedOps: TextOps, Nested: true},
	{Name: "cluster", Type: TypeText, AllowedOps: TextOps},
	{Name: "customer_id", Type: TypeText, AllowedOps: TextOps},
	{Name: "tire_size", Type: TypeText, AllowedOps: append(TextOps, OpPresence)},
	{Name: "make", Type: TypeText, AllowedOps: TextOps},
	{Name: "offset", Type: TypeInteger, AllowedOps: NumericOps},
}

func TestValidate_ValidQueries(t *testing.T) {
	valid := []string{
		"state=draft",
		"state!=cancelled",
		"year>2020",
		"total>=50000",
		"total>=50000.50",
		"active=true",
		"created_at>=2026-01-01",
		"name=John*",
		"make=*yota",
		"description=*testing*",
		"state=draft AND customer_id=customer_john-doe",
		"(state=draft OR state=issued) AND total>50000",
		"NOT state=cancelled",
		"(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo",
		"created_at:2026-01-01..2026-03-31",
		"ttl.duration>1d",
		"labels.dev=jane",
		"offset>=-10",
	}
	for _, q := range valid {
		t.Run(q, func(t *testing.T) {
			expr, err := parser.Parse(q, 0)
			if err != nil {
				t.Fatalf("parse: %v", err)
			}
			v := New(testFields)
			if err := v.Validate(expr); err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		})
	}
}

func TestValidate_UnknownField(t *testing.T) {
	expr, _ := parser.Parse("nonexistent=value", 0)
	v := New(testFields)
	if err := v.Validate(expr); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_OperatorNotAllowed(t *testing.T) {
	bad := []string{
		"state>draft",
		"state>=draft",
		"year=202*",
		"active>true",
	}
	for _, q := range bad {
		t.Run(q, func(t *testing.T) {
			expr, _ := parser.Parse(q, 0)
			v := New(testFields)
			if err := v.Validate(expr); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidate_TypeMismatch(t *testing.T) {
	bad := []string{
		"year=notanumber",
		"active=maybe",
		"created_at>=notadate",
	}
	for _, q := range bad {
		t.Run(q, func(t *testing.T) {
			expr, _ := parser.Parse(q, 0)
			v := New(testFields)
			if err := v.Validate(expr); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestValidate_PresenceAllowed(t *testing.T) {
	expr, _ := parser.Parse("tire_size", 0)
	v := New(testFields)
	if err := v.Validate(expr); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidate_PresenceNotAllowed(t *testing.T) {
	expr, _ := parser.Parse("active", 0)
	v := New(testFields)
	if err := v.Validate(expr); err == nil {
		t.Fatal("expected error")
	}
}

func TestValidate_MultipleErrors(t *testing.T) {
	expr, _ := parser.Parse("unknown_field=x AND year=notanum", 0)
	v := New(testFields)
	err := v.Validate(expr)
	if err == nil {
		t.Fatal("expected errors")
	}
	var el ErrorList
	if !errors.As(err, &el) {
		t.Fatalf("expected ErrorList, got %T", err)
	}
	if len(el) < 2 {
		t.Errorf("expected at least 2 errors, got %d", len(el))
	}
}

func TestValidate_EmptyConfig(t *testing.T) {
	expr, _ := parser.Parse("state=draft", 0)
	v := New(nil)
	if err := v.Validate(expr); err == nil {
		t.Fatal("expected error")
	}
}
