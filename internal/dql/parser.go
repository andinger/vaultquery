package dql

import (
	"fmt"
	"strconv"
)

// ParseError represents a parsing error with position information.
type ParseError struct {
	Pos     int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Pos, e.Message)
}

type parser struct {
	tokens []Token
	pos    int
}

// Parse lexes and parses a DQL query string.
func Parse(input string) (*Query, error) {
	tokens, err := Lex(input)
	if err != nil {
		return nil, err
	}
	p := &parser{tokens: tokens}
	return p.parseQuery()
}

func (p *parser) peek() Token {
	if p.pos >= len(p.tokens) {
		return Token{Type: EOF, Pos: -1}
	}
	return p.tokens[p.pos]
}

func (p *parser) advance() Token {
	tok := p.peek()
	p.pos++
	return tok
}

func (p *parser) expect(tokType string) (Token, error) {
	tok := p.advance()
	if tok.Type != tokType {
		return tok, &ParseError{
			Pos:     tok.Pos,
			Message: fmt.Sprintf("expected %s, got %s (%q)", tokType, tok.Type, tok.Literal),
		}
	}
	return tok, nil
}

func (p *parser) parseQuery() (*Query, error) {
	q := &Query{}

	tok := p.peek()
	switch tok.Type {
	case TABLE:
		p.advance()
		q.Mode = "TABLE"
		fields, err := p.parseFieldList()
		if err != nil {
			return nil, err
		}
		q.Fields = fields
	case LIST:
		p.advance()
		q.Mode = "LIST"
	default:
		return nil, &ParseError{
			Pos:     tok.Pos,
			Message: fmt.Sprintf("expected TABLE or LIST, got %s (%q)", tok.Type, tok.Literal),
		}
	}

	// Optional clauses
	for {
		tok = p.peek()
		switch tok.Type {
		case FROM:
			p.advance()
			str, err := p.expect(STRING)
			if err != nil {
				return nil, err
			}
			q.From = str.Literal
		case WHERE:
			p.advance()
			expr, err := p.parseExpr()
			if err != nil {
				return nil, err
			}
			q.Where = expr
		case SORT:
			p.advance()
			sorts, err := p.parseSortFields()
			if err != nil {
				return nil, err
			}
			q.Sort = sorts
		case LIMIT:
			p.advance()
			num, err := p.expect(NUMBER)
			if err != nil {
				return nil, err
			}
			n, convErr := strconv.Atoi(num.Literal)
			if convErr != nil {
				return nil, &ParseError{Pos: num.Pos, Message: "invalid limit value: " + num.Literal}
			}
			q.Limit = n
		case GROUP:
			p.advance()
			if _, err := p.expect(BY); err != nil {
				return nil, err
			}
			fields, err := p.parseFieldList()
			if err != nil {
				return nil, err
			}
			q.GroupBy = fields
		case FLATTEN:
			p.advance()
			fields, err := p.parseFieldList()
			if err != nil {
				return nil, err
			}
			q.Flatten = fields
		case EOF:
			return q, nil
		default:
			return nil, &ParseError{
				Pos:     tok.Pos,
				Message: fmt.Sprintf("unexpected token %s (%q)", tok.Type, tok.Literal),
			}
		}
	}
}

func (p *parser) parseFieldList() ([]string, error) {
	var fields []string
	tok := p.peek()
	if tok.Type != IDENT {
		return nil, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("expected field name, got %s (%q)", tok.Type, tok.Literal)}
	}
	fields = append(fields, p.advance().Literal)
	for p.peek().Type == COMMA {
		p.advance()
		tok = p.peek()
		if tok.Type != IDENT {
			return nil, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("expected field name after comma, got %s (%q)", tok.Type, tok.Literal)}
		}
		fields = append(fields, p.advance().Literal)
	}
	return fields, nil
}

func (p *parser) parseExpr() (Expr, error) {
	return p.parseOrExpr()
}

func (p *parser) parseOrExpr() (Expr, error) {
	left, err := p.parseAndExpr()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == OR {
		p.advance()
		right, err := p.parseAndExpr()
		if err != nil {
			return nil, err
		}
		left = LogicalExpr{Op: "OR", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseAndExpr() (Expr, error) {
	left, err := p.parsePrimary()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == AND {
		p.advance()
		right, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		left = LogicalExpr{Op: "AND", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parsePrimary() (Expr, error) {
	tok := p.peek()

	if tok.Type == LPAREN {
		p.advance()
		expr, err := p.parseExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		return ParenExpr{Inner: expr}, nil
	}

	if tok.Type != IDENT {
		return nil, &ParseError{
			Pos:     tok.Pos,
			Message: fmt.Sprintf("expected field name or '(', got %s (%q)", tok.Type, tok.Literal),
		}
	}

	field := p.advance()
	next := p.peek()

	// exists / !exists
	if next.Type == EXISTS {
		p.advance()
		return ExistsExpr{Field: field.Literal, Negated: false}, nil
	}
	if next.Type == NOT_EXISTS {
		p.advance()
		return ExistsExpr{Field: field.Literal, Negated: true}, nil
	}

	// comparison operators
	var op string
	switch next.Type {
	case EQ:
		op = "="
	case NEQ:
		op = "!="
	case LT:
		op = "<"
	case GT:
		op = ">"
	case LTE:
		op = "<="
	case GTE:
		op = ">="
	case CONTAINS:
		op = "contains"
	case NOT_CONTAINS:
		op = "!contains"
	default:
		return nil, &ParseError{
			Pos:     next.Pos,
			Message: fmt.Sprintf("expected operator after field %q, got %s (%q)", field.Literal, next.Type, next.Literal),
		}
	}
	p.advance()

	val := p.peek()
	if val.Type != STRING && val.Type != NUMBER && val.Type != IDENT {
		return nil, &ParseError{
			Pos:     val.Pos,
			Message: fmt.Sprintf("expected value after operator, got %s (%q)", val.Type, val.Literal),
		}
	}
	p.advance()

	return ComparisonExpr{Field: field.Literal, Op: op, Value: val.Literal}, nil
}

func (p *parser) parseSortFields() ([]SortField, error) {
	var fields []SortField
	sf, err := p.parseSortField()
	if err != nil {
		return nil, err
	}
	fields = append(fields, sf)
	for p.peek().Type == COMMA {
		p.advance()
		sf, err = p.parseSortField()
		if err != nil {
			return nil, err
		}
		fields = append(fields, sf)
	}
	return fields, nil
}

func (p *parser) parseSortField() (SortField, error) {
	tok := p.peek()
	if tok.Type != IDENT {
		return SortField{}, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("expected field name for SORT, got %s (%q)", tok.Type, tok.Literal)}
	}
	field := p.advance().Literal
	desc := false
	if p.peek().Type == DESC {
		p.advance()
		desc = true
	} else if p.peek().Type == ASC {
		p.advance()
	}
	return SortField{Field: field, Desc: desc}, nil
}
