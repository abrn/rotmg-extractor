package localsrc

import (
	"os"
	"path/filepath"
	"testing"
)

// makeBuild writes a minimal build tree: a Data dir (named dataName) under
// parent, containing il2cpp_data/Metadata/global-metadata.dat, plus a
// GameAssembly binary at gaPath (relative to root). Returns nothing; fails on
// error.
func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestLocateWindowsLayout(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "RotMG Exalt_Data", "il2cpp_data", "Metadata", "global-metadata.dat"), "META")
	writeFile(t, filepath.Join(root, "GameAssembly.dll"), "GA")
	writeFile(t, filepath.Join(root, "UnityPlayer.dll"), "UP")

	b, err := Locate(root)
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if filepath.Base(b.DataDir) != "RotMG Exalt_Data" {
		t.Errorf("DataDir = %q", b.DataDir)
	}
	if filepath.Base(b.GameAssembly) != "GameAssembly.dll" {
		t.Errorf("GameAssembly = %q", b.GameAssembly)
	}
	native := b.NativeFiles()
	if len(native) != 3 { // metadata + GameAssembly + UnityPlayer
		t.Errorf("NativeFiles = %v", native)
	}
}

func TestLocateMacAppDescent(t *testing.T) {
	// A downloaded mac build: filesDir/RotMGExalt.app/Contents/...
	root := t.TempDir()
	app := filepath.Join(root, "RotMGExalt.app")
	writeFile(t, filepath.Join(app, "Contents", "Resources", "Data", "il2cpp_data", "Metadata", "global-metadata.dat"), "META")
	writeFile(t, filepath.Join(app, "Contents", "Frameworks", "GameAssembly.dylib"), "GA")
	writeFile(t, filepath.Join(app, "Contents", "Frameworks", "UnityPlayer.dylib"), "UP")

	b, err := Locate(root) // pass the parent dir, not the .app
	if err != nil {
		t.Fatalf("Locate: %v", err)
	}
	if filepath.Base(b.AppPath) != "RotMGExalt.app" {
		t.Errorf("AppPath = %q, expected descent into .app", b.AppPath)
	}
	if filepath.Base(b.GameAssembly) != "GameAssembly.dylib" {
		t.Errorf("GameAssembly = %q", b.GameAssembly)
	}
}

func TestLocateMissing(t *testing.T) {
	if _, err := Locate(t.TempDir()); err == nil {
		t.Error("expected error for a dir with no build")
	}
}
