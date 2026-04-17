package main

import (
	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
)

// jsonNode is the JSON-serializable representation of an AST node.
type jsonNode struct {
	Type      string      `json:"type"`
	Position  *jsonPos    `json:"position,omitempty"`
	Op        string      `json:"op,omitempty"`
	Field     string      `json:"field,omitempty"`
	Value     string      `json:"value,omitempty"`
	ValueT    string      `json:"value_type,omitempty"`
	EndValue  string      `json:"end_value,omitempty"`
	EndValueT string      `json:"end_value_type,omitempty"`
	Selector  string      `json:"selector,omitempty"`
	FuncName  string      `json:"func_name,omitempty"`
	Args      []jsonArg   `json:"args,omitempty"`
	Children  []*jsonNode `json:"children,omitempty"`
}

type jsonPos struct {
	Offset int `json:"offset"`
	Length int `json:"length"`
}

type jsonArg struct {
	Type  string `json:"type"`
	Value string `json:"value"`
}

// astToJSON converts an AST expression into a JSON-serializable structure.
func astToJSON(expr ast.Expression, showPos bool) *jsonNode {
	if expr == nil {
		return nil
	}

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		opName := "AND"
		if e.Op == token.Or {
			opName = "OR"
		}
		node := &jsonNode{
			Type: "BinaryExpr",
			Op:   opName,
			Children: []*jsonNode{
				astToJSON(e.Left, showPos),
				astToJSON(e.Right, showPos),
			},
		}
		setPos(node, e.Position, showPos)
		return node

	case *ast.UnaryExpr:
		node := &jsonNode{
			Type:     "UnaryExpr",
			Op:       "NOT",
			Children: []*jsonNode{astToJSON(e.Expr, showPos)},
		}
		setPos(node, e.Position, showPos)
		return node

	case *ast.QualifierExpr:
		node := &jsonNode{
			Type:   "QualifierExpr",
			Op:     token.OperatorSymbol(e.Operator),
			Field:  e.Field.String(),
			Value:  e.Value.Raw,
			ValueT: e.Value.Type.String(),
		}
		if e.HasFieldFunc() {
			node.FuncName = e.FieldFunc.Name
		}
		if e.IsRange() {
			node.EndValue = e.EndValue.Raw
			node.EndValueT = e.EndValue.Type.String()
		}
		setPos(node, e.Position, showPos)
		return node

	case *ast.PresenceExpr:
		node := &jsonNode{
			Type:  "PresenceExpr",
			Field: e.Field.String(),
		}
		setPos(node, e.Position, showPos)
		return node

	case *ast.SelectorExpr:
		node := &jsonNode{
			Type:     "SelectorExpr",
			Selector: e.Selector,
			Children: []*jsonNode{astToJSON(e.Base, showPos)},
		}
		if e.Inner != nil {
			node.Children = append(node.Children, astToJSON(e.Inner, showPos))
		}
		setPos(node, e.Position, showPos)
		return node

	case *ast.GroupExpr:
		node := &jsonNode{
			Type:     "GroupExpr",
			Children: []*jsonNode{astToJSON(e.Expr, showPos)},
		}
		setPos(node, e.Position, showPos)
		return node

	case *ast.FuncCallExpr:
		node := &jsonNode{
			Type:     "FuncCallExpr",
			FuncName: e.Name,
		}
		for _, arg := range e.Args {
			ja := jsonArg{}
			switch {
			case arg.Field != nil:
				ja.Type = "field"
				ja.Value = arg.Field.String()
			case arg.Value != nil:
				ja.Type = arg.Value.Type.String()
				ja.Value = arg.Value.Raw
			case arg.Call != nil:
				ja.Type = "call"
				ja.Value = arg.Call.Name
				node.Children = append(node.Children, astToJSON(arg.Call, showPos))
			}
			node.Args = append(node.Args, ja)
		}
		setPos(node, e.Position, showPos)
		return node

	default:
		return nil
	}
}

func setPos(node *jsonNode, pos token.Position, showPos bool) {
	if showPos {
		node.Position = &jsonPos{Offset: pos.Offset, Length: pos.Length}
	}
}
