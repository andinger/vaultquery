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
	TABLE         = "TABLE"
	LIST          = "LIST"
	FROM          = "FROM"
	WHERE         = "WHERE"
	SORT          = "SORT"
	LIMIT         = "LIMIT"
	GROUP         = "GROUP"
	BY            = "BY"
	FLATTEN       = "FLATTEN"
	AND           = "AND"
	OR            = "OR"
	ASC           = "ASC"
	DESC          = "DESC"
	CONTAINS      = "CONTAINS"
	NOT_CONTAINS  = "NOT_CONTAINS"
	EXISTS        = "EXISTS"
	NOT_EXISTS    = "NOT_EXISTS"

	// Operators
	EQ  = "="
	NEQ = "!="
	LT  = "<"
	GT  = ">"
	LTE = "<="
	GTE = ">="

	// Punctuation
	LPAREN = "("
	RPAREN = ")"
	COMMA  = ","
)

var keywords = map[string]string{
	"table":    TABLE,
	"list":     LIST,
	"from":     FROM,
	"where":    WHERE,
	"sort":     SORT,
	"limit":    LIMIT,
	"group":    GROUP,
	"by":       BY,
	"flatten":  FLATTEN,
	"and":      AND,
	"or":       OR,
	"asc":      ASC,
	"desc":     DESC,
	"contains": CONTAINS,
	"exists":   EXISTS,
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
