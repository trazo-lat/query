package eval

import (
	"fmt"
	"strings"
	"time"

	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
)

// matcher is a compiled function that evaluates a query against a data accessor.
type matcher func(get func(field string) (any, bool)) bool

// compileMatcher walks the AST and produces a closure tree for fast evaluation.
func compileMatcher(expr ast.Expression) matcher {
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		left := compileMatcher(e.Left)
		right := compileMatcher(e.Right)
		if e.Op == token.And {
			return func(get func(string) (any, bool)) bool {
				return left(get) && right(get)
			}
		}
		return func(get func(string) (any, bool)) bool {
			return left(get) || right(get)
		}

	case *ast.UnaryExpr:
		inner := compileMatcher(e.Expr)
		return func(get func(string) (any, bool)) bool {
			return !inner(get)
		}

	case *ast.GroupExpr:
		return compileMatcher(e.Expr)

	case *ast.QualifierExpr:
		return compileQualifier(e)

	case *ast.PresenceExpr:
		field := e.Field.String()
		return func(get func(string) (any, bool)) bool {
			_, ok := get(field)
			return ok
		}

	case *ast.SelectorExpr:
		return compileMatcher(e.Base)

	default:
		return func(func(string) (any, bool)) bool { return false }
	}
}

func compileQualifier(e *ast.QualifierExpr) matcher {
	field := e.Field.String()

	// Range: field BETWEEN start AND end
	if e.IsRange() {
		return compileRange(field, &e.Value, e.EndValue)
	}

	// Wildcard: pattern matching
	if e.IsWildcard() {
		return compileWildcard(field, e.Value.Str)
	}

	// Standard comparison
	return compileComparison(field, e.Operator, &e.Value)
}

func compileRange(field string, start, end *ast.Value) matcher {
	return func(get func(string) (any, bool)) bool {
		raw, ok := get(field)
		if !ok {
			return false
		}
		return compareValues(raw, start, token.Gte) && compareValues(raw, end, token.Lte)
	}
}

func compileWildcard(field, pattern string) matcher {
	prefix := strings.HasPrefix(pattern, "*")
	suffix := strings.HasSuffix(pattern, "*")
	inner := strings.Trim(pattern, "*")

	return func(get func(string) (any, bool)) bool {
		raw, ok := get(field)
		if !ok {
			return false
		}
		s := strings.ToLower(fmt.Sprint(raw))
		lowerInner := strings.ToLower(inner)

		switch {
		case prefix && suffix:
			return strings.Contains(s, lowerInner)
		case prefix:
			return strings.HasSuffix(s, lowerInner)
		case suffix:
			return strings.HasPrefix(s, lowerInner)
		default:
			return s == lowerInner
		}
	}
}

func compileComparison(field string, op token.Type, expected *ast.Value) matcher {
	return func(get func(string) (any, bool)) bool {
		raw, ok := get(field)
		if !ok {
			return false
		}

		switch op { //nolint:exhaustive // only comparison operators
		case token.Eq:
			return equalValues(raw, expected)
		case token.Neq:
			return !equalValues(raw, expected)
		default:
			return compareValues(raw, expected, op)
		}
	}
}

func equalValues(actual any, expected *ast.Value) bool {
	switch expected.Type {
	case ast.ValueString:
		return strings.EqualFold(fmt.Sprint(actual), expected.Str)
	case ast.ValueInteger:
		return toInt64(actual) == expected.Int
	case ast.ValueFloat:
		return toFloat64(actual) == expected.Float
	case ast.ValueBoolean:
		return toBool(actual) == expected.Bool
	case ast.ValueDate:
		return toTime(actual).Equal(expected.Date)
	case ast.ValueDuration:
		return toDuration(actual) == expected.Duration
	default:
		return fmt.Sprint(actual) == expected.Raw
	}
}

func compareValues(actual any, expected *ast.Value, op token.Type) bool {
	switch expected.Type {
	case ast.ValueInteger:
		a := toInt64(actual)
		b := expected.Int
		return compareOrdered(a, b, op)
	case ast.ValueFloat:
		a := toFloat64(actual)
		b := expected.Float
		return compareOrdered(a, b, op)
	case ast.ValueDate:
		a := toTime(actual)
		b := expected.Date
		switch op { //nolint:exhaustive // only relational
		case token.Gt:
			return a.After(b)
		case token.Gte:
			return !a.Before(b)
		case token.Lt:
			return a.Before(b)
		case token.Lte:
			return !a.After(b)
		default:
			return a.Equal(b)
		}
	case ast.ValueDuration:
		a := int64(toDuration(actual))
		b := int64(expected.Duration)
		return compareOrdered(a, b, op)
	default:
		a := fmt.Sprint(actual)
		b := expected.Raw
		return compareOrdered(a, b, op)
	}
}

type ordered interface {
	~int64 | ~float64 | ~string
}

func compareOrdered[T ordered](a, b T, op token.Type) bool {
	switch op { //nolint:exhaustive // only relational operators
	case token.Gt:
		return a > b
	case token.Gte:
		return a >= b
	case token.Lt:
		return a < b
	case token.Lte:
		return a <= b
	case token.Eq:
		return a == b
	case token.Neq:
		return a != b
	default:
		return false
	}
}

// Type coercion helpers

func toInt64(v any) int64 {
	switch n := v.(type) {
	case int:
		return int64(n)
	case int32:
		return int64(n)
	case int64:
		return n
	case float32:
		return int64(n)
	case float64:
		return int64(n)
	default:
		return 0
	}
}

func toFloat64(v any) float64 {
	switch n := v.(type) {
	case int:
		return float64(n)
	case int32:
		return float64(n)
	case int64:
		return float64(n)
	case float32:
		return float64(n)
	case float64:
		return n
	default:
		return 0
	}
}

func toBool(v any) bool {
	switch b := v.(type) {
	case bool:
		return b
	case string:
		return strings.EqualFold(b, "true")
	default:
		return false
	}
}

func toTime(v any) time.Time {
	switch t := v.(type) {
	case time.Time:
		return t
	case string:
		if parsed, err := time.Parse("2006-01-02", t); err == nil {
			return parsed
		}
		if parsed, err := time.Parse(time.RFC3339, t); err == nil {
			return parsed
		}
	}
	return time.Time{}
}

func toDuration(v any) time.Duration {
	switch d := v.(type) {
	case time.Duration:
		return d
	case string:
		if parsed, err := time.ParseDuration(d); err == nil {
			return parsed
		}
	}
	return 0
}
