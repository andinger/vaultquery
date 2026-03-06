package executor

// Result holds the output of a DQL query execution.
type Result struct {
	Mode    string           `json:"mode"`
	Fields  []string         `json:"fields,omitempty"`
	Results []map[string]any `json:"results"`
}
