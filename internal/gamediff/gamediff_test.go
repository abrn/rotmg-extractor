package gamediff

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeMerged(t *testing.T, dir string, object, ground string) {
	t.Helper()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "object.xml"), []byte(object), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "ground.xml"), []byte(ground), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestCompare(t *testing.T) {
	old := t.TempDir()
	neu := t.TempDir()

	writeMerged(t, old, `<?xml version="1.0"?>
<Objects>
  <Object type="0x1" id="Keep"><DisplayId>Keeper</DisplayId><HP>100</HP></Object>
  <Object type="0x2" id="Change"><HP>50</HP></Object>
  <Object type="0x3" id="Gone"><HP>1</HP></Object>
</Objects>`, `<?xml version="1.0"?><GroundTypes></GroundTypes>`)

	writeMerged(t, neu, `<?xml version="1.0"?>
<Objects>
  <Object type="0x1" id="Keep"><DisplayId>Keeper</DisplayId><HP>100</HP></Object>
  <Object type="0x2" id="Change"><HP>75</HP></Object>
  <Object type="0x4" id="Fresh"><HP>10</HP></Object>
</Objects>`, `<?xml version="1.0"?><GroundTypes></GroundTypes>`)

	s, err := Compare(old, neu)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}

	a, r, c := s.Objects.Counts()
	if a != 1 || r != 1 || c != 1 {
		t.Fatalf("object counts = +%d -%d ~%d, want +1 -1 ~1", a, r, c)
	}
	if s.Objects.Added[0].ID != "Fresh" {
		t.Errorf("added = %q, want Fresh", s.Objects.Added[0].ID)
	}
	if s.Objects.Removed[0].ID != "Gone" {
		t.Errorf("removed = %q, want Gone", s.Objects.Removed[0].ID)
	}
	if s.Objects.Changed[0].ID != "Change" {
		t.Errorf("changed = %q, want Change", s.Objects.Changed[0].ID)
	}

	// Changelog should mention the changes and the version.
	md := s.Markdown("6.11.0.0.0", "abc123", "2026-06-20 00:00:00")
	for _, want := range []string{"6.11.0.0.0", "Fresh", "Gone", "Change", "Keeper"} {
		if want == "Keeper" {
			// Keep is unchanged, so its DisplayId should NOT appear.
			if strings.Contains(md, "Keeper") {
				t.Errorf("changelog should not mention unchanged entry Keeper")
			}
			continue
		}
		if !strings.Contains(md, want) {
			t.Errorf("changelog missing %q", want)
		}
	}
}

func TestCompareNoPrevious(t *testing.T) {
	neu := t.TempDir()
	writeMerged(t, neu, `<Objects><Object id="A"/></Objects>`, `<GroundTypes/>`)

	s, err := Compare(filepath.Join(neu, "missing"), neu)
	if err != nil {
		t.Fatalf("Compare: %v", err)
	}
	if a, _, _ := s.Objects.Counts(); a != 1 {
		t.Errorf("added = %d, want 1 (all new when no previous)", a)
	}
}
