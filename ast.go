package query

import (
	"fmt"
	"strings"
	"time"
)

// Node is the common interface for all AST nodes.
type Node interface {
	Pos() Position
	node() // marker method restricting implementations to this package
}

// Expression is an AST node that represents a query expression.
type Expression interface {
	Node
	expr() // marker method restricting implementations to this package
}

// FieldPath represents a dotted field path like "labels.dev" as ["labels", "dev"].
type FieldPath []string

// String returns the dotted representation of the field path.
func (fp FieldPath) String() string {
	return strings.Join(fp, ".")
}

// ValueType identifies the type of a parsed value.
type ValueType int

// Value type constants.
const (
	ValueString   ValueType = iota // plain string
	ValueInteger                   // integer number
	ValueFloat                     // floating-point number
	ValueBoolean                   // true or false
	ValueDate                      // date (YYYY-MM-DD)
	ValueDuration                  // duration (1d, 4h, etc.)
)

var valueTypeNames = [...]string{
	ValueString:   "string",
	ValueInteger:  "integer",
	ValueFloat:    "float",
	ValueBoolean:  "boolean",
	ValueDate:     "date",
	ValueDuration: "duration",
}

// String returns the name of the value type.
func (v ValueType) String() string {
	if int(v) < len(valueTypeNames) {
		return valueTypeNames[v]
	}
	return fmt.Sprintf("ValueType(%d)", v)
}

// Value represents a typed value in a qualifier expression.
type Value struct {
	Type     ValueType
	Raw      string        // original string from the query
	Str      string        // for string values
	Int      int64         // for integer values
	Float    float64       // for float values
	Bool     bool          // for boolean values
	Date     time.Time     // for date values
	Duration time.Duration // for duration values
	Wildcard bool          // true if the value contains wildcards
}

// BinaryExpr represents a binary logical expression: left AND right, left OR right.
type BinaryExpr struct {
	Op       TokenType  // TokenAnd or TokenOr
	Left     Expression // left operand
	Right    Expression // right operand
	Position Position
}

// Pos returns the position of the binary expression.
func (e *BinaryExpr) Pos() Position { return e.Position }
func (e *BinaryExpr) node()         {}
func (e *BinaryExpr) expr()         {}

// UnaryExpr represents a unary expression: NOT expr.
type UnaryExpr struct {
	Op       TokenType  // TokenNot
	Expr     Expression // operand
	Position Position
}

// Pos returns the position of the unary expression.
func (e *UnaryExpr) Pos() Position { return e.Position }
func (e *UnaryExpr) node()         {}
func (e *UnaryExpr) expr()         {}

// QualifierExpr represents a field comparison: field op value.
// For range expressions (field:start..end), EndValue is non-nil.
type QualifierExpr struct {
	Field    FieldPath // field path (e.g., ["labels", "dev"])
	Operator TokenType // comparison operator token type
	Value    Value     // primary value
	EndValue *Value    // end value for range expressions (field:start..end)
	Position Position
}

// Pos returns the position of the qualifier expression.
func (e *QualifierExpr) Pos() Position { return e.Position }
func (e *QualifierExpr) node()         {}
func (e *QualifierExpr) expr()         {}

// PresenceExpr represents a field presence check: just the field name with no operator.
type PresenceExpr struct {
	Field    FieldPath // field path
	Position Position
}

// Pos returns the position of the presence expression.
func (e *PresenceExpr) Pos() Position { return e.Position }
func (e *PresenceExpr) node()         {}
func (e *PresenceExpr) expr()         {}

// SelectorExpr represents a selector expression: expr @first, expr @last, or expr @(inner).
type SelectorExpr struct {
	Base     Expression // base expression
	Selector string     // "first", "last", or "" for @(...)
	Inner    Expression // inner expression for @(...)
	Position Position
}

// Pos returns the position of the selector expression.
func (e *SelectorExpr) Pos() Position { return e.Position }
func (e *SelectorExpr) node()         {}
func (e *SelectorExpr) expr()         {}

// GroupExpr represents a parenthesized expression: (expression).
type GroupExpr struct {
	Expr     Expression // inner expression
	Position Position
}

// Pos returns the position of the group expression.
func (e *GroupExpr) Pos() Position { return e.Position }
func (e *GroupExpr) node()         {}
func (e *GroupExpr) expr()         {}

// Walk traverses the AST depth-first, calling fn for each node.
// If fn returns false, children of that node are not visited.
func Walk(expr Expression, fn func(Expression) bool) {
	if expr == nil || !fn(expr) {
		return
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		Walk(e.Left, fn)
		Walk(e.Right, fn)
	case *UnaryExpr:
		Walk(e.Expr, fn)
	case *GroupExpr:
		Walk(e.Expr, fn)
	case *SelectorExpr:
		Walk(e.Base, fn)
		if e.Inner != nil {
			Walk(e.Inner, fn)
		}
	case *QualifierExpr, *PresenceExpr:
		// leaf nodes, no children
	}
}

// String serializes an AST expression back to a query string.
func String(expr Expression) string {
	if expr == nil {
		return ""
	}
	var buf strings.Builder
	writeExpr(&buf, expr)
	return buf.String()
}

func writeExpr(buf *strings.Builder, expr Expression) {
	switch e := expr.(type) {
	case *BinaryExpr:
		writeExpr(buf, e.Left)
		if e.Op == TokenAnd {
			buf.WriteString(" AND ")
		} else {
			buf.WriteString(" OR ")
		}
		writeExpr(buf, e.Right)
	case *UnaryExpr:
		buf.WriteString("NOT ")
		writeExpr(buf, e.Expr)
	case *QualifierExpr:
		buf.WriteString(e.Field.String())
		if e.EndValue != nil {
			buf.WriteByte(':')
			buf.WriteString(e.Value.Raw)
			buf.WriteString("..")
			buf.WriteString(e.EndValue.Raw)
		} else {
			buf.WriteString(operatorString(e.Operator))
			buf.WriteString(e.Value.Raw)
		}
	case *PresenceExpr:
		buf.WriteString(e.Field.String())
	case *GroupExpr:
		buf.WriteByte('(')
		writeExpr(buf, e.Expr)
		buf.WriteByte(')')
	case *SelectorExpr:
		writeExpr(buf, e.Base)
		buf.WriteByte('@')
		if e.Selector != "" {
			buf.WriteString(e.Selector)
		} else if e.Inner != nil {
			buf.WriteByte('(')
			writeExpr(buf, e.Inner)
			buf.WriteByte(')')
		}
	}
}

func operatorString(op TokenType) string {
	switch op {
	case TokenEq:
		return "="
	case TokenNeq:
		return "!="
	case TokenGt:
		return ">"
	case TokenGte:
		return ">="
	case TokenLt:
		return "<"
	case TokenLte:
		return "<="
	case TokenRange:
		return ":"
	default:
		return "="
	}
}
