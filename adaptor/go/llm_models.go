package microscope

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

type listModelsRequest struct {
	Provider string `json:"provider"`
	APIKey   string `json:"api_key"`
}

type providerModel struct {
	ID        string `json:"id"`
	Label     string `json:"label"`
	CreatedAt string `json:"created_at,omitempty"`
}

func (h *Handler) listLLMModels(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 8<<10)
	var input listModelsRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	input.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	input.APIKey = strings.TrimSpace(input.APIKey)
	if input.Provider == "" || input.APIKey == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "provider and api_key are required"})
		return
	}

	models, err := listProviderModels(r.Context(), input.Provider, input.APIKey)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"models": models})
}

func listProviderModels(ctx context.Context, provider, apiKey string) ([]providerModel, error) {
	switch provider {
	case "openai":
		return listOpenAIModels(ctx, apiKey)
	case "cursor":
		return listCursorModels(ctx, apiKey)
	case "anthropic":
		return listAnthropicModels(ctx, apiKey)
	case "gemini":
		return listGeminiModels(ctx, apiKey)
	default:
		return nil, fmt.Errorf("unsupported provider %q", provider)
	}
}

func providerHTTPClient() *http.Client {
	return &http.Client{Timeout: 25 * time.Second}
}

func providerGET(ctx context.Context, endpoint, apiKey string, headers map[string]string) ([]byte, int, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, 0, err
	}
	for key, value := range headers {
		req.Header.Set(key, value)
	}
	if apiKey != "" && req.Header.Get("Authorization") == "" && req.Header.Get("x-api-key") == "" && req.Header.Get("x-goog-api-key") == "" {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	res, err := providerHTTPClient().Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 4<<20))
	return raw, res.StatusCode, nil
}

