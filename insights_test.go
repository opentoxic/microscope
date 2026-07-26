package microscope

import "testing"

func TestModelSupportsCustomTemperature(t *testing.T) {
	cases := map[string]bool{
		"gpt-4o":                    true,
		"gpt-4o-mini":               true,
		"o1-mini":                   false,
		"o3-mini":                   false,
		"gpt-5":                     false,
		"claude-4.6-sonnet-thinking": false,
	}
	for model, want := range cases {
		if got := modelSupportsCustomTemperature(model); got != want {
			t.Fatalf("modelSupportsCustomTemperature(%q) = %v, want %v", model, got, want)
		}
	}
}
