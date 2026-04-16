package query

import "strings"

// Validator validates a parsed AST against a set of field configurations.
type Validator struct {
	fields map[string]FieldConfig
	nested map[string]FieldConfig // nested field prefixes (e.g., "labels")
	errors ErrorList
}

// NewValidator creates a validator with the given field configs.
func NewValidator(fields []FieldConfig) *Validator {
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
func (v *Validator) Validate(expr Expression) error {
	v.errors = nil
	v.validate(expr)
	return v.errors.errOrNil()
}

func (v *Validator) validate(expr Expression) {
	if expr == nil {
		return
	}

	switch e := expr.(type) {
	case *BinaryExpr:
		v.validate(e.Left)
		v.validate(e.Right)
	case *UnaryExpr:
		v.validate(e.Expr)
	case *GroupExpr:
		v.validate(e.Expr)
	case *SelectorExpr:
		v.validate(e.Base)
		if e.Inner != nil {
			v.validate(e.Inner)
		}
	case *QualifierExpr:
		v.validateQualifier(e)
	case *PresenceExpr:
		v.validatePresence(e)
	}
}

func (v *Validator) validateQualifier(q *QualifierExpr) {
	fieldName := q.Field.String()
	cfg, ok := v.resolveField(fieldName)
	if !ok {
		v.errors.add(newError(ErrFieldNotFound, q.Position,
			"unknown field %q", fieldName))
		return
	}

	op := tokenTypeToOp(q.Operator, q.Value.Wildcard)
	if !cfg.AllowsOp(op) {
		v.errors.add(newError(ErrOperatorNotAllowed, q.Position,
			"operator %q is not allowed for field %q (type %s)", string(op), fieldName, cfg.Type))
		return
	}

	if !typeCompatible(cfg.Type, q.Value) {
		v.errors.add(newError(ErrTypeMismatch, q.Position,
			"value type %s is not compatible with field %q (type %s)", q.Value.Type, fieldName, cfg.Type))
	}

	// Validate end value for range expressions
	if q.EndValue != nil {
		if !typeCompatible(cfg.Type, *q.EndValue) {
			v.errors.add(newError(ErrTypeMismatch, q.Position,
				"range end value type %s is not compatible with field %q (type %s)", q.EndValue.Type, fieldName, cfg.Type))
		}
	}
}

func (v *Validator) validatePresence(p *PresenceExpr) {
	fieldName := p.Field.String()
	cfg, ok := v.resolveField(fieldName)
	if !ok {
		v.errors.add(newError(ErrFieldNotFound, p.Position,
			"unknown field %q", fieldName))
		return
	}

	if !cfg.AllowsOp(OpPresence) {
		v.errors.add(newError(ErrOperatorNotAllowed, p.Position,
			"presence check is not allowed for field %q", fieldName))
	}
}

// resolveField looks up a field by exact name, then by nested prefix.
func (v *Validator) resolveField(name string) (FieldConfig, bool) {
	if cfg, ok := v.fields[name]; ok {
		return cfg, true
	}

	// Check for nested field: "labels.dev" matches "labels" with Nested=true
	parts := strings.SplitN(name, ".", 2)
	if len(parts) == 2 {
		if cfg, ok := v.nested[parts[0]]; ok {
			return cfg, true
		}
	}

	return FieldConfig{}, false
}

// tokenTypeToOp maps a token type to the corresponding Op.
func tokenTypeToOp(tt TokenType, wildcard bool) Op {
	if wildcard {
		return OpWildcard
	}
	switch tt {
	case TokenEq:
		return OpEq
	case TokenNeq:
		return OpNeq
	case TokenGt:
		return OpGt
	case TokenGte:
		return OpGte
	case TokenLt:
		return OpLt
	case TokenLte:
		return OpLte
	case TokenRange:
		return OpRange
	default:
		return OpEq
	}
}

// typeCompatible checks if a value is compatible with a field type.
func typeCompatible(fieldType FieldValueType, val Value) bool {
	switch fieldType {
	case TypeText:
		return val.Type == ValueString || val.Wildcard
	case TypeInteger:
		return val.Type == ValueInteger
	case TypeDecimal:
		return val.Type == ValueInteger || val.Type == ValueFloat
	case TypeBoolean:
		return val.Type == ValueBoolean
	case TypeDate, TypeDatetime:
		return val.Type == ValueDate
	case TypeDuration:
		return val.Type == ValueDuration
	default:
		return false
	}
}
