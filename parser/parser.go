package parser

import (
	"strconv"
	"time"

	"github.com/trazo-lat/query/ast"
	"github.com/trazo-lat/query/token"
)

// parser builds an AST from a token stream using recursive descent.
type parser struct {
	tokens []token.Token
	pos    int
	errors ErrorList
}

// Parse lexes and parses a query string into an AST.
func Parse(input string, maxLength int) (ast.Expression, error) {
	tokens, err := Lex(input, maxLength)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	expr := p.parseExpression()
	if err := p.errors.errOrNil(); err != nil {
		return nil, err
	}
	if p.peek().Type != token.EOF {
		tok := p.peek()
		p.errors.add(newError(ErrUnexpectedToken, tok.Pos,
			"unexpected token %s, expected end of query", tok))
	}
	if err := p.errors.errOrNil(); err != nil {
		return nil, err
	}
	return expr, nil
}

func (p *parser) parseExpression() ast.Expression {
	return p.parseLogicalOr()
}

func (p *parser) parseLogicalOr() ast.Expression {
	left := p.parseLogicalAnd()
	for p.peek().Type == token.Or {
		op := p.advance()
		right := p.parseLogicalAnd()
		if right == nil {
			break
		}
		left = &ast.BinaryExpr{
			Op:       token.Or,
			Left:     left,
			Right:    right,
			Position: op.Pos,
		}
	}
	return left
}

func (p *parser) parseLogicalAnd() ast.Expression {
	left := p.parseTerm()
	for p.peek().Type == token.And {
		op := p.advance()
		right := p.parseTerm()
		if right == nil {
			break
		}
		left = &ast.BinaryExpr{
			Op:       token.And,
			Left:     left,
			Right:    right,
			Position: op.Pos,
		}
	}
	return left
}

func (p *parser) parseTerm() ast.Expression {
	if p.peek().Type == token.Not {
		op := p.advance()
		expr := p.parseTerm()
		if expr == nil {
			return nil
		}
		return &ast.UnaryExpr{
			Op:       token.Not,
			Expr:     expr,
			Position: op.Pos,
		}
	}
	if p.peek().Type == token.LParen {
		open := p.advance()
		expr := p.parseExpression()
		if p.peek().Type != token.RParen {
			p.errors.add(newError(ErrSyntax, p.peek().Pos, "expected ')', got %s", p.peek()))
			return expr
		}
		p.advance()
		return &ast.GroupExpr{
			Expr:     expr,
			Position: open.Pos,
		}
	}
	return p.parseQualifier()
}

func (p *parser) parseQualifier() ast.Expression {
	if p.peek().Type != token.Ident {
		tok := p.peek()
		if tok.Type == token.EOF {
			p.errors.add(newError(ErrUnexpectedEOF, tok.Pos, "unexpected end of query, expected field name"))
		} else {
			p.errors.add(newError(ErrUnexpectedToken, tok.Pos, "expected field name, got %s", tok))
		}
		return nil
	}

	startPos := p.peek().Pos
	field := p.parseFieldName()
	tok := p.peek()

	if tok.Type == token.Colon {
		p.advance()
		return p.parseRangeExpr(field, startPos)
	}
	if tok.Type.IsOperator() {
		p.advance()
		val := p.parseValue()
		if val == nil {
			return nil
		}
		return &ast.QualifierExpr{
			Field:    field,
			Operator: tok.Type,
			Value:    *val,
			Position: startPos,
		}
	}
	return &ast.PresenceExpr{
		Field:    field,
		Position: startPos,
	}
}

func (p *parser) parseRangeExpr(field ast.FieldPath, startPos token.Position) ast.Expression {
	startVal := p.parseValue()
	if startVal == nil {
		return nil
	}
	if p.peek().Type != token.Range {
		p.errors.add(newError(ErrSyntax, p.peek().Pos,
			"expected '..' in range expression, got %s", p.peek()))
		return nil
	}
	p.advance()
	endVal := p.parseValue()
	if endVal == nil {
		return nil
	}
	return &ast.QualifierExpr{
		Field:    field,
		Operator: token.Range,
		Value:    *startVal,
		EndValue: endVal,
		Position: startPos,
	}
}

func (p *parser) parseFieldName() ast.FieldPath {
	var parts []string
	parts = append(parts, p.advance().Value)
	for p.peek().Type == token.Dot {
		p.advance()
		if p.peek().Type != token.Ident {
			p.errors.add(newError(ErrSyntax, p.peek().Pos,
				"expected field name after '.', got %s", p.peek()))
			break
		}
		parts = append(parts, p.advance().Value)
	}
	return ast.FieldPath(parts)
}

func (p *parser) parseValue() *ast.Value {
	tok := p.peek()
	switch tok.Type {
	case token.String:
		p.advance()
		return &ast.Value{Type: ast.ValueString, Raw: tok.Value, Str: tok.Value}
	case token.Integer:
		p.advance()
		n, err := strconv.ParseInt(tok.Value, 10, 64)
		if err != nil {
			p.errors.add(newError(ErrInvalidValue, tok.Pos, "invalid integer %q", tok.Value))
			return nil
		}
		return &ast.Value{Type: ast.ValueInteger, Raw: tok.Value, Int: n}
	case token.Float:
		p.advance()
		f, err := strconv.ParseFloat(tok.Value, 64)
		if err != nil {
			p.errors.add(newError(ErrInvalidValue, tok.Pos, "invalid float %q", tok.Value))
			return nil
		}
		return &ast.Value{Type: ast.ValueFloat, Raw: tok.Value, Float: f}
	case token.Boolean:
		p.advance()
		return &ast.Value{Type: ast.ValueBoolean, Raw: tok.Value, Bool: tok.Value == "true"}
	case token.Date:
		p.advance()
		d, err := time.Parse("2006-01-02", tok.Value)
		if err != nil {
			p.errors.add(newError(ErrInvalidDate, tok.Pos, "invalid date %q", tok.Value))
			return nil
		}
		return &ast.Value{Type: ast.ValueDate, Raw: tok.Value, Date: d}
	case token.Duration:
		p.advance()
		dur, err := ParseDuration(tok.Value)
		if err != nil {
			p.errors.add(newError(ErrInvalidDuration, tok.Pos, "invalid duration %q", tok.Value))
			return nil
		}
		return &ast.Value{Type: ast.ValueDuration, Raw: tok.Value, Duration: dur}
	case token.Wildcard:
		p.advance()
		return &ast.Value{Type: ast.ValueString, Raw: tok.Value, Str: tok.Value, Wildcard: true}
	case token.EOF:
		p.errors.add(newError(ErrUnexpectedEOF, tok.Pos, "expected value, got end of query"))
		return nil
	default:
		p.errors.add(newError(ErrUnexpectedToken, tok.Pos, "expected value, got %s", tok))
		p.advance()
		return nil
	}
}

func (p *parser) peek() token.Token {
	if p.pos >= len(p.tokens) {
		return token.Token{Type: token.EOF}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() token.Token {
	tok := p.peek()
	if p.pos < len(p.tokens) {
		p.pos++
	}
	return tok
}
