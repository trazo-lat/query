# query

[![CI](https://github.com/trazo-lat/query/actions/workflows/ci.yml/badge.svg)](https://github.com/trazo-lat/query/actions/workflows/ci.yml)

Pure Go query language library for Trazo. Handles lexing, parsing, AST construction, and validation of a unified query syntax used across all clients (CLI, Web UI, API, VS Code, WASM).

Zero external dependencies. This library is the shared baseline for query parsing and validation -- all semantic evaluation (SQL generation, execution) lives in the consumer.

## Install

```bash
go get github.com/trazo-lat/query
```

## Packages

| Package | Purpose |
|---------|---------|
| `query` | Top-level API: `Parse()`, `Validate()`, `ParseAndValidate()` |
| `query/token` | Lexical token types and position tracking |
| `query/ast` | AST nodes, `Visitor[T]` pattern, `Walk`, `String` |
| `query/parser` | Lexer and recursive descent parser |
| `query/validate` | Field configuration and AST validation |

## Usage

### Parse and validate

```go
import (
    "github.com/trazo-lat/query"
    "github.com/trazo-lat/query/validate"
)

expr, err := query.Parse("state=draft AND total>50000")

fields := []validate.FieldConfig{
    {Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
    {Name: "total", Type: validate.TypeDecimal, AllowedOps: validate.NumericOps},
}
err = query.Validate(expr, fields)
```

### Generate SQL via Visitor

Implement `ast.Visitor[T]` to transform the AST into any output format:

```go
import (
    "github.com/trazo-lat/query/ast"
    "github.com/trazo-lat/query/token"
)

type sqlVisitor struct {
    params []any
}

func (v *sqlVisitor) VisitQualifier(e *ast.QualifierExpr) string {
    field := e.Field.String()
    if e.IsWildcard() {
        v.params = append(v.params, ast.WildcardToLike(e.Value.Str))
        return fmt.Sprintf("%s LIKE $%d", field, len(v.params))
    }
    op := ast.SQLOperator(e.Operator, false)
    v.params = append(v.params, e.Value.Any())
    return fmt.Sprintf("%s %s $%d", field, op, len(v.params))
}

func (v *sqlVisitor) VisitBinary(e *ast.BinaryExpr) string {
    left := ast.Visit[string](v, e.Left)
    right := ast.Visit[string](v, e.Right)
    if e.Op == token.And {
        return left + " AND " + right
    }
    return left + " OR " + right
}
// ... implement remaining methods ...

// Usage:
v := &sqlVisitor{}
where := ast.Visit[string](v, expr)
// where = "state = $1 AND total > $2"
// v.params = ["draft", 50000]
```

### Generate JSON

```go
type jsonVisitor struct{}

func (v *jsonVisitor) VisitQualifier(e *ast.QualifierExpr) map[string]any {
    return map[string]any{
        "field": e.Field.String(),
        "op":    token.OperatorSymbol(e.Operator),
        "value": e.Value.Any(),
    }
}
// ... same pattern for other node types ...
```

### Build filter functions

```go
type filterVisitor struct{}

func (v *filterVisitor) VisitQualifier(e *ast.QualifierExpr) func(map[string]any) bool {
    field := e.Field.String()
    expected := e.Value.Any()
    return func(obj map[string]any) bool {
        return fmt.Sprint(obj[field]) == fmt.Sprint(expected)
    }
}
// ... build composable predicate functions ...
```

### Inspect the AST

```go
// List all fields referenced in a query
fields := ast.Fields(expr) // []ast.FieldPath

// Extract all field=value comparisons
quals := ast.Qualifiers(expr) // []*ast.QualifierExpr

// Check if it's a single condition (no AND/OR)
simple := ast.IsSimple(expr) // bool

// Measure nesting depth
depth := ast.Depth(expr) // int

// Walk the tree with custom logic
ast.Walk(expr, func(e ast.Expression) bool {
    // process each node
    return true // return false to skip children
})

// Round-trip back to query string
str := ast.String(expr) // "state=draft AND total>50000"
```

## Query Syntax

```
state=draft                                        # equality
state!=cancelled                                   # not equal
year>2020                                          # comparison
name=John*                                         # wildcard (prefix, suffix, contains)
tire_size                                          # presence check
state=draft AND customer_id=customer_john-doe      # logical AND
(state=draft OR state=issued) AND total>50000      # grouping with precedence
NOT state=cancelled                                # negation
created_at:2026-01-01..2026-03-31                  # date range
ttl.duration>1d                                    # duration comparison
labels.dev=jane                                    # nested field access
```

## Operators

| Operator | Allowed Types | Description |
|----------|--------------|-------------|
| `=` | all | Equality or wildcard match |
| `!=` | all | Not equal |
| `>` `>=` `<` `<=` | number, date, duration | Comparison |
| `..` | number, date, duration | Inclusive range (`field:start..end`) |
| `NOT` | expression | Boolean negation |
| `AND` | expressions | Logical AND (higher precedence than OR) |
| `OR` | expressions | Logical OR |

## License

Apache License 2.0
