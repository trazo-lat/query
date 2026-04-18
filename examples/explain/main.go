// Example: AST inspection and debugging.
//
// Demonstrates how to parse a query expression and inspect its AST
// structure using the library's public API — the same approach used
// by the `query explain` CLI command.
//
// Run:
//
//	go run ./examples/explain
package main

import (
	"fmt"
	"strings"

	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/parser"
	"github.com/trazo-lat/query/token"
)

func main() {
	queries := []string{
		"status=active AND priority>3",
		"(state=draft OR state=issued) AND NOT cancelled",
		"items@first",
		"price:10..50",
		"lower(name)=john*",
	}

	for _, q := range queries {
		fmt.Printf("Query: %s\n", q)
		fmt.Println(strings.Repeat("─", 60))

		// 1. Lex — show the token stream
		tokens, err := parser.Lex(q, 0)
		if err != nil {
			fmt.Printf("  Lex error: %v\n\n", err)
			continue
		}
		fmt.Print("  Tokens: ")
		for i, tok := range tokens {
			if i > 0 {
				fmt.Print(" ")
			}
			fmt.Print(tok)
		}
		fmt.Println()

		// 2. Parse — build the AST
		expr, err := parser.Parse(q, 0)
		if err != nil {
			fmt.Printf("  Parse error: %v\n\n", err)
			continue
		}

		// 3. AST utilities
		fmt.Printf("  Fields:  %v\n", fieldNames(ast.Fields(expr)))
		fmt.Printf("  Depth:   %d\n", ast.Depth(expr))
		fmt.Printf("  Simple:  %v\n", ast.IsSimple(expr))
		fmt.Printf("  String:  %s\n", ast.String(expr))

		// 4. Tree view
		fmt.Println("  Tree:")
		printTree(expr, "    ", true)

		fmt.Println()
	}

	// Demonstrate error formatting with source pointers
	fmt.Println("Error formatting")
	fmt.Println(strings.Repeat("─", 60))

	badQueries := []string{
		"AND foo",
		"status=",
		"$invalid",
	}

	for _, q := range badQueries {
		_, err := parser.Parse(q, 0)
		if err == nil {
			continue
		}
		for _, e := range parser.Errors(err) {
			fmt.Printf("  error: %s\n", e.Message)
			fmt.Printf("    %s\n", q)
			offset := e.Position.Offset
			if offset > len(q) {
				offset = len(q)
			}
			fmt.Printf("    %s^\n", strings.Repeat(" ", offset))
		}
	}
}

func printTree(expr ast.Expression, prefix string, isLast bool) {
	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	switch e := expr.(type) {
	case *ast.BinaryExpr:
		op := "AND"
		if e.Op == token.Or {
			op = "OR"
		}
		fmt.Printf("%s%s%sExpr\n", prefix, connector, op)
		printTree(e.Left, childPrefix, false)
		printTree(e.Right, childPrefix, true)

	case *ast.UnaryExpr:
		fmt.Printf("%s%sNOT\n", prefix, connector)
		printTree(e.Expr, childPrefix, true)

	case *ast.QualifierExpr:
		fmt.Printf("%s%s%s %s %s\n", prefix, connector,
			e.Field, token.OperatorSymbol(e.Operator), e.Value.Raw)
		if e.IsRange() {
			fmt.Printf("%s└── range end: %s\n", childPrefix, e.EndValue.Raw)
		}

	case *ast.PresenceExpr:
		fmt.Printf("%s%s%s (presence)\n", prefix, connector, e.Field)

	case *ast.SelectorExpr:
		if e.Selector != "" {
			fmt.Printf("%s%s@%s\n", prefix, connector, e.Selector)
		} else {
			fmt.Printf("%s%s@(...)\n", prefix, connector)
		}
		printTree(e.Base, childPrefix, e.Inner == nil)
		if e.Inner != nil {
			printTree(e.Inner, childPrefix, true)
		}

	case *ast.GroupExpr:
		fmt.Printf("%s%s(...)\n", prefix, connector)
		printTree(e.Expr, childPrefix, true)

	case *ast.FuncCallExpr:
		args := make([]string, len(e.Args))
		for i, a := range e.Args {
			args[i] = a.String()
		}
		fmt.Printf("%s%s%s(%s)\n", prefix, connector, e.Name, strings.Join(args, ", "))
	}
}

func fieldNames(fps []ast.FieldPath) []string {
	names := make([]string, len(fps))
	for i, fp := range fps {
		names[i] = fp.String()
	}
	return names
}
