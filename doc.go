// Package query provides a query language lexer, parser, AST, and validator
// for the Trazo platform.
//
// It implements a unified query syntax used across all clients (CLI, Web UI,
// API, WASM) with zero external dependencies. All semantic evaluation
// (SQL generation, execution) lives in the consumer.
//
// Basic usage:
//
//	expr, err := query.Parse("state=draft AND total>50000")
//	if err != nil {
//	    log.Fatal(err)
//	}
//
//	fields := []query.FieldConfig{
//	    {Name: "state", Type: query.TypeText, AllowedOps: query.TextOps},
//	    {Name: "total", Type: query.TypeDecimal, AllowedOps: query.NumericOps},
//	}
//	if err := query.Validate(expr, fields); err != nil {
//	    log.Fatal(err)
//	}
package query
