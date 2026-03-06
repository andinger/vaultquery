package indexer

import (
	"testing"
)

func TestExtractTags(t *testing.T) {
	data := []byte(`---
tags:
  - frontmatter-tag
---
# Title

Some text with #inline-tag and #another.

` + "```" + `
#not-a-tag in code block
` + "```" + `

More text #final-tag.
`)
	tags := ExtractTags(data, []string{"frontmatter-tag"})

	want := map[string]bool{
		"frontmatter-tag": true,
		"inline-tag":      true,
		"another":         true,
		"final-tag":       true,
	}

	if len(tags) != len(want) {
		t.Fatalf("expected %d tags, got %d: %v", len(want), len(tags), tags)
	}
	for _, tag := range tags {
		if !want[tag] {
			t.Errorf("unexpected tag: %q", tag)
		}
	}
}

func TestExtractTagsNested(t *testing.T) {
	data := []byte("Text with #parent/child tag")
	tags := ExtractTags(data, nil)
	if len(tags) != 1 || tags[0] != "parent/child" {
		t.Errorf("expected [parent/child], got %v", tags)
	}
}

func TestExtractTagsDedup(t *testing.T) {
	data := []byte("#tag #tag #TAG")
	tags := ExtractTags(data, nil)
	if len(tags) != 1 {
		t.Errorf("expected 1 tag (dedup), got %d: %v", len(tags), tags)
	}
}

func TestExtractTagsNoFalsePositives(t *testing.T) {
	data := []byte("# Heading\nno#tag here\na#b")
	tags := ExtractTags(data, nil)
	if len(tags) != 0 {
		t.Errorf("expected 0 tags, got %v", tags)
	}
}

func TestExtractLinks(t *testing.T) {
	data := []byte(`---
title: Test
---
# Test

Link to [[Page One]] and [[Page Two|display text]].
Also [[Page Three#heading]] reference.

` + "```" + `
[[not-a-link]]
` + "```" + `
`)
	links := ExtractLinks(data)

	want := map[string]bool{
		"Page One":   true,
		"Page Two":   true,
		"Page Three": true,
	}

	if len(links) != len(want) {
		t.Fatalf("expected %d links, got %d: %v", len(want), len(links), links)
	}
	for _, link := range links {
		if !want[link] {
			t.Errorf("unexpected link: %q", link)
		}
	}
}

func TestExtractLinksDedup(t *testing.T) {
	data := []byte("[[Page]] and [[Page]] again")
	links := ExtractLinks(data)
	if len(links) != 1 {
		t.Errorf("expected 1 link (dedup), got %d: %v", len(links), links)
	}
}

func TestExtractTasks(t *testing.T) {
	data := []byte(`---
title: Test
---
# Section One

- [ ] Uncompleted task
- [x] Completed task
- Regular list item

## Section Two

  - [ ] Indented task
  - [X] Also completed
`)
	tasks := ExtractTasks(data)

	if len(tasks) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(tasks))
	}

	// Task 1
	if tasks[0].Text != "Uncompleted task" || tasks[0].Completed || tasks[0].Section != "Section One" {
		t.Errorf("task 0: %+v", tasks[0])
	}

	// Task 2
	if tasks[1].Text != "Completed task" || !tasks[1].Completed || tasks[1].Section != "Section One" {
		t.Errorf("task 1: %+v", tasks[1])
	}

	// Task 3
	if tasks[2].Text != "Indented task" || tasks[2].Completed || tasks[2].Section != "Section Two" {
		t.Errorf("task 2: %+v", tasks[2])
	}

	// Task 4
	if tasks[3].Text != "Also completed" || !tasks[3].Completed || tasks[3].Section != "Section Two" {
		t.Errorf("task 3: %+v", tasks[3])
	}
}

func TestExtractTasksInCodeBlock(t *testing.T) {
	data := []byte("```\n- [ ] not a task\n```\n- [ ] real task")
	tasks := ExtractTasks(data)
	if len(tasks) != 1 || tasks[0].Text != "real task" {
		t.Errorf("expected 1 task 'real task', got %v", tasks)
	}
}

func TestExtractTasksNoFrontmatter(t *testing.T) {
	data := []byte("- [ ] task one\n- [x] task two")
	tasks := ExtractTasks(data)
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(tasks))
	}
}
