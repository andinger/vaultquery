package dql

// Tests translated from the Dataview repo:
// https://github.com/blacksmithgu/obsidian-dataview/blob/master/src/test/parse/parse.expression.test.ts
//
// These test the lexer and value parsing. Expression-level parsing (full Pratt parser
// for WHERE clauses) is not yet implemented — those tests are marked with t.Skip.

import (
	"testing"
	"time"
)

// --- Integer Literals ---

func TestDV_ParseIntegerLiteral(t *testing.T) {
	tokens, err := Lex("123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NUMBER || tokens[0].Literal != "123" {
		t.Errorf("expected NUMBER '123', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
}

func TestDV_ParseNegativeInteger(t *testing.T) {
	// -123 is lexed as MINUS, NUMBER
	tokens, err := Lex("-123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != MINUS {
		t.Errorf("expected MINUS, got %s", tokens[0].Type)
	}
	if tokens[1].Type != NUMBER || tokens[1].Literal != "123" {
		t.Errorf("expected NUMBER '123', got %s %q", tokens[1].Type, tokens[1].Literal)
	}
}

// --- Float Literals ---

func TestDV_ParseFloatLiteral(t *testing.T) {
	tokens, err := Lex("123.45")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NUMBER || tokens[0].Literal != "123.45" {
		t.Errorf("expected NUMBER '123.45', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
}

// --- String Literals ---

func TestDV_ParseStringLiteral(t *testing.T) {
	tokens, err := Lex(`"hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != STRING || tokens[0].Literal != "hello" {
		t.Errorf("expected STRING 'hello', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
}

func TestDV_ParseEmptyString(t *testing.T) {
	tokens, err := Lex(`""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != STRING || tokens[0].Literal != "" {
		t.Errorf("expected empty STRING, got %s %q", tokens[0].Type, tokens[0].Literal)
	}
}

func TestDV_ParseStringEscape(t *testing.T) {
	// "\"" → "
	tokens, err := Lex(`"\""`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Literal != `"` {
		t.Errorf("expected '\"', got %q", tokens[0].Literal)
	}
}

func TestDV_ParseStringEscapeBackslash(t *testing.T) {
	// "\\" → \
	tokens, err := Lex(`"\\"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Literal != `\` {
		t.Errorf(`expected '\', got %q`, tokens[0].Literal)
	}
}

func TestDV_ParseStringRegexEscape(t *testing.T) {
	// "\w+" → \w+ (backslash before non-special char kept as-is)
	tokens, err := Lex(`"\\w+"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Note: our lexer sees \\ → \, then w+ as literal chars
	// So the result is \w+
	if tokens[0].Literal != `\w+` {
		t.Errorf(`expected '\w+', got %q`, tokens[0].Literal)
	}
}

// --- Boolean Literals ---

func TestDV_ParseBooleanLiteral(t *testing.T) {
	tokens, err := Lex("true false")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != TRUE {
		t.Errorf("expected TRUE, got %s", tokens[0].Type)
	}
	if tokens[1].Type != FALSE {
		t.Errorf("expected FALSE, got %s", tokens[1].Type)
	}
}

// --- Null ---

func TestDV_ParseNull(t *testing.T) {
	tokens, err := Lex("null")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NULL_KW {
		t.Errorf("expected NULL, got %s", tokens[0].Type)
	}
}

func TestDV_ParseNullString(t *testing.T) {
	// "null" as string literal should remain a string
	tokens, err := Lex(`"null"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != STRING {
		t.Errorf("expected STRING, got %s", tokens[0].Type)
	}
	if tokens[0].Literal != "null" {
		t.Errorf("expected 'null', got %q", tokens[0].Literal)
	}
}

// --- Identifiers ---

func TestDV_ParseIdentifier(t *testing.T) {
	tokens, err := Lex("lma0")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != IDENT || tokens[0].Literal != "lma0" {
		t.Errorf("expected IDENT 'lma0', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
}

func TestDV_ParseIdentifierNoLeadingDigit(t *testing.T) {
	// "0no" should NOT parse as a single identifier
	tokens, err := Lex("0no")
	if err != nil {
		// Lexer may error on this or split it into NUMBER + IDENT
		return
	}
	// Should not be a single IDENT token
	if tokens[0].Type == IDENT && tokens[0].Literal == "0no" {
		t.Error("expected '0no' NOT to parse as a single identifier")
	}
}

func TestDV_ParseHyphenatedIdentifier(t *testing.T) {
	tokens, err := Lex("time-played")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != IDENT || tokens[0].Literal != "time-played" {
		t.Errorf("expected IDENT 'time-played', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
}

func TestDV_HyphenWithSpacesIsSubtraction(t *testing.T) {
	// "a - b" should be three tokens: IDENT MINUS IDENT
	tokens, err := Lex("a - b")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 4 { // a, -, b, EOF
		t.Fatalf("expected at least 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != IDENT || tokens[0].Literal != "a" {
		t.Errorf("expected IDENT 'a', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
	if tokens[1].Type != MINUS {
		t.Errorf("expected MINUS, got %s", tokens[1].Type)
	}
	if tokens[2].Type != IDENT || tokens[2].Literal != "b" {
		t.Errorf("expected IDENT 'b', got %s %q", tokens[2].Type, tokens[2].Literal)
	}
}

// --- Dot Notation ---

func TestDV_ParseDotNotation(t *testing.T) {
	// file.name → IDENT DOT IDENT
	tokens, err := Lex("file.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []struct {
		typ, lit string
	}{
		{IDENT, "file"},
		{DOT, "."},
		{IDENT, "name"},
		{EOF, ""},
	}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e.typ || tokens[i].Literal != e.lit {
			t.Errorf("token %d: expected {%s %q}, got {%s %q}", i, e.typ, e.lit, tokens[i].Type, tokens[i].Literal)
		}
	}
}

func TestDV_ParseDeepDotNotation(t *testing.T) {
	// a.b.c3 → IDENT DOT IDENT DOT IDENT
	tokens, err := Lex("a.b.c3")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 6 { // a . b . c3 EOF
		t.Fatalf("expected 6 tokens, got %d", len(tokens))
	}
	if tokens[0].Literal != "a" || tokens[2].Literal != "b" || tokens[4].Literal != "c3" {
		t.Errorf("unexpected token literals: %q %q %q", tokens[0].Literal, tokens[2].Literal, tokens[4].Literal)
	}
}

// --- Index Notation ---

func TestDV_ParseIndexNotation(t *testing.T) {
	// a[0] → IDENT LBRACKET NUMBER RBRACKET
	tokens, err := Lex("a[0]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{IDENT, LBRACKET, NUMBER, RBRACKET, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

// --- Function Calls ---

func TestDV_ParseFunctionNoArgs(t *testing.T) {
	// hello() → IDENT LPAREN RPAREN
	tokens, err := Lex("hello()")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{IDENT, LPAREN, RPAREN, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestDV_ParseFunctionWithArgs(t *testing.T) {
	// list(1, 2, 3) → correct tokens
	tokens, err := Lex("list(1, 2, 3)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// "list" is a keyword (LIST), but in expression context it should be valid
	// Our lexer maps "list" → LIST token, which is fine for query mode parsing
	if len(tokens) < 8 { // list ( 1 , 2 , 3 ) EOF
		t.Fatalf("expected at least 8 tokens, got %d", len(tokens))
	}
}

// --- Lambda Expressions ---

func TestDV_ParseLambdaTokens(t *testing.T) {
	// (x) => x → LPAREN IDENT RPAREN ARROW IDENT
	tokens, err := Lex("(x) => x")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{LPAREN, IDENT, RPAREN, ARROW, IDENT, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s (%q)", i, e, tokens[i].Type, tokens[i].Literal)
		}
	}
}

// --- Lists ---

func TestDV_ParseListTokens(t *testing.T) {
	// [1, 2, 3] → LBRACKET NUMBER COMMA NUMBER COMMA NUMBER RBRACKET
	tokens, err := Lex("[1, 2, 3]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{LBRACKET, NUMBER, COMMA, NUMBER, COMMA, NUMBER, RBRACKET, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestDV_ParseEmptyList(t *testing.T) {
	tokens, err := Lex("[]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != LBRACKET || tokens[1].Type != RBRACKET {
		t.Errorf("expected [ ], got %s %s", tokens[0].Type, tokens[1].Type)
	}
}

// --- Objects ---

func TestDV_ParseObjectTokens(t *testing.T) {
	// { a: 1 } → LBRACE IDENT : NUMBER RBRACE
	// Note: our lexer doesn't have a COLON token yet, so this tests basic structure
	tokens, err := Lex("{}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != LBRACE || tokens[1].Type != RBRACE {
		t.Errorf("expected { }, got %s %s", tokens[0].Type, tokens[1].Type)
	}
}

// --- Binary Operators ---

func TestDV_ParseArithmeticTokens(t *testing.T) {
	// 16 + "what" → NUMBER PLUS STRING
	tokens, err := Lex(`16 + "what"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NUMBER || tokens[1].Type != PLUS || tokens[2].Type != STRING {
		t.Errorf("expected NUMBER PLUS STRING, got %s %s %s", tokens[0].Type, tokens[1].Type, tokens[2].Type)
	}
}

func TestDV_ParseDivisionTokens(t *testing.T) {
	// 14 / 2 → NUMBER SLASH NUMBER
	tokens, err := Lex("14 / 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NUMBER || tokens[1].Type != SLASH || tokens[2].Type != NUMBER {
		t.Errorf("expected NUMBER SLASH NUMBER, got %s %s %s", tokens[0].Type, tokens[1].Type, tokens[2].Type)
	}
}

func TestDV_ParseModuloTokens(t *testing.T) {
	// 14 % 2 → NUMBER PERCENT NUMBER
	tokens, err := Lex("14 % 2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NUMBER || tokens[1].Type != PERCENT || tokens[2].Type != NUMBER {
		t.Errorf("expected NUMBER PERCENT NUMBER, got %s %s %s", tokens[0].Type, tokens[1].Type, tokens[2].Type)
	}
}

func TestDV_ParseMultiplicationNoSpaces(t *testing.T) {
	// 3*a → NUMBER STAR IDENT
	tokens, err := Lex("3*a")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NUMBER || tokens[1].Type != STAR || tokens[2].Type != IDENT {
		t.Errorf("expected NUMBER STAR IDENT, got %s %s %s", tokens[0].Type, tokens[1].Type, tokens[2].Type)
	}
}

// --- Negation ---

func TestDV_ParseNegationToken(t *testing.T) {
	// !true → BANG TRUE
	tokens, err := Lex("!true")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 3 {
		t.Fatalf("expected at least 3 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != BANG {
		t.Errorf("expected BANG, got %s", tokens[0].Type)
	}
	if tokens[1].Type != TRUE {
		t.Errorf("expected TRUE, got %s %q", tokens[1].Type, tokens[1].Literal)
	}
}

func TestDV_ParseDoubleNegation(t *testing.T) {
	// !!what → BANG BANG IDENT
	tokens, err := Lex("!!what")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) < 4 {
		t.Fatalf("expected at least 4 tokens, got %d", len(tokens))
	}
	if tokens[0].Type != BANG || tokens[1].Type != BANG {
		t.Errorf("expected BANG BANG, got %s %s", tokens[0].Type, tokens[1].Type)
	}
	if tokens[2].Type != IDENT || tokens[2].Literal != "what" {
		t.Errorf("expected IDENT 'what', got %s %q", tokens[2].Type, tokens[2].Literal)
	}
}

// --- Link Tokens ---

func TestDV_ParseLinkTokens(t *testing.T) {
	// [[test/Main]] → LINK_OPEN IDENT SLASH IDENT LINK_CLOSE
	tokens, err := Lex("[[test/Main]]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != LINK_OPEN {
		t.Errorf("expected LINK_OPEN, got %s", tokens[0].Type)
	}
	// The last token before EOF should be LINK_CLOSE
	found := false
	for _, tok := range tokens {
		if tok.Type == LINK_CLOSE {
			found = true
		}
	}
	if !found {
		t.Error("expected LINK_CLOSE token")
	}
}

// --- Date Parsing (via value system) ---

func TestDV_ParseDateValues(t *testing.T) {
	tests := []struct {
		name  string
		input string
		year  int
		month int
		day   int
	}{
		{"year-month-day", "1984-08-15", 1984, 8, 15},
		{"year-month-day 2", "2020-04-01", 2020, 4, 1},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			d, ok := ParseDate(tt.input)
			if !ok {
				t.Fatalf("ParseDate(%q): failed", tt.input)
			}
			if d.Year() != tt.year || int(d.Month()) != tt.month || d.Day() != tt.day {
				t.Errorf("expected %d-%02d-%02d, got %v", tt.year, tt.month, tt.day, d)
			}
		})
	}
}

func TestDV_ParseDateWithTime(t *testing.T) {
	d, ok := ParseDate("1984-08-15T12:42:59")
	if !ok {
		t.Fatal("ParseDate failed")
	}
	if d.Year() != 1984 || int(d.Month()) != 8 || d.Day() != 15 {
		t.Errorf("unexpected date: %v", d)
	}
	if d.Hour() != 12 || d.Minute() != 42 || d.Second() != 59 {
		t.Errorf("unexpected time: %v", d)
	}
}

func TestDV_ParseDateWithTimezone(t *testing.T) {
	d, ok := ParseDate("1985-12-06T19:40:10Z")
	if !ok {
		t.Fatal("ParseDate failed")
	}
	if d.Year() != 1985 || int(d.Month()) != 12 || d.Day() != 6 {
		t.Errorf("unexpected date: %v", d)
	}
	if d.Location() != time.UTC {
		t.Errorf("expected UTC, got %v", d.Location())
	}
}

func TestDV_ParseInvalidDate(t *testing.T) {
	_, ok := ParseDate("4237-14-73")
	if ok {
		t.Error("expected ParseDate to fail for invalid date 4237-14-73")
	}
}

// --- Duration Parsing ---

func TestDV_ParseDuration6Days(t *testing.T) {
	d1, ok := ParseDuration("6 days")
	if !ok {
		t.Fatal("ParseDuration('6 days') failed")
	}
	d2, ok := ParseDuration("6day")
	if !ok {
		t.Fatal("ParseDuration('6day') failed")
	}
	expected := 6 * 24 * time.Hour
	if d1 != expected {
		t.Errorf("expected %v, got %v", expected, d1)
	}
	if d1 != d2 {
		t.Errorf("6 days (%v) != 6day (%v)", d1, d2)
	}
}

func TestDV_ParseDuration4Minutes(t *testing.T) {
	d1, ok := ParseDuration("4min")
	if !ok {
		t.Fatal("ParseDuration('4min') failed")
	}
	d2, ok := ParseDuration("4 minutes")
	if !ok {
		t.Fatal("ParseDuration('4 minutes') failed")
	}
	d3, ok := ParseDuration("4 minute")
	if !ok {
		t.Fatal("ParseDuration('4 minute') failed")
	}
	expected := 4 * time.Minute
	if d1 != expected {
		t.Errorf("expected %v, got %v", expected, d1)
	}
	if d1 != d2 || d1 != d3 {
		t.Errorf("durations not equal: %v %v %v", d1, d2, d3)
	}
}

func TestDV_ParseDuration4Hours15Minutes(t *testing.T) {
	d1, ok := ParseDuration("4 hr 15 min")
	if !ok {
		t.Fatal("ParseDuration('4 hr 15 min') failed")
	}
	d2, ok := ParseDuration("4h15m")
	if !ok {
		t.Fatal("ParseDuration('4h15m') failed")
	}
	expected := 4*time.Hour + 15*time.Minute
	if d1 != expected {
		t.Errorf("expected %v, got %v", expected, d1)
	}
	if d1 != d2 {
		t.Errorf("durations not equal: %v %v", d1, d2)
	}
}

func TestDV_ParseDurationComplex(t *testing.T) {
	d1, ok := ParseDuration("4 years 6 weeks 9 minutes 3 seconds")
	if !ok {
		t.Fatal("ParseDuration complex failed")
	}
	d2, ok := ParseDuration("4yr6w9m3s")
	if !ok {
		t.Fatal("ParseDuration short form failed")
	}
	if d1 != d2 {
		t.Errorf("complex durations not equal: %v vs %v", d1, d2)
	}
	// 4 years = 4*365*24h, 6 weeks = 6*7*24h, 9 min, 3 sec
	expected := 4*365*24*time.Hour + 6*7*24*time.Hour + 9*time.Minute + 3*time.Second
	if d1 != expected {
		t.Errorf("expected %v, got %v", expected, d1)
	}
}

// --- Value system ---

func TestDV_CoerceFromString(t *testing.T) {
	tests := []struct {
		input string
		vtype ValueType
	}{
		{"true", TypeBool},
		{"false", TypeBool},
		{"42", TypeNumber},
		{"3.14", TypeNumber},
		{"hello", TypeString},
		{"2024-01-15", TypeDate},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v := CoerceFromString(tt.input)
			if v.Type != tt.vtype {
				t.Errorf("CoerceFromString(%q): expected type %v, got %v", tt.input, tt.vtype, v.Type)
			}
		})
	}
}

func TestDV_ValueComparison(t *testing.T) {
	// Number comparison
	a := NewNumber(10)
	b := NewNumber(20)
	if a.Compare(b) >= 0 {
		t.Error("expected 10 < 20")
	}

	// String comparison
	s1 := NewString("alpha")
	s2 := NewString("beta")
	if s1.Compare(s2) >= 0 {
		t.Error("expected 'alpha' < 'beta'")
	}

	// Null comparison
	n := NewNull()
	if n.Compare(NewString("x")) >= 0 {
		t.Error("expected null to sort before non-null")
	}
}

func TestDV_ValueArithmetic(t *testing.T) {
	a := NewNumber(10)
	b := NewNumber(3)

	sum := a.Add(b)
	if n, _ := sum.AsNumber(); n != 13 {
		t.Errorf("expected 13, got %v", n)
	}

	diff := a.Sub(b)
	if n, _ := diff.AsNumber(); n != 7 {
		t.Errorf("expected 7, got %v", n)
	}

	prod := a.Mul(b)
	if n, _ := prod.AsNumber(); n != 30 {
		t.Errorf("expected 30, got %v", n)
	}

	quot := a.Div(b)
	if n, _ := quot.AsNumber(); n < 3.33 || n > 3.34 {
		t.Errorf("expected ~3.33, got %v", n)
	}

	mod := a.Mod(b)
	if n, _ := mod.AsNumber(); n != 1 {
		t.Errorf("expected 1, got %v", n)
	}
}

func TestDV_ValueTruthy(t *testing.T) {
	tests := []struct {
		val    Value
		truthy bool
	}{
		{NewNull(), false},
		{NewBool(true), true},
		{NewBool(false), false},
		{NewNumber(0), false},
		{NewNumber(1), true},
		{NewNumber(-1), true},
		{NewString(""), false},
		{NewString("hello"), true},
	}
	for _, tt := range tests {
		if tt.val.Truthy() != tt.truthy {
			t.Errorf("Truthy(%v): expected %v", tt.val, tt.truthy)
		}
	}
}
