// Package parser implements the lexer and recursive descent parser for the
// query language. It converts a query string into an [ast.Expression] tree.
//
// Most consumers should use the top-level [query.Parse] function instead of
// calling this package directly.
package parser
