package dql

import (
	"fmt"
	"math"
	"strconv"
	"strings"
	"time"
)

// ValueType represents the runtime type of a DQL value.
type ValueType int

const (
	TypeNull ValueType = iota
	TypeNumber
	TypeString
	TypeBool
	TypeDate
	TypeDuration
	TypeLink
	TypeList
	TypeObject
)

func (t ValueType) String() string {
	switch t {
	case TypeNull:
		return "null"
	case TypeNumber:
		return "number"
	case TypeString:
		return "string"
	case TypeBool:
		return "boolean"
	case TypeDate:
		return "date"
	case TypeDuration:
		return "duration"
	case TypeLink:
		return "link"
	case TypeList:
		return "list"
	case TypeObject:
		return "object"
	default:
		return "unknown"
	}
}

// Value represents a runtime DQL value with a tagged type.
type Value struct {
	Type  ValueType
	Inner any // float64 | string | bool | time.Time | time.Duration | []Value | map[string]Value
}

// Constructors

func NewNull() Value                          { return Value{Type: TypeNull} }
func NewNumber(n float64) Value               { return Value{Type: TypeNumber, Inner: n} }
func NewString(s string) Value                { return Value{Type: TypeString, Inner: s} }
func NewBool(b bool) Value                    { return Value{Type: TypeBool, Inner: b} }
func NewDate(t time.Time) Value               { return Value{Type: TypeDate, Inner: t} }
func NewDuration(d time.Duration) Value       { return Value{Type: TypeDuration, Inner: d} }
func NewLink(target string) Value             { return Value{Type: TypeLink, Inner: target} }
func NewList(items []Value) Value             { return Value{Type: TypeList, Inner: items} }
func NewObject(fields map[string]Value) Value { return Value{Type: TypeObject, Inner: fields} }

// Accessors

func (v Value) AsNumber() (float64, bool) {
	if v.Type == TypeNumber {
		return v.Inner.(float64), true
	}
	return 0, false
}

func (v Value) AsString() (string, bool) {
	if v.Type == TypeString {
		return v.Inner.(string), true
	}
	return "", false
}

func (v Value) AsBool() (bool, bool) {
	if v.Type == TypeBool {
		return v.Inner.(bool), true
	}
	return false, false
}

func (v Value) AsDate() (time.Time, bool) {
	if v.Type == TypeDate {
		return v.Inner.(time.Time), true
	}
	return time.Time{}, false
}

func (v Value) AsDuration() (time.Duration, bool) {
	if v.Type == TypeDuration {
		return v.Inner.(time.Duration), true
	}
	return 0, false
}

func (v Value) AsLink() (string, bool) {
	if v.Type == TypeLink {
		return v.Inner.(string), true
	}
	return "", false
}

func (v Value) AsList() ([]Value, bool) {
	if v.Type == TypeList {
		return v.Inner.([]Value), true
	}
	return nil, false
}

func (v Value) AsObject() (map[string]Value, bool) {
	if v.Type == TypeObject {
		return v.Inner.(map[string]Value), true
	}
	return nil, false
}

// IsNull returns true if the value is null.
func (v Value) IsNull() bool {
	return v.Type == TypeNull
}

// Truthy returns the boolean interpretation of the value (Dataview semantics).
func (v Value) Truthy() bool {
	switch v.Type {
	case TypeNull:
		return false
	case TypeBool:
		return v.Inner.(bool)
	case TypeNumber:
		return v.Inner.(float64) != 0
	case TypeString:
		return v.Inner.(string) != ""
	case TypeList:
		return len(v.Inner.([]Value)) > 0
	case TypeObject:
		return len(v.Inner.(map[string]Value)) > 0
	default:
		return true
	}
}

// ToString returns a human-readable string representation.
func (v Value) ToString() string {
	switch v.Type {
	case TypeNull:
		return "-"
	case TypeNumber:
		n := v.Inner.(float64)
		if n == math.Trunc(n) {
			return strconv.FormatInt(int64(n), 10)
		}
		return strconv.FormatFloat(n, 'f', -1, 64)
	case TypeString:
		return v.Inner.(string)
	case TypeBool:
		if v.Inner.(bool) {
			return "true"
		}
		return "false"
	case TypeDate:
		return v.Inner.(time.Time).Format("2006-01-02")
	case TypeDuration:
		return v.Inner.(time.Duration).String()
	case TypeLink:
		return "[[" + v.Inner.(string) + "]]"
	case TypeList:
		items := v.Inner.([]Value)
		parts := make([]string, len(items))
		for i, item := range items {
			parts[i] = item.ToString()
		}
		return strings.Join(parts, ", ")
	case TypeObject:
		return fmt.Sprintf("%v", v.Inner)
	default:
		return ""
	}
}

