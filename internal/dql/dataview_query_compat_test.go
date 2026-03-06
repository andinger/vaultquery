package dql

// Tests translated from the Dataview repo:
// https://github.com/blacksmithgu/obsidian-dataview/blob/master/src/test/parse/parse.query.test.ts

import (
	"testing"
)

// --- Query Type Parsing ---

func TestDV_QueryTypeAlone(t *testing.T) {
	// Dataview: testQueryTypeAlone for each query type
	types := []struct {
		input string
		mode  string
	}{
		{"TABLE", "TABLE"},
		{"table", "TABLE"},
		{"LIST", "LIST"},
		{"list", "LIST"},
		{"TASK", "TASK"},
		{"task", "TASK"},
		{"CALENDAR", "CALENDAR"},
		{"calendar", "CALENDAR"},
	}
	for _, tt := range types {
		t.Run(tt.input, func(t *testing.T) {
			q, err := Parse(tt.input)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if q.Mode != tt.mode {
				t.Errorf("expected mode %s, got %s", tt.mode, q.Mode)
			}
		})
	}
}

func TestDV_QueryTypeWithWhitespace(t *testing.T) {
	// Dataview tests query types with trailing whitespace/newlines
	suffixes := []string{"", " ", "\n", " \n", "\n "}
	for _, mode := range []string{"TABLE", "LIST", "TASK", "CALENDAR"} {
		for _, suffix := range suffixes {
			q, err := Parse(mode + suffix)
			if err != nil {
				t.Fatalf("Parse(%q): unexpected error: %v", mode+suffix, err)
			}
			if q.Mode != mode {
				t.Errorf("Parse(%q): expected mode %s, got %s", mode+suffix, mode, q.Mode)
			}
		}
	}
}

func TestDV_InvalidQueryType(t *testing.T) {
	_, err := Parse("vehicle")
	if err == nil {
		t.Fatal("expected error for invalid query type 'vehicle'")
	}
}

// --- FROM Sources ---

func TestDV_LinkSource(t *testing.T) {
	// list from [[Stuff]]
	q, err := Parse("list from [[Stuff]]")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "LIST" {
		t.Errorf("expected LIST, got %s", q.Mode)
	}
	ls, ok := q.FromSource.(LinkSource)
	if !ok {
		t.Fatalf("expected LinkSource, got %T", q.FromSource)
	}
	if ls.Target != "Stuff" {
		t.Errorf("expected target 'Stuff', got %q", ls.Target)
	}
	if ls.Outgoing {
		t.Error("expected incoming link (Outgoing=false)")
	}
}

func TestDV_TagSource(t *testing.T) {
	// task from #games
	q, err := Parse("task from #games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TASK" {
		t.Errorf("expected TASK, got %s", q.Mode)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "games" {
		t.Errorf("expected tag 'games', got %q", ts.Tag)
	}
}

func TestDV_FromTagOr(t *testing.T) {
	// FROM #games or #gaming → BooleanFromSource OR
	q, err := Parse("LIST FROM #games OR #gaming")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bs, ok := q.FromSource.(BooleanFromSource)
	if !ok {
		t.Fatalf("expected BooleanFromSource, got %T", q.FromSource)
	}
	if bs.Op != "OR" {
		t.Errorf("expected OR, got %s", bs.Op)
	}
	left, ok := bs.Left.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource on left, got %T", bs.Left)
	}
	if left.Tag != "games" {
		t.Errorf("expected 'games', got %q", left.Tag)
	}
	right, ok := bs.Right.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource on right, got %T", bs.Right)
	}
	if right.Tag != "gaming" {
		t.Errorf("expected 'gaming', got %q", right.Tag)
	}
}

// --- Comments ---

