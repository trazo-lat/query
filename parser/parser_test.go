package parser

import (
	"testing"

	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
)

func TestParse_SimpleEquality(t *testing.T) {
	expr := mustParse(t, "state=draft")
	q := assertQualifier(t, expr)
	if q.Field.String() != "state" {
		t.Errorf("field: got %q, want %q", q.Field.String(), "state")
	}
	if q.Operator != token.Eq {
		t.Errorf("operator: got %v, want Eq", q.Operator)
	}
}

func TestParse_Precedence(t *testing.T) {
	expr := mustParse(t, "a=1 OR b=2 AND c=3")
	b, ok := expr.(*ast.BinaryExpr)
	if !ok || b.Op != token.Or {
		t.Fatal("expected OR at top level")
	}
	right, ok := b.Right.(*ast.BinaryExpr)
	if !ok || right.Op != token.And {
		t.Fatal("expected AND on right")
	}
}

func TestParse_GroupNOT(t *testing.T) {
	expr := mustParse(t, "NOT (state=draft OR state=issued)")
	u, ok := expr.(*ast.UnaryExpr)
	if !ok {
		t.Fatalf("expected UnaryExpr, got %T", expr)
	}
	_, ok = u.Expr.(*ast.GroupExpr)
	if !ok {
		t.Fatalf("expected GroupExpr, got %T", u.Expr)
	}
}

func TestParse_Range(t *testing.T) {
	expr := mustParse(t, "created_at:2026-01-01..2026-03-31")
	q := assertQualifier(t, expr)
	if q.Operator != token.Range {
		t.Errorf("operator: got %v, want Range", q.Operator)
	}
	if q.EndValue == nil {
		t.Fatal("expected EndValue")
	}
}

func TestParse_Presence(t *testing.T) {
	expr := mustParse(t, "tire_size")
	_, ok := expr.(*ast.PresenceExpr)
	if !ok {
		t.Fatalf("expected PresenceExpr, got %T", expr)
	}
}

func TestParse_DottedField(t *testing.T) {
	expr := mustParse(t, "labels.dev=jane")
	q := assertQualifier(t, expr)
	if q.Field.String() != "labels.dev" {
		t.Errorf("field: got %q, want %q", q.Field.String(), "labels.dev")
	}
}

func TestParse_RoundTrip(t *testing.T) {
	examples := []string{
		"state=draft",
		"state!=cancelled",
		"year>2020",
		"tire_size",
		"name=John*",
		"NOT state=cancelled",
		"state=draft AND total>50000",
		"(state=draft OR state=issued) AND total>50000",
		"(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo",
		"created_at:2026-01-01..2026-03-31",
	}
	for _, q := range examples {
		t.Run(q, func(t *testing.T) {
			expr := mustParse(t, q)
			got := ast.String(expr)
			if got != q {
				t.Errorf("round-trip:\n  got:  %q\n  want: %q", got, q)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	bad := []string{
		"(state=draft",
		"",
		"=draft",
		"a=1 AND",
	}
	for _, input := range bad {
		t.Run(input, func(t *testing.T) {
			_, err := Parse(input, 0)
			if err == nil {
				t.Error("expected error")
			}
		})
	}
}

func mustParse(t *testing.T, input string) ast.Expression {
	t.Helper()
	expr, err := Parse(input, 0)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return expr
}

func assertQualifier(t *testing.T, expr ast.Expression) *ast.QualifierExpr {
	t.Helper()
	q, ok := expr.(*ast.QualifierExpr)
	if !ok {
		t.Fatalf("expected *QualifierExpr, got %T", expr)
	}
	return q
}
