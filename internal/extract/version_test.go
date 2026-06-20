package extract

import (
	"os"
	"path/filepath"
	"testing"
)

func TestScanAnchored(t *testing.T) {
	cases := []struct {
		name string
		data []byte
		want string
	}{
		{
			// Two-digit component, non-printable separator (\x90) — the real
			// layout that broke the original single-digit regex.
			name: "real-layout",
			data: []byte("127.0.0.1127.0.0.1\x90\x04\x14" + "6.11.0.1.0\x00\x10*Client*"),
			want: "6.11.0.1.0",
		},
		{
			name: "no-anchor",
			data: []byte("6.11.0.1.0 floating with no anchor"),
			want: "",
		},
		{
			// Garbage after a 127.0.0.1: long/4-digit components must not match.
			name: "garbage-after-anchor",
			data: []byte("127.0.0.11313.53.43.99140151501618.184"),
			want: "",
		},
		{
			name: "single-digit-version",
			data: []byte("127.0.0.1\x002.5.0.0.0\x00"),
			want: "2.5.0.0.0",
		},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := scanAnchored(c.data); got != c.want {
				t.Errorf("scanAnchored = %q, want %q", got, c.want)
			}
		})
	}
}

func TestScanVersionFiles(t *testing.T) {
	dir := t.TempDir()
	// First file has no version; second (metadata-like) has it.
	noVer := filepath.Join(dir, "empty.bin")
	meta := filepath.Join(dir, "global-metadata.dat")
	os.WriteFile(noVer, []byte("nothing here"), 0o644)
	os.WriteFile(meta, []byte("127.0.0.1\x906.11.0.1.0\x00"), 0o644)

	got, err := ScanVersion(filepath.Join(dir, "missing.dat"), noVer, meta)
	if err != nil {
		t.Fatalf("ScanVersion: %v", err)
	}
	if got != "6.11.0.1.0" {
		t.Errorf("ScanVersion = %q, want 6.11.0.1.0", got)
	}
}
