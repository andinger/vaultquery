package dql

import (
	"fmt"
	"strconv"
	"strings"
)

// ParseError represents a parsing error with position information.
type ParseError struct {
	Pos     int
	Message string
}

func (e *ParseError) Error() string {
	return fmt.Sprintf("parse error at position %d: %s", e.Pos, e.Message)
}

const maxParseDepth = 128

type parser struct {
	tokens []Token
	pos    int
	depth  int
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

func (p *parser) peekAt(offset int) Token {
	pos := p.pos + offset
	if pos < 0 || pos >= len(p.tokens) {
		return Token{Type: EOF, Pos: -1}
	}
	return p.tokens[pos]
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

func (p *parser) enterDepth() error {
	p.depth++
	if p.depth > maxParseDepth {
		return &ParseError{Pos: p.peek().Pos, Message: "maximum nesting depth exceeded"}
	}
	return nil
}

func (p *parser) leaveDepth() {
	p.depth--
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

// parseFieldDef parses a single field definition: an expression with optional AS alias.
func (p *parser) parseFieldDef() (FieldDef, error) {
	expr, err := p.parseValueExpr()
	if err != nil {
		return FieldDef{}, err
	}

	fd := FieldDef{Expr: expr}

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

// --- Value expression parser (for TABLE fields, SORT, FLATTEN, GROUP BY) ---
// Precedence: add/sub < mul/div < unary < postfix (dot, index, call) < atom

func (p *parser) parseValueExpr() (Expr, error) {
	return p.parseComparisonExpr()
}

func (p *parser) parseComparisonExpr() (Expr, error) {
	left, err := p.parseAddExpr()
	if err != nil {
		return nil, err
	}
	// Check for comparison operators
	switch p.peek().Type {
	case EQ, NEQ, LT, GT, LTE, GTE:
		opTok := p.advance()
		right, err := p.parseAddExpr()
		if err != nil {
			return nil, err
		}
		// Convert to a ComparisonExpr with field and value extracted
		// For complex expressions, we use the evaluator path
		field := exprToFieldName(left)
		val := exprToStringValue(right)
		var op string
		switch opTok.Type {
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
		}
		return ComparisonExpr{Field: field, Op: op, Value: val}, nil
	}
	return left, nil
}

// exprToFieldName extracts a field name from an expression (for backward compat).
func exprToFieldName(expr Expr) string {
	if fa, ok := expr.(FieldAccessExpr); ok {
		return FieldName(fa.Parts)
	}
	return ""
}

// exprToStringValue extracts a string value from a literal expression.
func exprToStringValue(expr Expr) string {
	if lit, ok := expr.(LiteralExpr); ok {
		if s, ok := lit.Val.AsString(); ok {
			return s
		}
		if n, ok := lit.Val.AsNumber(); ok {
			if n == float64(int64(n)) {
				return strconv.FormatInt(int64(n), 10)
			}
			return strconv.FormatFloat(n, 'f', -1, 64)
		}
		if b, ok := lit.Val.AsBool(); ok {
			if b {
				return "true"
			}
			return "false"
		}
	}
	return ""
}

func (p *parser) parseAddExpr() (Expr, error) {
	left, err := p.parseMulExpr()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == PLUS || p.peek().Type == MINUS {
		op := p.advance()
		right, err := p.parseMulExpr()
		if err != nil {
			return nil, err
		}
		left = ArithmeticExpr{Op: op.Literal, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseMulExpr() (Expr, error) {
	left, err := p.parseUnaryExpr()
	if err != nil {
		return nil, err
	}
	for p.peek().Type == STAR || p.peek().Type == SLASH || p.peek().Type == PERCENT {
		op := p.advance()
		right, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}
		left = ArithmeticExpr{Op: op.Literal, Left: left, Right: right}
	}
	return left, nil
}

func (p *parser) parseUnaryExpr() (Expr, error) {
	if p.peek().Type == BANG {
		p.advance()
		inner, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}
		return NegationExpr{Inner: inner}, nil
	}
	if p.peek().Type == MINUS {
		p.advance()
		inner, err := p.parseUnaryExpr()
		if err != nil {
			return nil, err
		}
		return ArithmeticExpr{Op: "-", Left: LiteralExpr{Val: NewNumber(0)}, Right: inner}, nil
	}
	return p.parsePostfixExpr()
}

func (p *parser) parsePostfixExpr() (Expr, error) {
	expr, err := p.parseAtomExpr()
	if err != nil {
		return nil, err
	}
	for {
		switch p.peek().Type {
		case DOT:
			p.advance()
			next := p.peek()
			if next.Type != IDENT {
				return nil, &ParseError{Pos: next.Pos, Message: fmt.Sprintf("expected field name after '.', got %s (%q)", next.Type, next.Literal)}
			}
			part := p.advance().Literal
			// Merge into FieldAccessExpr if possible
			if fa, ok := expr.(FieldAccessExpr); ok {
				expr = FieldAccessExpr{Parts: append(fa.Parts, part)}
			} else {
				expr = IndexExpr{Object: expr, Index: LiteralExpr{Val: NewString(part)}}
			}
		case LBRACKET:
			p.advance()
			idx, err := p.parseValueExpr()
			if err != nil {
				return nil, err
			}
			if _, err := p.expect(RBRACKET); err != nil {
				return nil, err
			}
			expr = IndexExpr{Object: expr, Index: idx}
		case LPAREN:
			// Function call — extract name from preceding expression
			name := exprToFuncName(expr)
			p.advance()
			var args []Expr
			if p.peek().Type != RPAREN {
				arg, err := p.parseValueExpr()
				if err != nil {
					return nil, err
				}
				args = append(args, arg)
				for p.peek().Type == COMMA {
					p.advance()
					arg, err = p.parseValueExpr()
					if err != nil {
						return nil, err
					}
					args = append(args, arg)
				}
			}
			if _, err := p.expect(RPAREN); err != nil {
				return nil, err
			}
			expr = FunctionCallExpr{Name: name, Args: args}
		default:
			return expr, nil
		}
	}
}

func (p *parser) parseAtomExpr() (Expr, error) {
	tok := p.peek()
	switch tok.Type {
	case NUMBER:
		p.advance()
		n, _ := strconv.ParseFloat(tok.Literal, 64)
		return LiteralExpr{Val: NewNumber(n)}, nil
	case STRING:
		p.advance()
		return LiteralExpr{Val: NewString(tok.Literal)}, nil
	case TRUE:
		p.advance()
		return LiteralExpr{Val: NewBool(true)}, nil
	case FALSE:
		p.advance()
		return LiteralExpr{Val: NewBool(false)}, nil
	case NULL_KW:
		p.advance()
		return LiteralExpr{Val: NewNull()}, nil
	case IDENT:
		p.advance()
		return FieldAccessExpr{Parts: []string{tok.Literal}}, nil
	case LPAREN:
		// Could be parenthesized expression or lambda
		if err := p.enterDepth(); err != nil {
			return nil, err
		}
		defer p.leaveDepth()
		saved := p.pos
		p.advance()
		// Try lambda: (x) => expr or (x, y) => expr
		if p.peek().Type == RPAREN {
			// () => expr
			p.advance()
			if p.peek().Type == ARROW {
				p.advance()
				body, err := p.parseValueExpr()
				if err != nil {
					return nil, err
				}
				return LambdaExpr{Params: nil, Body: body}, nil
			}
			// Not a lambda, backtrack — this shouldn't normally happen
			p.pos = saved
		} else if p.peek().Type == IDENT {
			// Could be (x) => ... or (x, y) => ... or (expr)
			if p.isLambdaStart() {
				params, err := p.parseLambdaParams()
				if err != nil {
					p.pos = saved
				} else {
					p.advance() // consume =>
					body, err := p.parseValueExpr()
					if err != nil {
						return nil, err
					}
					return LambdaExpr{Params: params, Body: body}, nil
				}
			}
		}
		// Fall through: parenthesized expression
		if p.pos == saved {
			p.advance() // consume (
		}
		expr, err := p.parseValueExpr()
		if err != nil {
			return nil, err
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		return ParenExpr{Inner: expr}, nil
	case LBRACKET:
		// List literal: [expr, expr, ...]
		p.advance()
		var items []Expr
		if p.peek().Type != RBRACKET {
			item, err := p.parseValueExpr()
			if err != nil {
				return nil, err
			}
			items = append(items, item)
			for p.peek().Type == COMMA {
				p.advance()
				item, err = p.parseValueExpr()
				if err != nil {
					return nil, err
				}
				items = append(items, item)
			}
		}
		if _, err := p.expect(RBRACKET); err != nil {
			return nil, err
		}
		return FunctionCallExpr{Name: "list", Args: items}, nil
	default:
		return nil, &ParseError{Pos: tok.Pos, Message: fmt.Sprintf("expected expression, got %s (%q)", tok.Type, tok.Literal)}
	}
}

// isLambdaStart checks if the current position (after opening paren) looks like
// lambda parameters: ident (COMMA ident)* RPAREN ARROW
func (p *parser) isLambdaStart() bool {
	saved := p.pos
	defer func() { p.pos = saved }()

	for {
		if p.peek().Type != IDENT {
			return false
		}
		p.advance()
		if p.peek().Type == RPAREN {
			p.advance()
			return p.peek().Type == ARROW
		}
		if p.peek().Type != COMMA {
			return false
		}
		p.advance()
	}
}

// parseLambdaParams parses (already past the open paren): ident (COMMA ident)* RPAREN
func (p *parser) parseLambdaParams() ([]string, error) {
	var params []string
	for {
		tok, err := p.expect(IDENT)
		if err != nil {
			return nil, err
		}
		params = append(params, tok.Literal)
		if p.peek().Type == RPAREN {
			p.advance()
			return params, nil
		}
		if _, err := p.expect(COMMA); err != nil {
			return nil, err
		}
	}
}

// exprToFuncName extracts a function name from the callee expression.
func exprToFuncName(expr Expr) string {
	if fa, ok := expr.(FieldAccessExpr); ok {
		return FieldName(fa.Parts)
	}
	return ""
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
		if err := p.enterDepth(); err != nil {
			return nil, err
		}
		defer p.leaveDepth()
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

	// Handle negation: !expr, NOT expr, or !contains(...)
	if tok.Type == BANG || tok.Type == NOT {
		p.advance()
		inner, err := p.parsePrimary()
		if err != nil {
			return nil, err
		}
		return NegationExpr{Inner: inner}, nil
	}
	if tok.Type == NOT_CONTAINS {
		// !contains used as a standalone function call: !contains(file.path, "xyz")
		p.advance()
		if p.peek().Type == LPAREN {
			p.advance()
			var args []Expr
			for p.peek().Type != RPAREN && p.peek().Type != EOF {
				arg, err := p.parseFieldDef()
				if err != nil {
					return nil, err
				}
				args = append(args, arg.Expr)
				if p.peek().Type == COMMA {
					p.advance()
				}
			}
			if _, err := p.expect(RPAREN); err != nil {
				return nil, err
			}
			return NegationExpr{Inner: FunctionCallExpr{Name: "contains", Args: args}}, nil
		}
		// Fallback: treat as an error
		return nil, &ParseError{Pos: tok.Pos, Message: "expected '(' after !contains"}
	}

	// Handle contains(...) as a standalone function call in WHERE
	if tok.Type == CONTAINS && p.peekAt(1).Type == LPAREN {
		p.advance() // consume "contains"
		p.advance() // consume "("
		var args []Expr
		for p.peek().Type != RPAREN && p.peek().Type != EOF {
			arg, err := p.parseFieldDef()
			if err != nil {
				return nil, err
			}
			args = append(args, arg.Expr)
			if p.peek().Type == COMMA {
				p.advance()
			}
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		return FunctionCallExpr{Name: "contains", Args: args}, nil
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

	// Function call in WHERE: contains(field, value), regextest(...), etc.
	if next.Type == LPAREN && !strings.Contains(fieldName, ".") {
		p.advance() // consume (
		var args []Expr
		for p.peek().Type != RPAREN && p.peek().Type != EOF {
			arg, err := p.parseFieldDef()
			if err != nil {
				return nil, err
			}
			args = append(args, arg.Expr)
			if p.peek().Type == COMMA {
				p.advance()
			}
		}
		if _, err := p.expect(RPAREN); err != nil {
			return nil, err
		}
		return FunctionCallExpr{Name: fieldName, Args: args}, nil
	}

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
	expr, err := p.parseValueExpr()
	if err != nil {
		return SortField{}, err
	}

	// Extract simple field name for backward compat
	fieldName := ""
	if fa, ok := expr.(FieldAccessExpr); ok {
		fieldName = FieldName(fa.Parts)
	}

	desc := false
	if p.peek().Type == DESC {
		p.advance()
		desc = true
	} else if p.peek().Type == ASC {
		p.advance()
	}
	return SortField{Field: fieldName, Expr: expr, Desc: desc}, nil
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
