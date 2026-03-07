package eval

import (
	"fmt"
	"math"
	"regexp"
	"strings"
	"time"

	"github.com/andinger/vaultquery/internal/dql"
)

const (
	maxPadWidth        = 10_000
	maxRegexPatternLen = 10_240
)

// RegisterBuiltins adds all built-in DQL functions to the evaluator.
func RegisterBuiltins(ev *Evaluator) {
	// Constructors
	ev.RegisterFunc("object", fnObject)
	ev.RegisterFunc("list", fnList)
	ev.RegisterFunc("date", fnDate)
	ev.RegisterFunc("dur", fnDur)
	ev.RegisterFunc("number", fnNumber)
	ev.RegisterFunc("string", fnString)
	ev.RegisterFunc("link", fnLink)
	ev.RegisterFunc("typeof", fnTypeof)

	// Numeric
	ev.RegisterFunc("round", fnRound)
	ev.RegisterFunc("floor", fnFloor)
	ev.RegisterFunc("ceil", fnCeil)
	ev.RegisterFunc("min", fnMin)
	ev.RegisterFunc("max", fnMax)
	ev.RegisterFunc("sum", fnSum)
	ev.RegisterFunc("product", fnProduct)
	ev.RegisterFunc("average", fnAverage)
	ev.RegisterFunc("minby", fnMinBy)
	ev.RegisterFunc("maxby", fnMaxBy)

	// Arrays
	ev.RegisterFunc("contains", fnContains)
	ev.RegisterFunc("icontains", fnIcontains)
	ev.RegisterFunc("econtains", fnEcontains)
	ev.RegisterFunc("sort", fnSort)
	ev.RegisterFunc("reverse", fnReverse)
	ev.RegisterFunc("length", fnLength)
	ev.RegisterFunc("flat", fnFlat)
	ev.RegisterFunc("slice", fnSlice)
	ev.RegisterFunc("unique", fnUnique)
	ev.RegisterFunc("join", fnJoin)
	ev.RegisterFunc("all", fnAll)
	ev.RegisterFunc("any", fnAny)
	ev.RegisterFunc("none", fnNone)
	ev.RegisterFunc("nonnull", fnNonnull)

	// Strings
	ev.RegisterFunc("lower", fnLower)
	ev.RegisterFunc("upper", fnUpper)
	ev.RegisterFunc("split", fnSplit)
	ev.RegisterFunc("replace", fnReplace)
	ev.RegisterFunc("regextest", fnRegextest)
	ev.RegisterFunc("regexmatch", fnRegexmatch)
	ev.RegisterFunc("regexreplace", fnRegexreplace)
	ev.RegisterFunc("startswith", fnStartswith)
	ev.RegisterFunc("endswith", fnEndswith)
	ev.RegisterFunc("substring", fnSubstring)
	ev.RegisterFunc("truncate", fnTruncate)
	ev.RegisterFunc("padleft", fnPadleft)
	ev.RegisterFunc("padright", fnPadright)

	// Utility
	ev.RegisterFunc("default", fnDefault)
	ev.RegisterFunc("choice", fnChoice)
	ev.RegisterFunc("dateformat", fnDateformat)
	ev.RegisterFunc("durationformat", fnDurationformat)
	ev.RegisterFunc("striptime", fnStriptime)
	ev.RegisterFunc("meta", fnMeta)
	ev.RegisterFunc("currencyformat", fnCurrencyformat)
}

// --- Constructors ---

func fnObject(args []dql.Value, _ *EvalContext) dql.Value {
	obj := make(map[string]dql.Value)
	// Args are key-value pairs: object("key1", val1, "key2", val2, ...)
	for i := 0; i+1 < len(args); i += 2 {
		key := args[i].ToString()
		obj[key] = args[i+1]
	}
	return dql.NewObject(obj)
}

func fnList(args []dql.Value, _ *EvalContext) dql.Value {
	return dql.NewList(args)
}

func fnDate(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	lower := strings.ToLower(s)
	if lower == "today" || lower == "now" {
		return dql.NewDate(time.Now())
	}
	if d, ok := dql.ParseDate(s); ok {
		return dql.NewDate(d)
	}
	return dql.NewNull()
}

func fnDur(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	if d, ok := dql.ParseDuration(s); ok {
		return dql.NewDuration(d)
	}
	return dql.NewNull()
}

