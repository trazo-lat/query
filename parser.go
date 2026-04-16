package query

import (
	"strconv"
	"time"
)

// parser builds an AST from a token stream using recursive descent.
type parser struct {
	tokens []Token
	pos    int
	errors ErrorList
}

// parse converts a token stream into an AST.
func parse(tokens []Token) (Expression, error) {
	p := &parser{tokens: tokens}
	expr := p.parseExpression()
	if err := p.errors.errOrNil(); err != nil {
		return nil, err
	}
	if p.peek().Type != TokenEOF {
		tok := p.peek()
		p.errors.add(newError(ErrUnexpectedToken, tok.Pos,
			"unexpected token %s, expected end of query", tok))
	}
	if err := p.errors.errOrNil(); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *parser) parseExpression() Expression {
	return p.parseLogicalOr()
}

// parseLogicalOr handles: logical_and { "OR" logical_and }
func (p *parser) parseLogicalOr() Expression {
	left := p.parseLogicalAnd()
	for p.peek().Type == TokenOr {
		op := p.advance()
		right := p.parseLogicalAnd()
		if right == nil {
			break
		}
		left = &BinaryExpr{
			Op:       TokenOr,
			Left:     left,
			Right:    right,
			Position: op.Pos,
		}
	}
	return left
}

// parseLogicalAnd handles: term { "AND" term }
func (p *parser) parseLogicalAnd() Expression {
	left := p.parseTerm()
	for p.peek().Type == TokenAnd {
		op := p.advance()
		right := p.parseTerm()
		if right == nil {
			break
		}
		left = &BinaryExpr{
			Op:       TokenAnd,
			Left:     left,
			Right:    right,
			Position: op.Pos,
		}
	}
	return left
}

// parseTerm handles: [ "NOT" ] ( qualifier | "(" expression ")" )
func (p *parser) parseTerm() Expression {
	// Handle NOT prefix
	if p.peek().Type == TokenNot {
		op := p.advance()
		expr := p.parseTerm()
		if expr == nil {
			return nil
		}
		return &UnaryExpr{
			Op:       TokenNot,
			Expr:     expr,
			Position: op.Pos,
		}
	}

	// Handle parenthesized group
	if p.peek().Type == TokenLParen {
		open := p.advance()
		expr := p.parseExpression()
		if p.peek().Type != TokenRParen {
			p.errors.add(newError(ErrSyntax, p.peek().Pos, "expected ')', got %s", p.peek()))
			return expr
		}
		p.advance() // consume ')'
		return &GroupExpr{
			Expr:     expr,
			Position: open.Pos,
		}
	}

	return p.parseQualifier()
}

// parseQualifier handles: field_name [ operator value ] with optional range syntax.
func (p *parser) parseQualifier() Expression {
	if p.peek().Type != TokenIdent {
		tok := p.peek()
		if tok.Type == TokenEOF {
			p.errors.add(newError(ErrUnexpectedEOF, tok.Pos, "unexpected end of query, expected field name"))
		} else {
			p.errors.add(newError(ErrUnexpectedToken, tok.Pos,
				"expected field name, got %s", tok))
		}
		return nil
	}

	startPos := p.peek().Pos
	field := p.parseFieldName()

	// Check for operator
	tok := p.peek()

	// Range syntax: field:value..value
	if tok.Type == TokenColon {
		p.advance() // consume ':'
		return p.parseRangeExpr(field, startPos)
	}

	// Standard comparison operators
	if tok.Type.IsOperator() {
		p.advance() // consume operator
		val := p.parseValue()
		if val == nil {
			return nil
		}
		return &QualifierExpr{
			Field:    field,
			Operator: tok.Type,
			Value:    *val,
			Position: startPos,
		}
	}

	// No operator — presence check
	return &PresenceExpr{
		Field:    field,
		Position: startPos,
	}
}

// parseRangeExpr handles: value ".." value (after the colon was consumed).
func (p *parser) parseRangeExpr(field FieldPath, startPos Position) Expression {
	startVal := p.parseValue()
	if startVal == nil {
		return nil
	}

	if p.peek().Type != TokenRange {
		p.errors.add(newError(ErrSyntax, p.peek().Pos,
			"expected '..' in range expression, got %s", p.peek()))
		return nil
	}
	p.advance() // consume '..'

	endVal := p.parseValue()
	if endVal == nil {
		return nil
	}

	return &QualifierExpr{
		Field:    field,
		Operator: TokenRange,
		Value:    *startVal,
		EndValue: endVal,
		Position: startPos,
	}
}

// parseFieldName handles: identifier { "." identifier }
func (p *parser) parseFieldName() FieldPath {
	var parts []string
	parts = append(parts, p.advance().Value)

	for p.peek().Type == TokenDot {
		p.advance() // consume '.'
		if p.peek().Type != TokenIdent {
			p.errors.add(newError(ErrSyntax, p.peek().Pos,
				"expected field name after '.', got %s", p.peek()))
			break
		}
		parts = append(parts, p.advance().Value)
	}
	return FieldPath(parts)
}

// parseValue reads the next value token and converts it to a typed Value.
func (p *parser) parseValue() *Value {
	tok := p.peek()
	switch tok.Type {
	case TokenString:
		p.advance()
		return &Value{Type: ValueString, Raw: tok.Value, Str: tok.Value}
	case TokenInteger:
		p.advance()
		n, err := strconv.ParseInt(tok.Value, 10, 64)
		if err != nil {
			p.errors.add(newError(ErrInvalidValue, tok.Pos, "invalid integer %q", tok.Value))
			return nil
		}
		return &Value{Type: ValueInteger, Raw: tok.Value, Int: n}
	case TokenFloat:
		p.advance()
		f, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			p.errors.add(newError(ErrInvalidValue, tok.Pos, "invalid float %q", tok.Value))
			return nil
		}
		return &Value{Type: ValueFloat, Raw: tok.Value, Float: f}
	case TokenBoolean:
		p.advance()
		return &Value{Type: ValueBoolean, Raw: tok.Value, Bool: tok.Value == "true"}
	case TokenDate:
		p.advance()
		d, err := time.Parse("2006-01-02", tok.Value)
		if err != nil {
			p.errors.add(newError(ErrInvalidDate, tok.Pos, "invalid date %q", tok.Value))
			return nil
		}
		return &Value{Type: ValueDate, Raw: tok.Value, Date: d}
	case TokenDuration:
		p.advance()
		dur, err := parseDuration(tok.Value)
		if err != nil {
			p.errors.add(newError(ErrInvalidDuration, tok.Pos, "invalid duration %q", tok.Value))
			return nil
		}
		return &Value{Type: ValueDuration, Raw: tok.Value, Duration: dur}
	case TokenWildcard:
		p.advance()
		return &Value{Type: ValueString, Raw: tok.Value, Str: tok.Value, Wildcard: true}
	case TokenEOF:
		p.errors.add(newError(ErrUnexpectedEOF, tok.Pos, "expected value, got end of query"))
		return nil
	default:
		p.errors.add(newError(ErrUnexpectedToken, tok.Pos, "expected value, got %s", tok))
		p.advance()
		return nil
	}
}

func (p *parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: TokenEOF, Pos: Position{Offset: 0}}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}
