package microscope

import (
	"runtime"
	"strings"
)

// KnownLanguages are the runtimes microscope SDKs report metrics for.
var KnownLanguages = []string{"go", "python", "node", "ruby", "php", "elixir"}

// goRuntimeMetricContent samples the current Go process and returns metric
// entry content shaped the same way every other language SDK reports its
// own runtime metrics: a name, a language tag, a primary concurrency value
// and unit, plus language-specific extras.
func goRuntimeMetricContent() map[string]any {
	var memory runtime.MemStats
	runtime.ReadMemStats(&memory)
	goroutines := runtime.NumGoroutine()
	return map[string]any{
		"name":         "go.runtime",
		"language":     "go",
		"value":        goroutines,
		"unit":         "goroutines",
		"goroutines":   goroutines,
		"memory_mb":    float64(memory.HeapAlloc) / 1024 / 1024,
		"heap_mb":      float64(memory.HeapAlloc) / 1024 / 1024,
		"heap_objects": memory.HeapObjects,
		"gc_cycles":    memory.NumGC,
	}
}

// DetectLanguage reports which runtime produced a metric entry's content.
// It prefers the explicit "language" field SDKs set, and falls back to the
// "<language>.runtime" naming convention for older entries that predate it.
func DetectLanguage(content map[string]any) string {
	if content == nil {
		return ""
	}
	if lang, ok := content["language"].(string); ok && lang != "" {
		return lang
	}
	name, _ := content["name"].(string)
	prefix, _, found := strings.Cut(name, ".")
	if !found {
		return ""
	}
	for _, lang := range KnownLanguages {
		if lang == prefix {
			return lang
		}
	}
	return ""
}
