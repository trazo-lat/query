package query

import (
	"testing"
)

func TestParse_SimpleEquality(t *testing.T) {
	expr := mustParse(t, "state=draft")
	q, ok := expr.(*QualifierExpr)
	if !ok {
		t.Fatalf("expected *QualifierExpr, got %T", expr)
	}
	if q.Field.String() != "state" {
		t.Errorf("field: got %q, want %q", q.Field.String(), "state")
	}
	if q.Operator != TokenEq {
		t.Errorf("operator: got %v, want %v", q.Operator, TokenEq)
	}
	if q.Value.Str != "draft" {
		t.Errorf("value: got %q, want %q", q.Value.Str, "draft")
	}
}

func TestParse_NotEqual(t *testing.T) {
	expr := mustParse(t, "state!=cancelled")
	q := assertQualifier(t, expr)
	if q.Operator != TokenNeq {
		t.Errorf("operator: got %v, want %v", q.Operator, TokenNeq)
	}
	if q.Value.Str != "cancelled" {
		t.Errorf("value: got %q, want %q", q.Value.Str, "cancelled")
	}
}

func TestParse_ComparisonOperators(t *testing.T) {
	tests := []struct {
		input string
		op    TokenType
		val   int64
	}{
		{"year>2020", TokenGt, 2020},
		{"year>=2020", TokenGte, 2020},
		{"year<2025", TokenLt, 2025},
		{"year<=2025", TokenLte, 2025},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr := mustParse(t, tt.input)
			q := assertQualifier(t, expr)
			if q.Operator != tt.op {
				t.Errorf("operator: got %v, want %v", q.Operator, tt.op)
			}
			if q.Value.Int != tt.val {
				t.Errorf("value: got %d, want %d", q.Value.Int, tt.val)
			}
		})
	}
}

func TestParse_Presence(t *testing.T) {
	expr := mustParse(t, "tire_size")
	p, ok := expr.(*PresenceExpr)
	if !ok {
		t.Fatalf("expected *PresenceExpr, got %T", expr)
	}
	if p.Field.String() != "tire_size" {
		t.Errorf("field: got %q, want %q", p.Field.String(), "tire_size")
	}
}

func TestParse_Wildcard(t *testing.T) {
	tests := []struct {
		input    string
		wildcard string
	}{
		{"name=John*", "John*"},
		{"make=*yota", "*yota"},
		{"description=*testing*", "*testing*"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr := mustParse(t, tt.input)
			q := assertQualifier(t, expr)
			if !q.Value.Wildcard {
				t.Error("expected Wildcard=true")
			}
			if q.Value.Str != tt.wildcard {
				t.Errorf("value: got %q, want %q", q.Value.Str, tt.wildcard)
			}
		})
	}
}

func TestParse_NOT(t *testing.T) {
	expr := mustParse(t, "NOT state=cancelled")
	u, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected *UnaryExpr, got %T", expr)
	}
	if u.Op != TokenNot {
		t.Errorf("op: got %v, want %v", u.Op, TokenNot)
	}
	q := assertQualifier(t, u.Expr)
	if q.Field.String() != "state" {
		t.Errorf("field: got %q, want %q", q.Field.String(), "state")
	}
}

func TestParse_AND(t *testing.T) {
	expr := mustParse(t, "state=draft AND customer_id=customer_john-doe")
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr, got %T", expr)
	}
	if b.Op != TokenAnd {
		t.Errorf("op: got %v, want %v", b.Op, TokenAnd)
	}
	left := assertQualifier(t, b.Left)
	if left.Field.String() != "state" {
		t.Errorf("left field: got %q, want %q", left.Field.String(), "state")
	}
	right := assertQualifier(t, b.Right)
	if right.Field.String() != "customer_id" {
		t.Errorf("right field: got %q, want %q", right.Field.String(), "customer_id")
	}
}

func TestParse_OR(t *testing.T) {
	expr := mustParse(t, "state=draft OR state=issued")
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr, got %T", expr)
	}
	if b.Op != TokenOr {
		t.Errorf("op: got %v, want %v", b.Op, TokenOr)
	}
}

func TestParse_Precedence_ANDBindsTighterThanOR(t *testing.T) {
	// a=1 OR b=2 AND c=3 should parse as OR(a=1, AND(b=2, c=3))
	expr := mustParse(t, "a=1 OR b=2 AND c=3")
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr (OR), got %T", expr)
	}
	if b.Op != TokenOr {
		t.Errorf("top-level op: got %v, want OR", b.Op)
	}
	// Left is a simple qualifier
	assertQualifier(t, b.Left)
	// Right is AND
	right, ok := b.Right.(*BinaryExpr)
	if !ok {
		t.Fatalf("right: expected *BinaryExpr (AND), got %T", b.Right)
	}
	if right.Op != TokenAnd {
		t.Errorf("right op: got %v, want AND", right.Op)
	}
}

