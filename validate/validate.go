package validate

import (
	"fmt"
	"strings"

	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
)

// ErrorKind classifies validation errors.
type ErrorKind int

// Validation error kind constants.
const (
	ErrFieldNotFound      ErrorKind = iota // field not in config
	ErrOperatorNotAllowed                  // operator not permitted for field
	ErrTypeMismatch                        // value type incompatible with field type
)

// Error is a structured validation error.
//
//nolint:revive // Error is the canonical name for this package
type Error struct {
	Message  string
	Position token.Position
	Kind     ErrorKind
}

// Error implements the error interface.
func (e *Error) Error() string {
	return fmt.Sprintf("position %d: %s", e.Position.Offset, e.Message)
}

// ErrorList is a collection of validation errors.
type ErrorList []*Error

// Error implements the error interface.
func (el ErrorList) Error() string {
	switch len(el) {
	case 0:
		return "no errors"
	case 1:
		return el[0].Error()
	default:
		msgs := make([]string, len(el))
		for i, e := range el {
			msgs[i] = e.Error()
		}
		return strings.Join(msgs, "; ")
	}
}

// Unwrap returns the underlying errors.
func (el ErrorList) Unwrap() []error {
	errs := make([]error, len(el))
	for i, e := range el {
		errs[i] = e
	}
	return errs
}

// Validator validates a parsed AST against field configurations.
type Validator struct {
	fields map[string]FieldConfig
	nested map[string]FieldConfig
	errors ErrorList
}

// New creates a validator with the given field configs.
func New(fields []FieldConfig) *Validator {
	v := &Validator{
		fields: make(map[string]FieldConfig, len(fields)),
		nested: make(map[string]FieldConfig),
	}
	for _, f := range fields {
		v.fields[f.Name] = f
		if f.Nested {
			v.nested[f.Name] = f
		}
	}
	return v
}

// Validate checks the AST against the field configurations.
// It collects all errors rather than stopping at the first one.
func (v *Validator) Validate(expr ast.Expression) error {
	v.errors = nil
	v.validate(expr)
	if len(v.errors) == 0 {
		return nil
	}
	return v.errors
}

func (v *Validator) validate(expr ast.Expression) {
	if expr == nil {
		return
	}
	switch e := expr.(type) {
	case *ast.BinaryExpr:
		v.validate(e.Left)
		v.validate(e.Right)
	case *ast.UnaryExpr:
		v.validate(e.Expr)
	case *ast.GroupExpr:
		v.validate(e.Expr)
	case *ast.SelectorExpr:
		v.validateSelector(e)
	case *ast.FuncCallExpr:
		v.validateFuncCallFields(e)
	case *ast.QualifierExpr:
		v.validateQualifier(e)
	case *ast.PresenceExpr:
		v.validatePresence(e)
	}
}

func (v *Validator) validateQualifier(q *ast.QualifierExpr) {
	// If the qualifier has a field transform function (e.g., lower(name)=john),
	// validate the field references inside the function args instead.
	if q.HasFieldFunc() {
		v.validateFuncCallFields(q.FieldFunc)
		return
	}

	fieldName := q.Field.String()
	cfg, ok := v.resolveField(fieldName)
	if !ok {
		v.addError(ErrFieldNotFound, q.Position, "unknown field %q", fieldName)
		return
	}
	op := tokenTypeToOp(q.Operator, q.Value.Wildcard)
	if !cfg.AllowsOp(op) {
		v.addError(ErrOperatorNotAllowed, q.Position,
			"operator %q is not allowed for field %q (type %s)", string(op), fieldName, cfg.Type)
		return
	}
	if !typeCompatible(cfg.Type, q.Value) {
		v.addError(ErrTypeMismatch, q.Position,
			"value type %s is not compatible with field %q (type %s)", q.Value.Type, fieldName, cfg.Type)
	}
	if q.EndValue != nil {
		if !typeCompatible(cfg.Type, *q.EndValue) {
			v.addError(ErrTypeMismatch, q.Position,
				"range end value type %s is not compatible with field %q (type %s)", q.EndValue.Type, fieldName, cfg.Type)
		}
	}
}

// validateFuncCallFields validates field references inside function arguments.
func (v *Validator) validateFuncCallFields(fc *ast.FuncCallExpr) {
	for _, arg := range fc.Args {
		if arg.Field != nil {
			fieldName := arg.Field.String()
			if _, ok := v.resolveField(fieldName); !ok {
				v.addError(ErrFieldNotFound, fc.Position, "unknown field %q in function %s()", fieldName, fc.Name)
			}
		}
		if arg.Call != nil {
			v.validateFuncCallFields(arg.Call)
		}
	}
}

// validateSelector checks that the selector's base field is declared and
// recurses into the inner expression. The base acts as a list reference, so
// we only require the field to exist — OpPresence is not required.
func (v *Validator) validateSelector(s *ast.SelectorExpr) {
	switch b := s.Base.(type) {
	case *ast.PresenceExpr:
		fieldName := b.Field.String()
		if _, ok := v.resolveField(fieldName); !ok {
			v.addError(ErrFieldNotFound, b.Position, "unknown field %q", fieldName)
		}
	case *ast.QualifierExpr:
		v.validateQualifier(b)
	default:
		v.validate(s.Base)
	}
	if s.Inner != nil {
		v.validate(s.Inner)
	}
}

func (v *Validator) validatePresence(p *ast.PresenceExpr) {
	fieldName := p.Field.String()
	cfg, ok := v.resolveField(fieldName)
	if !ok {
		v.addError(ErrFieldNotFound, p.Position, "unknown field %q", fieldName)
		return
	}
	if !cfg.AllowsOp(OpPresence) {
		v.addError(ErrOperatorNotAllowed, p.Position,
			"presence check is not allowed for field %q", fieldName)
	}
}

func (v *Validator) resolveField(name string) (FieldConfig, bool) {
	if cfg, ok := v.fields[name]; ok {
		return cfg, true
	}
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		if cfg, ok := v.nested[parts[0]]; ok {
			return cfg, true
		}
	}
	return FieldConfig{}, false
}

func (v *Validator) addError(kind ErrorKind, pos token.Position, format string, args ...any) {
	v.errors = append(v.errors, &Error{
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
		Kind:     kind,
	})
}

func tokenTypeToOp(tt token.Type, wildcard bool) Op {
	if wildcard {
		return OpWildcard
	}
	switch tt { //nolint:exhaustive // only comparison operators
	case token.Eq:
		return OpEq
	case token.Neq:
		return OpNeq
	case token.Gt:
		return OpGt
	case token.Gte:
		return OpGte
	case token.Lt:
		return OpLt
	case token.Lte:
		return OpLte
	case token.Range:
		return OpRange
	default:
		return OpEq
	}
}

func typeCompatible(fieldType FieldValueType, val ast.Value) bool {
	switch fieldType {
	case TypeText:
		return val.Type == ast.ValueString || val.Wildcard
	case TypeInteger:
		return val.Type == ast.ValueInteger
	case TypeDecimal:
		return val.Type == ast.ValueInteger || val.Type == ast.ValueFloat
	case TypeBoolean:
		return val.Type == ast.ValueBoolean
	case TypeDate, TypeDatetime:
		return val.Type == ast.ValueDate
	case TypeDuration:
		return val.Type == ast.ValueDuration
	default:
		return false
	}
}
