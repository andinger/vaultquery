package indexer

import (
	"reflect"
	"sort"
	"testing"
)

func TestFrontmatterWithFields(t *testing.T) {
	input := []byte(`---
type: Kubernetes Cluster
customer: Acme Corp
replicas: 3
enabled: true
tags:
  - linux
  - production
---
# Acme Production Cluster
Some body text.
`)
	fields, title, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Acme Production Cluster" {
		t.Errorf("title = %q, want %q", title, "Acme Production Cluster")
	}
	assertField(t, fields, "type", []string{"Kubernetes Cluster"})
	assertField(t, fields, "customer", []string{"Acme Corp"})
	assertField(t, fields, "replicas", []string{"3"})
	assertField(t, fields, "enabled", []string{"true"})
	assertField(t, fields, "tags", []string{"linux", "production"})
}

func TestNoFrontmatter(t *testing.T) {
	input := []byte("# Just a Title\nSome content.\n")
	fields, title, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Errorf("expected empty fields, got %v", fields)
	}
	if title != "Just a Title" {
		t.Errorf("title = %q, want %q", title, "Just a Title")
	}
}

func TestEmptyFrontmatter(t *testing.T) {
	input := []byte("---\n---\n# Title After Empty\n")
	fields, title, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	if len(fields) != 0 {
		t.Errorf("expected empty fields, got %v", fields)
	}
	if title != "Title After Empty" {
		t.Errorf("title = %q, want %q", title, "Title After Empty")
	}
}

func TestTitleFromHeading(t *testing.T) {
	input := []byte(`---
type: note
---
Some preamble text.
# The Real Title
More text.
`)
	_, title, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	if title != "The Real Title" {
		t.Errorf("title = %q, want %q", title, "The Real Title")
	}
}

func TestNoHeading(t *testing.T) {
	input := []byte(`---
type: note
---
Just body text, no heading.
`)
	_, title, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	if title != "" {
		t.Errorf("title = %q, want empty", title)
	}
}

func TestNestedMapSkipped(t *testing.T) {
	input := []byte(`---
simple: value
nested:
  key1: val1
  key2: val2
---
`)
	fields, _, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := fields["nested"]; ok {
		t.Error("nested map should be skipped")
	}
	assertField(t, fields, "simple", []string{"value"})
}

func TestArrayOfMixedTypes(t *testing.T) {
	input := []byte(`---
mixed:
  - hello
  - 42
  - true
---
`)
	fields, _, err := ParseFrontmatter(input)
	if err != nil {
		t.Fatal(err)
	}
	got := fields["mixed"]
	sort.Strings(got)
	want := []string{"42", "hello", "true"}
	sort.Strings(want)
	if !reflect.DeepEqual(got, want) {
		t.Errorf("mixed = %v, want %v", got, want)
	}
}

func assertField(t *testing.T, fields map[string][]string, key string, want []string) {
	t.Helper()
	got, ok := fields[key]
	if !ok {
		t.Errorf("field %q not found", key)
		return
	}
	sortedGot := make([]string, len(got))
	copy(sortedGot, got)
	sort.Strings(sortedGot)
	sortedWant := make([]string, len(want))
	copy(sortedWant, want)
	sort.Strings(sortedWant)
	if !reflect.DeepEqual(sortedGot, sortedWant) {
		t.Errorf("field %q = %v, want %v", key, got, want)
	}
}
