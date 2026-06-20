package builddiff

import (
	"os"
	"path/filepath"
	"testing"
)

func writeTree(t *testing.T, dir string, files map[string]string) {
	t.Helper()
	for name, body := range files {
		p := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCompare(t *testing.T) {
	old := t.TempDir()
	neu := t.TempDir()

	writeTree(t, old, map[string]string{
		"same.xml":    "a\nb\nc\n",
		"changed.xml": "a\nb\nc\n",
		"removed.xml": "x\ny\n",
	})
	writeTree(t, neu, map[string]string{
		"same.xml":    "a\nb\nc\n",
		"changed.xml": "a\nB\nc\nd\n", // -b +B +d  => added 2, removed 1
		"added.xml":   "1\n2\n3\n",    // +3
	})

	d, err := Compare(old, neu)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if d.NewFiles != 1 {
		t.Errorf("NewFiles = %d, want 1", d.NewFiles)
	}
	if d.DelFiles != 1 {
		t.Errorf("DelFiles = %d, want 1", d.DelFiles)
	}
	if d.ChangedFiles != 1 {
		t.Errorf("ChangedFiles = %d, want 1", d.ChangedFiles)
	}
	// added: changed.xml (B, d) = 2  + added.xml (1,2,3) = 3  => 5
	if d.AddedLines != 5 {
		t.Errorf("AddedLines = %d, want 5", d.AddedLines)
	}
	// removed: changed.xml (b) = 1  + removed.xml (x,y) = 2  => 3
	if d.RemovedLines != 3 {
		t.Errorf("RemovedLines = %d, want 3", d.RemovedLines)
	}
}

func TestCompareMissingOld(t *testing.T) {
	neu := t.TempDir()
	writeTree(t, neu, map[string]string{"a.xml": "1\n2\n"})

	d, err := Compare(filepath.Join(neu, "does-not-exist"), neu)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if d.NewFiles != 1 || d.AddedLines != 2 {
		t.Errorf("got %+v, want NewFiles=1 AddedLines=2", d)
	}
}
