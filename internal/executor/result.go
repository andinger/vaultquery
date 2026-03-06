package executor

// Result holds the output of a DQL query execution.
type Result struct {
	Mode    string           `json:"mode"`
	Fields  []string         `json:"fields,omitempty"`
	Results []map[string]any `json:"results"`
}

// GroupedResult represents a single group in GROUP BY output.
type GroupedResult struct {
	Key  any              `json:"key"`
	Rows []map[string]any `json:"rows"`
}

// GroupedQueryResult holds results after GROUP BY.
type GroupedQueryResult struct {
	Mode   string          `json:"mode"`
	Fields []string        `json:"fields,omitempty"`
	Groups []GroupedResult `json:"groups"`
}
