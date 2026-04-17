package main

import (
	"fmt"
	"strings"

	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
)

// renderTree renders an AST expression as a tree with box-drawing characters.
func renderTree(expr ast.Expression, showPositions bool) string {
	var buf strings.Builder
	writeNode(&buf, expr, "", true, true, showPositions)
	return buf.String()
}

// writeNode writes a single AST node and its children to the buffer.
// isRoot controls whether the connector prefix is omitted (for the root node).
// isLast controls whether └── or ├── is used.
func writeNode(buf *strings.Builder, expr ast.Expression, prefix string, isRoot, isLast bool, showPos bool) {
	// Determine connector
	connector := "├── "
	childPrefix := prefix + "│   "
	if isRoot {
		connector = ""
		childPrefix = ""
	} else if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		opName := "AndExpr"
		if e.Op == token.Or {
			opName = "OrExpr"
		}
		buf.WriteString(prefix + connector + opName)
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')
		writeNode(buf, e.Left, childPrefix, false, false, showPos)
		writeNode(buf, e.Right, childPrefix, false, true, showPos)

	case *ast.UnaryExpr:
		buf.WriteString(prefix + connector + "NotExpr")
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')
		writeNode(buf, e.Expr, childPrefix, false, true, showPos)

	case *ast.QualifierExpr:
		label := fmt.Sprintf("QualifierExpr (%s)", token.OperatorSymbol(e.Operator))
		buf.WriteString(prefix + connector + label)
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')

		// Count children for isLast logic
		children := 2 // field + value (always present)
		if e.HasFieldFunc() {
			children++
		}
		if e.IsRange() {
			children++ // end value
		}
		idx := 0

		if e.HasFieldFunc() {
			idx++
			writeLeaf(buf, childPrefix, idx == children, showPos,
				fmt.Sprintf("FieldFunc: %s(%s)", e.FieldFunc.Name, e.Field.String()),
				e.FieldFunc.Position)
		} else {
			idx++
			writeLeaf(buf, childPrefix, idx == children, showPos,
				fmt.Sprintf("Field: %s", e.Field.String()),
				e.Position)
		}

		idx++
		writeLeaf(buf, childPrefix, idx == children, showPos,
			fmt.Sprintf("Value: %s (%s)", e.Value.Raw, e.Value.Type),
			e.Position)

		if e.IsRange() {
			idx++
			writeLeaf(buf, childPrefix, idx == children, showPos,
				fmt.Sprintf("EndValue: %s (%s)", e.EndValue.Raw, e.EndValue.Type),
				e.Position)
		}

	case *ast.PresenceExpr:
		buf.WriteString(prefix + connector + "PresenceExpr")
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')
		writeLeaf(buf, childPrefix, true, showPos,
			fmt.Sprintf("Field: %s", e.Field.String()),
			e.Position)

	case *ast.SelectorExpr:
		selectorLabel := "SelectorExpr"
		if e.Selector != "" {
			selectorLabel = fmt.Sprintf("SelectorExpr (@%s)", e.Selector)
		} else {
			selectorLabel = "SelectorExpr (@(...))"
		}
		buf.WriteString(prefix + connector + selectorLabel)
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')

		hasInner := e.Inner != nil
		writeNode(buf, e.Base, childPrefix, false, !hasInner, showPos)
		if hasInner {
			writeNode(buf, e.Inner, childPrefix, false, true, showPos)
		}

	case *ast.GroupExpr:
		buf.WriteString(prefix + connector + "GroupExpr")
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')
		writeNode(buf, e.Expr, childPrefix, false, true, showPos)

	case *ast.FuncCallExpr:
		label := fmt.Sprintf("FuncCallExpr (%s)", e.Name)
		buf.WriteString(prefix + connector + label)
		writePos(buf, e.Position, showPos)
		buf.WriteByte('\n')

		for i, arg := range e.Args {
			last := i == len(e.Args)-1
			if arg.Call != nil {
				writeNode(buf, arg.Call, childPrefix, false, last, showPos)
			} else {
				writeLeaf(buf, childPrefix, last, showPos,
					fmt.Sprintf("Arg: %s", arg.String()),
					e.Position)
			}
		}
	}
}

// writeLeaf writes a leaf node (no children) to the buffer.
func writeLeaf(buf *strings.Builder, prefix string, isLast, showPos bool, label string, pos token.Position) {
	connector := "├── "
	if isLast {
		connector = "└── "
	}
	buf.WriteString(prefix + connector + label)
	writePos(buf, pos, showPos)
	buf.WriteByte('\n')
}

// writePos appends a position annotation if showPos is true.
func writePos(buf *strings.Builder, pos token.Position, showPos bool) {
	if showPos {
		fmt.Fprintf(buf, " [%d:%d]", pos.Offset, pos.Length)
	}
}
