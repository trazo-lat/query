// Package eval compiles a query string into an executable program that can
// be evaluated against data. It combines parsing, validation, and evaluation
// into a single pipeline.
//
// # Compile and Match
//
//	fields := []validate.FieldConfig{
//	    {Name: "state", Type: validate.TypeText, AllowedOps: validate.TextOps},
//	    {Name: "total", Type: validate.TypeDecimal, AllowedOps: validate.NumericOps},
//	}
//
//	prog, err := eval.Compile("state=draft AND total>50000", fields)
//	if err != nil { ... }
//
//	prog.Match(map[string]any{"state": "draft", "total": 60000}) // true
//	prog.Match(map[string]any{"state": "draft", "total": 100})   // false
//
// # Restrict allowed operations
//
//	prog, err := eval.Compile(q, fields,
//	    eval.WithAllowedOps(validate.OpEq, validate.OpNeq),  // no >, <, etc.
//	    eval.WithAllowedFields("state", "name"),              // only these fields
//	    eval.WithMaxDepth(3),                                 // limit nesting
//	)
//
// # Custom data accessor
//
//	prog.MatchFunc(func(field string) (any, bool) {
//	    return myRecord.Get(field)
//	})
package eval
