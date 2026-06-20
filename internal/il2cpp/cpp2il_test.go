package il2cpp

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestParseOutputFormats(t *testing.T) {
	text := "\x1b[34mINFO\x1b[0m: starting\n" +
		"Available output formats:\n" +
		"dll_il_recovery - Recovered managed DLLs\n" +
		"metadata_json: Metadata JSON dump\n" +
		"method_dump    Human-readable method dumps\n"

	got := parseOutputFormats(text)
	want := []string{"dll_il_recovery", "metadata_json", "method_dump"}
	if len(got) != len(want) {
		t.Fatalf("parseOutputFormats length = %d (%v), want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseOutputFormats[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestParseOutputFormatsFromCpp2ILList(t *testing.T) {
	text := "===Cpp2IL by Samboy063===\n" +
		"A Tool to Reverse Unity's \"il2cpp\" Build Process.\n" +
		"Version 2022.1.0-pre-release.21\n" +
		"[Info] [Program] Available output formats:\n" +
		"  ID: dummydll   Name: DLL output format for backwards compatibility.\n" +
		"  ID: dll_il_recovery   Name: DLL files with IL Recovery\n" +
		"  ID: diffable-cs   Name: Diffable C#\n"

	got := parseOutputFormats(text)
	want := []string{"dummydll", "dll_il_recovery", "diffable-cs"}
	if len(got) != len(want) {
		t.Fatalf("parseOutputFormats length = %d (%v), want %d", len(got), got, len(want))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("parseOutputFormats[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func TestStageInputUsesDumpableMetadata(t *testing.T) {
	dir := t.TempDir()
	sourceData := filepath.Join(dir, "source", "RotMGExalt_Data")
	if err := os.MkdirAll(filepath.Join(sourceData, "il2cpp_data", "Metadata"), 0o755); err != nil {
		t.Fatal(err)
	}
	gameAssembly := filepath.Join(dir, "source", "GameAssembly.dll")
	metadata := filepath.Join(dir, "global-metadata.decrypted.dat")
	if err := os.WriteFile(gameAssembly, []byte("game assembly"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(metadata, []byte("dumpable metadata"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceData, "globalgamemanagers"), []byte("unity version"), 0o644); err != nil {
		t.Fatal(err)
	}

	stage := filepath.Join(dir, "stage")
	c := &Cpp2IL{}
	err := c.stageInput(Input{
		DataDir:      sourceData,
		GameAssembly: gameAssembly,
		Metadata:     metadata,
	}, stage)
	if err != nil {
		t.Fatalf("stageInput: %v", err)
	}

	stagedMetadata := filepath.Join(stage, "RotMGExalt_Data", "il2cpp_data", "Metadata", "global-metadata.dat")
	got, err := os.ReadFile(stagedMetadata)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "dumpable metadata" {
		t.Fatalf("staged metadata = %q, want dumpable metadata", got)
	}
	if _, err := os.Stat(filepath.Join(stage, "GameAssembly.dll")); err != nil {
		t.Fatalf("staged GameAssembly missing: %v", err)
	}
}

func TestLinkOrCopyFileCreatesUsableLinkFromOtherWorkingDir(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.bin")
	dst := filepath.Join(dir, "nested", "linked.bin")
	if err := os.WriteFile(src, []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := linkOrCopyFile(src, dst); err != nil {
		t.Fatalf("linkOrCopyFile: %v", err)
	}

	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	if err := os.Chdir(filepath.Join(dir, "nested")); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("staged file should be readable from another cwd: %v", err)
	}
	if string(got) != "ok" {
		t.Fatalf("staged file = %q, want ok", got)
	}
}

func TestResolveCpp2ILBinaryAcceptsDirectPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "cpp2il")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}
	want, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := ResolveCpp2ILBinary(path, ""); got != want {
		t.Fatalf("ResolveCpp2ILBinary direct path = %q, want %q", got, want)
	}
}

func TestResolveCpp2ILBinaryMakesRelativeDirectPathAbsolute(t *testing.T) {
	dir := t.TempDir()
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer os.Chdir(oldwd)
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}

	path := filepath.Join("tools", "il2cpp", "cpp2il")
	if runtime.GOOS == "windows" {
		path += ".exe"
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("binary"), 0o755); err != nil {
		t.Fatal(err)
	}

	want, err := filepath.Abs(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := ResolveCpp2ILBinary(path, ""); got != want {
		t.Fatalf("ResolveCpp2ILBinary relative direct path = %q, want %q", got, want)
	}
}