func fnNumber(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	switch args[0].Type {
	case dql.TypeNumber:
		return args[0]
	case dql.TypeString:
		v := dql.CoerceFromString(args[0].Inner.(string))
		if v.Type == dql.TypeNumber {
			return v
		}
	case dql.TypeBool:
		if args[0].Inner.(bool) {
			return dql.NewNumber(1)
		}
		return dql.NewNumber(0)
	}
	return dql.NewNull()
}

func fnString(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewString("")
	}
	return dql.NewString(args[0].ToString())
}

func fnLink(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	return dql.NewLink(s)
}

func fnTypeof(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewString("null")
	}
	// Use Dataview-compatible type names
	switch args[0].Type {
	case dql.TypeNull:
		return dql.NewString("null")
	case dql.TypeNumber:
		return dql.NewString("number")
	case dql.TypeString:
		return dql.NewString("string")
	case dql.TypeBool:
		return dql.NewString("boolean")
	case dql.TypeDate:
		return dql.NewString("date")
	case dql.TypeDuration:
		return dql.NewString("duration")
	case dql.TypeLink:
		return dql.NewString("link")
	case dql.TypeList:
		return dql.NewString("array")
	case dql.TypeObject:
		return dql.NewString("object")
	default:
		return dql.NewString("unknown")
	}
}

// --- Numeric ---

func fnRound(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	n, ok := args[0].AsNumber()
	if !ok {
		return dql.NewNull()
	}
	precision := 0
	if len(args) > 1 {
		if p, ok := args[1].AsNumber(); ok {
			precision = int(p)
		}
	}
	pow := math.Pow(10, float64(precision))
	return dql.NewNumber(math.Round(n*pow) / pow)
}

func fnFloor(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if n, ok := args[0].AsNumber(); ok {
		return dql.NewNumber(math.Floor(n))
	}
	return dql.NewNull()
}

func fnCeil(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if n, ok := args[0].AsNumber(); ok {
		return dql.NewNumber(math.Ceil(n))
	}
	return dql.NewNull()
}

func fnMin(args []dql.Value, _ *EvalContext) dql.Value {
	return aggregateList(args, func(a, b dql.Value) dql.Value {
		if a.Compare(b) <= 0 {
			return a
		}
		return b
	})
}

func fnMax(args []dql.Value, _ *EvalContext) dql.Value {
	return aggregateList(args, func(a, b dql.Value) dql.Value {
		if a.Compare(b) >= 0 {
			return a
		}
		return b
	})
}

func fnSum(args []dql.Value, _ *EvalContext) dql.Value {
	items := flattenArgs(args)
	result := dql.NewNumber(0)
	for _, item := range items {
		result = result.Add(item)
	}
	return result
}

func fnProduct(args []dql.Value, _ *EvalContext) dql.Value {
	items := flattenArgs(args)
	if len(items) == 0 {
		return dql.NewNumber(0)
	}
	result := dql.NewNumber(1)
	for _, item := range items {
		result = result.Mul(item)
	}
	return result
}

func fnAverage(args []dql.Value, _ *EvalContext) dql.Value {
	items := flattenArgs(args)
	if len(items) == 0 {
		return dql.NewNull()
	}
	sum := dql.NewNumber(0)
	for _, item := range items {
		sum = sum.Add(item)
	}
	return sum.Div(dql.NewNumber(float64(len(items))))
}

func fnMinBy(args []dql.Value, _ *EvalContext) dql.Value { return dql.NewNull() } // needs lambda
func fnMaxBy(args []dql.Value, _ *EvalContext) dql.Value { return dql.NewNull() } // needs lambda

// --- Arrays ---

func fnContains(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewBool(false)
	}
	return dql.NewBool(valueContainsCheck(args[0], args[1]))
}

func fnIcontains(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewBool(false)
	}
	// Case-insensitive string contains
	if s, ok := args[0].AsString(); ok {
		if n, ok := args[1].AsString(); ok {
			return dql.NewBool(strings.Contains(strings.ToLower(s), strings.ToLower(n)))
		}
	}
	return fnContains(args, nil)
}

