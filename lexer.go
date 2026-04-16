package query

import (
	"strings"
	"time"
	"unicode"
)

// lexer tokenizes a query string into a sequence of tokens.
type lexer struct {
	input         string
	pos           int // current read position
	start         int // start of current token
	tokens        []Token
	errors        ErrorList
	afterOperator bool // true after emitting a comparison operator
}

// lex tokenizes the input query string and returns the token stream.
func lex(input string, maxLength int) ([]Token, error) {
	if maxLength > 0 && len(input) > maxLength {
		return nil, ErrorList{newError(ErrQueryTooLong, Position{},
			"query length %d exceeds maximum of %d characters", len(input), maxLength)}
	}

	l := &lexer{input: input}
	l.run()

	if err := l.errors.errOrNil(); err != nil {
		return nil, err
	}
	return l.tokens, nil
}

func (l *lexer) run() {
	for l.pos < len(l.input) {
		l.skipWhitespace()
		if l.pos >= len(l.input) {
			break
		}

		l.start = l.pos

		if l.afterOperator {
			l.lexValue()
			continue
		}

		ch := l.input[l.pos]
		switch {
		case ch == '(':
			l.emit(TokenLParen, "(")
			l.pos++
		case ch == ')':
			l.emit(TokenRParen, ")")
			l.pos++
		case ch == '@':
			l.emit(TokenAt, "@")
			l.pos++
		case ch == '!' && l.peek(1) == '=':
			l.emit(TokenNeq, "!=")
			l.pos += 2
			l.afterOperator = true
		case ch == '>' && l.peek(1) == '=':
			l.emit(TokenGte, ">=")
			l.pos += 2
			l.afterOperator = true
		case ch == '<' && l.peek(1) == '=':
			l.emit(TokenLte, "<=")
			l.pos += 2
			l.afterOperator = true
		case ch == '=':
			l.emit(TokenEq, "=")
			l.pos++
			l.afterOperator = true
		case ch == '>':
			l.emit(TokenGt, ">")
			l.pos++
			l.afterOperator = true
		case ch == '<':
			l.emit(TokenLt, "<")
			l.pos++
			l.afterOperator = true
		case ch == ':':
			l.emit(TokenColon, ":")
			l.pos++
			l.afterOperator = true
		case ch == '.' && l.peek(1) == '.':
			l.emit(TokenRange, "..")
			l.pos += 2
			l.afterOperator = true
		case ch == '.':
			l.emit(TokenDot, ".")
			l.pos++
		case isIdentStart(ch):
			l.lexIdentOrKeyword()
		default:
			l.errors.add(newError(ErrSyntax, Position{Offset: l.pos, Length: 1},
				"unexpected character %q", string(ch)))
			l.pos++
		}
	}

	l.tokens = append(l.tokens, Token{Type: TokenEOF, Pos: Position{Offset: l.pos}})
}

func (l *lexer) lexIdentOrKeyword() {
	start := l.pos
	for l.pos < len(l.input) && isIdentChar(l.input[l.pos]) {
		l.pos++
	}
	word := l.input[start:l.pos]
	pos := Position{Offset: start, Length: l.pos - start}

	switch word {
	case "AND":
		l.tokens = append(l.tokens, Token{Type: TokenAnd, Value: word, Pos: pos})
	case "OR":
		l.tokens = append(l.tokens, Token{Type: TokenOr, Value: word, Pos: pos})
	case "NOT":
		l.tokens = append(l.tokens, Token{Type: TokenNot, Value: word, Pos: pos})
	default:
		l.tokens = append(l.tokens, Token{Type: TokenIdent, Value: word, Pos: pos})
	}
}

// lexValue reads a value token after an operator. Values end at whitespace, ')' or EOF.
// Values can contain wildcards (*), escape sequences (\*, \\, \(, \)), dates, durations, etc.
func (l *lexer) lexValue() {
	l.afterOperator = false
	l.skipWhitespace()
	if l.pos >= len(l.input) {
		return
	}

	start := l.pos
	var buf strings.Builder
	hasWildcard := false

	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' || ch == ')' {
			break
		}
		// Stop at '..' (range separator) so parser can handle range expressions
		if ch == '.' && l.peek(1) == '.' {
			break
		}
		if ch == '\\' && l.pos+1 < len(l.input) {
			next := l.input[l.pos+1]
			switch next {
			case '*', '\\', '(', ')':
				buf.WriteByte(next)
				l.pos += 2
				continue
			}
		}
		if ch == '*' {
			hasWildcard = true
		}
		buf.WriteByte(ch)
		l.pos++
	}

	raw := l.input[start:l.pos]
	value := buf.String()
	pos := Position{Offset: start, Length: l.pos - start}

	if len(value) == 0 {
		return
	}

	tok := l.classifyValue(raw, value, hasWildcard, pos)
	l.tokens = append(l.tokens, tok)
}

