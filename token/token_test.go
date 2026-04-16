package token

import "testing"

func TestPosition_String(t *testing.T) {
	p := Position{Offset: 5, Length: 3}
	want := "offset 5, length 3"
	if p.String() != want {
		t.Errorf("got %q, want %q", p.String(), want)
	}
}

func TestType_String(t *testing.T) {
	tests := []struct {
		typ  Type
		want string
	}{
		{Illegal, "ILLEGAL"},
		{EOF, "EOF"},
		{Ident, "IDENT"},
		{String, "STRING"},
		{Integer, "INTEGER"},
		{Float, "FLOAT"},
		{Date, "DATE"},
		{Duration, "DURATION"},
		{Boolean, "BOOLEAN"},
		{Eq, "="},
		{Neq, "!="},
		{Gt, ">"},
		{Gte, ">="},
		{Lt, "<"},
		{Lte, "<="},
		{Range, ".."},
		{Colon, ":"},
		{And, "AND"},
		{Or, "OR"},
		{Not, "NOT"},
		{LParen, "("},
		{RParen, ")"},
		{At, "@"},
		{Dot, "."},
		{Comma, ","},
		{Wildcard, "*"},
		{Type(9999), "Type(9999)"},
	}
	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.want {
			t.Errorf("Type(%d).String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}

func TestType_IsOperator(t *testing.T) {
	operators := []Type{Eq, Neq, Gt, Gte, Lt, Lte, Colon}
	for _, op := range operators {
		if !op.IsOperator() {
			t.Errorf("%v.IsOperator() = false, want true", op)
		}
	}
	nonOperators := []Type{Ident, String, And, Or, Not, LParen, RParen, Dot, Comma, Range}
	for _, op := range nonOperators {
		if op.IsOperator() {
			t.Errorf("%v.IsOperator() = true, want false", op)
		}
	}
}

func TestType_IsLogical(t *testing.T) {
	if !And.IsLogical() {
		t.Error("And.IsLogical() = false, want true")
	}
	if !Or.IsLogical() {
		t.Error("Or.IsLogical() = false, want true")
	}
	if Not.IsLogical() {
		t.Error("Not.IsLogical() = true, want false")
	}
	if Eq.IsLogical() {
		t.Error("Eq.IsLogical() = true, want false")
	}
}

func TestToken_String(t *testing.T) {
	tests := []struct {
		tok  Token
		want string
	}{
		{Token{Type: Ident, Value: "name"}, `IDENT("name")`},
		{Token{Type: EOF}, "EOF"},
		{Token{Type: Eq, Value: "="}, `=("=")`},
	}
	for _, tt := range tests {
		if got := tt.tok.String(); got != tt.want {
			t.Errorf("got %q, want %q", got, tt.want)
		}
	}
}

func TestOperatorSymbol(t *testing.T) {
	tests := []struct {
		op   Type
		want string
	}{
		{Eq, "="},
		{Neq, "!="},
		{Gt, ">"},
		{Gte, ">="},
		{Lt, "<"},
		{Lte, "<="},
		{Range, ".."},
		{Colon, ":"},
		{Ident, "="}, // default
	}
	for _, tt := range tests {
		if got := OperatorSymbol(tt.op); got != tt.want {
			t.Errorf("OperatorSymbol(%v) = %q, want %q", tt.op, got, tt.want)
		}
	}
}
