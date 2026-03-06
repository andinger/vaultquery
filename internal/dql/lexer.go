package dql

import "fmt"

// Lex tokenizes the input string into a slice of tokens.
func Lex(input string) ([]Token, error) {
	var tokens []Token
	i := 0
	for i < len(input) {
		ch := input[i]

		// Skip whitespace
		if ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r' {
			i++
			continue
		}

		switch {
		case ch == '(':
			tokens = append(tokens, Token{Type: LPAREN, Literal: "(", Pos: i})
			i++
		case ch == ')':
			tokens = append(tokens, Token{Type: RPAREN, Literal: ")", Pos: i})
			i++
		case ch == ',':
			tokens = append(tokens, Token{Type: COMMA, Literal: ",", Pos: i})
			i++
		case ch == '+':
			tokens = append(tokens, Token{Type: PLUS, Literal: "+", Pos: i})
			i++
		case ch == '-':
			tokens = append(tokens, Token{Type: MINUS, Literal: "-", Pos: i})
			i++
		case ch == '*':
			tokens = append(tokens, Token{Type: STAR, Literal: "*", Pos: i})
			i++
		case ch == '/':
			if i+1 < len(input) && input[i+1] == '/' {
				// Comment — skip to end of line
				i += 2
				for i < len(input) && input[i] != '\n' {
					i++
				}
			} else {
				tokens = append(tokens, Token{Type: SLASH, Literal: "/", Pos: i})
				i++
			}
		case ch == '%':
			tokens = append(tokens, Token{Type: PERCENT, Literal: "%", Pos: i})
			i++
		case ch == '.':
			tokens = append(tokens, Token{Type: DOT, Literal: ".", Pos: i})
			i++
		case ch == '#':
			tokens = append(tokens, Token{Type: HASH, Literal: "#", Pos: i})
			i++
		case ch == '{':
			tokens = append(tokens, Token{Type: LBRACE, Literal: "{", Pos: i})
			i++
		case ch == '}':
			tokens = append(tokens, Token{Type: RBRACE, Literal: "}", Pos: i})
			i++
		case ch == '[':
			pos := i
			if i+1 < len(input) && input[i+1] == '[' {
				tokens = append(tokens, Token{Type: LINK_OPEN, Literal: "[[", Pos: pos})
				i += 2
			} else {
				tokens = append(tokens, Token{Type: LBRACKET, Literal: "[", Pos: pos})
				i++
			}
		case ch == ']':
			pos := i
			if i+1 < len(input) && input[i+1] == ']' {
				tokens = append(tokens, Token{Type: LINK_CLOSE, Literal: "]]", Pos: pos})
				i += 2
			} else {
				tokens = append(tokens, Token{Type: RBRACKET, Literal: "]", Pos: pos})
				i++
			}
		case ch == '=':
			pos := i
			if i+1 < len(input) && input[i+1] == '>' {
				tokens = append(tokens, Token{Type: ARROW, Literal: "=>", Pos: pos})
				i += 2
			} else {
				tokens = append(tokens, Token{Type: EQ, Literal: "=", Pos: pos})
				i++
			}
		case ch == '!':
			pos := i
			i++
			if i < len(input) && input[i] == '=' {
				tokens = append(tokens, Token{Type: NEQ, Literal: "!=", Pos: pos})
				i++
			} else if i < len(input) && isLetter(input[i]) {
				// Peek ahead: could be !contains or !exists
				start := i
				for i < len(input) && isIdentChar(input[i]) {
					i++
				}
				word := toLower(input[start:i])
				switch word {
				case "contains":
					tokens = append(tokens, Token{Type: NOT_CONTAINS, Literal: "!contains", Pos: pos})
				case "exists":
					tokens = append(tokens, Token{Type: NOT_EXISTS, Literal: "!exists", Pos: pos})
				default:
					// General negation: emit BANG then rewind to let the identifier be lexed normally
					tokens = append(tokens, Token{Type: BANG, Literal: "!", Pos: pos})
					i = start
				}
			} else {
				tokens = append(tokens, Token{Type: BANG, Literal: "!", Pos: pos})
			}
		case ch == '<':
			pos := i
			i++
			if i < len(input) && input[i] == '=' {
				tokens = append(tokens, Token{Type: LTE, Literal: "<=", Pos: pos})
				i++
			} else {
				tokens = append(tokens, Token{Type: LT, Literal: "<", Pos: pos})
			}
		case ch == '>':
			pos := i
			i++
			if i < len(input) && input[i] == '=' {
				tokens = append(tokens, Token{Type: GTE, Literal: ">=", Pos: pos})
				i++
			} else {
				tokens = append(tokens, Token{Type: GT, Literal: ">", Pos: pos})
			}
		case ch == '\'' || ch == '"':
			pos := i
			quote := ch
			i++
			start := i
			var s []byte
			for i < len(input) {
				if input[i] == '\\' && i+1 < len(input) {
					next := input[i+1]
					if next == quote {
						s = append(s, input[start:i]...)
						s = append(s, quote)
						i += 2
						start = i
						continue
					}
					if next == '\\' {
						s = append(s, input[start:i]...)
						s = append(s, '\\')
						i += 2
						start = i
						continue
					}
				}
				if input[i] == quote {
					break
				}
				i++
			}
			if i >= len(input) {
				return nil, fmt.Errorf("unterminated string at position %d", pos)
			}
			s = append(s, input[start:i]...)
			tokens = append(tokens, Token{Type: STRING, Literal: string(s), Pos: pos})
			i++ // skip closing quote
		case isDigit(ch):
			pos := i
			for i < len(input) && (isDigit(input[i]) || input[i] == '.') {
				i++
			}
			tokens = append(tokens, Token{Type: NUMBER, Literal: input[pos:i], Pos: pos})
		case isLetter(ch) || ch == '_':
			pos := i
			for i < len(input) && isIdentChar(input[i]) {
				i++
			}
			// Greedily consume hyphens followed by alphanumeric (no whitespace) for
			// Dataview-style identifiers like "time-played" or "my-date"
			for i < len(input) && input[i] == '-' && i+1 < len(input) && isIdentChar(input[i+1]) {
				i++ // consume hyphen
				for i < len(input) && isIdentChar(input[i]) {
					i++
				}
			}
			lit := input[pos:i]
			tokType := LookupIdent(lit)
			tokens = append(tokens, Token{Type: tokType, Literal: lit, Pos: pos})
		default:
			return nil, fmt.Errorf("unexpected character at position %d: %q", i, string(ch))
		}
	}
	tokens = append(tokens, Token{Type: EOF, Literal: "", Pos: i})
	return tokens, nil
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z')
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

func isIdentChar(ch byte) bool {
	return isLetter(ch) || isDigit(ch) || ch == '_'
}
