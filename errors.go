package query

import (
	"errors"
	"fmt"
	"strings"
)

// ErrorKind classifies the type of query error.
type ErrorKind int

// Error kind constants.
const (
	ErrSyntax             ErrorKind = iota // general syntax error
	ErrUnexpectedToken                     // unexpected token encountered
	ErrUnexpectedEOF                       // premature end of input
	ErrInvalidValue                        // malformed value literal
	ErrFieldNotFound                       // field not in config
	ErrOperatorNotAllowed                  // operator not permitted for field
	ErrTypeMismatch                        // value type incompatible with field type
	ErrQueryTooLong                        // query exceeds max length
	ErrInvalidWildcard                     // unsupported wildcard pattern
	ErrInvalidDate                         // malformed date literal
	ErrInvalidDuration                     // malformed duration literal
)

var kindNames = [...]string{
	ErrSyntax:             "syntax error",
	ErrUnexpectedToken:    "unexpected token",
	ErrUnexpectedEOF:      "unexpected end of input",
	ErrInvalidValue:       "invalid value",
	ErrFieldNotFound:      "unknown field",
	ErrOperatorNotAllowed: "operator not allowed",
	ErrTypeMismatch:       "type mismatch",
	ErrQueryTooLong:       "query too long",
	ErrInvalidWildcard:    "invalid wildcard",
	ErrInvalidDate:        "invalid date",
	ErrInvalidDuration:    "invalid duration",
}

// String returns the human-readable name of the error kind.
func (k ErrorKind) String() string {
	if int(k) < len(kindNames) {
		return kindNames[k]
	}
	return fmt.Sprintf("ErrorKind(%d)", k)
}

// QueryError is a structured error with position info from parsing or validation.
//
//nolint:revive // QueryError is the canonical public name matching the spec
type QueryError struct {
	Message  string
	Position Position
	Kind     ErrorKind
}

// Error implements the error interface.
func (e *QueryError) Error() string {
	return fmt.Sprintf("position %d: %s", e.Position.Offset, e.Message)
}

// ErrorList is a collection of query errors.
type ErrorList []*QueryError

// Error implements the error interface, joining all error messages.
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

// Unwrap returns the underlying errors for errors.Is/As compatibility.
func (el ErrorList) Unwrap() []error {
	errs := make([]error, len(el))
	for i, e := range el {
		errs[i] = e
	}
	return errs
}

func (el *ErrorList) add(err *QueryError) {
	*el = append(*el, err)
}

func (el ErrorList) errOrNil() error {
	if len(el) == 0 {
		return nil
	}
	return el
}

// IsQueryError reports whether err (or any error in its chain) is a *QueryError.
func IsQueryError(err error) bool {
	var qe *QueryError
	return errors.As(err, &qe)
}

// Errors extracts all *QueryError values from err.
// Returns nil if err is not a query error.
func Errors(err error) []*QueryError {
	var el ErrorList
	if errors.As(err, &el) {
		return []*QueryError(el)
	}
	var qe *QueryError
	if errors.As(err, &qe) {
		return []*QueryError{qe}
	}
	return nil
}

func newError(kind ErrorKind, pos Position, format string, args ...any) *QueryError {
	return &QueryError{
		Message:  fmt.Sprintf(format, args...),
		Position: pos,
		Kind:     kind,
	}
}
