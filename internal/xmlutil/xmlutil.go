// Package xmlutil provides an encoding/xml decoder that tolerates the character
// sets seen in RotMG assets. Some files (e.g. assets_manifest) declare
// ISO-8859-1, which the standard decoder rejects unless a CharsetReader is set.
package xmlutil

import (
	"bufio"
	"encoding/xml"
	"io"
	"strings"
	"unicode/utf8"
)

// NewDecoder returns an xml.Decoder with a CharsetReader that handles UTF-8,
// ASCII and Latin-1/Windows-1252, and passes through anything else unchanged
// (best effort) rather than failing.
func NewDecoder(r io.Reader) *xml.Decoder {
	dec := xml.NewDecoder(r)
	dec.CharsetReader = charsetReader
	return dec
}

func charsetReader(charset string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(charset)) {
	case "", "utf-8", "utf8", "us-ascii", "ascii":
		return input, nil
	case "iso-8859-1", "iso8859-1", "latin1", "latin-1", "windows-1252", "cp1252":
		return &latin1Reader{br: bufio.NewReader(input)}, nil
	default:
		// Unknown charset: pass through so the file is still scanned.
		return input, nil
	}
}

// latin1Reader converts a Latin-1 byte stream to UTF-8 on the fly. (Windows-1252
// is treated as Latin-1, which is exact for the ASCII range and close enough for
// the high range in the data we hash/bucket.)
type latin1Reader struct {
	br  *bufio.Reader
	buf [utf8.UTFMax]byte
	rem []byte
}

func (l *latin1Reader) Read(p []byte) (int, error) {
	n := 0
	for n < len(p) {
		if len(l.rem) > 0 {
			c := copy(p[n:], l.rem)
			n += c
			l.rem = l.rem[c:]
			continue
		}
		b, err := l.br.ReadByte()
		if err != nil {
			if n > 0 {
				return n, nil
			}
			return 0, err
		}
		sz := utf8.EncodeRune(l.buf[:], rune(b))
		c := copy(p[n:], l.buf[:sz])
		n += c
		if c < sz {
			l.rem = append(l.rem[:0], l.buf[c:sz]...)
		}
	}
	return n, nil
}
