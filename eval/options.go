package eval

import "github.com/trazo-lat/query/validate"

type options struct {
	maxLength     int
	maxDepth      int
	allowedOps    []validate.Op
	allowedFields []string
}

func defaultOpts() options {
	return options{
		maxLength: 256,
	}
}

// Option configures query compilation.
type Option func(*options)

// WithMaxLength sets the maximum query string length. 0 disables the limit.
func WithMaxLength(n int) Option {
	return func(o *options) { o.maxLength = n }
}

// WithMaxDepth limits the nesting depth of the expression tree.
// 0 means no limit.
func WithMaxDepth(n int) Option {
	return func(o *options) { o.maxDepth = n }
}

// WithAllowedOps restricts which operators are permitted in the query.
// If empty, all operators from the field config are allowed.
func WithAllowedOps(ops ...validate.Op) Option {
	return func(o *options) { o.allowedOps = ops }
}

// WithAllowedFields restricts which fields can appear in the query.
// If empty, all fields from the config are allowed.
func WithAllowedFields(fields ...string) Option {
	return func(o *options) { o.allowedFields = fields }
}
