package dql

import (
	"testing"
	"time"
)

func TestValueConstructorsAndAccessors(t *testing.T) {
	tests := []struct {
		name string
		val  Value
		typ  ValueType
	}{
		{"null", NewNull(), TypeNull},
		{"number", NewNumber(42), TypeNumber},
		{"string", NewString("hello"), TypeString},
		{"bool", NewBool(true), TypeBool},
		{"date", NewDate(time.Date(2024, 1, 15, 0, 0, 0, 0, time.UTC)), TypeDate},
		{"duration", NewDuration(24 * time.Hour), TypeDuration},
		{"link", NewLink("My Page"), TypeLink},
		{"list", NewList([]Value{NewNumber(1), NewNumber(2)}), TypeList},
		{"object", NewObject(map[string]Value{"a": NewNumber(1)}), TypeObject},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.val.Type != tt.typ {
				t.Errorf("expected type %v, got %v", tt.typ, tt.val.Type)
			}
		})
	}
}

func TestValueAccessors(t *testing.T) {
	n := NewNumber(3.14)
	if v, ok := n.AsNumber(); !ok || v != 3.14 {
		t.Errorf("AsNumber: got %v, %v", v, ok)
	}
	if _, ok := n.AsString(); ok {
		t.Error("expected AsString to return false for number")
	}

	s := NewString("test")
	if v, ok := s.AsString(); !ok || v != "test" {
		t.Errorf("AsString: got %v, %v", v, ok)
	}

	b := NewBool(true)
	if v, ok := b.AsBool(); !ok || !v {
		t.Errorf("AsBool: got %v, %v", v, ok)
	}

	d := NewDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC))
	if v, ok := d.AsDate(); !ok || v.Year() != 2024 {
		t.Errorf("AsDate: got %v, %v", v, ok)
	}

	dur := NewDuration(time.Hour)
	if v, ok := dur.AsDuration(); !ok || v != time.Hour {
		t.Errorf("AsDuration: got %v, %v", v, ok)
	}

	lnk := NewLink("Page")
	if v, ok := lnk.AsLink(); !ok || v != "Page" {
		t.Errorf("AsLink: got %v, %v", v, ok)
	}

	lst := NewList([]Value{NewNumber(1)})
	if v, ok := lst.AsList(); !ok || len(v) != 1 {
		t.Errorf("AsList: got %v, %v", v, ok)
	}

	obj := NewObject(map[string]Value{"x": NewNumber(1)})
	if v, ok := obj.AsObject(); !ok || len(v) != 1 {
		t.Errorf("AsObject: got %v, %v", v, ok)
	}
}

func TestValueTruthy(t *testing.T) {
	tests := []struct {
		val  Value
		want bool
	}{
		{NewNull(), false},
		{NewBool(true), true},
		{NewBool(false), false},
		{NewNumber(0), false},
		{NewNumber(1), true},
		{NewNumber(-1), true},
		{NewString(""), false},
		{NewString("x"), true},
		{NewList(nil), false},
		{NewList([]Value{NewNumber(1)}), true},
		{NewObject(nil), false},
		{NewObject(map[string]Value{"a": NewNumber(1)}), true},
		{NewDate(time.Now()), true},
		{NewLink("page"), true},
	}
	for _, tt := range tests {
		if got := tt.val.Truthy(); got != tt.want {
			t.Errorf("Truthy(%v) = %v, want %v", tt.val, got, tt.want)
		}
	}
}