// Compare returns -1, 0, or 1 comparing v to other.
// Null is less than everything. Mismatched types compare by type ordinal.
func (v Value) Compare(other Value) int {
	if v.Type == TypeNull && other.Type == TypeNull {
		return 0
	}
	if v.Type == TypeNull {
		return -1
	}
	if other.Type == TypeNull {
		return 1
	}

	// Coerce for comparison: string that looks like a number vs number
	a, b := v, other
	if a.Type != b.Type {
		a, b = coerceForComparison(a, b)
	}

	if a.Type != b.Type {
		// Fall back to type ordinal
		if a.Type < b.Type {
			return -1
		}
		return 1
	}

	switch a.Type {
	case TypeNumber:
		an, bn := a.Inner.(float64), b.Inner.(float64)
		if an < bn {
			return -1
		}
		if an > bn {
			return 1
		}
		return 0
	case TypeString:
		as, bs := strings.ToLower(a.Inner.(string)), strings.ToLower(b.Inner.(string))
		if as < bs {
			return -1
		}
		if as > bs {
			return 1
		}
		return 0
	case TypeBool:
		ab, bb := a.Inner.(bool), b.Inner.(bool)
		if ab == bb {
			return 0
		}
		if !ab {
			return -1
		}
		return 1
	case TypeDate:
		at, bt := a.Inner.(time.Time), b.Inner.(time.Time)
		if at.Before(bt) {
			return -1
		}
		if at.After(bt) {
			return 1
		}
		return 0
	case TypeDuration:
		ad, bd := a.Inner.(time.Duration), b.Inner.(time.Duration)
		if ad < bd {
			return -1
		}
		if ad > bd {
			return 1
		}
		return 0
	case TypeLink:
		al, bl := strings.ToLower(a.Inner.(string)), strings.ToLower(b.Inner.(string))
		if al < bl {
			return -1
		}
		if al > bl {
			return 1
		}
		return 0
	default:
		return 0
	}
}

// Add performs addition between values (number+number, date+duration, string concat).
func (v Value) Add(other Value) Value {
	if v.Type == TypeNumber && other.Type == TypeNumber {
		return NewNumber(v.Inner.(float64) + other.Inner.(float64))
	}
	if v.Type == TypeDate && other.Type == TypeDuration {
		return NewDate(v.Inner.(time.Time).Add(other.Inner.(time.Duration)))
	}
	if v.Type == TypeDuration && other.Type == TypeDate {
		return NewDate(other.Inner.(time.Time).Add(v.Inner.(time.Duration)))
	}
	if v.Type == TypeDuration && other.Type == TypeDuration {
		return NewDuration(v.Inner.(time.Duration) + other.Inner.(time.Duration))
	}
	if v.Type == TypeString && other.Type == TypeString {
		return NewString(v.Inner.(string) + other.Inner.(string))
	}
	return NewNull()
}

// Sub performs subtraction between values.
func (v Value) Sub(other Value) Value {
	if v.Type == TypeNumber && other.Type == TypeNumber {
		return NewNumber(v.Inner.(float64) - other.Inner.(float64))
	}
	if v.Type == TypeDate && other.Type == TypeDuration {
		return NewDate(v.Inner.(time.Time).Add(-other.Inner.(time.Duration)))
	}
	if v.Type == TypeDate && other.Type == TypeDate {
		return NewDuration(v.Inner.(time.Time).Sub(other.Inner.(time.Time)))
	}
	if v.Type == TypeDuration && other.Type == TypeDuration {
		return NewDuration(v.Inner.(time.Duration) - other.Inner.(time.Duration))
	}
	return NewNull()
}

// Mul performs multiplication.
func (v Value) Mul(other Value) Value {
	if v.Type == TypeNumber && other.Type == TypeNumber {
		return NewNumber(v.Inner.(float64) * other.Inner.(float64))
	}
	return NewNull()
}

// Div performs division.
func (v Value) Div(other Value) Value {
	if v.Type == TypeNumber && other.Type == TypeNumber {
		d := other.Inner.(float64)
		if d == 0 {
			return NewNull()
		}
		return NewNumber(v.Inner.(float64) / d)
	}
	return NewNull()
}

// Mod performs modulo.
func (v Value) Mod(other Value) Value {
	if v.Type == TypeNumber && other.Type == TypeNumber {
		d := other.Inner.(float64)
		if d == 0 {
			return NewNull()
		}
		return NewNumber(math.Mod(v.Inner.(float64), d))
	}
	return NewNull()
}

// Negate returns the logical negation of the value.
func (v Value) Negate() Value {
	return NewBool(!v.Truthy())
}

// coerceForComparison attempts to make two values the same type for comparison.
func coerceForComparison(a, b Value) (Value, Value) {
	// String to Number
	if a.Type == TypeString && b.Type == TypeNumber {
		if n, err := strconv.ParseFloat(a.Inner.(string), 64); err == nil {
			return NewNumber(n), b
		}
	}
	if a.Type == TypeNumber && b.Type == TypeString {
		if n, err := strconv.ParseFloat(b.Inner.(string), 64); err == nil {
			return a, NewNumber(n)
		}
	}
	return a, b
}

// CoerceFromString attempts to infer a typed Value from a raw string
// (used when reading from the EAV store).
func CoerceFromString(s string) Value {
	if s == "" {
		return NewNull()
	}

	// Boolean
	lower := strings.ToLower(s)
	if lower == "true" {
		return NewBool(true)
	}
	if lower == "false" {
		return NewBool(false)
	}

	// Number
	if n, err := strconv.ParseFloat(s, 64); err == nil {
		return NewNumber(n)
	}

	// Date (ISO formats)
	for _, layout := range []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return NewDate(t)
		}
	}

	return NewString(s)
}
