package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindMCPBinary(t *testing.T) {
	t.Run("none present", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if got := findMCPBinary(); got != "" {
			t.Errorf("findMCPBinary() = %q, want empty", got)
		}
	})

	t.Run("executable at ./bin/mcp", func(t *testing.T) {
		dir := t.TempDir()
		t.Chdir(dir)
		if err := os.Mkdir("bin", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("bin", "mcp"), []byte("#!/bin/sh\n"), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := findMCPBinary(); got != "./bin/mcp" {
			t.Errorf("findMCPBinary() = %q, want ./bin/mcp", got)
		}
	})

	t.Run("non-executable file still found via stat fallback", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := os.Mkdir("bin", 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join("bin", "mcp"), []byte("x"), 0o644); err != nil {
			t.Fatal(err)
		}
		if got := findMCPBinary(); got == "" {
			t.Error("findMCPBinary() = empty, want a path")
		}
	})

	t.Run("directory named mcp is ignored", func(t *testing.T) {
		t.Chdir(t.TempDir())
		if err := os.MkdirAll(filepath.Join("bin", "mcp"), 0o755); err != nil {
			t.Fatal(err)
		}
		if got := findMCPBinary(); got != "" {
			t.Errorf("findMCPBinary() = %q, want empty (a directory is not a binary)", got)
		}
	})
}