func listOpenAIModels(ctx context.Context, apiKey string) ([]providerModel, error) {
	after := ""
	models := make([]providerModel, 0, 64)
	for page := 0; page < 20; page++ {
		endpoint := "https://api.openai.com/v1/models"
		if after != "" {
			endpoint += "?after=" + url.QueryEscape(after)
		}
		raw, status, err := providerGET(ctx, endpoint, apiKey, map[string]string{
			"Authorization": "Bearer " + apiKey,
		})
		if err != nil {
			return nil, err
		}
		if status >= 400 {
			return nil, fmt.Errorf("openai returned %d: %s", status, strings.TrimSpace(string(raw)))
		}
		var parsed struct {
			Data []struct {
				ID      string `json:"id"`
				Created int64  `json:"created"`
			} `json:"data"`
			HasMore bool `json:"has_more"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		for _, item := range parsed.Data {
			if !openAIChatModel(item.ID) {
				continue
			}
			created := ""
			if item.Created > 0 {
				created = time.Unix(item.Created, 0).UTC().Format(time.RFC3339)
			}
			models = append(models, providerModel{ID: item.ID, Label: item.ID, CreatedAt: created})
			after = item.ID
		}
		if !parsed.HasMore || len(parsed.Data) == 0 {
			break
		}
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("openai returned no chat-capable models")
	}
	return sortProviderModels(models), nil
}

func openAIChatModel(id string) bool {
	lower := strings.ToLower(id)
	excluded := []string{
		"embedding", "whisper", "tts", "dall-e", "davinci", "babbage", "moderation",
		"realtime", "audio", "transcribe", "search", "sora", "omni-moderation",
		"curie", "ada", "codex", "edit", "similarity", "instruct",
	}
	for _, token := range excluded {
		if strings.Contains(lower, token) {
			return false
		}
	}
	// Include known chat families and any future GPT-family IDs not yet listed.
	if strings.HasPrefix(lower, "gpt-") || strings.HasPrefix(lower, "o1") || strings.HasPrefix(lower, "o3") || strings.HasPrefix(lower, "o4") || strings.HasPrefix(lower, "chatgpt") {
		return true
	}
	return strings.Contains(lower, "gpt") || strings.HasPrefix(lower, "o")
}

func listCursorModels(ctx context.Context, apiKey string) ([]providerModel, error) {
	models, err := listCursorModelsWithAuth(ctx, apiKey, true)
	if err == nil && len(models) > 0 {
		return models, nil
	}
	// Cursor docs also allow Bearer auth.
	models, bearerErr := listCursorModelsWithAuth(ctx, apiKey, false)
	if bearerErr != nil && err != nil {
		return nil, err
	}
	if len(models) == 0 {
		if err != nil {
			return nil, err
		}
		return nil, fmt.Errorf("cursor returned no models")
	}
	return models, nil
}

func listCursorModelsWithAuth(ctx context.Context, apiKey string, basic bool) ([]providerModel, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.cursor.com/v1/models", nil)
	if err != nil {
		return nil, err
	}
	if basic {
		req.SetBasicAuth(apiKey, "")
	} else {
		req.Header.Set("Authorization", "Bearer "+apiKey)
	}
	res, err := providerHTTPClient().Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 2<<20))
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("cursor returned %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Items []struct {
			ID          string `json:"id"`
			DisplayName string `json:"displayName"`
			Description string `json:"description"`
		} `json:"items"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return nil, err
	}
	models := make([]providerModel, 0, len(parsed.Items))
	for _, item := range parsed.Items {
		if strings.TrimSpace(item.ID) == "" {
			continue
		}
		label := strings.TrimSpace(item.DisplayName)
		if label == "" {
			label = item.ID
		}
		models = append(models, providerModel{ID: item.ID, Label: label})
	}
	return sortProviderModels(models), nil
}

func listAnthropicModels(ctx context.Context, apiKey string) ([]providerModel, error) {
	afterID := ""
	models := make([]providerModel, 0, 32)
	for page := 0; page < 20; page++ {
		query := url.Values{}
		query.Set("limit", "1000")
		if afterID != "" {
			query.Set("after_id", afterID)
		}
		endpoint := "https://api.anthropic.com/v1/models?" + query.Encode()
		raw, status, err := providerGET(ctx, endpoint, apiKey, map[string]string{
			"x-api-key":         apiKey,
			"anthropic-version": "2023-06-01",
		})
		if err != nil {
			return nil, err
		}
		if status >= 400 {
			return nil, fmt.Errorf("anthropic returned %d: %s", status, strings.TrimSpace(string(raw)))
		}
		var parsed struct {
			Data []struct {
				ID          string `json:"id"`
				DisplayName string `json:"display_name"`
				CreatedAt   string `json:"created_at"`
			} `json:"data"`
			HasMore bool   `json:"has_more"`
			LastID  string `json:"last_id"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		for _, item := range parsed.Data {
			label := strings.TrimSpace(item.DisplayName)
			if label == "" {
				label = item.ID
			}
			models = append(models, providerModel{ID: item.ID, Label: label, CreatedAt: item.CreatedAt})
		}
		if !parsed.HasMore || parsed.LastID == "" || parsed.LastID == afterID {
			break
		}
		afterID = parsed.LastID
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("anthropic returned no models")
	}
	return sortProviderModels(models), nil
}

func listGeminiModels(ctx context.Context, apiKey string) ([]providerModel, error) {
	pageToken := ""
	models := make([]providerModel, 0, 64)
	seen := make(map[string]struct{})
	for page := 0; page < 20; page++ {
		query := url.Values{}
		query.Set("pageSize", "1000")
		if pageToken != "" {
			query.Set("pageToken", pageToken)
		}
		endpoint := "https://generativelanguage.googleapis.com/v1beta/models?" + query.Encode()
		raw, status, err := providerGET(ctx, endpoint, apiKey, map[string]string{
			"x-goog-api-key": apiKey,
		})
		if err != nil {
			return nil, err
		}
		if status >= 400 {
			return nil, fmt.Errorf("gemini returned %d: %s", status, strings.TrimSpace(string(raw)))
		}
		var parsed struct {
			Models []struct {
				Name                       string   `json:"name"`
				DisplayName                string   `json:"displayName"`
				BaseModelID                string   `json:"baseModelId"`
				SupportedGenerationMethods []string `json:"supportedGenerationMethods"`
			} `json:"models"`
			NextPageToken string `json:"nextPageToken"`
		}
		if err := json.Unmarshal(raw, &parsed); err != nil {
			return nil, err
		}
		for _, item := range parsed.Models {
			if !geminiChatModel(item.Name, item.SupportedGenerationMethods) {
				continue
			}
			id := strings.TrimPrefix(item.Name, "models/")
			if item.BaseModelID != "" {
				id = item.BaseModelID
			}
			if _, ok := seen[id]; ok {
				continue
			}
			seen[id] = struct{}{}
			label := strings.TrimSpace(item.DisplayName)
			if label == "" {
				label = id
			}
			models = append(models, providerModel{ID: id, Label: label})
		}
		if parsed.NextPageToken == "" {
			break
		}
		pageToken = parsed.NextPageToken
	}
	if len(models) == 0 {
		return nil, fmt.Errorf("gemini returned no generative models")
	}
	return sortProviderModels(models), nil
}

func geminiChatModel(name string, methods []string) bool {
	lower := strings.ToLower(name)
	excluded := []string{"embedding", "aqa", "imagen", "veo", "tts", "lyria", "gemma"}
	for _, token := range excluded {
		if strings.Contains(lower, token) {
			return false
		}
	}
	if len(methods) == 0 {
		return strings.Contains(lower, "gemini")
	}
	for _, method := range methods {
		if method == "generateContent" || method == "countTokens" {
			return true
		}
	}
	return false
}

func sortProviderModels(models []providerModel) []providerModel {
	sort.Slice(models, func(i, j int) bool {
		ai, aj := models[i].CreatedAt, models[j].CreatedAt
		if ai != "" && aj != "" && ai != aj {
			return ai > aj
		}
		if ai != "" && aj == "" {
			return true
		}
		if ai == "" && aj != "" {
			return false
		}
		return strings.ToLower(models[i].Label) < strings.ToLower(models[j].Label)
	})
	return models
}