func TestDV_Comments(t *testing.T) {
	// Comments at beginning, middle, end, and inline
	q, err := Parse(
		"// This is a comment at the beginning\n" +
			"TABLE customer, status\n" +
			"// This is a comment\n" +
			"// This is a second comment\n" +
			"FROM #clients\n" +
			"// This is a third comment\n" +
			"WHERE status = 'active'\n" +
			"// This is a fourth comment\n" +
			"\n" +
			"   // This is a comment with whitespace prior\n" +
			"SORT customer ASC\n" +
			"// This is a comment at the end")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" {
		t.Errorf("expected TABLE, got %s", q.Mode)
	}
	names := FieldDefNames(q.Fields)
	if len(names) != 2 {
		t.Fatalf("expected 2 fields, got %d", len(names))
	}
	if names[0] != "customer" || names[1] != "status" {
		t.Errorf("unexpected fields: %v", names)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "clients" {
		t.Errorf("expected tag 'clients', got %q", ts.Tag)
	}
}

// --- Named Fields ---

func TestDV_SimpleNamedField(t *testing.T) {
	// TABLE time-played FROM #games
	q, err := Parse("TABLE time-played FROM #games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	names := FieldDefNames(q.Fields)
	if len(names) != 1 || names[0] != "time-played" {
		t.Errorf("expected [time-played], got %v", names)
	}
}

func TestDV_FieldWithAlias(t *testing.T) {
	// TABLE rating AS rate
	q, err := Parse("TABLE rating AS rate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Fields) != 1 {
		t.Fatalf("expected 1 field, got %d", len(q.Fields))
	}
	if q.Fields[0].Alias != "rate" {
		t.Errorf("expected alias 'rate', got %q", q.Fields[0].Alias)
	}
	names := FieldDefNames(q.Fields)
	if names[0] != "rate" {
		t.Errorf("expected display name 'rate', got %q", names[0])
	}
}

// --- Sort Fields ---

func TestDV_SortFieldDescending(t *testing.T) {
	// SORT time-played DESC — tests both hyphenated ident and DESC
	q, err := Parse("LIST SORT time-played DESC")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Sort) != 1 {
		t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
	}
	if q.Sort[0].Field != "time-played" {
		t.Errorf("expected field 'time-played', got %q", q.Sort[0].Field)
	}
	if !q.Sort[0].Desc {
		t.Error("expected Desc=true")
	}
}

func TestDV_SortFieldDescendingFull(t *testing.T) {
	// SORT length DESCENDING — full keyword
	q, err := Parse("LIST SORT length DESCENDING")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(q.Sort) != 1 {
		t.Fatalf("expected 1 sort field, got %d", len(q.Sort))
	}
	if !q.Sort[0].Desc {
		t.Error("expected Desc=true for DESCENDING")
	}
}

func TestDV_SortFieldAscendingFull(t *testing.T) {
	q, err := Parse("LIST SORT name ASCENDING")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Sort[0].Desc {
		t.Error("expected Desc=false for ASCENDING")
	}
}

// --- Task Queries ---

func TestDV_TaskFromTag(t *testing.T) {
	q, err := Parse("task from #games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TASK" {
		t.Errorf("expected TASK, got %s", q.Mode)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "games" {
		t.Errorf("expected 'games', got %q", ts.Tag)
	}
}

// --- List Queries ---

func TestDV_ListWithoutID(t *testing.T) {
	// LIST WITHOUT ID ... FROM #games
	// Note: Dataview supports LIST WITHOUT ID file.name FROM #games
	// where file.name is a format expression. We parse WITHOUT ID but
	// don't yet parse the format expression after it.
	q, err := Parse("LIST WITHOUT ID FROM #games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "LIST" {
		t.Errorf("expected LIST, got %s", q.Mode)
	}
	if !q.WithoutID {
		t.Error("expected WithoutID=true")
	}
}

// --- Table Queries ---

func TestDV_TableMinimal(t *testing.T) {
	// TABLE time-played, rating, length FROM #games
	q, err := Parse("TABLE time-played, rating, length FROM #games")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" {
		t.Errorf("expected TABLE, got %s", q.Mode)
	}
	names := FieldDefNames(q.Fields)
	if len(names) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(names))
	}
	expected := []string{"time-played", "rating", "length"}
	for i, e := range expected {
		if names[i] != e {
			t.Errorf("field %d: expected %q, got %q", i, e, names[i])
		}
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "games" {
		t.Errorf("expected 'games', got %q", ts.Tag)
	}
}

