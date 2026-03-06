package indexer

import (
	"strings"

	"github.com/andinger/vaultquery/internal/index"
)

// ExtractTags extracts inline #tags from markdown content (excluding code blocks)
// and combines them with frontmatter tags.
func ExtractTags(data []byte, frontmatterTags []string) []string {
	seen := make(map[string]bool)
	var tags []string

	// Add frontmatter tags first
	for _, t := range frontmatterTags {
		lower := strings.ToLower(t)
		if !seen[lower] {
			seen[lower] = true
			tags = append(tags, t)
		}
	}

	// Scan content for inline #tags
	content := skipFrontmatter(data)
	lines := strings.Split(string(content), "\n")
	inCodeBlock := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Find #tag patterns
		for i := 0; i < len(line); i++ {
			if line[i] != '#' {
				continue
			}
			// # must be at start or preceded by whitespace
			if i > 0 && !isWhitespace(line[i-1]) {
				continue
			}
			// Must be followed by a letter
			if i+1 >= len(line) || !isTagChar(line[i+1]) {
				continue
			}
			// Read the tag
			start := i + 1
			j := start
			for j < len(line) && isTagChar(line[j]) {
				j++
			}
			// Allow nested tags with /
			for j < len(line) && line[j] == '/' {
				j++
				for j < len(line) && isTagChar(line[j]) {
					j++
				}
			}
			tag := line[start:j]
			lower := strings.ToLower(tag)
			if !seen[lower] {
				seen[lower] = true
				tags = append(tags, tag)
			}
			i = j - 1
		}
	}

	return tags
}

// ExtractLinks extracts [[wikilink]] targets from markdown content.
func ExtractLinks(data []byte) []string {
	content := skipFrontmatter(data)
	s := string(content)
	seen := make(map[string]bool)
	var links []string
	inCodeBlock := false

	lines := strings.Split(s, "\n")
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		for i := 0; i < len(line)-1; i++ {
			if line[i] == '[' && line[i+1] == '[' {
				end := strings.Index(line[i+2:], "]]")
				if end < 0 {
					break
				}
				target := line[i+2 : i+2+end]
				// Handle [[target|display]] and [[target#heading]]
				if pipe := strings.IndexByte(target, '|'); pipe >= 0 {
					target = target[:pipe]
				}
				if hash := strings.IndexByte(target, '#'); hash >= 0 {
					target = target[:hash]
				}
				target = strings.TrimSpace(target)
				if target != "" && !seen[target] {
					seen[target] = true
					links = append(links, target)
				}
				i = i + 2 + end + 1
			}
		}
	}

	return links
}

// ExtractTasks extracts task items (- [ ] text / - [x] text) from markdown content.
func ExtractTasks(data []byte) []index.TaskInfo {
	content := skipFrontmatter(data)
	lines := strings.Split(string(content), "\n")
	var tasks []index.TaskInfo
	currentSection := ""
	inCodeBlock := false

	for lineNum, line := range lines {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") {
			inCodeBlock = !inCodeBlock
			continue
		}
		if inCodeBlock {
			continue
		}

		// Track heading sections
		if strings.HasPrefix(trimmed, "#") {
			// Strip # prefix
			h := strings.TrimLeft(trimmed, "#")
			currentSection = strings.TrimSpace(h)
			continue
		}

		// Check for task items: "- [ ] text" or "- [x] text" (with optional indentation)
		stripped := strings.TrimLeft(line, " \t")
		if len(stripped) < 6 {
			continue
		}

		if strings.HasPrefix(stripped, "- [ ] ") || strings.HasPrefix(stripped, "- [x] ") ||
			strings.HasPrefix(stripped, "- [X] ") {
			completed := stripped[3] == 'x' || stripped[3] == 'X'
			text := strings.TrimSpace(stripped[6:])
			tasks = append(tasks, index.TaskInfo{
				Line:      lineNum + 1, // 1-based
				Text:      text,
				Completed: completed,
				Section:   currentSection,
			})
		}
	}

	return tasks
}

func skipFrontmatter(data []byte) []byte {
	s := string(data)
	if !strings.HasPrefix(s, "---\n") {
		return data
	}
	end := strings.Index(s[4:], "\n---\n")
	if end < 0 {
		return data
	}
	return []byte(s[4+end+5:])
}

func isWhitespace(ch byte) bool {
	return ch == ' ' || ch == '\t' || ch == '\n' || ch == '\r'
}

func isTagChar(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') ||
		(ch >= '0' && ch <= '9') || ch == '_' || ch == '-'
}
