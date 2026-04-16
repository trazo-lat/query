// Package query provides a query language lexer, parser, AST, and validator
// for the Trazo platform.
//
// It implements a unified query syntax used across all clients (CLI, Web UI,
// API, WASM) with zero external dependencies. All semantic evaluation
// (SQL generation, execution) lives in the consumer.
//
// # Sub-packages
//
//   - [token] — lexical token types and position tracking
//   - [ast] — AST node types, Visitor pattern, Walk, String
//   - [parser] — lexer and recursive descent parser
//   - [validate] — field configuration and AST validation
//
// # Basic usage
//
//	expr, err := query.Parse("state=draft AND total>50000")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fields := []validate.FieldConfig{
//	    {Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
//	    {Name: "total", Type: validate.TypeDecimal, AllowedOps: validate.NumericOps},
//	}
//	if err := query.Validate(expr, fields); err != nil {
//	    log.Fatal(err)
//	}
//
// # Code generation via Visitor
//
// Implement [ast.Visitor] to transform the AST into SQL, JSON, filter functions,
// React components, or any other target. See the package examples for SQL and
// JSON visitors.
package query