func TestDV_TableWithoutID(t *testing.T) {
	q, err := Parse("TABLE WITHOUT ID name, value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "TABLE" {
		t.Errorf("expected TABLE, got %s", q.Mode)
	}
	if !q.WithoutID {
		t.Error("expected WithoutID=true")
	}
	names := FieldDefNames(q.Fields)
	if len(names) != 2 || names[0] != "name" || names[1] != "value" {
		t.Errorf("expected [name, value], got %v", names)
	}
}

func TestDV_TableWithoutIDWeirdSpacing(t *testing.T) {
	q, err := Parse("TABLE    WITHOUT     ID   name, value")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !q.WithoutID {
		t.Error("expected WithoutID=true")
	}
	names := FieldDefNames(q.Fields)
	if len(names) != 2 || names[0] != "name" || names[1] != "value" {
		t.Errorf("expected [name, value], got %v", names)
	}
}

// --- Calendar Queries ---

func TestDV_CalendarMinimal(t *testing.T) {
	// CALENDAR my-date FROM #games WHERE foo > 100
	q, err := Parse("CALENDAR FROM #games WHERE foo > 100")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if q.Mode != "CALENDAR" {
		t.Errorf("expected CALENDAR, got %s", q.Mode)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "games" {
		t.Errorf("expected tag 'games', got %q", ts.Tag)
	}
}

// --- FROM source parsing (from parse.expression.test.ts) ---

func TestDV_FromSimpleSources(t *testing.T) {
	// FROM "hello" → FolderSource
	q, err := Parse(`LIST FROM "hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fs, ok := q.FromSource.(FolderSource)
	if !ok {
		t.Fatalf("expected FolderSource, got %T", q.FromSource)
	}
	if fs.Path != "hello" {
		t.Errorf("expected 'hello', got %q", fs.Path)
	}

	// FROM #neat → TagSource
	q, err = Parse("LIST FROM #neat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "neat" {
		t.Errorf("expected 'neat', got %q", ts.Tag)
	}
}

func TestDV_FromNegatedWithMinus(t *testing.T) {
	// FROM -"hello" → NegatedFromSource(FolderSource)
	q, err := Parse(`LIST FROM -"hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, ok := q.FromSource.(NegatedFromSource)
	if !ok {
		t.Fatalf("expected NegatedFromSource, got %T", q.FromSource)
	}
	fs, ok := ns.Inner.(FolderSource)
	if !ok {
		t.Fatalf("expected FolderSource inside negation, got %T", ns.Inner)
	}
	if fs.Path != "hello" {
		t.Errorf("expected 'hello', got %q", fs.Path)
	}
}

func TestDV_FromNegatedWithBang(t *testing.T) {
	// FROM !"hello" → NegatedFromSource(FolderSource)
	q, err := Parse(`LIST FROM !"hello"`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, ok := q.FromSource.(NegatedFromSource)
	if !ok {
		t.Fatalf("expected NegatedFromSource, got %T", q.FromSource)
	}
	_, ok = ns.Inner.(FolderSource)
	if !ok {
		t.Fatalf("expected FolderSource inside negation, got %T", ns.Inner)
	}
}

func TestDV_FromNegatedTag(t *testing.T) {
	// FROM -#neat → NegatedFromSource(TagSource)
	q, err := Parse("LIST FROM -#neat")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, ok := q.FromSource.(NegatedFromSource)
	if !ok {
		t.Fatalf("expected NegatedFromSource, got %T", q.FromSource)
	}
	ts, ok := ns.Inner.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource inside negation, got %T", ns.Inner)
	}
	if ts.Tag != "neat" {
		t.Errorf("expected 'neat', got %q", ts.Tag)
	}
}

func TestDV_FromParenSource(t *testing.T) {
	// FROM ("lma0") → FolderSource
	q, err := Parse(`LIST FROM ("lma0")`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	fs, ok := q.FromSource.(FolderSource)
	if !ok {
		t.Fatalf("expected FolderSource, got %T", q.FromSource)
	}
	if fs.Path != "lma0" {
		t.Errorf("expected 'lma0', got %q", fs.Path)
	}

	// FROM (#neat0) → TagSource
	q, err = Parse("LIST FROM (#neat0)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "neat0" {
		t.Errorf("expected 'neat0', got %q", ts.Tag)
	}
}

func TestDV_FromBinarySource(t *testing.T) {
	// FROM "lma0" OR #neat → BooleanFromSource
	q, err := Parse(`LIST FROM "lma0" OR #neat`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bs, ok := q.FromSource.(BooleanFromSource)
	if !ok {
		t.Fatalf("expected BooleanFromSource, got %T", q.FromSource)
	}
	if bs.Op != "OR" {
		t.Errorf("expected OR, got %s", bs.Op)
	}
	if _, ok := bs.Left.(FolderSource); !ok {
		t.Errorf("expected FolderSource on left, got %T", bs.Left)
	}
	if _, ok := bs.Right.(TagSource); !ok {
		t.Errorf("expected TagSource on right, got %T", bs.Right)
	}
}

func TestDV_FromBinarySourceAnd(t *testing.T) {
	// FROM "meme" AND #dirty → BooleanFromSource AND
	q, err := Parse(`LIST FROM "meme" AND #dirty`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bs, ok := q.FromSource.(BooleanFromSource)
	if !ok {
		t.Fatalf("expected BooleanFromSource, got %T", q.FromSource)
	}
	if bs.Op != "AND" {
		t.Errorf("expected AND, got %s", bs.Op)
	}
}

func TestDV_FromNegatedParenSource(t *testing.T) {
	// FROM -(#neat) → NegatedFromSource(TagSource)
	q, err := Parse("LIST FROM -(#neat)")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, ok := q.FromSource.(NegatedFromSource)
	if !ok {
		t.Fatalf("expected NegatedFromSource, got %T", q.FromSource)
	}
	if _, ok := ns.Inner.(TagSource); !ok {
		t.Errorf("expected TagSource inside negation, got %T", ns.Inner)
	}
}

func TestDV_FromNegatedBinarySource(t *testing.T) {
	// FROM -("meme" AND #dirty) → NegatedFromSource(BooleanFromSource)
	q, err := Parse(`LIST FROM -("meme" AND #dirty)`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ns, ok := q.FromSource.(NegatedFromSource)
	if !ok {
		t.Fatalf("expected NegatedFromSource, got %T", q.FromSource)
	}
	bs, ok := ns.Inner.(BooleanFromSource)
	if !ok {
		t.Fatalf("expected BooleanFromSource inside negation, got %T", ns.Inner)
	}
	if bs.Op != "AND" {
		t.Errorf("expected AND, got %s", bs.Op)
	}
}

func TestDV_FromNegatedFolderAndTag(t *testing.T) {
	// FROM -"meme" AND #dirty → BooleanFromSource(NegatedFromSource(FolderSource), TagSource)
	// Note: negation binds tighter than AND
	q, err := Parse(`LIST FROM -"meme" AND #dirty`)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	bs, ok := q.FromSource.(BooleanFromSource)
	if !ok {
		t.Fatalf("expected BooleanFromSource, got %T", q.FromSource)
	}
	if bs.Op != "AND" {
		t.Errorf("expected AND, got %s", bs.Op)
	}
	ns, ok := bs.Left.(NegatedFromSource)
	if !ok {
		t.Fatalf("expected NegatedFromSource on left, got %T", bs.Left)
	}
	if _, ok := ns.Inner.(FolderSource); !ok {
		t.Errorf("expected FolderSource inside negation, got %T", ns.Inner)
	}
	if _, ok := bs.Right.(TagSource); !ok {
		t.Errorf("expected TagSource on right, got %T", bs.Right)
	}
}

// --- Nested/Hierarchical tags ---

func TestDV_NestedTagInFrom(t *testing.T) {
	q, err := Parse("LIST FROM #daily/2021/20/08")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	ts, ok := q.FromSource.(TagSource)
	if !ok {
		t.Fatalf("expected TagSource, got %T", q.FromSource)
	}
	if ts.Tag != "daily/2021/20/08" {
		t.Errorf("expected 'daily/2021/20/08', got %q", ts.Tag)
	}
}
