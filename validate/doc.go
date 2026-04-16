// Package validate checks a parsed AST against field configurations.
//
// Consumers declare their fields using [FieldConfig] and pass them to
// [New] to create a validator. The validator checks that all fields in the
// query exist, that operators are allowed for each field's type, and that
// value types are compatible.
//
//	fields := []validate.FieldConfig{
//	    {Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
//	    {Name: "total", Type: validate.TypeDecimal, AllowedOps: validate.NumericOps},
//	}
//	v := validate.New(fields)
//	if err := v.Validate(expr); err != nil {
//	    // handle validation errors
//	}
package validate
