package dql

import (
	"regexp"
	"strconv"
	"strings"
	"time"
)

// ParseDate attempts to parse a date string in various formats.
func ParseDate(s string) (time.Time, bool) {
	layouts := []string{
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05",
		"2006-01-02T15:04",
		"2006-01-02",
		"January 2, 2006",
		"Jan 2, 2006",
		"2006/01/02",
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// ParseDateFromFilename extracts a date from a filename like "2024-01-15.md" or "2024-01-15 Note.md".
func ParseDateFromFilename(filename string) (time.Time, bool) {
	re := regexp.MustCompile(`(\d{4}-\d{2}-\d{2})`)
	match := re.FindString(filename)
	if match == "" {
		return time.Time{}, false
	}
	return ParseDate(match)
}

var durPattern = regexp.MustCompile(`(?i)(\d+)\s*(years?|yrs?|months?|mos?|weeks?|wks?|w|days?|d|hours?|hrs?|h|minutes?|mins?|m|seconds?|secs?|s)`)

// ParseDuration parses a Dataview-style duration string like "1 day", "2h 30m", "1 year 2 months".
func ParseDuration(s string) (time.Duration, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0, false
	}

	// Try Go standard format first
	if d, err := time.ParseDuration(s); err == nil {
		return d, true
	}

	matches := durPattern.FindAllStringSubmatch(s, -1)
	if len(matches) == 0 {
		return 0, false
	}

	var total time.Duration
	for _, m := range matches {
		n, err := strconv.Atoi(m[1])
		if err != nil || n > 1_000_000 {
			continue
		}
		unit := strings.ToLower(m[2])
		switch {
		case strings.HasPrefix(unit, "year"), strings.HasPrefix(unit, "yr"):
			total += time.Duration(n) * 365 * 24 * time.Hour
		case strings.HasPrefix(unit, "month"), strings.HasPrefix(unit, "mo"):
			total += time.Duration(n) * 30 * 24 * time.Hour
		case strings.HasPrefix(unit, "week"), strings.HasPrefix(unit, "wk"), unit == "w":
			total += time.Duration(n) * 7 * 24 * time.Hour
		case strings.HasPrefix(unit, "day"), unit == "d":
			total += time.Duration(n) * 24 * time.Hour
		case strings.HasPrefix(unit, "hour"), strings.HasPrefix(unit, "hr"), unit == "h":
			total += time.Duration(n) * time.Hour
		case strings.HasPrefix(unit, "minute"), strings.HasPrefix(unit, "min"), unit == "m":
			total += time.Duration(n) * time.Minute
		case strings.HasPrefix(unit, "second"), strings.HasPrefix(unit, "sec"), unit == "s":
			total += time.Duration(n) * time.Second
		}
	}

	if total == 0 {
		return 0, false
	}
	return total, true
}
