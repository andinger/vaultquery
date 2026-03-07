package dql

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"2024-01-15", "2024-01-15", true},
		{"2024-01-15T10:30:00", "2024-01-15", true},
		{"2024/01/15", "2024-01-15", true},
		{"not a date", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, ok := ParseDate(tt.input)
			if ok != tt.ok {
				t.Fatalf("ParseDate(%q) ok=%v, want %v", tt.input, ok, tt.ok)
			}
			if ok && d.Format("2006-01-02") != tt.want {
				t.Errorf("ParseDate(%q) = %s, want %s", tt.input, d.Format("2006-01-02"), tt.want)
			}
		})
	}
}

func TestParseDateFromFilename(t *testing.T) {
	tests := []struct {
		input string
		want  string
		ok    bool
	}{
		{"2024-01-15.md", "2024-01-15", true},
		{"2024-01-15 Daily Note.md", "2024-01-15", true},
		{"random-file.md", "", false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, ok := ParseDateFromFilename(tt.input)
			if ok != tt.ok {
				t.Fatalf("ok=%v, want %v", ok, tt.ok)
			}
			if ok && d.Format("2006-01-02") != tt.want {
				t.Errorf("got %s, want %s", d.Format("2006-01-02"), tt.want)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input string
		want  time.Duration
		ok    bool
	}{
		{"1 day", 24 * time.Hour, true},
		{"2 days", 48 * time.Hour, true},
		{"1 hour", time.Hour, true},
		{"2h 30m", 2*time.Hour + 30*time.Minute, true},
		{"1 week", 7 * 24 * time.Hour, true},
		{"1 year", 365 * 24 * time.Hour, true},
		{"1 month", 30 * 24 * time.Hour, true},
		{"30s", 30 * time.Second, true},
		{"5m", 5 * time.Minute, true},
		{"100 years", 100 * 365 * 24 * time.Hour, true},
		{"999999999999 years", 0, false},
		{"", 0, false},
		{"abc", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			d, ok := ParseDuration(tt.input)
			if ok != tt.ok {
				t.Fatalf("ParseDuration(%q) ok=%v, want %v", tt.input, ok, tt.ok)
			}
			if ok && d != tt.want {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, d, tt.want)
			}
		})
	}
}
