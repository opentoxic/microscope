package microscope

import (
	"encoding/json"
	"testing"
)

func TestOpenAIChatModelFilter(t *testing.T) {
	cases := map[string]bool{
		"gpt-4o":                 true,
		"gpt-4o-mini":            true,
		"gpt-5":                  true,
		"o1-mini":                true,
		"o3-mini":                true,
		"text-embedding-3-small": false,
		"whisper-1":              false,
		"dall-e-3":               false,
		"tts-1":                  false,
	}
	for id, want := range cases {
		if got := openAIChatModel(id); got != want {
			t.Fatalf("openAIChatModel(%q) = %v, want %v", id, got, want)
		}
	}
}

func TestGeminiChatModelFilter(t *testing.T) {
	if !geminiChatModel("models/gemini-2.0-flash", []string{"generateContent"}) {
		t.Fatal("expected gemini flash to be included")
	}
	if geminiChatModel("models/text-embedding-004", []string{"embedContent"}) {
		t.Fatal("expected embedding model to be excluded")
	}
	if !geminiChatModel("models/gemini-2.5-pro", nil) {
		t.Fatal("expected gemini model without methods to be included")
	}
}

func TestSortProviderModels(t *testing.T) {
	models := sortProviderModels([]providerModel{
		{ID: "b", Label: "Beta"},
		{ID: "a", Label: "Alpha", CreatedAt: "2025-01-02T00:00:00Z"},
		{ID: "c", Label: "Gamma", CreatedAt: "2025-01-03T00:00:00Z"},
	})
	if models[0].ID != "c" || models[1].ID != "a" {
		t.Fatalf("unexpected sort order: %#v", models)
	}
}

func TestListCursorModelsParsesItems(t *testing.T) {
	raw := `{
		"items": [
			{"id":"composer-2","displayName":"Composer 2"},
			{"id":"claude-4.6-sonnet-thinking","displayName":"Claude 4.6 Sonnet (Thinking)"}
		]
	}`
	var parsed struct {
		Items []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
		} `json:"items"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Items) != 2 || parsed.Items[0].ID != "composer-2" {
		t.Fatalf("unexpected cursor payload: %#v", parsed.Items)
	}
}

func TestListAnthropicModelsParsesData(t *testing.T) {
	raw := `{
		"data": [
			{"id":"claude-sonnet-4-20250514","display_name":"Claude Sonnet 4","created_at":"2025-05-14T00:00:00Z"}
		],
		"has_more": false,
		"last_id": "claude-sonnet-4-20250514"
	}`
	var parsed struct {
		Data []struct {
			ID          string `json:"id"`
			DisplayName string `json:"display_name"`
		} `json:"data"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Data) != 1 || parsed.Data[0].DisplayName != "Claude Sonnet 4" {
		t.Fatalf("unexpected anthropic payload: %#v", parsed.Data)
	}
}

func TestListGeminiModelsParsesModels(t *testing.T) {
	raw := `{
		"models": [
			{
				"name": "models/gemini-2.0-flash",
				"displayName": "Gemini 2.0 Flash",
				"baseModelId": "gemini-2.0-flash",
				"supportedGenerationMethods": ["generateContent"]
			}
		]
	}`
	var parsed struct {
		Models []struct {
			Name        string `json:"name"`
			DisplayName string `json:"displayName"`
			BaseModelID string `json:"baseModelId"`
		} `json:"models"`
	}
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		t.Fatal(err)
	}
	if len(parsed.Models) != 1 || parsed.Models[0].BaseModelID != "gemini-2.0-flash" {
		t.Fatalf("unexpected gemini payload: %#v", parsed.Models)
	}
}
