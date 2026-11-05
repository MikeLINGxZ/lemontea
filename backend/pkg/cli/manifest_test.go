package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestManifestLoadSaveRoundtrip verifies that LoadManifest reads back what SaveManifest wrote.
func TestManifestLoadSaveRoundtrip(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")

	original := Manifest{
		Name:        "lark-cli",
		Version:     "1.2.3",
		Description: "feishu cli",
		Executable:  "lark",
		Isolation:   IsolationIsolated,
		Tools: []Tool{
			{
				Name:            "lark_send",
				Description:     "send a message",
				InputSchema:     json.RawMessage(`{"type":"object","properties":{"to":{"type":"string"}}}`),
				ArgvTemplate:    []string{"message", "send", "--to", "{to}"},
				OutputMode:      OutputJSON,
				TimeoutSeconds:  30,
				RequiresConfirm: true,
				Enabled:         true,
			},
		},
	}

	if err := SaveManifest(path, original); err != nil {
		t.Fatalf("save: %v", err)
	}
	loaded, err := LoadManifest(path)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if loaded.Name != original.Name || loaded.Executable != original.Executable {
		t.Fatalf("top-level mismatch: %+v", loaded)
	}
	if len(loaded.Tools) != 1 || loaded.Tools[0].Name != "lark_send" {
		t.Fatalf("tools mismatch: %+v", loaded.Tools)
	}
	if loaded.Tools[0].OutputMode != OutputJSON {
		t.Fatalf("output mode mismatch: %v", loaded.Tools[0].OutputMode)
	}
}

// TestManifestLoadMissingReturnsEmpty verifies LoadManifest on a missing file returns an empty manifest (not error).
func TestManifestLoadMissingReturnsEmpty(t *testing.T) {
	m, err := LoadManifest(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("expected nil err on missing file, got %v", err)
	}
	if m.Name != "" || len(m.Tools) != 0 {
		t.Fatalf("expected empty manifest, got %+v", m)
	}
}

// TestManifestValidateRejectsUnknownPlaceholders verifies Validate flags argv_template tokens that have no matching input field.
func TestManifestValidateRejectsUnknownPlaceholders(t *testing.T) {
	m := Manifest{
		Executable: "x",
		Tools: []Tool{
			{
				Name:         "t",
				InputSchema:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}}}`),
				ArgvTemplate: []string{"--a", "{a}", "--b", "{undeclared}"},
				OutputMode:   OutputText,
			},
		},
	}
	if err := Validate(m); err == nil {
		t.Fatalf("expected validation error for undeclared placeholder")
	}
}

// TestManifestValidateAcceptsCleanManifest verifies a manifest where every {placeholder} maps to an input field passes.
func TestManifestValidateAcceptsCleanManifest(t *testing.T) {
	m := Manifest{
		Executable: "x",
		Tools: []Tool{
			{
				Name:         "t",
				InputSchema:  json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"},"b":{"type":"string"}}}`),
				ArgvTemplate: []string{"--a", "{a}", "--b", "{b}"},
				OutputMode:   OutputText,
			},
		},
	}
	if err := Validate(m); err != nil {
		t.Fatalf("expected clean manifest to validate, got %v", err)
	}
}

// TestManifestSaveAtomicWritesViaTmp verifies SaveManifest uses a tmp+rename pattern (visible by inspecting after save).
func TestManifestSaveAtomicWritesViaTmp(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "manifest.json")
	if err := SaveManifest(path, Manifest{Executable: "x"}); err != nil {
		t.Fatalf("save: %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("manifest missing: %v", err)
	}
	// No leftover tmp file.
	if _, err := os.Stat(path + ".tmp"); !os.IsNotExist(err) {
		t.Fatalf("unexpected tmp file lingering: err=%v", err)
	}
}

// TestRepairManifestExecutableBackfillsMissingExecutable verifies older manifests are repaired from the installed package metadata.
func TestRepairManifestExecutableBackfillsMissingExecutable(t *testing.T) {
	installDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(installDir, "node_modules", ".bin"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(installDir, "node_modules", "lark-cli"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "node_modules", "lark-cli", "package.json"), []byte(`{"name":"lark-cli","version":"1.0.0","bin":"index.js"}`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(installDir, "node_modules", ".bin", "lark-cli"), []byte("#!/usr/bin/env bash\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	manifestPath := filepath.Join(t.TempDir(), "manifest.json")
	if err := SaveManifest(manifestPath, Manifest{Name: "lark-cli", Tools: []Tool{}}); err != nil {
		t.Fatal(err)
	}

	repaired, err := RepairManifestExecutable(manifestPath, installDir, Manifest{Name: "lark-cli", Tools: []Tool{}})
	if err != nil {
		t.Fatalf("RepairManifestExecutable returned error: %v", err)
	}
	if filepath.Base(repaired.Executable) != "lark-cli" {
		t.Fatalf("expected repaired executable to target lark-cli, got %q", repaired.Executable)
	}

	loaded, err := LoadManifest(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Executable == "" {
		t.Fatalf("expected repaired manifest to be persisted: %+v", loaded)
	}
}
