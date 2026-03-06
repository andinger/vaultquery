package indexer

import (
	"bytes"
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// ParseFrontmatter extracts YAML frontmatter fields and the first # heading title.
func ParseFrontmatter(data []byte) (fields map[string][]string, title string, err error) {
	fields = make(map[string][]string)

	content := data
	if bytes.HasPrefix(data, []byte("---\n")) {
		end := bytes.Index(data[4:], []byte("\n---\n"))
		if end >= 0 {
			yamlBlock := data[4 : 4+end]
			content = data[4+end+5:]

			var raw map[string]any
			if err := yaml.Unmarshal(yamlBlock, &raw); err != nil {
				return fields, "", err
			}

			for k, v := range raw {
				converted := convertValue(v)
				if converted != nil {
					fields[k] = converted
				}
			}
		}
	}

	title = extractTitle(content)
	return fields, title, nil
}

func convertValue(v any) []string {
	switch val := v.(type) {
	case string:
		return []string{val}
	case int:
		return []string{fmt.Sprint(val)}
	case float64:
		return []string{fmt.Sprint(val)}
	case bool:
		return []string{fmt.Sprint(val)}
	case []any:
		var result []string
		for _, elem := range val {
			switch e := elem.(type) {
			case string:
				result = append(result, e)
			case int:
				result = append(result, fmt.Sprint(e))
			case float64:
				result = append(result, fmt.Sprint(e))
			case bool:
				result = append(result, fmt.Sprint(e))
			default:
				// Skip nested maps/complex types in arrays
			}
		}
		return result
	case map[string]any:
		// Skip nested maps
		return nil
	default:
		return []string{fmt.Sprint(val)}
	}
}

func extractTitle(content []byte) string {
	for _, line := range strings.Split(string(content), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "# ") {
			return strings.TrimSpace(trimmed[2:])
		}
	}
	return ""
}
