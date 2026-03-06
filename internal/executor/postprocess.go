package executor

import (
	"github.com/andinger/vaultquery/internal/dql"
	"github.com/andinger/vaultquery/internal/eval"
)

// applyFlatten expands rows where the FLATTEN expression evaluates to a list.
// Each list element produces a duplicate row with the flattened value.
func applyFlatten(rows []map[string]any, flattenDefs []dql.FieldDef, ev *eval.Evaluator) []map[string]any {
	if len(flattenDefs) == 0 {
		return rows
	}

	for _, fd := range flattenDefs {
		fieldName := dql.FieldDefName(fd)
		alias := fd.Alias
		if alias == "" {
			alias = fieldName
		}

		var expanded []map[string]any
		for _, row := range rows {
			// Get the value to flatten
			rawVal, exists := row[fieldName]
			if !exists {
				expanded = append(expanded, row)
				continue
			}

			// Convert to dql.Value to check if it's a list
			val := anyToValue(rawVal)
			items, isList := val.AsList()
			if !isList || len(items) == 0 {
				expanded = append(expanded, row)
				continue
			}

			// Create one row per list element
			for _, item := range items {
				newRow := copyRow(row)
				newRow[alias] = valueToAny(item)
				expanded = append(expanded, newRow)
			}
		}
		rows = expanded
	}

	return rows
}

// applyGroupBy groups rows by the GROUP BY expression values.
func applyGroupBy(rows []map[string]any, groupByDefs []dql.FieldDef, ev *eval.Evaluator) []GroupedResult {
	if len(groupByDefs) == 0 {
		return nil
	}

	// Group by the first GROUP BY expression (Dataview typically uses single GROUP BY)
	fd := groupByDefs[0]
	fieldName := dql.FieldDefName(fd)

	groups := make(map[string]*GroupedResult)
	var order []string

	for _, row := range rows {
		rawVal, exists := row[fieldName]
		var key string
		if exists && rawVal != nil {
			key = anyToValue(rawVal).ToString()
		} else {
			key = "-"
		}

		if _, ok := groups[key]; !ok {
			groups[key] = &GroupedResult{Key: rawVal}
			order = append(order, key)
		}
		groups[key].Rows = append(groups[key].Rows, row)
	}

	result := make([]GroupedResult, len(order))
	for i, key := range order {
		result[i] = *groups[key]
	}
	return result
}

func copyRow(row map[string]any) map[string]any {
	newRow := make(map[string]any, len(row))
	for k, v := range row {
		newRow[k] = v
	}
	return newRow
}

func anyToValue(v any) dql.Value {
	if v == nil {
		return dql.NewNull()
	}
	switch val := v.(type) {
	case string:
		return dql.NewString(val)
	case float64:
		return dql.NewNumber(val)
	case int:
		return dql.NewNumber(float64(val))
	case bool:
		return dql.NewBool(val)
	case []string:
		items := make([]dql.Value, len(val))
		for i, s := range val {
			items[i] = dql.NewString(s)
		}
		return dql.NewList(items)
	case dql.Value:
		return val
	default:
		return dql.NewString(dql.NewNull().ToString())
	}
}

func valueToAny(v dql.Value) any {
	switch v.Type {
	case dql.TypeNull:
		return nil
	case dql.TypeNumber:
		return v.Inner.(float64)
	case dql.TypeString:
		return v.Inner.(string)
	case dql.TypeBool:
		return v.Inner.(bool)
	default:
		return v.ToString()
	}
}
