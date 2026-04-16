package query

import (
	"testing"
)

func TestLex_SimpleOperators(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   []TokenType
		values []string
	}{
		{
			name:   "equality",
			input:  "state=draft",
			want:   []TokenType{TokenIdent, TokenEq, TokenString, TokenEOF},
			values: []string{"state", "=", "draft", ""},
		},
		{
			name:   "not equal",
			input:  "state!=cancelled",
			want:   []TokenType{TokenIdent, TokenNeq, TokenString, TokenEOF},
			values: []string{"state", "!=", "cancelled", ""},
		},
		{
			name:   "greater than",
			input:  "year>2020",
			want:   []TokenType{TokenIdent, TokenGt, TokenInteger, TokenEOF},
			values: []string{"year", ">", "2020", ""},
		},
		{
			name:   "greater than or equal",
			input:  "total>=50000",
			want:   []TokenType{TokenIdent, TokenGte, TokenInteger, TokenEOF},
			values: []string{"total", ">=", "50000", ""},
		},
		{
			name:   "less than",
			input:  "year<2025",
			want:   []TokenType{TokenIdent, TokenLt, TokenInteger, TokenEOF},
			values: []string{"year", "<", "2025", ""},
		},
		{
			name:   "less than or equal",
			input:  "total<=99999",
			want:   []TokenType{TokenIdent, TokenLte, TokenInteger, TokenEOF},
			values: []string{"total", "<=", "99999", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokenTypes(t, tokens, tt.want)
			assertTokenValues(t, tokens, tt.values)
		})
	}
}

func TestLex_Keywords(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		want   []TokenType
		values []string
	}{
		{
			name:   "AND keyword",
			input:  "a=1 AND b=2",
			want:   []TokenType{TokenIdent, TokenEq, TokenInteger, TokenAnd, TokenIdent, TokenEq, TokenInteger, TokenEOF},
			values: []string{"a", "=", "1", "AND", "b", "=", "2", ""},
		},
		{
			name:   "OR keyword",
			input:  "a=1 OR b=2",
			want:   []TokenType{TokenIdent, TokenEq, TokenInteger, TokenOr, TokenIdent, TokenEq, TokenInteger, TokenEOF},
			values: []string{"a", "=", "1", "OR", "b", "=", "2", ""},
		},
		{
			name:   "NOT keyword",
			input:  "NOT a=1",
			want:   []TokenType{TokenNot, TokenIdent, TokenEq, TokenInteger, TokenEOF},
			values: []string{"NOT", "a", "=", "1", ""},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			assertTokenTypes(t, tokens, tt.want)
			assertTokenValues(t, tokens, tt.values)
		})
	}
}

func TestLex_Parentheses(t *testing.T) {
	tokens, err := lex("(a=1 OR b=2)", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokenType{TokenLParen, TokenIdent, TokenEq, TokenInteger, TokenOr, TokenIdent, TokenEq, TokenInteger, TokenRParen, TokenEOF}
	assertTokenTypes(t, tokens, want)
}

func TestLex_Wildcards(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  TokenType
		value string
	}{
		{"prefix", "name=John*", TokenWildcard, "John*"},
		{"suffix", "name=*ohn", TokenWildcard, "*ohn"},
		{"contains", "name=*oh*", TokenWildcard, "*oh*"},
		{"star only", "name=*", TokenWildcard, "*"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			// Third token is the value
			if tokens[2].Type != tt.want {
				t.Errorf("got type %v, want %v", tokens[2].Type, tt.want)
			}
			if tokens[2].Value != tt.value {
				t.Errorf("got value %q, want %q", tokens[2].Value, tt.value)
			}
		})
	}
}

func TestLex_InvalidWildcard(t *testing.T) {
	_, err := lex("name=a*b*c", 0)
	if err == nil {
		t.Fatal("expected error for invalid wildcard pattern")
	}
	if !IsQueryError(err) {
		t.Fatalf("expected QueryError, got %T", err)
	}
}

