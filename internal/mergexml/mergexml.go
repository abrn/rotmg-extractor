// Package mergexml consolidates the many extracted RotMG XML files into three
// combined files by top-level element tag:
//
//	<Object> elements from every file  -> object.xml
//	<Ground> elements from every file  -> ground.xml
//	everything else                    -> misc.xml
//
// Each element is copied verbatim (exact source bytes) so attributes and
// formatting are preserved without a re-encoding round-trip.
package mergexml

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"

	"rotmg-extractor/internal/logx"
	"rotmg-extractor/internal/xmlutil"
)

// bucket describes one output file and the root element wrapping its contents.
type bucket struct {
	key  string
	file string
	root string
}

var buckets = []bucket{
	{key: "object", file: "object.xml", root: "Objects"},
	{key: "ground", file: "ground.xml", root: "GroundTypes"},
	{key: "misc", file: "misc.xml", root: "Misc"},
}

// classify maps a top-level element tag to its bucket key.
func classify(tag string) string {
	switch tag {
	case "Object":
		return "object"
	case "Ground":
		return "ground"
	default:
		return "misc"
	}
}

// Merge scans srcDir recursively for .xml files, buckets every top-level element
// by tag, and writes object.xml / ground.xml / misc.xml into outDir.
func Merge(log *logx.Logger, srcDir, outDir string) error {
	files, err := xmlFiles(srcDir)
	if err != nil {
		return err
	}
	if len(files) == 0 {
		log.Warn("No .xml files found under %s - skipping XML merge", srcDir)
		return nil
	}

	log.Success("Merging %d XML file(s)...", len(files))
	log.Indent()
	defer log.Dedent()

	out := map[string]*bytes.Buffer{}
	counts := map[string]int{}
	for _, b := range buckets {
		out[b.key] = &bytes.Buffer{}
	}

	for _, f := range files {
		if err := bucketFile(f, out, counts); err != nil {
			log.Warn("Skipping %s: %v", filepath.Base(f), err)
		}
	}

	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, b := range buckets {
		path := filepath.Join(outDir, b.file)
		if err := writeMerged(path, b.root, out[b.key].Bytes()); err != nil {
			return fmt.Errorf("writing %s: %w", b.file, err)
		}
		log.Info("%s: %d element(s) -> %s", b.key, counts[b.key], b.file)
	}
	return nil
}

// bucketFile reads one XML file and appends each top-level element's raw bytes
// to the appropriate bucket.
func bucketFile(path string, out map[string]*bytes.Buffer, counts map[string]int) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	data = bytes.TrimPrefix(data, []byte{0xef, 0xbb, 0xbf}) // UTF-8 BOM

	dec := xmlutil.NewDecoder(bytes.NewReader(data))
	depth := 0
	var elemStart int64
	var elemKey string

	for {
		before := dec.InputOffset()
		tok, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		switch t := tok.(type) {
		case xml.StartElement:
			depth++
			if depth == 2 { // direct child of the root element
				elemStart = indexByteFrom(data, '<', before)
				elemKey = classify(t.Name.Local)
			}
		case xml.EndElement:
			if depth == 2 {
				end := dec.InputOffset() // just past this element's '>'
				buf := out[elemKey]
				buf.Write(bytes.TrimRight(data[elemStart:end], " \t"))
				buf.WriteByte('\n')
				counts[elemKey]++
			}
			depth--
		}
	}
	return nil
}

// writeMerged writes a combined file with an XML header and a wrapping root.
func writeMerged(path, root string, inner []byte) error {
	var buf bytes.Buffer
	buf.WriteString(xml.Header) // <?xml version="1.0" encoding="UTF-8"?>\n
	buf.WriteString("<" + root + ">\n")
	buf.Write(inner)
	buf.WriteString("</" + root + ">\n")
	return os.WriteFile(path, buf.Bytes(), 0o644)
}

// xmlFiles returns all .xml files under dir, sorted for deterministic output.
func xmlFiles(dir string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(dir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && filepath.Ext(path) == ".xml" {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// indexByteFrom returns the index of the first c at or after `from`, or `from`
// if not found.
func indexByteFrom(b []byte, c byte, from int64) int64 {
	i := bytes.IndexByte(b[from:], c)
	if i < 0 {
		return from
	}
	return from + int64(i)
}
