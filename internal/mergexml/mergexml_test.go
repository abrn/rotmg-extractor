package mergexml

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"rotmg-extractor/internal/logx"
)

func TestMerge(t *testing.T) {
	src := t.TempDir()
	out := t.TempDir()

	write := func(name, body string) {
		if err := os.WriteFile(filepath.Join(src, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	write("objects.xml", `<?xml version="1.0"?>
<Objects>
  <Object type="0x1" id="A"><Class>Equipment</Class></Object>
  <Object type="0x2" id="B"/>
</Objects>`)

	write("players.xml", `<?xml version="1.0"?>
<Objects>
  <Object type="0x3" id="Rogue"/>
</Objects>`)

	write("ground.xml", `<?xml version="1.0"?>
<GroundTypes>
  <Ground type="0xb" id="Grass"/>
</GroundTypes>`)

	write("terrains.xml", `<?xml version="1.0"?>
<TerrainTypes>
  <Terrain type="0x0" id="None"/>
  <Terrain type="0x1" id="Mountains"/>
</TerrainTypes>`)

	log := logx.New(logx.LevelError, false)
	if err := Merge(log, src, out); err != nil {
		t.Fatalf("Merge: %v", err)
	}

	objects := readFile(t, out, "object.xml")
	if got := strings.Count(objects, "<Object "); got != 3 {
		t.Errorf("object.xml: want 3 <Object>, got %d", got)
	}
	if strings.Count(objects, "<Ground") != 0 || strings.Count(objects, "<Terrain") != 0 {
		t.Error("object.xml contaminated with non-Object elements")
	}
	if !strings.Contains(objects, `id="Rogue"`) {
		t.Error("object.xml missing element from second file")
	}
	if !strings.Contains(objects, `type="0x1" id="A"`) {
		t.Error("object.xml did not preserve attributes verbatim")
	}
	if !strings.HasPrefix(objects, "<?xml") || !strings.Contains(objects, "<Objects>") {
		t.Error("object.xml missing header/root wrapper")
	}

	ground := readFile(t, out, "ground.xml")
	if got := strings.Count(ground, "<Ground "); got != 1 {
		t.Errorf("ground.xml: want 1 <Ground>, got %d", got)
	}

	misc := readFile(t, out, "misc.xml")
	if got := strings.Count(misc, "<Terrain "); got != 2 {
		t.Errorf("misc.xml: want 2 <Terrain>, got %d", got)
	}
	if strings.Contains(misc, "<Object") || strings.Contains(misc, "<Ground") {
		t.Error("misc.xml leaked Object/Ground elements")
	}
}

func readFile(t *testing.T, dir, name string) string {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(dir, name))
	if err != nil {
		t.Fatalf("reading %s: %v", name, err)
	}
	return string(b)
}
