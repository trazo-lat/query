package ast

import "testing"

// TestMarkers explicitly invokes the marker methods so they show up in
// coverage. They're normally called implicitly by interface satisfaction.
func TestMarkers(t *testing.T) {
	nodes := []Expression{
		&BinaryExpr{},
		&UnaryExpr{},
		&QualifierExpr{},
		&PresenceExpr{},
		&GroupExpr{},
		&SelectorExpr{},
		&FuncCallExpr{},
	}
	for _, n := range nodes {
		// Accessing .expr() / .node() through the concrete types — these
		// are unexported so we need a type switch to call them.
		switch e := n.(type) {
		case *BinaryExpr:
			e.node()
			e.expr()
		case *UnaryExpr:
			e.node()
			e.expr()
		case *QualifierExpr:
			e.node()
			e.expr()
		case *PresenceExpr:
			e.node()
			e.expr()
		case *GroupExpr:
			e.node()
			e.expr()
		case *SelectorExpr:
			e.node()
			e.expr()
		case *FuncCallExpr:
			e.node()
			e.expr()
		}
	}
}

func TestWalk_FuncCallNested(t *testing.T) {
	fp := FieldPath{"name"}
	inner := &FuncCallExpr{Name: "lower", Args: []FuncArg{{Field: &fp}}}
	outer := &FuncCallExpr{Name: "outer", Args: []FuncArg{{Call: inner}, {Field: &fp}}}

	count := 0
	Walk(outer, func(Expression) bool {
		count++
		return true
	})
	if count != 2 {
		t.Errorf("got %d visits, want 2", count)
	}
}
