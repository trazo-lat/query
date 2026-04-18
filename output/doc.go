// Package output provides formatters for rendering AST expressions
// into human-readable or machine-readable representations.
//
// Built-in formatters (JSON, Tree) use [ast.Visitor] internally to walk
// the AST. Custom formatters can be plugged in via the [Formatter] interface.
//
// Convenience functions for built-in formats:
//
//	data, err := output.AsJSON(expr)
//	data, err := output.AsTree(expr, output.WithPositions())
//
// Custom formatters:
//
//	output.Format(os.Stdout, expr, myYAMLFormatter)
package output
