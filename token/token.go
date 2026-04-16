package token

import "fmt"

// Position represents a location within a query string.
type Position struct {
	Offset int // byte offset from the start of the query
	Length int // length in bytes of the token
}

// String returns a human-readable representation of the position.
func (p Position) String() string {
	return fmt.Sprintf("offset %d, length %d", p.Offset, p.Length)
}

// Type represents the type of a lexical token.
type Type int

// Token type constants.
const (
	Illegal  Type = iota // unexpected character
	EOF                  // end of input
	Ident                // field names and identifiers
	String               // string values (after operators)
	Integer              // integer values
	Float                // float values (e.g., 50000.50)
	Date                 // date values (2026-01-01)
	Duration             // duration values (1d, 4h, 30m, 2w)
	Boolean              // true, false
	Eq                   // =
	Neq                  // !=
	Gt                   // >
	Gte                  // >=
	Lt                   // <
	Lte                  // <=
	Range                // ..
	Colon                // :
	And                  // AND
	Or                   // OR
	Not                  // NOT
	LParen               // (
	RParen               // )
	At                   // @
	Dot                  // .
	Comma                // ,
	Wildcard             // * (within string values)
)

var typeNames = [...]string{
	Illegal:  "ILLEGAL",
	EOF:      "EOF",
	Ident:    "IDENT",
	String:   "STRING",
	Integer:  "INTEGER",
	Float:    "FLOAT",
	Date:     "DATE",
	Duration: "DURATION",
	Boolean:  "BOOLEAN",
	Eq:       "=",
	Neq:      "!=",
	Gt:       ">",
	Gte:      ">=",
	Lt:       "<",
	Lte:      "<=",
	Range:    "..",
	Colon:    ":",
	And:      "AND",
	Or:       "OR",
	Not:      "NOT",
	LParen:   "(",
	RParen:   ")",
	At:       "@",
	Dot:      ".",
	Comma:    ",",
	Wildcard: "*",
}

// String returns the human-readable name of the token type.
func (t Type) String() string {
	if int(t) < len(typeNames) {
		return typeNames[t]
	}
	return fmt.Sprintf("Type(%d)", t)
}

// IsOperator reports whether the token type is a comparison operator.
func (t Type) IsOperator() bool {
	switch t { //nolint:exhaustive // only operator tokens return true
	case Eq, Neq, Gt, Gte, Lt, Lte, Colon:
		return true
	default:
		return false
	}
}

// IsLogical reports whether the token type is a logical operator (AND, OR).
func (t Type) IsLogical() bool {
	return t == And || t == Or
}

// Token represents a single lexical token with its type, value, and position.
type Token struct {
	Type  Type
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

// OperatorSymbol returns the string representation of a comparison operator.
func OperatorSymbol(op Type) string {
	switch op { //nolint:exhaustive // only comparison operators are relevant
	case Eq:
		return "="
	case Neq:
		return "!="
	case Gt:
		return ">"
	case Gte:
		return ">="
	case Lt:
		return "<"
	case Lte:
		return "<="
	case Range:
		return ".."
	case Colon:
		return ":"
	default:
		return "="
	}
}