func TestParse_Grouping(t *testing.T) {
	expr := mustParse(t, "(state=draft OR state=issued) AND total>50000")
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr (AND), got %T", expr)
	}
	if b.Op != TokenAnd {
		t.Errorf("op: got %v, want AND", b.Op)
	}
	// Left should be a GroupExpr containing an OR
	g, ok := b.Left.(*GroupExpr)
	if !ok {
		t.Fatalf("left: expected *GroupExpr, got %T", b.Left)
	}
	inner, ok := g.Expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("group inner: expected *BinaryExpr, got %T", g.Expr)
	}
	if inner.Op != TokenOr {
		t.Errorf("group inner op: got %v, want OR", inner.Op)
	}
}

func TestParse_NestedGroups(t *testing.T) {
	expr := mustParse(t, "((a=1 OR b=2) AND c=3)")
	g, ok := expr.(*GroupExpr)
	if !ok {
		t.Fatalf("expected *GroupExpr, got %T", expr)
	}
	b, ok := g.Expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("inner: expected *BinaryExpr (AND), got %T", g.Expr)
	}
	if b.Op != TokenAnd {
		t.Errorf("inner op: got %v, want AND", b.Op)
	}
}

func TestParse_NOTWithGroup(t *testing.T) {
	expr := mustParse(t, "NOT (state=draft OR state=issued)")
	u, ok := expr.(*UnaryExpr)
	if !ok {
		t.Fatalf("expected *UnaryExpr, got %T", expr)
	}
	_, ok = u.Expr.(*GroupExpr)
	if !ok {
		t.Fatalf("inner: expected *GroupExpr, got %T", u.Expr)
	}
}

func TestParse_DottedField(t *testing.T) {
	expr := mustParse(t, "labels.dev=jane")
	q := assertQualifier(t, expr)
	if q.Field.String() != "labels.dev" {
		t.Errorf("field: got %q, want %q", q.Field.String(), "labels.dev")
	}
}

func TestParse_DateValue(t *testing.T) {
	expr := mustParse(t, "created_at>=2026-01-01")
	q := assertQualifier(t, expr)
	if q.Value.Type != ValueDate {
		t.Errorf("value type: got %v, want %v", q.Value.Type, ValueDate)
	}
	if q.Value.Date.Year() != 2026 || q.Value.Date.Month() != 1 || q.Value.Date.Day() != 1 {
		t.Errorf("date: got %v, want 2026-01-01", q.Value.Date)
	}
}

func TestParse_DurationValue(t *testing.T) {
	expr := mustParse(t, "ttl.duration>1d")
	q := assertQualifier(t, expr)
	if q.Value.Type != ValueDuration {
		t.Errorf("value type: got %v, want %v", q.Value.Type, ValueDuration)
	}
	if q.Value.Raw != "1d" {
		t.Errorf("raw: got %q, want %q", q.Value.Raw, "1d")
	}
}

func TestParse_BooleanValue(t *testing.T) {
	expr := mustParse(t, "active=true")
	q := assertQualifier(t, expr)
	if q.Value.Type != ValueBoolean {
		t.Errorf("value type: got %v, want %v", q.Value.Type, ValueBoolean)
	}
	if !q.Value.Bool {
		t.Error("expected Bool=true")
	}
}

func TestParse_FloatValue(t *testing.T) {
	expr := mustParse(t, "total>=50000.50")
	q := assertQualifier(t, expr)
	if q.Value.Type != ValueFloat {
		t.Errorf("value type: got %v, want %v", q.Value.Type, ValueFloat)
	}
	if q.Value.Float != 50000.50 {
		t.Errorf("float: got %f, want %f", q.Value.Float, 50000.50)
	}
}

func TestParse_RangeExpression(t *testing.T) {
	expr := mustParse(t, "created_at:2026-01-01..2026-03-31")
	q := assertQualifier(t, expr)
	if q.Operator != TokenRange {
		t.Errorf("operator: got %v, want %v", q.Operator, TokenRange)
	}
	if q.Value.Type != ValueDate {
		t.Errorf("start type: got %v, want %v", q.Value.Type, ValueDate)
	}
	if q.EndValue == nil {
		t.Fatal("expected EndValue for range")
	}
	if q.EndValue.Type != ValueDate {
		t.Errorf("end type: got %v, want %v", q.EndValue.Type, ValueDate)
	}
	if q.Value.Raw != "2026-01-01" {
		t.Errorf("start: got %q, want %q", q.Value.Raw, "2026-01-01")
	}
	if q.EndValue.Raw != "2026-03-31" {
		t.Errorf("end: got %q, want %q", q.EndValue.Raw, "2026-03-31")
	}
}

