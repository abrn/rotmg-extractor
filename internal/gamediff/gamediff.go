// Package gamediff produces a semantic diff of RotMG game data between two
// builds: which <Object> and <Ground> entries (keyed by their id attribute)
// were added, removed, or changed. It reads the consolidated object.xml and
// ground.xml produced by the mergexml package.
package gamediff

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Change identifies a single added/removed/changed entry.
type Change struct {
	ID   string
	Name string
}

// Category holds the per-element-kind diff results.
type Category struct {
	Added   []Change
	Removed []Change
	Changed []Change
}

// Summary is the full game-data diff.
type Summary struct {
	Objects Category
	Ground  Category
}

// Empty reports whether nothing changed.
func (s Summary) Empty() bool {
	return len(s.Objects.Added)+len(s.Objects.Removed)+len(s.Objects.Changed)+
		len(s.Ground.Added)+len(s.Ground.Removed)+len(s.Ground.Changed) == 0
}

// Compare diffs the merged XML in newDir against oldDir.
func Compare(oldDir, newDir string) (Summary, error) {
	objects, err := diffFile(filepath.Join(oldDir, "object.xml"), filepath.Join(newDir, "object.xml"))
	if err != nil {
		return Summary{}, fmt.Errorf("diffing objects: %w", err)
	}
	ground, err := diffFile(filepath.Join(oldDir, "ground.xml"), filepath.Join(newDir, "ground.xml"))
	if err != nil {
		return Summary{}, fmt.Errorf("diffing ground: %w", err)
	}
	return Summary{Objects: objects, Ground: ground}, nil
}

type entry struct {
	hash string
	name string
}

func diffFile(oldPath, newPath string) (Category, error) {
	oldEntries, err := parseEntries(oldPath)
	if err != nil {
		return Category{}, err
	}
	newEntries, err := parseEntries(newPath)
	if err != nil {
		return Category{}, err
	}

	var c Category
	for id, e := range newEntries {
		old, ok := oldEntries[id]
		switch {
		case !ok:
			c.Added = append(c.Added, Change{ID: id, Name: e.name})
		case old.hash != e.hash:
			c.Changed = append(c.Changed, Change{ID: id, Name: e.name})
		}
	}
	for id, e := range oldEntries {
		if _, ok := newEntries[id]; !ok {
			c.Removed = append(c.Removed, Change{ID: id, Name: e.name})
		}
	}

	sortChanges(c.Added)
	sortChanges(c.Removed)
	sortChanges(c.Changed)
	return c, nil
}

var displayIDRe = regexp.MustCompile(`(?s)<DisplayId>(.*?)</DisplayId>`)

// parseEntries reads every top-level element from a merged file into a map of
// id -> {content hash, display name}. A missing file yields an empty map.
func parseEntries(path string) (map[string]entry, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return map[string]entry{}, nil
	}
	if err != nil {
		return nil, err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf})

	res := map[string]entry{}
	dec := xml.NewDecoder(bytes.NewReader(data))
	depth := 0
	var start int64
	var id string

	for {
		before := dec.InputOffset()
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 {
				start = indexByteFrom(data, '<', before)
				id = attrValue(t.Attr, "id")
			}
		case xml.EndElement:
			if depth == 2 && id != "" {
				raw := bytes.TrimSpace(data[start:dec.InputOffset()])
				sum := sha256.Sum256(raw)
				name := id
				if m := displayIDRe.FindSubmatch(raw); m != nil {
					name = strings.TrimSpace(string(m[1]))
				}
				res[id] = entry{hash: hex.EncodeToString(sum[:8]), name: name}
			}
			depth--
		}
	}
	return res, nil
}

func attrValue(attrs []xml.Attr, name string) string {
	for _, a := range attrs {
		if a.Name.Local == name {
			return a.Value
		}
	}
	return ""
}

func sortChanges(c []Change) {
	sort.Slice(c, func(i, j int) bool { return c[i].ID < c[j].ID })
}

func indexByteFrom(b []byte, c byte, from int64) int64 {
	i := bytes.IndexByte(b[from:], c)
	if i < 0 {
		return from
	}
	return from + int64(i)
}