func TestLex_DateLiterals(t *testing.T) {
	tokens, err := lex("created_at>=2026-01-01", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTokenTypes(t, tokens, []TokenType{TokenIdent, TokenGte, TokenDate, TokenEOF})
	if tokens[2].Value != "2026-01-01" {
		t.Errorf("got %q, want %q", tokens[2].Value, "2026-01-01")
	}
}

func TestLex_DurationLiterals(t *testing.T) {
	tests := []struct {
		input string
		value string
	}{
		{"ttl>1d", "1d"},
		{"ttl>4h", "4h"},
		{"ttl>30m", "30m"},
		{"ttl>2w", "2w"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[2].Type != TokenDuration {
				t.Errorf("got type %v, want %v", tokens[2].Type, TokenDuration)
			}
			if tokens[2].Value != tt.value {
				t.Errorf("got %q, want %q", tokens[2].Value, tt.value)
			}
		})
	}
}

func TestLex_BooleanLiterals(t *testing.T) {
	tests := []struct {
		input string
		value string
	}{
		{"active=true", "true"},
		{"active=false", "false"},
	}

	for _, tt := range tests {
		t.Run(tt.value, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[2].Type != TokenBoolean {
				t.Errorf("got type %v, want %v", tokens[2].Type, TokenBoolean)
			}
			if tokens[2].Value != tt.value {
				t.Errorf("got %q, want %q", tokens[2].Value, tt.value)
			}
		})
	}
}

