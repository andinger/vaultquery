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
		// Check for WITHOUT ID
		if p.peek().Type == WITHOUT {
			p.advance()
			if _, err := p.expect(ID); err != nil {
				return nil, err
			}
			q.WithoutID = true
		}
		// Parse field list if next token is an identifier (not a clause keyword)
		if p.isFieldStart() {
			fields, err := p.parseFieldDefList()
			if err != nil {
				return nil, err
			}
			q.Fields = fields
		}
	case LIST:
		p.advance()
		q.Mode = "LIST"
		// LIST WITHOUT ID expr
		if p.peek().Type == WITHOUT {
			p.advance()
			if _, err := p.expect(ID); err != nil {
				return nil, err
			}
			q.WithoutID = true
		}
	case TASK:
		p.advance()
		q.Mode = "TASK"
	case CALENDAR:
		p.advance()
		q.Mode = "CALENDAR"
	default:
		return nil, &ParseError{
			Pos:     tok.Pos,
			Message: fmt.Sprintf("expected TABLE, LIST, TASK, or CALENDAR, got %s (%q)", tok.Type, tok.Literal),
		}
	}

	// Optional clauses
	for {
		tok = p.peek()
		switch tok.Type {
		case FROM:
			p.advance()
			source, err := p.parseFromSource()
			if err != nil {
				return nil, err
			}
			q.FromSource = source
			// Set From string for backward compat when it's a simple folder
			if fs, ok := source.(FolderSource); ok {
				q.From = fs.Path
			}
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
			fields, err := p.parseFieldDefList()
			if err != nil {
				return nil, err
			}
			q.GroupBy = fields
		case FLATTEN:
			p.advance()
			fields, err := p.parseFieldDefList()
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

// isFieldStart returns true if the current token starts a field expression
// (i.e., is not a clause keyword like FROM, WHERE, etc.)
func (p *parser) isFieldStart() bool {
	tok := p.peek()
	switch tok.Type {
	case FROM, WHERE, SORT, LIMIT, GROUP, FLATTEN, EOF:
		return false
	}
	return true
}

// parseFieldDefList parses comma-separated field definitions with optional AS aliases.
func (p *parser) parseFieldDefList() ([]FieldDef, error) {
	var fields []FieldDef
	fd, err := p.parseFieldDef()
	if err != nil {
		return nil, err
	}
	fields = append(fields, fd)
	for p.peek().Type == COMMA {
		p.advance()
		fd, err = p.parseFieldDef()
		if err != nil {
			return nil, err
		}
		fields = append(fields, fd)
	}
	return fields, nil
}

// parseFieldDef parses a single field: either a simple identifier or an expression AS alias.
func (p *parser) parseFieldDef() (FieldDef, error) {
	tok := p.peek()
	if tok.Type != IDENT {
		return FieldDef{}, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("expected field name, got %s (%q)", tok.Type, tok.Literal)}
	}

	// For now, parse a simple dotted identifier for field definitions.
	// Full expression parsing in field defs will come in Phase 1.
	name := p.advance().Literal

	// Handle dotted access: file.name
	for p.peek().Type == DOT {
		p.advance()
		next := p.peek()
		if next.Type != IDENT {
			return FieldDef{}, &ParseError{Pos: next.Pos, Message: fmt.Sprintf("expected field name after '.', got %s (%q)", next.Type, next.Literal)}
		}
		name += "." + p.advance().Literal
	}

	fd := FieldDef{
		Expr: FieldAccessExpr{Parts: splitDotted(name)},
	}

	// Optional AS alias
	if p.peek().Type == AS {
		p.advance()
		aliasTok := p.peek()
		if aliasTok.Type == IDENT {
			fd.Alias = p.advance().Literal
		} else if aliasTok.Type == STRING {
			fd.Alias = p.advance().Literal
		} else {
			return FieldDef{}, &ParseError{Pos: aliasTok.Pos, Message: fmt.Sprintf("expected alias name after AS, got %s (%q)", aliasTok.Type, aliasTok.Literal)}
		}
	}

	return fd, nil
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

	// Consume dotted field parts: file.name, file.tags, etc.
	fieldName := field.Literal
	for p.peek().Type == DOT {
		p.advance()
		next := p.peek()
		if next.Type != IDENT {
			return nil, &ParseError{Pos: next.Pos, Message: fmt.Sprintf("expected field name after '.', got %s (%q)", next.Type, next.Literal)}
		}
		fieldName += "." + p.advance().Literal
	}

	next := p.peek()

	// exists / !exists
	if next.Type == EXISTS {
		p.advance()
		return ExistsExpr{Field: fieldName, Negated: false}, nil
	}
	if next.Type == NOT_EXISTS {
		p.advance()
		return ExistsExpr{Field: fieldName, Negated: true}, nil
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
			Message: fmt.Sprintf("expected operator after field %q, got %s (%q)", fieldName, next.Type, next.Literal),
		}
	}
	p.advance()

	val := p.peek()
	switch val.Type {
	case STRING, NUMBER, IDENT, TRUE, FALSE, NULL_KW:
		// valid value tokens
	default:
		return nil, &ParseError{
			Pos:     val.Pos,
			Message: fmt.Sprintf("expected value after operator, got %s (%q)", val.Type, val.Literal),
		}
	}
	p.advance()

	// Map true/false/null keywords to their string representation for ComparisonExpr
	valStr := val.Literal
	if val.Type == TRUE {
		valStr = "true"
	} else if val.Type == FALSE {
		valStr = "false"
	} else if val.Type == NULL_KW {
		valStr = ""
	}

	return ComparisonExpr{Field: fieldName, Op: op, Value: valStr}, nil
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
	fieldName := p.advance().Literal

	// Handle dotted sort fields
	for p.peek().Type == DOT {
		p.advance()
		next := p.peek()
		if next.Type != IDENT {
			return SortField{}, &ParseError{Pos: next.Pos, Message: fmt.Sprintf("expected field name after '.', got %s (%q)", next.Type, next.Literal)}
		}
		fieldName += "." + p.advance().Literal
	}

	desc := false
	if p.peek().Type == DESC {
		p.advance()
		desc = true
	} else if p.peek().Type == ASC {
		p.advance()
	}
	return SortField{Field: fieldName, Desc: desc}, nil
}

