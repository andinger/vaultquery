package eval

import (
	"path/filepath"
	"strings"
	"time"

	"github.com/andinger/vaultquery/internal/dql"
)

// FileMetadata holds additional file metadata for file.* field resolution.
type FileMetadata struct {
	Size     int64
	Mtime    int64
	Ctime    int64
	Tags     []string
	InLinks  []string
	OutLinks []string
	Aliases  []string
}

// NewEvalContextWithMeta creates a context with file.* field support.
func NewEvalContextWithMeta(path, title string, fields map[string]dql.Value, meta *FileMetadata) *EvalContext {
	ctx := NewEvalContext(path, title, fields)
	if meta != nil {
		populateFileFields(ctx, path, title, meta)
	}
	return ctx
}

func populateFileFields(ctx *EvalContext, path, title string, meta *FileMetadata) {
	// Derive file.* fields
	name := filepath.Base(path)
	ext := filepath.Ext(name)
	nameNoExt := strings.TrimSuffix(name, ext)
	folder := filepath.Dir(path)
	if folder == "." {
		folder = ""
	}

	fileFields := map[string]dql.Value{
		"file.name":   dql.NewString(nameNoExt),
		"file.folder": dql.NewString(folder),
		"file.path":   dql.NewString(path),
		"file.ext":    dql.NewString(ext),
		"file.link":   dql.NewLink(nameNoExt),
		"file.size":   dql.NewNumber(float64(meta.Size)),
	}

	// Timestamps
	if meta.Mtime > 0 {
		mtime := time.Unix(meta.Mtime, 0)
		fileFields["file.mtime"] = dql.NewDate(mtime)
		fileFields["file.mday"] = dql.NewDate(truncateToDay(mtime))
	}
	if meta.Ctime > 0 {
		ctime := time.Unix(meta.Ctime, 0)
		fileFields["file.ctime"] = dql.NewDate(ctime)
		fileFields["file.cday"] = dql.NewDate(truncateToDay(ctime))
	}

	// Tags
	if len(meta.Tags) > 0 {
		items := make([]dql.Value, len(meta.Tags))
		for i, t := range meta.Tags {
			items[i] = dql.NewString(t)
		}
		fileFields["file.tags"] = dql.NewList(items)
		fileFields["file.etags"] = dql.NewList(items) // etags = explicit tags (same for now)
	} else {
		fileFields["file.tags"] = dql.NewList(nil)
		fileFields["file.etags"] = dql.NewList(nil)
	}

	// Links
	if len(meta.InLinks) > 0 {
		items := make([]dql.Value, len(meta.InLinks))
		for i, l := range meta.InLinks {
			items[i] = dql.NewLink(l)
		}
		fileFields["file.inlinks"] = dql.NewList(items)
	} else {
		fileFields["file.inlinks"] = dql.NewList(nil)
	}

	if len(meta.OutLinks) > 0 {
		items := make([]dql.Value, len(meta.OutLinks))
		for i, l := range meta.OutLinks {
			items[i] = dql.NewLink(l)
		}
		fileFields["file.outlinks"] = dql.NewList(items)
	} else {
		fileFields["file.outlinks"] = dql.NewList(nil)
	}

	// Aliases
	if len(meta.Aliases) > 0 {
		items := make([]dql.Value, len(meta.Aliases))
		for i, a := range meta.Aliases {
			items[i] = dql.NewString(a)
		}
		fileFields["file.aliases"] = dql.NewList(items)
	} else {
		fileFields["file.aliases"] = dql.NewList(nil)
	}

	// file.day from filename date pattern
	if d, ok := dql.ParseDateFromFilename(name); ok {
		fileFields["file.day"] = dql.NewDate(d)
	}

	// file.frontmatter as Object containing all frontmatter fields
	if len(ctx.Fields) > 0 {
		fileFields["file.frontmatter"] = dql.NewObject(ctx.Fields)
	} else {
		fileFields["file.frontmatter"] = dql.NewObject(nil)
	}

	// Add all file.* fields to context
	for k, v := range fileFields {
		ctx.Fields[k] = v
	}
}

func truncateToDay(t time.Time) time.Time {
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, t.Location())
}
