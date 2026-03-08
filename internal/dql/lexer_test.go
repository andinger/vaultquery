package dql

import (
	"testing"
)

func TestLexSimpleTable(t *testing.T) {
	tokens, err := Lex("TABLE name FROM \"Clients\"")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []struct {
		typ string
		lit string
	}{
		{TABLE, "TABLE"},
		{IDENT, "name"},
		{FROM, "FROM"},
		{STRING, "Clients"},
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

func TestLexAllOperators(t *testing.T) {
	tokens, err := Lex("= != < > <= >=")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{EQ, NEQ, LT, GT, LTE, GTE, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected type %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestLexQuotedStrings(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{"double quotes", `"hello world"`, "hello world"},
		{"single quotes", `'hello world'`, "hello world"},
		{"escaped double", `"say \"hello\""`, `say "hello"`},
		{"escaped single", `'it\'s'`, "it's"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[0].Type != STRING || tokens[0].Literal != tt.want {
				t.Errorf("expected STRING %q, got %s %q", tt.want, tokens[0].Type, tokens[0].Literal)
			}
		})
	}
}

func TestLexNumbers(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"42", "42"},
		{"3.14", "3.14"},
		{"0", "0"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[0].Type != NUMBER || tokens[0].Literal != tt.want {
				t.Errorf("expected NUMBER %q, got %s %q", tt.want, tokens[0].Type, tokens[0].Literal)
			}
		})
	}
}

func TestLexKeywordsCaseInsensitive(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"table", TABLE},
		{"TABLE", TABLE},
		{"Table", TABLE},
		{"where", WHERE},
		{"WHERE", WHERE},
		{"from", FROM},
		{"sort", SORT},
		{"limit", LIMIT},
		{"and", AND},
		{"or", OR},
		{"asc", ASC},
		{"desc", DESC},
		{"contains", CONTAINS},
		{"exists", EXISTS},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if tokens[0].Type != tt.want {
				t.Errorf("expected type %s, got %s", tt.want, tokens[0].Type)
			}
		})
	}
}

func TestLexEmptyInput(t *testing.T) {
	tokens, err := Lex("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 1 || tokens[0].Type != EOF {
		t.Errorf("expected single EOF token, got %v", tokens)
	}
}

func TestLexUnterminatedString(t *testing.T) {
	_, err := Lex(`"hello`)
	if err == nil {
		t.Fatal("expected error for unterminated string")
	}
}

func TestLexNotContainsAndNotExists(t *testing.T) {
	tokens, err := Lex("!contains !exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != NOT_CONTAINS {
		t.Errorf("expected NOT_CONTAINS, got %s", tokens[0].Type)
	}
	if tokens[1].Type != NOT_EXISTS {
		t.Errorf("expected NOT_EXISTS, got %s", tokens[1].Type)
	}
}

func TestLexDottedFieldAccess(t *testing.T) {
	tokens, err := Lex("file.name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []struct {
		typ string
		lit string
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

func TestLexArithmeticTokens(t *testing.T) {
	tokens, err := Lex("a + b - c * d / e % f")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{IDENT, PLUS, IDENT, MINUS, IDENT, STAR, IDENT, SLASH, IDENT, PERCENT, IDENT, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestLexLinkAndHash(t *testing.T) {
	tokens, err := Lex("[[page]] #tag")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []struct {
		typ string
		lit string
	}{
		{LINK_OPEN, "[["},
		{IDENT, "page"},
		{LINK_CLOSE, "]]"},
		{HASH, "#"},
		{IDENT, "tag"},
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

func TestLexArrow(t *testing.T) {
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

func TestLexBoolAndNull(t *testing.T) {
	tokens, err := Lex("true false null")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{TRUE, FALSE, NULL_KW, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestLexBrackets(t *testing.T) {
	tokens, err := Lex("a[0] {}")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{IDENT, LBRACKET, NUMBER, RBRACKET, LBRACE, RBRACE, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestLexPunctuation(t *testing.T) {
	tokens, err := Lex("(a, b)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	expected := []string{LPAREN, IDENT, COMMA, IDENT, RPAREN, EOF}
	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}
	for i, e := range expected {
		if tokens[i].Type != e {
			t.Errorf("token %d: expected %s, got %s", i, e, tokens[i].Type)
		}
	}
}

func TestLexBackslashBang(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		wantToks []struct {
			typ string
			lit string
		}
		wantErr bool
	}{
		{
			name:  "backslash-bang-equals becomes NEQ",
			input: `\!=`,
			wantToks: []struct {
				typ string
				lit string
			}{
				{NEQ, "!="},
				{EOF, ""},
			},
		},
		{
			name:  "backslash-bang-paren becomes BANG LPAREN",
			input: `\!(`,
			wantToks: []struct {
				typ string
				lit string
			}{
				{BANG, "!"},
				{LPAREN, "("},
				{EOF, ""},
			},
		},
		{
			name:  "backslash-bang-contains becomes NOT_CONTAINS",
			input: `\!contains`,
			wantToks: []struct {
				typ string
				lit string
			}{
				{NOT_CONTAINS, "!contains"},
				{EOF, ""},
			},
		},
		{
			name:    "lone backslash errors",
			input:   `\`,
			wantErr: true,
		},
		{
			name:    "backslash not followed by bang errors",
			input:   `\a`,
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tokens, err := Lex(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(tokens) != len(tt.wantToks) {
				t.Fatalf("expected %d tokens, got %d: %v", len(tt.wantToks), len(tokens), tokens)
			}
			for i, e := range tt.wantToks {
				if tokens[i].Type != e.typ || tokens[i].Literal != e.lit {
					t.Errorf("token %d: expected {%s %q}, got {%s %q}", i, e.typ, e.lit, tokens[i].Type, tokens[i].Literal)
				}
			}
		})
	}
}

func TestLexPositionTracking(t *testing.T) {
	tokens, err := Lex("TABLE name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Pos != 0 {
		t.Errorf("expected pos 0, got %d", tokens[0].Pos)
	}
	if tokens[1].Pos != 6 {
		t.Errorf("expected pos 6, got %d", tokens[1].Pos)
	}
}
