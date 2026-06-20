package unityassets

import (
	"encoding/binary"
	"fmt"
	"os"
	"strings"
)

// TextAssets reads every TextAsset object out of the file. Object payloads are
// read by seeking, so the large asset bodies are never all held in memory.
func (sf *SerializedFile) TextAssets() ([]TextAsset, error) {
	f, err := os.Open(sf.path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var assets []TextAsset
	for _, obj := range sf.objects {
		if sf.classID(obj.typeIndex) != classIDTextAsset {
			continue
		}
		payload := make([]byte, obj.byteSize)
		if _, err := f.ReadAt(payload, obj.byteStart); err != nil {
			return nil, fmt.Errorf("reading TextAsset payload at %d: %w", obj.byteStart, err)
		}
		ta, err := parseTextAsset(payload, sf.order)
		if err != nil {
			return nil, err
		}
		assets = append(assets, ta)
	}
	return assets, nil
}

// parseTextAsset reads a TextAsset object body, whose layout is two aligned
// length-prefixed byte arrays: m_Name followed by m_Script.
func parseTextAsset(b []byte, order binary.ByteOrder) (TextAsset, error) {
	r := &reader{b: b, order: order}

	name, err := r.alignedBytes()
	if err != nil {
		return TextAsset{}, fmt.Errorf("reading m_Name: %w", err)
	}
	script, err := r.alignedBytes()
	if err != nil {
		return TextAsset{}, fmt.Errorf("reading m_Script: %w", err)
	}
	return TextAsset{Name: string(name), Script: script}, nil
}

// alignedBytes reads a 4-byte length, that many bytes, then aligns to 4.
func (r *reader) alignedBytes() ([]byte, error) {
	if r.remaining() < 4 {
		return nil, fmt.Errorf("truncated length prefix")
	}
	n := int(r.u32())
	if n < 0 || n > r.remaining() {
		return nil, fmt.Errorf("invalid length %d (remaining %d)", n, r.remaining())
	}
	out := make([]byte, n)
	copy(out, r.b[r.pos:r.pos+n])
	r.pos += n
	r.align4()
	return out, nil
}

// DetectExtension guesses a sensible file extension from TextAsset content,
// mirroring the original extractor's behaviour.
func DetectExtension(script []byte) string {
	s := string(trimPrefix(script, 512))
	s = strings.TrimPrefix(s, "\ufeff") // UTF-8 BOM
	s = strings.TrimLeft(s, " \t\r\n")
	switch {
	case strings.HasPrefix(s, "<!DOCTYPE html>"), strings.HasPrefix(s, "<html"):
		return "html"
	case strings.HasPrefix(s, "<"):
		return "xml"
	case strings.HasPrefix(s, "{"), strings.HasPrefix(s, "["):
		return "json"
	case isPrintable(script):
		return "txt"
	default:
		return "bytes"
	}
}

func trimPrefix(b []byte, n int) []byte {
	if len(b) < n {
		return b
	}
	return b[:n]
}

// isPrintable reports whether the (prefix of the) data looks like UTF-8 text.
func isPrintable(b []byte) bool {
	limit := len(b)
	if limit > 512 {
		limit = 512
	}
	for _, c := range b[:limit] {
		if c == 0 {
			return false
		}
		if c < 0x09 || (c > 0x0d && c < 0x20) {
			return false
		}
	}
	return true
}