func TestValueCompare(t *testing.T) {
	tests := []struct {
		name string
		a, b Value
		want int
	}{
		{"null-null", NewNull(), NewNull(), 0},
		{"null-number", NewNull(), NewNumber(1), -1},
		{"number-null", NewNumber(1), NewNull(), 1},
		{"numbers-eq", NewNumber(5), NewNumber(5), 0},
		{"numbers-lt", NewNumber(3), NewNumber(5), -1},
		{"numbers-gt", NewNumber(7), NewNumber(5), 1},
		{"strings-eq", NewString("abc"), NewString("abc"), 0},
		{"strings-case", NewString("ABC"), NewString("abc"), 0},
		{"strings-lt", NewString("a"), NewString("b"), -1},
		{"bools", NewBool(false), NewBool(true), -1},
		{"string-number-coerce", NewString("42"), NewNumber(42), 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.a.Compare(tt.b)
			if got != tt.want {
				t.Errorf("Compare(%v, %v) = %d, want %d", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestValueArithmetic(t *testing.T) {
	// Number ops
	a, b := NewNumber(10), NewNumber(3)
	if v := a.Add(b); v.Type != TypeNumber || v.Inner.(float64) != 13 {
		t.Errorf("Add: %v", v)
	}
	if v := a.Sub(b); v.Type != TypeNumber || v.Inner.(float64) != 7 {
		t.Errorf("Sub: %v", v)
	}
	if v := a.Mul(b); v.Type != TypeNumber || v.Inner.(float64) != 30 {
		t.Errorf("Mul: %v", v)
	}
	if v := a.Div(b); v.Type != TypeNumber {
		t.Errorf("Div type: %v", v.Type)
	}
	if v := a.Mod(b); v.Type != TypeNumber || v.Inner.(float64) != 1 {
		t.Errorf("Mod: %v", v)
	}

	// Div by zero
	if v := a.Div(NewNumber(0)); v.Type != TypeNull {
		t.Errorf("Div by zero should be null, got %v", v)
	}

	// String concat
	if v := NewString("hello ").Add(NewString("world")); v.Type != TypeString || v.Inner.(string) != "hello world" {
		t.Errorf("String Add: %v", v)
	}

	// Date + Duration
	date := NewDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	dur := NewDuration(24 * time.Hour)
	result := date.Add(dur)
	if result.Type != TypeDate {
		t.Fatalf("Date+Duration type: %v", result.Type)
	}
	if got := result.Inner.(time.Time).Day(); got != 2 {
		t.Errorf("Date+Duration day: %d", got)
	}

	// Date - Date
	d1 := NewDate(time.Date(2024, 1, 3, 0, 0, 0, 0, time.UTC))
	d2 := NewDate(time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC))
	diff := d1.Sub(d2)
	if diff.Type != TypeDuration {
		t.Fatalf("Date-Date type: %v", diff.Type)
	}
	if diff.Inner.(time.Duration) != 48*time.Hour {
		t.Errorf("Date-Date: %v", diff)
	}

	// Type mismatch → null
	if v := NewString("x").Mul(NewNumber(3)); v.Type != TypeNull {
		t.Errorf("String*Number should be null, got %v", v)
	}
}

func TestValueToString(t *testing.T) {
	tests := []struct {
		val  Value
		want string
	}{
		{NewNull(), "-"},
		{NewNumber(42), "42"},
		{NewNumber(3.14), "3.14"},
		{NewString("hello"), "hello"},
		{NewBool(true), "true"},
		{NewBool(false), "false"},
		{NewDate(time.Date(2024, 6, 15, 0, 0, 0, 0, time.UTC)), "2024-06-15"},
		{NewLink("My Page"), "[[My Page]]"},
		{NewList([]Value{NewNumber(1), NewNumber(2), NewNumber(3)}), "1, 2, 3"},
	}
	for _, tt := range tests {
		if got := tt.val.ToString(); got != tt.want {
			t.Errorf("ToString(%v) = %q, want %q", tt.val, got, tt.want)
		}
	}
}

func TestCoerceFromString(t *testing.T) {
	tests := []struct {
		input string
		typ   ValueType
	}{
		{"", TypeNull},
		{"true", TypeBool},
		{"false", TypeBool},
		{"TRUE", TypeBool},
		{"42", TypeNumber},
		{"3.14", TypeNumber},
		{"2024-01-15", TypeDate},
		{"2024-01-15T10:30:00", TypeDate},
		{"hello world", TypeString},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			v := CoerceFromString(tt.input)
			if v.Type != tt.typ {
				t.Errorf("CoerceFromString(%q) type = %v, want %v", tt.input, v.Type, tt.typ)
			}
		})
	}
}

func TestValueNegate(t *testing.T) {
	if v := NewBool(true).Negate(); v.Type != TypeBool || v.Inner.(bool) != false {
		t.Errorf("Negate(true) = %v", v)
	}
	if v := NewNull().Negate(); v.Type != TypeBool || v.Inner.(bool) != true {
		t.Errorf("Negate(null) = %v", v)
	}
}

func TestValueTypeString(t *testing.T) {
	tests := []struct {
		typ  ValueType
		want string
	}{
		{TypeNull, "null"},
		{TypeNumber, "number"},
		{TypeString, "string"},
		{TypeBool, "boolean"},
		{TypeDate, "date"},
		{TypeDuration, "duration"},
		{TypeLink, "link"},
		{TypeList, "list"},
		{TypeObject, "object"},
	}
	for _, tt := range tests {
		if got := tt.typ.String(); got != tt.want {
			t.Errorf("%d.String() = %q, want %q", tt.typ, got, tt.want)
		}
	}
}
