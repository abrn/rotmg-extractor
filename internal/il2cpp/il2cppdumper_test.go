package il2cpp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestResolveIl2CppDumperBinaryDirectPath(t *testing.T) {
	path := filepath.Join(t.TempDir(), "Il2CppDumper")
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
	got, useDotnet := ResolveIl2CppDumperBinary(path, "")
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if useDotnet {
		t.Fatalf("useDotnet = true, want false for native binary")
	}
}

func TestResolveIl2CppDumperBinaryResolvesDllAsDotnet(t *testing.T) {
	dir := t.TempDir()
	dll := filepath.Join(dir, "Il2CppDumper.dll")
	if err := os.WriteFile(dll, []byte("managed"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, useDotnet := ResolveIl2CppDumperBinary(dir, "")
	want, err := filepath.Abs(dll)
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("path = %q, want %q", got, want)
	}
	if !useDotnet {
		t.Fatalf("useDotnet = false, want true for .dll")
	}
}

func TestIl2CppDumperArgsNative(t *testing.T) {
	d := &Il2CppDumper{BinPath: "/tools/Il2CppDumper", ExtraArgs: []string{"--foo"}}
	got := d.args(Input{GameAssembly: "/g/GameAssembly.dll", Metadata: "/m/meta.dat"}, "/out")
	want := []string{"/g/GameAssembly.dll", "/m/meta.dat", "/out", "--foo"}
	assertSlice(t, got, want)
	if d.program() != "/tools/Il2CppDumper" {
		t.Fatalf("program = %q, want /tools/Il2CppDumper", d.program())
	}
}

func TestIl2CppDumperArgsDotnet(t *testing.T) {
	d := &Il2CppDumper{BinPath: "/tools/Il2CppDumper.dll", UseDotnet: true}
	got := d.args(Input{GameAssembly: "/g/GameAssembly.dll", Metadata: "/m/meta.dat"}, "/out")
	want := []string{"/tools/Il2CppDumper.dll", "/g/GameAssembly.dll", "/m/meta.dat", "/out"}
	assertSlice(t, got, want)
	if d.program() != "dotnet" {
		t.Fatalf("program = %q, want dotnet", d.program())
	}
}

func TestIl2CppDumperWritesNonInteractiveConfig(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "Il2CppDumper")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Pre-existing config with a custom key that must be preserved.
	existing := `{"DumpMethod":true,"RequireAnyKey":true}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(existing), 0o644); err != nil {
		t.Fatal(err)
	}

	d := &Il2CppDumper{BinPath: bin, ForceVersion: "29"}
	if err := d.ensureConfig(); err != nil {
		t.Fatal(err)
	}

	var cfg map[string]any
	data, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["RequireAnyKey"] != false {
		t.Fatalf("RequireAnyKey = %v, want false", cfg["RequireAnyKey"])
	}
	if cfg["DumpMethod"] != true {
		t.Fatalf("DumpMethod not preserved: %v", cfg["DumpMethod"])
	}
	if cfg["ForceIl2CppVersion"] != true {
		t.Fatalf("ForceIl2CppVersion = %v, want true", cfg["ForceIl2CppVersion"])
	}
	if cfg["ForceVersion"] != float64(29) {
		t.Fatalf("ForceVersion = %v, want 29", cfg["ForceVersion"])
	}
}

func TestIl2CppDumperKeepConfigLeavesExisting(t *testing.T) {
	dir := t.TempDir()
	bin := filepath.Join(dir, "Il2CppDumper")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	original := []byte(`{"RequireAnyKey":true}`)
	if err := os.WriteFile(filepath.Join(dir, "config.json"), original, 0o644); err != nil {
		t.Fatal(err)
	}
	d := &Il2CppDumper{BinPath: bin, KeepConfig: true}
	if err := d.ensureConfig(); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dir, "config.json"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("config changed under KeepConfig: %s", got)
	}
}

func TestCollectOutputs(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "dump.cs"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "DummyDll"), 0o755); err != nil {
		t.Fatal(err)
	}
	got := collectOutputs(dir)
	if !sliceContains(got, "DummyDll") || !sliceContains(got, "dump.cs") {
		t.Fatalf("collectOutputs = %v, want DummyDll and dump.cs", got)
	}
	if sliceContains(got, "il2cpp.h") {
		t.Fatalf("collectOutputs included absent il2cpp.h: %v", got)
	}
}

func TestIl2CppDumperAvailable(t *testing.T) {
	var nilDumper *Il2CppDumper
	if nilDumper.Available() {
		t.Fatal("nil dumper should not be available")
	}
	if (&Il2CppDumper{}).Available() {
		t.Fatal("empty BinPath should not be available")
	}
	bin := filepath.Join(t.TempDir(), "Il2CppDumper")
	if err := os.WriteFile(bin, []byte("x"), 0o755); err != nil {
		t.Fatal(err)
	}
	if !(&Il2CppDumper{BinPath: bin}).Available() {
		t.Fatal("existing binary should be available")
	}
}

func assertSlice(t *testing.T, got, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len = %d (%v), want %d (%v)", len(got), got, len(want), want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("[%d] = %q, want %q (all: %v)", i, got[i], want[i], got)
		}
	}
}

func sliceContains(s []string, v string) bool {
	for _, x := range s {
		if x == v {
			return true
		}
	}
	return false
}
