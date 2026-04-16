package ast

import (
	"github.com/trazo-lat/query/token"
)

// Node is the common interface for all AST nodes.
type Node interface {
	Pos() token.Position
	node() // marker — restricts implementations to this package
}

// Expression is an AST node that represents a query expression.
type Expression interface {
	Node
	expr() // marker — restricts implementations to this package
}

// BinaryExpr represents a binary logical expression: left AND right, left OR right.
type BinaryExpr struct {
	Op       token.Type // token.And or token.Or
	Left     Expression
	Right    Expression
	Position token.Position
}

// Pos returns the position of the binary expression.
func (e *BinaryExpr) Pos() token.Position { return e.Position }
func (e *BinaryExpr) node()               {}
func (e *BinaryExpr) expr()               {}

// UnaryExpr represents a unary expression: NOT expr.
type UnaryExpr struct {
	Op       token.Type // token.Not
	Expr     Expression
	Position token.Position
}

// Pos returns the position of the unary expression.
func (e *UnaryExpr) Pos() token.Position { return e.Position }
func (e *UnaryExpr) node()               {}
func (e *UnaryExpr) expr()               {}

// QualifierExpr represents a field comparison: field op value.
// For range expressions (field:start..end), EndValue is non-nil.
type QualifierExpr struct {
	Field    FieldPath  // e.g., ["labels", "dev"]
	Operator token.Type // comparison operator
	Value    Value      // primary value
	EndValue *Value     // end value for range expressions
	Position token.Position
}

// Pos returns the position of the qualifier expression.
func (e *QualifierExpr) Pos() token.Position { return e.Position }
func (e *QualifierExpr) node()               {}
func (e *QualifierExpr) expr()               {}

// IsRange reports whether this is a range expression (field:start..end).
func (e *QualifierExpr) IsRange() bool { return e.EndValue != nil }

// IsWildcard reports whether this qualifier uses a wildcard value.
func (e *QualifierExpr) IsWildcard() bool { return e.Value.Wildcard }

// PresenceExpr represents a field presence check: just the field name with no operator.
type PresenceExpr struct {
	Field    FieldPath
	Position token.Position
}

// Pos returns the position of the presence expression.
func (e *PresenceExpr) Pos() token.Position { return e.Position }
func (e *PresenceExpr) node()               {}
func (e *PresenceExpr) expr()               {}

// SelectorExpr represents a selector expression: expr @first, expr @last, or expr @(inner).
type SelectorExpr struct {
	Base     Expression
	Selector string     // "first", "last", or "" for @(...)
	Inner    Expression // inner expression for @(...)
	Position token.Position
}

// Pos returns the position of the selector expression.
func (e *SelectorExpr) Pos() token.Position { return e.Position }
func (e *SelectorExpr) node()               {}
func (e *SelectorExpr) expr()               {}

// GroupExpr represents a parenthesized expression: (expression).
type GroupExpr struct {
	Expr     Expression
	Position token.Position
}

// Pos returns the position of the group expression.
func (e *GroupExpr) Pos() token.Position { return e.Position }
func (e *GroupExpr) node()               {}
func (e *GroupExpr) expr()               {}
