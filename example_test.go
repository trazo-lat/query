package query_test

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/trazo-lat/query"
	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
	"github.com/trazo-lat/query/validate"
)

func Example_parse() {
	expr, err := query.Parse("state=draft AND total>50000")
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println(ast.String(expr))
	// Output: state=draft AND total>50000
}

func Example_parseAndValidate() {
	fields := []validate.FieldConfig{
		{Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
		{Name: "total", Type: validate.TypeDecimal, AllowedOps: validate.NumericOps},
	}

	expr, err := query.ParseAndValidate("state=draft AND total>50000", fields)
	if err != nil {
		fmt.Println("error:", err)
		return
	}
	fmt.Println("fields:", len(ast.Fields(expr)))
	fmt.Println("simple:", ast.IsSimple(expr))
	// Output:
	// fields: 2
	// simple: false
}

// ---------------------------------------------------------------------------
// SQL generation via Visitor
// ---------------------------------------------------------------------------

// sqlVisitor transforms an AST into a SQL WHERE clause with numbered params.
type sqlVisitor struct {
	params []any
}

func (v *sqlVisitor) VisitBinary(e *ast.BinaryExpr) string {
	left := ast.Visit[string](v, e.Left)
	right := ast.Visit[string](v, e.Right)
	if e.Op == token.And {
		return left + " AND " + right
	}
	return left + " OR " + right
}

func (v *sqlVisitor) VisitUnary(e *ast.UnaryExpr) string {
	inner := ast.Visit[string](v, e.Expr)
	return "NOT (" + inner + ")"
}

func (v *sqlVisitor) VisitQualifier(e *ast.QualifierExpr) string {
	field := e.Field.String()
	if e.IsRange() {
		v.params = append(v.params, e.Value.Any(), e.EndValue.Any())
		return fmt.Sprintf("%s BETWEEN $%d AND $%d", field, len(v.params)-1, len(v.params))
	}
	if e.IsWildcard() {
		v.params = append(v.params, ast.WildcardToLike(e.Value.Str))
		return fmt.Sprintf("%s LIKE $%d", field, len(v.params))
	}
	op := ast.SQLOperator(e.Operator, false)
	v.params = append(v.params, e.Value.Any())
	return fmt.Sprintf("%s %s $%d", field, op, len(v.params))
}

func (v *sqlVisitor) VisitPresence(e *ast.PresenceExpr) string {
	return e.Field.String() + " IS NOT NULL"
}

func (v *sqlVisitor) VisitGroup(e *ast.GroupExpr) string {
	inner := ast.Visit[string](v, e.Expr)
	return "(" + inner + ")"
}

func (v *sqlVisitor) VisitSelector(e *ast.SelectorExpr) string {
	return ast.Visit[string](v, e.Base)
}

func Example_sqlGeneration() {
	expr, _ := query.Parse("(state=draft OR state=issued) AND total>50000")
	v := &sqlVisitor{}
	where := ast.Visit[string](v, expr)
	fmt.Println("WHERE", where)
	fmt.Println("params:", v.params)
	// Output:
	// WHERE (state = $1 OR state = $2) AND total > $3
	// params: [draft issued 50000]
}

func Example_sqlWildcard() {
	expr, _ := query.Parse("name=John*")
	v := &sqlVisitor{}
	where := ast.Visit[string](v, expr)
	fmt.Println("WHERE", where)
	fmt.Println("params:", v.params)
	// Output:
	// WHERE name LIKE $1
	// params: [John%]
}

func Example_sqlRange() {
	expr, _ := query.Parse("created_at:2026-01-01..2026-03-31")
	v := &sqlVisitor{}
	where := ast.Visit[string](v, expr)
	fmt.Println("WHERE", where)
	// Output:
	// WHERE created_at BETWEEN $1 AND $2
}

// ---------------------------------------------------------------------------
// JSON generation via Visitor
// ---------------------------------------------------------------------------

// jsonNode is a JSON-serializable representation of a query AST.
type jsonNode struct {
	Type     string      `json:"type"`
	Op       string      `json:"op,omitempty"`
	Field    string      `json:"field,omitempty"`
	Value    any         `json:"value,omitempty"`
	EndValue any         `json:"endValue,omitempty"`
	Wildcard bool        `json:"wildcard,omitempty"`
	Left     *jsonNode   `json:"left,omitempty"`
	Right    *jsonNode   `json:"right,omitempty"`
	Expr     *jsonNode   `json:"expr,omitempty"`
	Children []*jsonNode `json:"children,omitempty"`
}

type jsonVisitor struct{}

func (v *jsonVisitor) VisitBinary(e *ast.BinaryExpr) *jsonNode {
	op := "AND"
	if e.Op == token.Or {
		op = "OR"
	}
	return &jsonNode{
		Type:  "binary",
		Op:    op,
		Left:  ast.Visit[*jsonNode](v, e.Left),
		Right: ast.Visit[*jsonNode](v, e.Right),
	}
}

func (v *jsonVisitor) VisitUnary(e *ast.UnaryExpr) *jsonNode {
	return &jsonNode{
		Type: "not",
		Expr: ast.Visit[*jsonNode](v, e.Expr),
	}
}

func (v *jsonVisitor) VisitQualifier(e *ast.QualifierExpr) *jsonNode {
	n := &jsonNode{
		Type:     "qualifier",
		Field:    e.Field.String(),
		Op:       token.OperatorSymbol(e.Operator),
		Value:    e.Value.Any(),
		Wildcard: e.IsWildcard(),
	}
	if e.IsRange() {
		n.Op = ".."
		n.EndValue = e.EndValue.Any()
	}
	return n
}

func (v *jsonVisitor) VisitPresence(e *ast.PresenceExpr) *jsonNode {
	return &jsonNode{Type: "presence", Field: e.Field.String()}
}

func (v *jsonVisitor) VisitGroup(e *ast.GroupExpr) *jsonNode {
	return &jsonNode{
		Type: "group",
		Expr: ast.Visit[*jsonNode](v, e.Expr),
	}
}

func (v *jsonVisitor) VisitSelector(e *ast.SelectorExpr) *jsonNode {
	return ast.Visit[*jsonNode](v, e.Base)
}

func Example_jsonGeneration() {
	expr, _ := query.Parse("state=draft AND total>50000")
	jv := &jsonVisitor{}
	node := ast.Visit[*jsonNode](jv, expr)

	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(node)
	fmt.Print(buf.String())
	// Output:
	// {
	//   "type": "binary",
	//   "op": "AND",
	//   "left": {
	//     "type": "qualifier",
	//     "op": "=",
	//     "field": "state",
	//     "value": "draft"
	//   },
	//   "right": {
	//     "type": "qualifier",
	//     "op": ">",
	//     "field": "total",
	//     "value": 50000
	//   }
	// }
}

// ---------------------------------------------------------------------------
// Extracting fields and qualifiers
// ---------------------------------------------------------------------------

func Example_fields() {
	expr, _ := query.Parse("(labels.dev=jane OR labels.env=bar) AND NOT cluster=demo")
	for _, fp := range ast.Fields(expr) {
		fmt.Println(fp.String())
	}
	// Output:
	// labels.dev
	// labels.env
	// cluster
}

func Example_qualifiers() {
	expr, _ := query.Parse("state=draft AND year>2020 AND name=John*")
	for _, q := range ast.Qualifiers(expr) {
		fmt.Printf("%-10s %-3s %s\n", q.Field, token.OperatorSymbol(q.Operator), q.Value.Raw)
	}
	// Output:
	// state      =   draft
	// year       >   2020
	// name       =   John*
}

// ---------------------------------------------------------------------------
// Walk — custom traversal
// ---------------------------------------------------------------------------

func Example_walk() {
	expr, _ := query.Parse("(a=1 OR b=2) AND c=3")

	// Collect all field names
	var fields []string
	ast.Walk(expr, func(e ast.Expression) bool {
		switch n := e.(type) {
		case *ast.QualifierExpr:
			fields = append(fields, n.Field.String())
		case *ast.PresenceExpr:
			fields = append(fields, n.Field.String())
		}
		return true
	})
	fmt.Println(strings.Join(fields, ", "))
	// Output: a, b, c
}

func Example_depth() {
	expr, _ := query.Parse("(a=1 OR b=2) AND c=3")
	fmt.Println("depth:", ast.Depth(expr))
	// Output: depth: 4
}

// ---------------------------------------------------------------------------
// Filter function generation via Visitor
// ---------------------------------------------------------------------------

// filterVisitor builds a Go predicate function from an AST. This demonstrates
// how a consumer might evaluate queries against in-memory objects (e.g., for
// client-side filtering in CLI tools or WASM).
type filterVisitor struct{}

func (v *filterVisitor) VisitBinary(e *ast.BinaryExpr) func(map[string]any) bool {
	left := ast.Visit[func(map[string]any) bool](v, e.Left)
	right := ast.Visit[func(map[string]any) bool](v, e.Right)
	if e.Op == token.And {
		return func(obj map[string]any) bool { return left(obj) && right(obj) }
	}
	return func(obj map[string]any) bool { return left(obj) || right(obj) }
}

func (v *filterVisitor) VisitUnary(e *ast.UnaryExpr) func(map[string]any) bool {
	inner := ast.Visit[func(map[string]any) bool](v, e.Expr)
	return func(obj map[string]any) bool { return !inner(obj) }
}

func (v *filterVisitor) VisitQualifier(e *ast.QualifierExpr) func(map[string]any) bool {
	field := e.Field.String()
	expected := e.Value.Any()
	return func(obj map[string]any) bool {
		val, ok := obj[field]
		if !ok {
			return false
		}
		switch e.Operator {
		case token.Eq:
			return fmt.Sprint(val) == fmt.Sprint(expected)
		case token.Neq:
			return fmt.Sprint(val) != fmt.Sprint(expected)
		default:
			return false
		}
	}
}

func (v *filterVisitor) VisitPresence(e *ast.PresenceExpr) func(map[string]any) bool {
	field := e.Field.String()
	return func(obj map[string]any) bool {
		_, ok := obj[field]
		return ok
	}
}

func (v *filterVisitor) VisitGroup(e *ast.GroupExpr) func(map[string]any) bool {
	return ast.Visit[func(map[string]any) bool](v, e.Expr)
}

func (v *filterVisitor) VisitSelector(e *ast.SelectorExpr) func(map[string]any) bool {
	return ast.Visit[func(map[string]any) bool](v, e.Base)
}

func Example_filterFunction() {
	expr, _ := query.Parse("state=draft AND category!=archived")

	fv := &filterVisitor{}
	matches := ast.Visit[func(map[string]any) bool](fv, expr)

	items := []map[string]any{
		{"state": "draft", "category": "active"},
		{"state": "draft", "category": "archived"},
		{"state": "published", "category": "active"},
	}
	for _, item := range items {
		fmt.Printf("state=%-10s category=%-10s match=%v\n",
			item["state"], item["category"], matches(item))
	}
	// Output:
	// state=draft      category=active     match=true
	// state=draft      category=archived   match=false
	// state=published  category=active     match=false
}

// ---------------------------------------------------------------------------
// React component hint generation — shows how to build a UI description
// ---------------------------------------------------------------------------

// uiNode represents a React-like component tree for rendering a query editor.
type uiNode struct {
	Component string    `json:"component"`
	Props     uiProps   `json:"props,omitempty"`
	Children  []*uiNode `json:"children,omitempty"`
}

type uiProps struct {
	Field    string `json:"field,omitempty"`
	Operator string `json:"operator,omitempty"`
	Value    string `json:"value,omitempty"`
	Logic    string `json:"logic,omitempty"`
}

type uiVisitor struct{}

func (v *uiVisitor) VisitBinary(e *ast.BinaryExpr) *uiNode {
	logic := "AND"
	if e.Op == token.Or {
		logic = "OR"
	}
	return &uiNode{
		Component: "LogicGroup",
		Props:     uiProps{Logic: logic},
		Children: []*uiNode{
			ast.Visit[*uiNode](v, e.Left),
			ast.Visit[*uiNode](v, e.Right),
		},
	}
}

func (v *uiVisitor) VisitUnary(e *ast.UnaryExpr) *uiNode {
	return &uiNode{
		Component: "NegationWrapper",
		Children:  []*uiNode{ast.Visit[*uiNode](v, e.Expr)},
	}
}

func (v *uiVisitor) VisitQualifier(e *ast.QualifierExpr) *uiNode {
	return &uiNode{
		Component: "FilterChip",
		Props: uiProps{
			Field:    e.Field.String(),
			Operator: token.OperatorSymbol(e.Operator),
			Value:    e.Value.Raw,
		},
	}
}

func (v *uiVisitor) VisitPresence(e *ast.PresenceExpr) *uiNode {
	return &uiNode{
		Component: "FilterChip",
		Props:     uiProps{Field: e.Field.String(), Operator: "exists"},
	}
}

func (v *uiVisitor) VisitGroup(e *ast.GroupExpr) *uiNode {
	return ast.Visit[*uiNode](v, e.Expr)
}

func (v *uiVisitor) VisitSelector(e *ast.SelectorExpr) *uiNode {
	return ast.Visit[*uiNode](v, e.Base)
}

func Example_reactComponentTree() {
	expr, _ := query.Parse("state=draft AND total>50000")
	uv := &uiVisitor{}
	tree := ast.Visit[*uiNode](uv, expr)

	var buf strings.Builder
	enc := json.NewEncoder(&buf)
	enc.SetIndent("", "  ")
	enc.SetEscapeHTML(false)
	_ = enc.Encode(tree)
	fmt.Print(buf.String())
	// Output:
	// {
	//   "component": "LogicGroup",
	//   "props": {
	//     "logic": "AND"
	//   },
	//   "children": [
	//     {
	//       "component": "FilterChip",
	//       "props": {
	//         "field": "state",
	//         "operator": "=",
	//         "value": "draft"
	//       }
	//     },
	//     {
	//       "component": "FilterChip",
	//       "props": {
	//         "field": "total",
	//         "operator": ">",
	//         "value": "50000"
	//       }
	//     }
	//   ]
	// }
}