// classifyValue determines the token type for a lexed value.
func (l *lexer) classifyValue(raw, value string, hasWildcard bool, pos Position) Token {
	// Wildcard values
	if hasWildcard {
		if !isValidWildcard(value) {
			l.errors.add(newError(ErrInvalidWildcard, pos,
				"invalid wildcard pattern %q: only prefix (foo*), suffix (*foo), and contains (*foo*) patterns are allowed", raw))
		}
		return Token{Type: TokenWildcard, Value: value, Pos: pos}
	}

	// Boolean
	if value == "true" || value == "false" {
		return Token{Type: TokenBoolean, Value: value, Pos: pos}
	}

	// Date: YYYY-MM-DD
	if isDateLiteral(value) {
		if _, err := time.Parse("2006-01-02", value); err != nil {
			l.errors.add(newError(ErrInvalidDate, pos, "invalid date %q", value))
		}
		return Token{Type: TokenDate, Value: value, Pos: pos}
	}

	// Duration: digits followed by d/h/m/w
	if isDurationLiteral(value) {
		return Token{Type: TokenDuration, Value: value, Pos: pos}
	}

	// Number: try integer then float
	if isIntegerLiteral(value) {
		return Token{Type: TokenInteger, Value: value, Pos: pos}
	}
	if isFloatLiteral(value) {
		return Token{Type: TokenFloat, Value: value, Pos: pos}
	}

	// Default: string
	return Token{Type: TokenString, Value: value, Pos: pos}
}

func (l *lexer) skipWhitespace() {
	for l.pos < len(l.input) {
		ch := l.input[l.pos]
		if ch != ' ' && ch != '\t' && ch != '\n' && ch != '\r' {
			break
		}
		l.pos++
	}
}

func (l *lexer) emit(typ TokenType, value string) {
	l.tokens = append(l.tokens, Token{
		Type:  typ,
		Value: value,
		Pos:   Position{Offset: l.start, Length: len(value)},
	})
}

func (l *lexer) peek(offset int) byte {
	idx := l.pos + offset
	if idx >= len(l.input) {
		return 0
	}
	return l.input[idx]
}

func isIdentStart(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isIdentChar(ch byte) bool {
	return isIdentStart(ch) || (ch >= '0' && ch <= '9') || ch == '-'
}

// isValidWildcard checks that a wildcard pattern is only prefix, suffix, or contains.
// Allowed: "foo*", "*foo", "*foo*". Not allowed: "a*b", "a*b*c".
func isValidWildcard(s string) bool {
	idx := strings.Index(s, "*")
	if idx == -1 {
		return true
	}

	// Count non-consecutive wildcard positions
	// Allowed forms: *, *text, text*, *text*
	stripped := strings.ReplaceAll(s, "*", "")
	stars := len(s) - len(stripped)

	if stars == 1 {
		// * at start, end, or the entire string
		return s[0] == '*' || s[len(s)-1] == '*'
	}
	if stars == 2 {
		// *text* pattern
		return s[0] == '*' && s[len(s)-1] == '*'
	}
	return false
}

// isDateLiteral checks if a string matches the YYYY-MM-DD pattern.
func isDateLiteral(s string) bool {
	if len(s) != 10 {
		return false
	}
	for i, ch := range s {
		switch i {
		case 4, 7:
			if ch != '-' {
				return false
			}
		default:
			if !unicode.IsDigit(ch) {
				return false
			}
		}
	}
	return true
}

// isDurationLiteral checks if a string is a duration like 1d, 4h, 30m, 2w.
func isDurationLiteral(s string) bool {
	if len(s) < 2 {
		return false
	}
	suffix := s[len(s)-1]
	if suffix != 'd' && suffix != 'h' && suffix != 'm' && suffix != 'w' {
		return false
	}
	for _, ch := range s[:len(s)-1] {
		if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

func isIntegerLiteral(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	for _, ch := range s[start:] {
		if !unicode.IsDigit(ch) {
			return false
		}
	}
	return true
}

func isFloatLiteral(s string) bool {
	if len(s) == 0 {
		return false
	}
	start := 0
	if s[0] == '-' || s[0] == '+' {
		start = 1
	}
	if start >= len(s) {
		return false
	}
	hasDot := false
	for _, ch := range s[start:] {
		if ch == '.' {
			if hasDot {
				return false
			}
			hasDot = true
			continue
		}
		if !unicode.IsDigit(ch) {
			return false
		}
	}
	return hasDot
}

// parseDuration parses a duration literal like "1d", "4h", "30m", "2w"
// into a time.Duration. Go's time.ParseDuration does not support 'd' or 'w'.
func parseDuration(s string) (time.Duration, error) {
	if len(s) < 2 {
		return 0, newError(ErrInvalidDuration, Position{}, "invalid duration %q", s)
	}

	numStr := s[:len(s)-1]
	n := 0
	for _, r := range numStr {
		if r < '0' || r > '9' {
			return 0, newError(ErrInvalidDuration, Position{}, "invalid duration %q", s)
		}
		n = n*10 + int(r-'0')
	}

	switch s[len(s)-1] {
	case 'm':
		return time.Duration(n) * time.Minute, nil
	case 'h':
		return time.Duration(n) * time.Hour, nil
	case 'd':
		return time.Duration(n) * 24 * time.Hour, nil
	case 'w':
		return time.Duration(n) * 7 * 24 * time.Hour, nil
	default:
		return 0, newError(ErrInvalidDuration, Position{}, "invalid duration suffix in %q", s)
	}
}