func fnEcontains(args []dql.Value, _ *EvalContext) dql.Value {
	// Exact contains — no fuzzy substring matching within list elements
	if len(args) < 2 {
		return dql.NewBool(false)
	}
	return dql.NewBool(valueExactContains(args[0], args[1]))
}

func fnSort(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewList(nil)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewList(nil)
	}
	sorted := make([]dql.Value, len(items))
	copy(sorted, items)
	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[i].Compare(sorted[j]) > 0 {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return dql.NewList(sorted)
}

func fnReverse(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewList(nil)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewList(nil)
	}
	reversed := make([]dql.Value, len(items))
	for i, v := range items {
		reversed[len(items)-1-i] = v
	}
	return dql.NewList(reversed)
}

func fnLength(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNumber(0)
	}
	if items, ok := args[0].AsList(); ok {
		return dql.NewNumber(float64(len(items)))
	}
	if s, ok := args[0].AsString(); ok {
		return dql.NewNumber(float64(len(s)))
	}
	if obj, ok := args[0].AsObject(); ok {
		return dql.NewNumber(float64(len(obj)))
	}
	return dql.NewNumber(0)
}

func fnFlat(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewList(nil)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewList(nil)
	}
	var result []dql.Value
	for _, item := range items {
		if inner, ok := item.AsList(); ok {
			result = append(result, inner...)
		} else {
			result = append(result, item)
		}
	}
	return dql.NewList(result)
}

func fnSlice(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewList(nil)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewList(nil)
	}
	start := 0
	if n, ok := args[1].AsNumber(); ok {
		start = int(n)
	}
	end := len(items)
	if len(args) > 2 {
		if n, ok := args[2].AsNumber(); ok {
			end = int(n)
		}
	}
	if start < 0 {
		start = len(items) + start
	}
	if end < 0 {
		end = len(items) + end
	}
	if start < 0 {
		start = 0
	}
	if end > len(items) {
		end = len(items)
	}
	if start >= end {
		return dql.NewList(nil)
	}
	return dql.NewList(items[start:end])
}

func fnUnique(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewList(nil)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewList(nil)
	}
	var result []dql.Value
	seen := make(map[string]bool)
	for _, item := range items {
		key := item.ToString()
		if !seen[key] {
			seen[key] = true
			result = append(result, item)
		}
	}
	return dql.NewList(result)
}

func fnJoin(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewString("")
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewString(args[0].ToString())
	}
	sep := ", "
	if len(args) > 1 {
		if s, ok := args[1].AsString(); ok {
			sep = s
		}
	}
	parts := make([]string, len(items))
	for i, item := range items {
		parts[i] = item.ToString()
	}
	return dql.NewString(strings.Join(parts, sep))
}

func fnAll(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewBool(true)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewBool(args[0].Truthy())
	}
	for _, item := range items {
		if !item.Truthy() {
			return dql.NewBool(false)
		}
	}
	return dql.NewBool(true)
}

func fnAny(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewBool(false)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewBool(args[0].Truthy())
	}
	for _, item := range items {
		if item.Truthy() {
			return dql.NewBool(true)
		}
	}
	return dql.NewBool(false)
}

func fnNone(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewBool(true)
	}
	items, ok := args[0].AsList()
	if !ok {
		return dql.NewBool(!args[0].Truthy())
	}
	for _, item := range items {
		if item.Truthy() {
			return dql.NewBool(false)
		}
	}
	return dql.NewBool(true)
}

func fnNonnull(args []dql.Value, _ *EvalContext) dql.Value {
	var result []dql.Value
	for _, arg := range args {
		if items, ok := arg.AsList(); ok {
			for _, item := range items {
				if !item.IsNull() {
					result = append(result, item)
				}
			}
		} else if !arg.IsNull() {
			result = append(result, arg)
		}
	}
	return dql.NewList(result)
}

// --- Strings ---

func fnLower(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if s, ok := args[0].AsString(); ok {
		return dql.NewString(strings.ToLower(s))
	}
	return dql.NewNull()
}

func fnUpper(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if s, ok := args[0].AsString(); ok {
		return dql.NewString(strings.ToUpper(s))
	}
	return dql.NewNull()
}

func fnSplit(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewList(nil)
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewList(nil)
	}
	sep, ok := args[1].AsString()
	if !ok {
		return dql.NewList(nil)
	}
	parts := strings.Split(s, sep)
	items := make([]dql.Value, len(parts))
	for i, p := range parts {
		items[i] = dql.NewString(p)
	}
	return dql.NewList(items)
}