// --- FROM source parsing ---

func (p *parser) parseFromSource() (FromSource, error) {
	return p.parseFromOr()
}

func (p *parser) parseFromOr() (FromSource, error) {
	left, err := p.parseFromAnd()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == OR {
		p.advance()
		right, err := p.parseFromAnd()
		if err != nil {
			return nil, err
		}
		left = BooleanFromSource{Op: "OR", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseFromAnd() (FromSource, error) {
	left, err := p.parseFromPrimary()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == AND {
		p.advance()
		right, err := p.parseFromPrimary()
		if err != nil {
			return nil, err
		}
		left = BooleanFromSource{Op: "AND", Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseFromPrimary() (FromSource, error) {
	tok := p.peek()

	switch tok.Type {
	case STRING:
		// FROM "folder"
		p.advance()
		return FolderSource{Path: tok.Literal}, nil

	case HASH:
		// FROM #tag
		p.advance()
		tagTok := p.peek()
		if tagTok.Type != IDENT {
			return nil, &ParseError{Pos: tagTok.Pos, Message: fmt.Sprintf("expected tag name after '#', got %s (%q)", tagTok.Type, tagTok.Literal)}
		}
		tag := p.advance().Literal
		// Handle nested tags: #parent/child, #daily/2021/20/08
		for p.peek().Type == SLASH {
			p.advance()
			next := p.peek()
			if next.Type != IDENT && next.Type != NUMBER {
				break
			}
			tag += "/" + p.advance().Literal
		}
		return TagSource{Tag: tag}, nil

	case LINK_OPEN:
		// FROM [[page]]
		p.advance()
		pageTok := p.peek()
		if pageTok.Type != IDENT && pageTok.Type != STRING {
			return nil, &ParseError{Pos: pageTok.Pos, Message: fmt.Sprintf("expected page name in [[...]], got %s (%q)", pageTok.Type, pageTok.Literal)}
		}
		page := p.advance().Literal
		// Allow dotted/spaced page names
		for p.peek().Type == DOT || p.peek().Type == IDENT {
			if p.peek().Type == DOT {
				page += p.advance().Literal
			} else {
				page += " " + p.advance().Literal
			}
		}
		if _, err := p.expect(LINK_CLOSE); err != nil {
			return nil, err
		}
		return LinkSource{Target: page, Outgoing: false}, nil

	case IDENT:
		// FROM outgoing([[page]])
		if toLower(tok.Literal) == "outgoing" {
			p.advance()
			if _, err := p.expect(LPAREN); err != nil {
				return nil, err
			}
			if _, err := p.expect(LINK_OPEN); err != nil {
				return nil, err
			}
			pageTok := p.peek()
			if pageTok.Type != IDENT && pageTok.Type != STRING {
				return nil, &ParseError{Pos: pageTok.Pos, Message: "expected page name"}
			}
			page := p.advance().Literal
			for p.peek().Type == DOT || p.peek().Type == IDENT {
				if p.peek().Type == DOT {
					page += p.advance().Literal
				} else {
					page += " " + p.advance().Literal
				}
			}
			if _, err := p.expect(LINK_CLOSE); err != nil {
				return nil, err
			}
			if _, err := p.expect(RPAREN); err != nil {
				return nil, err
			}
			return LinkSource{Target: page, Outgoing: true}, nil
		}
		return nil, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("unexpected identifier in FROM: %q", tok.Literal)}

	case NOT, MINUS, BANG:
		// FROM NOT/- /! source
		p.advance()
		inner, err := p.parseFromPrimary()
		if err != nil {
			return nil, err
		}
		return NegatedFromSource{Inner: inner}, nil

	case LPAREN:
		// FROM (source)
		p.advance()
		inner, err := p.parseFromSource()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		return inner, nil

	default:
		return nil, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("expected FROM source, got %s (%q)", tok.Type, tok.Literal)}
	}
}

// splitDotted splits "file.name" into ["file", "name"].
func splitDotted(s string) []string {
	var parts []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '.' {
			parts = append(parts, s[start:i])
			start = i + 1
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// Helper functions for backward compatibility

// FieldDefName returns the simple field name from a FieldDef.
// If the expression is a FieldAccessExpr, it returns the dotted name.
func FieldDefName(fd FieldDef) string {
	if fa, ok := fd.Expr.(FieldAccessExpr); ok {
		return FieldName(fa.Parts)
	}
	return ""
}

// FieldDefNames extracts simple field names from a slice of FieldDefs.
func FieldDefNames(fds []FieldDef) []string {
	names := make([]string, len(fds))
	for i, fd := range fds {
		if fd.Alias != "" {
			names[i] = fd.Alias
		} else {
			names[i] = FieldDefName(fd)
		}
	}
	return names
}
