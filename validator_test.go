package query

import (
	"testing"
)

// testFields defines a standard set of field configs for validator tests.
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

func TestValidator_ValidQueries(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"simple equality", "state=draft"},
		{"not equal", "state!=cancelled"},
		{"integer comparison", "year>2020"},
		{"integer gte", "year>=2020"},
		{"decimal comparison", "total>=50000"},
		{"decimal float", "total>=50000.50"},
		{"integer as decimal", "total>=50000"},
		{"boolean", "active=true"},
		{"date gte", "created_at>=2026-01-01"},
		{"wildcard prefix", "name=John*"},
		{"wildcard suffix", "make=*yota"},
		{"wildcard contains", "description=*testing*"},
		{"AND expression", "state=draft AND customer_id=customer_john-doe"},
		{"OR expression", "state=draft OR state=issued"},
		{"NOT expression", "NOT state=cancelled"},
		{"grouped OR with AND", "(state=draft OR state=issued) AND total>50000"},
		{"complex nested", "(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo"},
		{"date range", "created_at:2026-01-01..2026-03-31"},
		{"nested duration", "ttl.duration>1d"},
		{"nested labels", "labels.dev=jane"},
		{"negative integer", "offset>=-10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := mustParse(t, tt.input)
			v := NewValidator(testFields)
			if err := v.Validate(expr); err != nil {
				t.Errorf("unexpected validation error: %v", err)
			}
		})
	}
}

func TestValidator_UnknownField(t *testing.T) {
	expr := mustParse(t, "nonexistent=value")
	v := NewValidator(testFields)
	err := v.Validate(expr)
	if err == nil {
		t.Fatal("expected validation error for unknown field")
	}
	errs := Errors(err)
	if len(errs) == 0 || errs[0].Kind != ErrFieldNotFound {
		t.Errorf("expected ErrFieldNotFound, got %v", err)
	}
}

func TestValidator_OperatorNotAllowed(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"gt on text", "state>draft"},
		{"gte on text", "state>=draft"},
		{"lt on text", "state<draft"},
		{"lte on text", "state<=draft"},
		{"wildcard on integer", "year=202*"},
		{"wildcard on boolean", "active=tru*"},
		{"gt on boolean", "active>true"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := mustParse(t, tt.input)
			v := NewValidator(testFields)
			err := v.Validate(expr)
			if err == nil {
				t.Fatal("expected validation error for disallowed operator")
			}
		})
	}
}

func TestValidator_TypeMismatch(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"string for integer", "year=notanumber"},
		{"string for boolean", "active=maybe"},
		{"string for date", "created_at>=notadate"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			expr := mustParse(t, tt.input)
			v := NewValidator(testFields)
			err := v.Validate(expr)
			if err == nil {
				t.Fatal("expected validation error for type mismatch")
			}
		})
	}
}

func TestValidator_PresenceNotAllowed(t *testing.T) {
	// "active" (boolean) does not have OpPresence in BoolOps
	expr := mustParse(t, "active")
	v := NewValidator(testFields)
	err := v.Validate(expr)
	if err == nil {
		t.Fatal("expected validation error for presence not allowed")
	}
	errs := Errors(err)
	if len(errs) == 0 || errs[0].Kind != ErrOperatorNotAllowed {
		t.Errorf("expected ErrOperatorNotAllowed, got %v", err)
	}
}

func TestValidator_PresenceAllowed(t *testing.T) {
	// "tire_size" has OpPresence in AllowedOps
	expr := mustParse(t, "tire_size")
	v := NewValidator(testFields)
	if err := v.Validate(expr); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidator_NestedField(t *testing.T) {
	expr := mustParse(t, "labels.dev=jane")
	v := NewValidator(testFields)
	if err := v.Validate(expr); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidator_UnknownNestedField(t *testing.T) {
	expr := mustParse(t, "unknown.sub=value")
	v := NewValidator(testFields)
	err := v.Validate(expr)
	if err == nil {
		t.Fatal("expected error for unknown nested field")
	}
}

func TestValidator_MultipleErrors(t *testing.T) {
	expr := mustParse(t, "unknown_field=x AND year=notanum")
	v := NewValidator(testFields)
	err := v.Validate(expr)
	if err == nil {
		t.Fatal("expected validation errors")
	}
	errs := Errors(err)
	if len(errs) < 2 {
		t.Errorf("expected at least 2 errors, got %d: %v", len(errs), err)
	}
}

func TestValidator_RangeTypeCheck(t *testing.T) {
	expr := mustParse(t, "created_at:2026-01-01..2026-03-31")
	v := NewValidator(testFields)
	if err := v.Validate(expr); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestValidator_EmptyFieldConfig(t *testing.T) {
	// Validator with no fields should reject everything
	expr := mustParse(t, "state=draft")
	v := NewValidator(nil)
	err := v.Validate(expr)
	if err == nil {
		t.Fatal("expected error with empty field config")
	}
}

func TestFieldConfig_AllowsOp(t *testing.T) {
	cfg := FieldConfig{
		Name:       "test",
		Type:       TypeText,
		AllowedOps: []Op{OpEq, OpNeq},
	}

	if !cfg.AllowsOp(OpEq) {
		t.Error("expected OpEq to be allowed")
	}
	if !cfg.AllowsOp(OpNeq) {
		t.Error("expected OpNeq to be allowed")
	}
	if cfg.AllowsOp(OpGt) {
		t.Error("expected OpGt to not be allowed")
	}
}
