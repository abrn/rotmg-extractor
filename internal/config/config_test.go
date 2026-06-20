package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultIL2CPPBackend(t *testing.T) {
	cfg := Default()
	if cfg.IL2CPP.Backend != "cpp2il" {
		t.Fatalf("default IL2CPP.Backend = %q, want cpp2il", cfg.IL2CPP.Backend)
	}
	if cfg.IL2CPP.Il2CppDumper.Dir != "tools/il2cpp/il2cppdumper" {
		t.Fatalf("default Il2CppDumper.Dir = %q, want tools/il2cpp/il2cppdumper", cfg.IL2CPP.Il2CppDumper.Dir)
	}
}

func TestLoadIl2CppDumperConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "extractor.yml")
	yaml := "il2cpp:\n" +
		"  enabled: true\n" +
		"  backend: il2cppdumper\n" +
		"  il2cppdumper:\n" +
		"    dir: tools/custom\n" +
		"    force_version: \"29\"\n" +
		"    keep_config: true\n" +
		"    extra_args: [--foo, --bar]\n"
	if err := os.WriteFile(path, []byte(yaml), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.IL2CPP.Backend != "il2cppdumper" {
		t.Fatalf("Backend = %q, want il2cppdumper", cfg.IL2CPP.Backend)
	}
	d := cfg.IL2CPP.Il2CppDumper
	if d.Dir != "tools/custom" {
		t.Fatalf("Dir = %q, want tools/custom", d.Dir)
	}
	if d.ForceVersion != "29" {
		t.Fatalf("ForceVersion = %q, want 29", d.ForceVersion)
	}
	if !d.KeepConfig {
		t.Fatalf("KeepConfig = false, want true")
	}
	if len(d.ExtraArgs) != 2 || d.ExtraArgs[0] != "--foo" || d.ExtraArgs[1] != "--bar" {
		t.Fatalf("ExtraArgs = %v, want [--foo --bar]", d.ExtraArgs)
	}
}
