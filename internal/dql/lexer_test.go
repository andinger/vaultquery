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

func TestLexIdentWithDotsAndHyphens(t *testing.T) {
	tokens, err := Lex("file.name my-field")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if tokens[0].Type != IDENT || tokens[0].Literal != "file.name" {
		t.Errorf("expected IDENT 'file.name', got %s %q", tokens[0].Type, tokens[0].Literal)
	}
	if tokens[1].Type != IDENT || tokens[1].Literal != "my-field" {
		t.Errorf("expected IDENT 'my-field', got %s %q", tokens[1].Type, tokens[1].Literal)
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
