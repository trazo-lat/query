package query

import "fmt"

// Position represents a location within a query string.
type Position struct {
	Offset int // byte offset from the start of the query
	Length int // length in bytes of the token
}

// Pos returns the position itself, satisfying the Node interface.
func (p Position) Pos() Position { return p }

// String returns a human-readable representation of the position.
func (p Position) String() string {
	return fmt.Sprintf("offset %d, length %d", p.Offset, p.Length)
}

// TokenType represents the type of a lexical token.
type TokenType int

// Token type constants.
const (
	TokenIllegal  TokenType = iota // unexpected character
	TokenEOF                       // end of input
	TokenIdent                     // field names and identifiers
	TokenString                    // string values (after operators)
	TokenInteger                   // integer values
	TokenFloat                     // float values (e.g., 50000.50)
	TokenDate                      // date values (2026-01-01)
	TokenDuration                  // duration values (1d, 4h, 30m, 2w)
	TokenBoolean                   // true, false
	TokenEq                        // =
	TokenNeq                       // !=
	TokenGt                        // >
	TokenGte                       // >=
	TokenLt                        // <
	TokenLte                       // <=
	TokenRange                     // ..
	TokenColon                     // :
	TokenAnd                       // AND
	TokenOr                        // OR
	TokenNot                       // NOT
	TokenLParen                    // (
	TokenRParen                    // )
	TokenAt                        // @
	TokenDot                       // .
	TokenWildcard                  // * (within string values)
)

var tokenNames = [...]string{
	TokenIllegal:  "ILLEGAL",
	TokenEOF:      "EOF",
	TokenIdent:    "IDENT",
	TokenString:   "STRING",
	TokenInteger:  "INTEGER",
	TokenFloat:    "FLOAT",
	TokenDate:     "DATE",
	TokenDuration: "DURATION",
	TokenBoolean:  "BOOLEAN",
	TokenEq:       "=",
	TokenNeq:      "!=",
	TokenGt:       ">",
	TokenGte:      ">=",
	TokenLt:       "<",
	TokenLte:      "<=",
	TokenRange:    "..",
	TokenColon:    ":",
	TokenAnd:      "AND",
	TokenOr:       "OR",
	TokenNot:      "NOT",
	TokenLParen:   "(",
	TokenRParen:   ")",
	TokenAt:       "@",
	TokenDot:      ".",
	TokenWildcard: "*",
}

// String returns the human-readable name of the token type.
func (t TokenType) String() string {
	if int(t) < len(tokenNames) {
		return tokenNames[t]
	}
	return fmt.Sprintf("TokenType(%d)", t)
}

// IsOperator reports whether the token type is a comparison operator.
func (t TokenType) IsOperator() bool {
	switch t { //nolint:exhaustive // only operator tokens return true
	case TokenEq, TokenNeq, TokenGt, TokenGte, TokenLt, TokenLte, TokenColon:
		return true
	default:
		return false
	}
}

// Token represents a single lexical token with its type, value, and position.
type Token struct {
	Type  TokenType
	Value string
	Pos   Position
}

// String returns a debug representation of the token.
func (t Token) String() string {
	if t.Value != "" {
		return fmt.Sprintf("%s(%q)", t.Type, t.Value)
	}
	return t.Type.String()
}