func fnReplace(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 3 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	old, ok := args[1].AsString()
	if !ok {
		return dql.NewNull()
	}
	new, ok := args[2].AsString()
	if !ok {
		return dql.NewNull()
	}
	return dql.NewString(strings.ReplaceAll(s, old, new))
}

func fnRegextest(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewBool(false)
	}
	pattern, ok := args[0].AsString()
	if !ok {
		return dql.NewBool(false)
	}
	s, ok := args[1].AsString()
	if !ok {
		return dql.NewBool(false)
	}
	if len(pattern) > maxRegexPatternLen {
		return dql.NewBool(false)
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return dql.NewBool(false)
	}
	return dql.NewBool(re.MatchString(s))
}

func fnRegexmatch(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	pattern, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	s, ok := args[1].AsString()
	if !ok {
		return dql.NewNull()
	}
	if len(pattern) > maxRegexPatternLen {
		return dql.NewNull()
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return dql.NewNull()
	}
	matches := re.FindStringSubmatch(s)
	if matches == nil {
		return dql.NewNull()
	}
	items := make([]dql.Value, len(matches))
	for i, m := range matches {
		items[i] = dql.NewString(m)
	}
	return dql.NewList(items)
}

func fnRegexreplace(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 3 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	pattern, ok := args[1].AsString()
	if !ok {
		return dql.NewNull()
	}
	repl, ok := args[2].AsString()
	if !ok {
		return dql.NewNull()
	}
	if len(pattern) > maxRegexPatternLen {
		return dql.NewNull()
	}
	re, err := regexp.Compile(pattern)
	if err != nil {
		return dql.NewNull()
	}
	return dql.NewString(re.ReplaceAllString(s, repl))
}

func fnStartswith(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewBool(false)
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewBool(false)
	}
	prefix, ok := args[1].AsString()
	if !ok {
		return dql.NewBool(false)
	}
	return dql.NewBool(strings.HasPrefix(s, prefix))
}

func fnEndswith(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewBool(false)
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewBool(false)
	}
	suffix, ok := args[1].AsString()
	if !ok {
		return dql.NewBool(false)
	}
	return dql.NewBool(strings.HasSuffix(s, suffix))
}

func fnSubstring(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	start := 0
	if n, ok := args[1].AsNumber(); ok {
		start = int(n)
	}
	if start < 0 {
		start = 0
	}
	if start >= len(s) {
		return dql.NewString("")
	}
	if len(args) > 2 {
		if end, ok := args[2].AsNumber(); ok {
			e := int(end)
			if e > len(s) {
				e = len(s)
			}
			if e < start {
				return dql.NewString("")
			}
			return dql.NewString(s[start:e])
		}
	}
	return dql.NewString(s[start:])
}

func fnTruncate(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	s, ok := args[0].AsString()
	if !ok {
		return dql.NewNull()
	}
	maxLen, ok := args[1].AsNumber()
	if !ok {
		return dql.NewNull()
	}
	n := int(maxLen)
	if n >= len(s) {
		return dql.NewString(s)
	}
	suffix := "..."
	if len(args) > 2 {
		if sf, ok := args[2].AsString(); ok {
			suffix = sf
		}
	}
	if n <= len(suffix) {
		return dql.NewString(s[:n])
	}
	return dql.NewString(s[:n-len(suffix)] + suffix)
}

func fnPadleft(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	s := args[0].ToString()
	width, ok := args[1].AsNumber()
	if !ok {
		return dql.NewNull()
	}
	pad := " "
	if len(args) > 2 {
		if p, ok := args[2].AsString(); ok && p != "" {
			pad = p
		}
	}
	w := int(width)
	if w > maxPadWidth {
		w = maxPadWidth
	}
	for len(s) < w {
		s = pad + s
	}
	return dql.NewString(s)
}

func fnPadright(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	s := args[0].ToString()
	width, ok := args[1].AsNumber()
	if !ok {
		return dql.NewNull()
	}
	pad := " "
	if len(args) > 2 {
		if p, ok := args[2].AsString(); ok && p != "" {
			pad = p
		}
	}
	w := int(width)
	if w > maxPadWidth {
		w = maxPadWidth
	}
	for len(s) < w {
		s = s + pad
	}
	return dql.NewString(s)
}

