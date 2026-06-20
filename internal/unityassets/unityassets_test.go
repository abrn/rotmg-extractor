package unityassets

import (
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectExtension(t *testing.T) {
	cases := []struct {
		name string
		in   string
		want string
	}{
		{"xml", `<?xml version="1.0"?><Objects/>`, "xml"},
		{"xml-no-decl", `<Objects></Objects>`, "xml"},
		{"xml-with-bom", "\ufeff<Objects/>", "xml"},
		{"html", "<!DOCTYPE html><html></html>", "html"},
		{"json-obj", `{"a":1}`, "json"},
		{"json-arr", `[1,2,3]`, "json"},
		{"json-leading-ws", "  \n{\"a\":1}", "json"},
		{"txt", "just some plain text", "txt"},
		{"binary", string([]byte{0x00, 0x01, 0x02, 0xff}), "bytes"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := DetectExtension([]byte(c.in)); got != c.want {
				t.Errorf("DetectExtension(%q) = %q, want %q", c.in, got, c.want)
			}
		})
	}
}

func TestParseTextAsset(t *testing.T) {
	// Build a TextAsset body: aligned m_Name then aligned m_Script.
	var b []byte
	appendAligned := func(s string) {
		var n [4]byte
		binary.LittleEndian.PutUint32(n[:], uint32(len(s)))
		b = append(b, n[:]...)
		b = append(b, s...)
		for len(b)%4 != 0 {
			b = append(b, 0)
		}
	}
	appendAligned("objects")             // name (7 bytes -> padded)
	appendAligned("<Objects></Objects>") // script

	ta, err := parseTextAsset(b, binary.LittleEndian)
	if err != nil {
		t.Fatalf("parseTextAsset: %v", err)
	}
	if ta.Name != "objects" {
		t.Errorf("Name = %q, want %q", ta.Name, "objects")
	}
	if string(ta.Script) != "<Objects></Objects>" {
		t.Errorf("Script = %q", ta.Script)
	}
}

func TestSanitize(t *testing.T) {
	if got := sanitize("dungeons/realm:boss"); got != "dungeons_realmboss" {
		t.Errorf("sanitize = %q", got)
	}
}

// TestRealInstall validates the parser against the installed game when present.
// It is skipped in CI / on machines without the build.
func TestRealInstall(t *testing.T) {
	data := "/Users/admin/.local/share/RealmOfTheMadGod/Production/RotMGExalt.app/Contents/Resources/Data"
	resources := filepath.Join(data, "resources.assets")
	if _, err := os.Stat(resources); err != nil {
		t.Skip("game not installed; skipping integration test")
	}

	sf, err := OpenSerializedFile(resources)
	if err != nil {
		t.Fatalf("OpenSerializedFile: %v", err)
	}
	assets, err := sf.TextAssets()
	if err != nil {
		t.Fatalf("TextAssets: %v", err)
	}
	if len(assets) == 0 {
		t.Fatal("expected TextAssets, got none")
	}

	var objects *TextAsset
	for i := range assets {
		if assets[i].Name == "objects" {
			objects = &assets[i]
			break
		}
	}
	if objects == nil {
		t.Fatal("did not find the 'objects' TextAsset")
	}
	if !strings.Contains(string(objects.Script), "<Objects>") {
		t.Errorf("objects asset does not look like RotMG XML: %.60q", objects.Script)
	}
	if DetectExtension(objects.Script) != "xml" {
		t.Errorf("objects detected as %q, want xml", DetectExtension(objects.Script))
	}
}
