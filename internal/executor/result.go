package executor

// Result holds the output of a DQL query execution.
type Result struct {
	Mode    string           `json:"mode"              toon:"mode"`
	Fields  []string         `json:"fields,omitempty"  toon:"fields"`
	Results []map[string]any `json:"results"           toon:"results"`
}

// GroupedResult represents a single group in GROUP BY output.
type GroupedResult struct {
	Key  any              `json:"key"  toon:"key"`
	Rows []map[string]any `json:"rows" toon:"rows"`
}

// GroupedQueryResult holds results after GROUP BY.
type GroupedQueryResult struct {
	Mode   string          `json:"mode"             toon:"mode"`
	Fields []string        `json:"fields,omitempty" toon:"fields"`
	Groups []GroupedResult `json:"groups"           toon:"groups"`
}
