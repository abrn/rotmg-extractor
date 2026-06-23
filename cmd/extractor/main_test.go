package main

import (
	"testing"

	"rotmg-extractor/internal/config"
	"rotmg-extractor/internal/logx"
)

func testLog() *logx.Logger { return logx.New(logx.ParseLevel("error"), false) }

func TestBuildIL2CPPDumperSelectsIl2CppDumper(t *testing.T) {
	cfg := config.Default()
	cfg.IL2CPP.Enabled = true
	cfg.IL2CPP.Backend = "il2cppdumper"
	d := buildIL2CPPDumper(cfg, testLog())
	if d == nil {
		t.Fatal("expected a dumper, got nil")
	}
	if d.Name() != "il2cppdumper" {
		t.Fatalf("Name() = %q, want il2cppdumper", d.Name())
	}
}

func TestBuildIL2CPPDumperDefaultsToCpp2IL(t *testing.T) {
	cfg := config.Default()
	cfg.IL2CPP.Enabled = true
	d := buildIL2CPPDumper(cfg, testLog())
	if d == nil {
		t.Fatal("expected a dumper, got nil")
	}
	if d.Name() != "cpp2il" {
		t.Fatalf("Name() = %q, want cpp2il", d.Name())
	}
}

func TestBuildIL2CPPDumperDisabledReturnsNil(t *testing.T) {
	cfg := config.Default() // Enabled defaults to false
	if buildIL2CPPDumper(cfg, testLog()) != nil {
		t.Fatal("expected nil when il2cpp is disabled")
	}
}
