package pipeline

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"rotmg-extractor/internal/logx"
)

func TestEssentialFiles(t *testing.T) {
	want := map[string]bool{
		"RotMG Exalt_Data/resources.assets":                         true,
		"RotMG Exalt_Data/globalgamemanagers":                       true,
		"RotMG Exalt_Data/level3":                                   true,
		"RotMG Exalt_Data/sharedassets1.assets":                     true,
		"RotMG Exalt_Data/il2cpp_data/Metadata/global-metadata.dat": true,
		"GameAssembly.dll":                                          true,
		"RotMGExalt.app/Contents/Frameworks/UnityPlayer.dylib":      true,
		// non-essential:
		"RotMG Exalt_Data/resources.resource":    false,
		"RotMG Exalt_Data/resources.assets.resS": false,
		"baselib.dll":                            false,
		"RotMG Exalt.exe":                        false,
	}
	for path, exp := range want {
		if got := essentialFiles(path); got != exp {
			t.Errorf("essentialFiles(%q) = %v, want %v", path, got, exp)
		}
	}
}

func TestPruneOldBuilds(t *testing.T) {
	publishDir := t.TempDir()
	mk := func(name string, age time.Duration) {
		p := filepath.Join(publishDir, name)
		if err := os.MkdirAll(p, 0o755); err != nil {
			t.Fatal(err)
		}
		os.WriteFile(filepath.Join(p, "x"), []byte("x"), 0o644)
		ts := time.Now().Add(-age)
		os.Chtimes(p, ts, ts)
	}
	mk("current", 0)
	mk("v-new", 1*time.Hour)
	mk("v-mid", 2*time.Hour)
	mk("v-old", 3*time.Hour)

	p := &Pipeline{Log: logx.New(logx.LevelError, false), KeepBuilds: 2}
	p.pruneOldBuilds(publishDir)

	exists := func(name string) bool {
		_, err := os.Stat(filepath.Join(publishDir, name))
		return err == nil
	}
	if !exists("current") {
		t.Error("current must never be pruned")
	}
	if !exists("v-new") || !exists("v-mid") {
		t.Error("the two newest builds should be kept")
	}
	if exists("v-old") {
		t.Error("the oldest build beyond KeepBuilds should be pruned")
	}
}

func TestPruneOldBuildsKeepAll(t *testing.T) {
	publishDir := t.TempDir()
	for _, n := range []string{"v1", "v2", "v3"} {
		os.MkdirAll(filepath.Join(publishDir, n), 0o755)
	}
	p := &Pipeline{Log: logx.New(logx.LevelError, false), KeepBuilds: 0} // keep all
	p.pruneOldBuilds(publishDir)
	for _, n := range []string{"v1", "v2", "v3"} {
		if _, err := os.Stat(filepath.Join(publishDir, n)); err != nil {
			t.Errorf("%s should be kept when KeepBuilds=0", n)
		}
	}
}
