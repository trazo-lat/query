package ast

import (
	"testing"

	"github.com/trazo-lat/query/token"
)

func TestWalk(t *testing.T) {
	// Build AST: a=1 AND (b=2 OR c=3)
	expr := &BinaryExpr{
		Op: token.And,
		Left: &QualifierExpr{
			Field:    FieldPath{"a"},
			Operator: token.Eq,
			Value:    Value{Type: ValueString, Raw: "1", Str: "1"},
		},
		Right: &GroupExpr{
			Expr: &BinaryExpr{
				Op: token.Or,
				Left: &QualifierExpr{
					Field:    FieldPath{"b"},
					Operator: token.Eq,
					Value:    Value{Type: ValueString, Raw: "2", Str: "2"},
				},
				Right: &QualifierExpr{
					Field:    FieldPath{"c"},
					Operator: token.Eq,
					Value:    Value{Type: ValueString, Raw: "3", Str: "3"},
				},
			},
		},
	}

	var fields []string
	Walk(expr, func(e Expression) bool {
		if q, ok := e.(*QualifierExpr); ok {
			fields = append(fields, q.Field.String())
		}
		return true
	})
	if len(fields) != 3 {
		t.Fatalf("got %d fields, want 3", len(fields))
	}
}

func TestFields(t *testing.T) {
	expr := &BinaryExpr{
		Op: token.And,
		Left: &QualifierExpr{
			Field: FieldPath{"state"},
		},
		Right: &PresenceExpr{
			Field: FieldPath{"labels", "dev"},
		},
	}
	fps := Fields(expr)
	if len(fps) != 2 {
		t.Fatalf("got %d fields, want 2", len(fps))
	}
}

func TestIsSimple(t *testing.T) {
	if !IsSimple(&QualifierExpr{}) {
		t.Error("QualifierExpr should be simple")
	}
	if !IsSimple(&PresenceExpr{}) {
		t.Error("PresenceExpr should be simple")
	}
	if IsSimple(&BinaryExpr{}) {
		t.Error("BinaryExpr should not be simple")
	}
}

func TestDepth(t *testing.T) {
	// Single qualifier: depth 1
	if d := Depth(&QualifierExpr{}); d != 1 {
		t.Errorf("qualifier depth: got %d, want 1", d)
	}
	// Binary with two qualifiers: depth 2
	expr := &BinaryExpr{
		Left:  &QualifierExpr{},
		Right: &QualifierExpr{},
	}
	if d := Depth(expr); d != 2 {
		t.Errorf("binary depth: got %d, want 2", d)
	}
}

func TestString_Nil(t *testing.T) {
	if got := String(nil); got != "" {
		t.Errorf("String(nil): got %q, want empty", got)
	}
}

func TestWalk_Nil(t *testing.T) {
	Walk(nil, func(e Expression) bool { return true })
}

func TestFieldPath(t *testing.T) {
	fp := FieldPath{"labels", "dev"}
	if fp.String() != "labels.dev" {
		t.Errorf("String: got %q", fp.String())
	}
	if fp.Root() != "labels" {
		t.Errorf("Root: got %q", fp.Root())
	}
	if !fp.IsNested() {
		t.Error("expected IsNested=true")
	}
	single := FieldPath{"state"}
	if single.IsNested() {
		t.Error("expected IsNested=false for single")
	}
}

func TestValueAny(t *testing.T) {
	v := Value{Type: ValueInteger, Int: 42}
	if got, ok := v.Any().(int64); !ok || got != 42 {
		t.Errorf("Any: got %v", v.Any())
	}
	v2 := Value{Type: ValueBoolean, Bool: true}
	if got, ok := v2.Any().(bool); !ok || !got {
		t.Errorf("Any: got %v", v2.Any())
	}
}
