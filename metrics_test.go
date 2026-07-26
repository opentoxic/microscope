package microscope

import "testing"

func TestGoRuntimeMetricContent(t *testing.T) {
	content := goRuntimeMetricContent()

	if content["name"] != "go.runtime" {
		t.Fatalf("expected name go.runtime, got %v", content["name"])
	}
	if content["language"] != "go" {
		t.Fatalf("expected language go, got %v", content["language"])
	}
	if content["unit"] != "goroutines" {
		t.Fatalf("expected unit goroutines, got %v", content["unit"])
	}
	if _, ok := content["value"].(int); !ok {
		t.Fatalf("expected value to be an int, got %T", content["value"])
	}
	if _, ok := content["memory_mb"].(float64); !ok {
		t.Fatalf("expected memory_mb to be a float64, got %T", content["memory_mb"])
	}
}

func TestDetectLanguageFromExplicitField(t *testing.T) {
	for _, lang := range KnownLanguages {
		content := map[string]any{"language": lang, "name": "irrelevant"}
		if got := DetectLanguage(content); got != lang {
			t.Fatalf("expected %s, got %s", lang, got)
		}
	}
}

func TestDetectLanguageFallsBackToNamePrefix(t *testing.T) {
	cases := map[string]string{
		"go.runtime":      "go",
		"python.runtime":  "python",
		"node.runtime":    "node",
		"ruby.runtime":    "ruby",
		"php.runtime":     "php",
		"elixir.runtime":  "elixir",
		"custom.event":    "",
		"unknownlanguage": "",
	}
	for name, want := range cases {
		got := DetectLanguage(map[string]any{"name": name})
		if got != want {
			t.Fatalf("name %q: expected %q, got %q", name, want, got)
		}
	}
}

func TestDetectLanguageHandlesNilAndEmpty(t *testing.T) {
	if got := DetectLanguage(nil); got != "" {
		t.Fatalf("expected empty string for nil content, got %q", got)
	}
	if got := DetectLanguage(map[string]any{}); got != "" {
		t.Fatalf("expected empty string for empty content, got %q", got)
	}
}
