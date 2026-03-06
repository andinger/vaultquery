package dql

// Token types
const (
	// Special
	EOF     = "EOF"
	ILLEGAL = "ILLEGAL"

	// Literals
	IDENT  = "IDENT"
	STRING = "STRING"
	NUMBER = "NUMBER"

	// Keywords
	TABLE        = "TABLE"
	LIST         = "LIST"
	TASK         = "TASK"
	CALENDAR     = "CALENDAR"
	FROM         = "FROM"
	WHERE        = "WHERE"
	SORT         = "SORT"
	LIMIT        = "LIMIT"
	GROUP        = "GROUP"
	BY           = "BY"
	FLATTEN      = "FLATTEN"
	AND          = "AND"
	OR           = "OR"
	NOT          = "NOT"
	ASC          = "ASC"
	DESC         = "DESC"
	CONTAINS     = "CONTAINS"
	NOT_CONTAINS = "NOT_CONTAINS"
	EXISTS       = "EXISTS"
	NOT_EXISTS   = "NOT_EXISTS"
	AS           = "AS"
	WITHOUT      = "WITHOUT"
	ID           = "ID"
	TRUE         = "TRUE"
	FALSE        = "FALSE"
	NULL_KW      = "NULL"

	// Operators
	EQ   = "="
	NEQ  = "!="
	LT   = "<"
	GT   = ">"
	LTE  = "<="
	GTE  = ">="
	BANG = "!"

	// Arithmetic
	PLUS    = "+"
	MINUS   = "-"
	STAR    = "*"
	SLASH   = "/"
	PERCENT = "%"

	// Punctuation
	LPAREN     = "("
	RPAREN     = ")"
	LBRACKET   = "["
	RBRACKET   = "]"
	LBRACE     = "{"
	RBRACE     = "}"
	COMMA      = ","
	DOT        = "."
	ARROW      = "=>"
	HASH       = "#"
	LINK_OPEN  = "[["
	LINK_CLOSE = "]]"
)

var keywords = map[string]string{
	"table":      TABLE,
	"list":       LIST,
	"task":       TASK,
	"calendar":   CALENDAR,
	"from":       FROM,
	"where":      WHERE,
	"sort":       SORT,
	"limit":      LIMIT,
	"group":      GROUP,
	"by":         BY,
	"flatten":    FLATTEN,
	"and":        AND,
	"or":         OR,
	"not":        NOT,
	"asc":        ASC,
	"ascending":  ASC,
	"desc":       DESC,
	"descending": DESC,
	"contains":   CONTAINS,
	"exists":     EXISTS,
	"as":         AS,
	"without":    WITHOUT,
	"id":         ID,
	"true":       TRUE,
	"false":      FALSE,
	"null":       NULL_KW,
}

// Token represents a lexical token.
type Token struct {
	Type    string
	Literal string
	Pos     int
}

// LookupIdent checks if an identifier is a keyword.
func LookupIdent(ident string) string {
	lower := toLower(ident)
	if tok, ok := keywords[lower]; ok {
		return tok
	}
	return IDENT
}

func toLower(s string) string {
	b := make([]byte, len(s))
	for i := range s {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 'a' - 'A'
		}
		b[i] = c
	}
	return string(b)
}