func TestParse_ComplexQuery(t *testing.T) {
	expr := mustParse(t, "(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo")
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr (AND), got %T", expr)
	}
	if b.Op != TokenAnd {
		t.Errorf("op: got %v, want AND", b.Op)
	}
	// Left is a group
	_, ok = b.Left.(*GroupExpr)
	if !ok {
		t.Fatalf("left: expected *GroupExpr, got %T", b.Left)
	}
	// Right is NOT
	u, ok := b.Right.(*UnaryExpr)
	if !ok {
		t.Fatalf("right: expected *UnaryExpr, got %T", b.Right)
	}
	if u.Op != TokenNot {
		t.Errorf("right op: got %v, want NOT", u.Op)
	}
}

func TestParse_RoundTrip(t *testing.T) {
	tests := []struct {
		input string
		want  string // expected round-trip output (empty means same as input)
	}{
		{"state=draft", ""},
		{"state!=cancelled", ""},
		{"year>2020", ""},
		{"total>=50000", ""},
		{"tire_size", ""},
		{"name=John*", ""},
		{"labels.dev=jane", ""},
		{"active=true", ""},
		{"NOT state=cancelled", ""},
		{"state=draft AND total>50000", ""},
		{"state=draft OR state=issued", ""},
		{"(state=draft OR state=issued) AND total>50000", ""},
		{"NOT (state=draft OR state=issued)", ""},
		{"(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo", ""},
		{"created_at:2026-01-01..2026-03-31", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			expr := mustParse(t, tt.input)
			got := String(expr)
			want := tt.want
			if want == "" {
				want = tt.input
			}
			if got != want {
				t.Errorf("round-trip:\n  got:  %q\n  want: %q", got, want)
			}
		})
	}
}

func TestParse_Errors(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"unclosed paren", "(state=draft"},
		{"empty input", ""},
		{"operator without field", "=draft"},
		{"missing right operand", "a=1 AND"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				return // lexer error is fine
			}
			_, err = parse(tokens)
			if err == nil {
				t.Error("expected parse error")
			}
		})
	}
}

func TestParse_PositionTracking(t *testing.T) {
	// "a=1 AND b=2"
	expr := mustParse(t, "a=1 AND b=2")
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr, got %T", expr)
	}
	// AND token is at offset 4
	if b.Position.Offset != 4 {
		t.Errorf("AND position: got offset %d, want 4", b.Position.Offset)
	}
}

func TestParse_MultipleAND(t *testing.T) {
	expr := mustParse(t, "a=1 AND b=2 AND c=3")
	// Should be AND(AND(a=1, b=2), c=3) — left-associative
	b, ok := expr.(*BinaryExpr)
	if !ok {
		t.Fatalf("expected *BinaryExpr, got %T", expr)
	}
	if b.Op != TokenAnd {
		t.Errorf("top op: got %v, want AND", b.Op)
	}
	left, ok := b.Left.(*BinaryExpr)
	if !ok {
		t.Fatalf("left: expected *BinaryExpr, got %T", b.Left)
	}
	if left.Op != TokenAnd {
		t.Errorf("left op: got %v, want AND", left.Op)
	}
}

func TestParse_DottedPresence(t *testing.T) {
	expr := mustParse(t, "labels.env")
	p, ok := expr.(*PresenceExpr)
	if !ok {
		t.Fatalf("expected *PresenceExpr, got %T", expr)
	}
	if p.Field.String() != "labels.env" {
		t.Errorf("field: got %q, want %q", p.Field.String(), "labels.env")
	}
}

func TestWalk(t *testing.T) {
	expr := mustParse(t, "a=1 AND (b=2 OR c=3)")
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
	want := []string{"a", "b", "c"}
	for i, f := range fields {
		if f != want[i] {
			t.Errorf("field[%d]: got %q, want %q", i, f, want[i])
		}
	}
}

func TestWalk_StopDescending(t *testing.T) {
	expr := mustParse(t, "a=1 AND (b=2 OR c=3)")
	var visited int
	Walk(expr, func(e Expression) bool {
		visited++
		// Stop descending into the group
		if _, ok := e.(*GroupExpr); ok {
			return false
		}
		return true
	})
	// Should visit: BinaryExpr(AND), QualifierExpr(a), GroupExpr — stops at group
	if visited != 3 {
		t.Errorf("visited %d nodes, want 3", visited)
	}
}

// mustParse is a test helper that lexes and parses, failing on error.
func mustParse(t *testing.T, input string) Expression {
	t.Helper()
	tokens, err := lex(input, 0)
	if err != nil {
		t.Fatalf("lex error: %v", err)
	}
	expr, err := parse(tokens)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	return expr
}

// assertQualifier asserts the expression is a *QualifierExpr and returns it.
func assertQualifier(t *testing.T, expr Expression) *QualifierExpr {
	t.Helper()
	q, ok := expr.(*QualifierExpr)
	if !ok {
		t.Fatalf("expected *QualifierExpr, got %T", expr)
	}
	return q
}