func TestLex_FloatLiterals(t *testing.T) {
	tokens, err := lex("total>=50000.50", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTokenTypes(t, tokens, []TokenType{TokenIdent, TokenGte, TokenFloat, TokenEOF})
	if tokens[2].Value != "50000.50" {
		t.Errorf("got %q, want %q", tokens[2].Value, "50000.50")
	}
}

func TestLex_DottedFieldNames(t *testing.T) {
	tokens, err := lex("labels.dev=jane", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokenType{TokenIdent, TokenDot, TokenIdent, TokenEq, TokenString, TokenEOF}
	assertTokenTypes(t, tokens, want)
	wantValues := []string{"labels", ".", "dev", "=", "jane", ""}
	assertTokenValues(t, tokens, wantValues)
}

func TestLex_EscapeSequences(t *testing.T) {
	tokens, err := lex(`name=hello\*world`, 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The escaped star should produce a regular string, not a wildcard
	if tokens[2].Type != TokenString {
		t.Errorf("got type %v, want %v", tokens[2].Type, TokenString)
	}
	if tokens[2].Value != "hello*world" {
		t.Errorf("got value %q, want %q", tokens[2].Value, "hello*world")
	}
}

func TestLex_ColonAndRange(t *testing.T) {
	tokens, err := lex("created_at:2026-01-01..2026-03-31", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokenType{TokenIdent, TokenColon, TokenDate, TokenRange, TokenDate, TokenEOF}
	assertTokenTypes(t, tokens, want)
	wantValues := []string{"created_at", ":", "2026-01-01", "..", "2026-03-31", ""}
	assertTokenValues(t, tokens, wantValues)
}

func TestLex_PresenceField(t *testing.T) {
	tokens, err := lex("tire_size", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTokenTypes(t, tokens, []TokenType{TokenIdent, TokenEOF})
	if tokens[0].Value != "tire_size" {
		t.Errorf("got %q, want %q", tokens[0].Value, "tire_size")
	}
}

func TestLex_ComplexQuery(t *testing.T) {
	tokens, err := lex("(state=draft OR state=issued) AND total>50000", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokenType{
		TokenLParen,
		TokenIdent, TokenEq, TokenString,
		TokenOr,
		TokenIdent, TokenEq, TokenString,
		TokenRParen,
		TokenAnd,
		TokenIdent, TokenGt, TokenInteger,
		TokenEOF,
	}
	assertTokenTypes(t, tokens, want)
}

func TestLex_PositionTracking(t *testing.T) {
	tokens, err := lex("a=1 AND b=2", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// a at offset 0
	if tokens[0].Pos.Offset != 0 {
		t.Errorf("token 'a': got offset %d, want 0", tokens[0].Pos.Offset)
	}
	// = at offset 1
	if tokens[1].Pos.Offset != 1 {
		t.Errorf("token '=': got offset %d, want 1", tokens[1].Pos.Offset)
	}
	// 1 at offset 2
	if tokens[2].Pos.Offset != 2 {
		t.Errorf("token '1': got offset %d, want 2", tokens[2].Pos.Offset)
	}
	// AND at offset 4
	if tokens[3].Pos.Offset != 4 {
		t.Errorf("token 'AND': got offset %d, want 4", tokens[3].Pos.Offset)
	}
	if tokens[3].Pos.Length != 3 {
		t.Errorf("token 'AND': got length %d, want 3", tokens[3].Pos.Length)
	}
}

func TestLex_EmptyInput(t *testing.T) {
	tokens, err := lex("", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != TokenEOF {
		t.Errorf("expected single EOF token, got %v", tokens)
	}
}

func TestLex_QueryTooLong(t *testing.T) {
	long := make([]byte, 300)
	for i := range long {
		long[i] = 'a'
	}
	_, err := lex(string(long), 256)
	if err == nil {
		t.Fatal("expected error for query too long")
	}
	errs := Errors(err)
	if len(errs) == 0 {
		t.Fatal("expected QueryError")
	}
	if errs[0].Kind != ErrQueryTooLong {
		t.Errorf("got kind %v, want %v", errs[0].Kind, ErrQueryTooLong)
	}
}

func TestLex_IllegalCharacter(t *testing.T) {
	_, err := lex("state=draft & active=true", 0)
	if err == nil {
		t.Fatal("expected error for illegal character")
	}
}

func TestLex_AtSymbol(t *testing.T) {
	tokens, err := lex("items@first", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTokenTypes(t, tokens, []TokenType{TokenIdent, TokenAt, TokenIdent, TokenEOF})
}

func TestLex_WhitespaceVariations(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{"spaces", "a=1 AND b=2"},
		{"tabs", "a=1\tAND\tb=2"},
		{"multiple spaces", "a=1   AND   b=2"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := lex(tt.input, 0)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			want := []TokenType{TokenIdent, TokenEq, TokenInteger, TokenAnd, TokenIdent, TokenEq, TokenInteger, TokenEOF}
			assertTokenTypes(t, tokens, want)
		})
	}
}

func TestLex_ValueEndsAtCloseParen(t *testing.T) {
	tokens, err := lex("(state=draft)", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := []TokenType{TokenLParen, TokenIdent, TokenEq, TokenString, TokenRParen, TokenEOF}
	assertTokenTypes(t, tokens, want)
	if tokens[3].Value != "draft" {
		t.Errorf("got value %q, want %q", tokens[3].Value, "draft")
	}
}

func TestLex_NegativeInteger(t *testing.T) {
	tokens, err := lex("offset>=-10", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTokenTypes(t, tokens, []TokenType{TokenIdent, TokenGte, TokenInteger, TokenEOF})
	if tokens[2].Value != "-10" {
		t.Errorf("got value %q, want %q", tokens[2].Value, "-10")
	}
}

func TestLex_IdentWithHyphen(t *testing.T) {
	tokens, err := lex("customer_id=customer_john-doe", 0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	assertTokenTypes(t, tokens, []TokenType{TokenIdent, TokenEq, TokenString, TokenEOF})
	if tokens[2].Value != "customer_john-doe" {
		t.Errorf("got value %q, want %q", tokens[2].Value, "customer_john-doe")
	}
}

// assertTokenTypes checks that the token stream has the expected types.
func assertTokenTypes(t *testing.T, got []Token, want []TokenType) {
	t.Helper()
	if len(got) != len(want) {
		types := make([]TokenType, len(got))
		for i, tok := range got {
			types[i] = tok.Type
		}
		t.Fatalf("got %d tokens %v, want %d %v", len(got), types, len(want), want)
	}
	for i, tok := range got {
		if tok.Type != want[i] {
			t.Errorf("token[%d]: got %v, want %v", i, tok.Type, want[i])
		}
	}
}

// assertTokenValues checks that the token stream has the expected values.
func assertTokenValues(t *testing.T, got []Token, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %d tokens, want %d values", len(got), len(want))
	}
	for i, tok := range got {
		if tok.Value != want[i] {
			t.Errorf("token[%d]: got value %q, want %q", i, tok.Value, want[i])
		}
	}
}
