package cli

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// TestGenerateParsesManifestAndOverridesExecutable verifies generated JSON is parsed and the executable is caller-controlled.
func TestGenerateParsesManifestAndOverridesExecutable(t *testing.T) {
	t.Helper()

	manifest, err := Generate(context.Background(), GenerateParams{
		HelpText:    "demo --help",
		PackageName: "demo-cli",
		PackageMeta: PackageMeta{Version: "1.0.0", Description: "demo"},
		Executable:  "/tmp/demo-cli",
		Caller: func(ctx context.Context, system, user string) (string, error) {
			return `{
				"name":"demo-cli",
				"version":"1.0.0",
				"description":"demo",
				"executable":"hallucinated",
				"isolation":"isolated",
				"tools":[
					{
						"name":"list_items",
						"description":"list items",
						"input_schema":{"type":"object","properties":{"query":{"type":"string"}}},
						"argv_template":["list","--query","{query}"],
						"output_mode":"json",
						"timeout_seconds":60,
						"requires_confirm":false,
						"enabled":true
					}
				]
			}`, nil
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if manifest.Executable != "/tmp/demo-cli" {
		t.Fatalf("expected executable override, got %q", manifest.Executable)
	}
	if len(manifest.Tools) != 1 || manifest.Tools[0].Name != "list_items" {
		t.Fatalf("unexpected tools: %+v", manifest.Tools)
	}
}

// TestGenerateAcceptsFencedJSON verifies markdown code fences are tolerated.
func TestGenerateAcceptsFencedJSON(t *testing.T) {
	t.Helper()

	manifest, err := Generate(context.Background(), GenerateParams{
		HelpText:    "demo --help",
		PackageName: "demo-cli",
		PackageMeta: PackageMeta{Version: "1.0.0"},
		Executable:  "/tmp/demo-cli",
		Caller: func(ctx context.Context, system, user string) (string, error) {
			return "```json\n{\n  \"name\":\"demo-cli\",\n  \"version\":\"1.0.0\",\n  \"description\":\"demo\",\n  \"executable\":\"ignored\",\n  \"isolation\":\"isolated\",\n  \"tools\":[]\n}\n```", nil
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if manifest.Name != "demo-cli" || manifest.Executable != "/tmp/demo-cli" {
		t.Fatalf("unexpected manifest: %+v", manifest)
	}
}

// TestGenerateFallsBackToStubManifest verifies malformed model output degrades to an editable empty manifest.
func TestGenerateFallsBackToStubManifest(t *testing.T) {
	t.Helper()

	manifest, err := Generate(context.Background(), GenerateParams{
		HelpText:    "demo --help",
		PackageName: "demo-cli",
		PackageMeta: PackageMeta{Version: "1.0.0", Description: "demo"},
		Executable:  "/tmp/demo-cli",
		Caller: func(ctx context.Context, system, user string) (string, error) {
			return "this is not json", nil
		},
	})
	if err != nil {
		t.Fatalf("expected fallback manifest without error, got %v", err)
	}
	if manifest.Name != "demo-cli" || manifest.Executable != "/tmp/demo-cli" {
		t.Fatalf("unexpected fallback manifest: %+v", manifest)
	}
	if len(manifest.Tools) != 0 {
		t.Fatalf("expected empty tools on fallback, got %+v", manifest.Tools)
	}
}

// TestGenerateBuildsValidStubSchema verifies the fallback manifest remains serializable for later editing.
func TestGenerateBuildsValidStubSchema(t *testing.T) {
	t.Helper()

	manifest, err := Generate(context.Background(), GenerateParams{
		HelpText:    "demo --help",
		PackageName: "demo-cli",
		PackageMeta: PackageMeta{Version: "1.0.0"},
		Executable:  "/tmp/demo-cli",
		Caller: func(ctx context.Context, system, user string) (string, error) {
			return `{"name":"demo-cli","version":"1.0.0","description":"demo","executable":"","isolation":"isolated","tools":[]}`, nil
		},
	})
	if err != nil {
		t.Fatalf("Generate returned error: %v", err)
	}
	if _, marshalErr := json.Marshal(manifest); marshalErr != nil {
		t.Fatalf("expected fallback manifest to remain serializable: %v", marshalErr)
	}
}

// TestGenerateFallsBackWhenCallerFails verifies transport/model failures still return an editable stub manifest.
func TestGenerateFallsBackWhenCallerFails(t *testing.T) {
	t.Helper()

	manifest, err := Generate(context.Background(), GenerateParams{
		HelpText:    "demo --help",
		PackageName: "demo-cli",
		PackageMeta: PackageMeta{Version: "1.0.0", Description: "demo"},
		Executable:  "/tmp/demo-cli",
		Caller: func(ctx context.Context, system, user string) (string, error) {
			return "", context.DeadlineExceeded
		},
	})
	if err != nil {
		t.Fatalf("expected fallback manifest without error, got %v", err)
	}
	if manifest.Name != "demo-cli" || manifest.Executable != "/tmp/demo-cli" {
		t.Fatalf("unexpected fallback manifest: %+v", manifest)
	}
	if len(manifest.Tools) != 0 {
		t.Fatalf("expected empty tools on caller failure, got %+v", manifest.Tools)
	}
}

// TestBuildGeneratorUserPromptTruncatesVerboseHelp verifies extremely large help text is clamped before prompting the model.
func TestBuildGeneratorUserPromptTruncatesVerboseHelp(t *testing.T) {
	t.Helper()

	helpText := strings.Repeat("subcommand --flag description\n", 800)
	prompt := buildGeneratorUserPrompt(GenerateParams{
		HelpText:    helpText,
		PackageName: "demo-cli",
		PackageMeta: PackageMeta{Version: "1.0.0", Description: "demo"},
		Executable:  "/tmp/demo-cli",
	})
	if len(prompt) >= len(helpText)+200 {
		t.Fatalf("expected prompt to truncate help text, prompt length=%d raw length=%d", len(prompt), len(helpText))
	}
	if !strings.Contains(prompt, "[help output truncated]") {
		t.Fatalf("expected prompt to contain truncation marker: %q", prompt)
	}
}
