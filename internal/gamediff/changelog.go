package gamediff

import (
	"fmt"
	"strings"
)

// maxList caps how many entries are listed per section in the changelog, to
// keep the file readable when a build changes thousands of entries.
const maxList = 500

// Counts returns the added/removed/changed totals for a category.
func (c Category) Counts() (added, removed, changed int) {
	return len(c.Added), len(c.Removed), len(c.Changed)
}

// Markdown renders a human-readable changelog for the build.
func (s Summary) Markdown(version, hash, timestamp string) string {
	var b strings.Builder

	title := version
	if title == "" {
		title = hash
	}
	fmt.Fprintf(&b, "# RotMG build changelog — %s\n", title)
	fmt.Fprintf(&b, "_build hash `%s` · %s_\n\n", hash, timestamp)

	if s.Empty() {
		b.WriteString("No game-data changes detected.\n")
		return b.String()
	}

	writeCategory(&b, "Objects", s.Objects)
	writeCategory(&b, "Ground", s.Ground)
	return b.String()
}

func writeCategory(b *strings.Builder, title string, c Category) {
	added, removed, changed := c.Counts()
	fmt.Fprintf(b, "## %s  (+%d  -%d  ~%d)\n\n", title, added, removed, changed)
	writeSection(b, "Added", c.Added)
	writeSection(b, "Removed", c.Removed)
	writeSection(b, "Changed", c.Changed)
}

func writeSection(b *strings.Builder, label string, changes []Change) {
	if len(changes) == 0 {
		return
	}
	fmt.Fprintf(b, "### %s (%d)\n", label, len(changes))
	limit := len(changes)
	if limit > maxList {
		limit = maxList
	}
	for _, c := range changes[:limit] {
		if c.Name != "" && c.Name != c.ID {
			fmt.Fprintf(b, "- `%s` %s\n", c.ID, c.Name)
		} else {
			fmt.Fprintf(b, "- `%s`\n", c.ID)
		}
	}
	if len(changes) > limit {
		fmt.Fprintf(b, "- … and %d more\n", len(changes)-limit)
	}
	b.WriteByte('\n')
}