// --- Utility ---

func fnDefault(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if !args[0].IsNull() && args[0].Truthy() {
		return args[0]
	}
	if len(args) > 1 {
		return args[1]
	}
	return dql.NewNull()
}

func fnChoice(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 3 {
		return dql.NewNull()
	}
	if args[0].Truthy() {
		return args[1]
	}
	return args[2]
}

func fnDateformat(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	d, ok := args[0].AsDate()
	if !ok {
		return dql.NewNull()
	}
	format, ok := args[1].AsString()
	if !ok {
		return dql.NewNull()
	}
	// Convert Dataview/Luxon format to Go format
	goFormat := convertDateFormat(format)
	return dql.NewString(d.Format(goFormat))
}

func fnDurationformat(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if d, ok := args[0].AsDuration(); ok {
		return dql.NewString(d.String())
	}
	return dql.NewNull()
}

func fnStriptime(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	if d, ok := args[0].AsDate(); ok {
		stripped := time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, d.Location())
		return dql.NewDate(stripped)
	}
	return dql.NewNull()
}

func fnMeta(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) == 0 {
		return dql.NewNull()
	}
	// meta() returns metadata about a link
	if target, ok := args[0].AsLink(); ok {
		return dql.NewObject(map[string]dql.Value{
			"path":    dql.NewString(target),
			"display": dql.NewString(target),
		})
	}
	return dql.NewNull()
}

func fnCurrencyformat(args []dql.Value, _ *EvalContext) dql.Value {
	if len(args) < 2 {
		return dql.NewNull()
	}
	n, ok := args[0].AsNumber()
	if !ok {
		return dql.NewNull()
	}
	currency, ok := args[1].AsString()
	if !ok {
		return dql.NewNull()
	}
	return dql.NewString(fmt.Sprintf("%s%.2f", currency, n))
}

// --- Helpers ---

func valueContainsCheck(haystack, needle dql.Value) bool {
	if items, ok := haystack.AsList(); ok {
		// Fuzzy: for string elements, check substring; for others, exact match
		ns, needleIsStr := needle.AsString()
		for _, item := range items {
			if needleIsStr {
				if is, ok := item.AsString(); ok {
					if strings.Contains(strings.ToLower(is), strings.ToLower(ns)) {
						return true
					}
					continue
				}
			}
			if item.Compare(needle) == 0 {
				return true
			}
		}
		return false
	}
	if hs, ok := haystack.AsString(); ok {
		if ns, ok := needle.AsString(); ok {
			return strings.Contains(strings.ToLower(hs), strings.ToLower(ns))
		}
	}
	return haystack.Compare(needle) == 0
}

// valueExactContains checks exact element membership (no substring matching).
func valueExactContains(haystack, needle dql.Value) bool {
	if items, ok := haystack.AsList(); ok {
		for _, item := range items {
			if item.Compare(needle) == 0 {
				return true
			}
		}
		return false
	}
	if hs, ok := haystack.AsString(); ok {
		if ns, ok := needle.AsString(); ok {
			return strings.Contains(hs, ns)
		}
	}
	return haystack.Compare(needle) == 0
}

func flattenArgs(args []dql.Value) []dql.Value {
	if len(args) == 1 {
		if items, ok := args[0].AsList(); ok {
			return items
		}
	}
	return args
}

func aggregateList(args []dql.Value, cmp func(a, b dql.Value) dql.Value) dql.Value {
	items := flattenArgs(args)
	if len(items) == 0 {
		return dql.NewNull()
	}
	result := items[0]
	for _, item := range items[1:] {
		result = cmp(result, item)
	}
	return result
}

// convertDateFormat converts common Luxon/Dataview date format tokens to Go layout.
func convertDateFormat(format string) string {
	replacer := strings.NewReplacer(
		"yyyy", "2006",
		"YYYY", "2006",
		"yy", "06",
		"MM", "01",
		"dd", "02",
		"HH", "15",
		"hh", "03",
		"mm", "04",
		"ss", "05",
		"EEEE", "Monday",
		"EEE", "Mon",
		"MMMM", "January",
		"MMM", "Jan",
	)
	return replacer.Replace(format)
}
