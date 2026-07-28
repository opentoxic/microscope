package microscope

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type insightRequest struct {
	Provider string         `json:"provider"`
	Model    string         `json:"model"`
	APIKey   string         `json:"api_key"`
	Period   string         `json:"period"`
	Context  string         `json:"context"`
	Entries  []insightEntry `json:"entries"`
}

type insightEntry struct {
	ID        string         `json:"id"`
	Type      string         `json:"type"`
	CreatedAt string         `json:"created_at"`
	Content   map[string]any `json:"content"`
}

type insightFinding struct {
	Title    string `json:"title"`
	Detail   string `json:"detail"`
	Severity string `json:"severity"`
}

type insightResponse struct {
	Summary            string           `json:"summary"`
	HealthScore        int              `json:"health_score"`
	Findings           []insightFinding `json:"findings"`
	Recommendations    []string         `json:"recommendations"`
	Metrics            map[string]any   `json:"metrics"`
	SignalDistribution []signalDist     `json:"signal_distribution"`
}

type signalDist struct {
	Type  string  `json:"type"`
	Count int     `json:"count"`
	Pct   float64 `json:"pct"`
}

func (h *Handler) analyzeInsights(w http.ResponseWriter, r *http.Request) {
	r.Body = http.MaxBytesReader(w, r.Body, 2<<20)
	var input insightRequest
	if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid JSON body"})
		return
	}
	input.Provider = strings.TrimSpace(strings.ToLower(input.Provider))
	input.Model = strings.TrimSpace(input.Model)
	input.APIKey = strings.TrimSpace(input.APIKey)
	if input.Provider == "" || input.Model == "" || input.APIKey == "" {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "provider, model, and api_key are required"})
		return
	}
	if len(input.Entries) == 0 {
		writeJSON(w, http.StatusUnprocessableEntity, map[string]string{"error": "at least one entry is required"})
		return
	}
	if len(input.Entries) > 120 {
		input.Entries = input.Entries[:120]
	}

	result, err := runInsightAnalysis(r.Context(), input)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, result)
}

func runInsightAnalysis(ctx context.Context, input insightRequest) (insightResponse, error) {
	prompt := buildInsightPrompt(input)
	raw, err := callLLM(ctx, input.Provider, input.Model, input.APIKey, prompt)
	if err != nil {
		return insightResponse{}, err
	}
	return parseInsightResponse(raw, input.Entries)
}

func buildInsightPrompt(input insightRequest) string {
	var b strings.Builder
	b.WriteString("You are an observability analyst for a Go microservice runtime recorder called Microscope.\n")
	b.WriteString("Analyze the telemetry and return ONLY valid JSON with this exact schema:\n")
	b.WriteString(`{"summary":"string","health_score":0-100,"findings":[{"title":"string","detail":"string","severity":"info|warning|critical"}],"recommendations":["string"],"metrics":{"error_rate":"string","avg_latency_ms":"number","dominant_signal":"string","risk_level":"string"},"signal_distribution":[{"type":"string","count":number,"pct":number}]}` + "\n")
	if input.Context != "" {
		b.WriteString("Context: " + input.Context + "\n")
	}
	if input.Period != "" {
		b.WriteString("Time window: " + input.Period + "\n")
	}
	b.WriteString(fmt.Sprintf("Entry count: %d\n", len(input.Entries)))
	b.WriteString("Entries:\n")
	for _, entry := range input.Entries {
		content, _ := json.Marshal(entry.Content)
		if len(content) > 900 {
			content = content[:900]
		}
		b.WriteString(fmt.Sprintf("- id=%s type=%s at=%s content=%s\n", entry.ID, entry.Type, entry.CreatedAt, string(content)))
	}
	return b.String()
}

func parseInsightResponse(raw string, entries []insightEntry) (insightResponse, error) {
	raw = strings.TrimSpace(raw)
	start := strings.Index(raw, "{")
	end := strings.LastIndex(raw, "}")
	if start >= 0 && end > start {
		raw = raw[start : end+1]
	}
	var parsed insightResponse
	if err := json.Unmarshal([]byte(raw), &parsed); err != nil {
		return fallbackInsight(entries), nil
	}
	if parsed.HealthScore < 0 {
		parsed.HealthScore = 0
	}
	if parsed.HealthScore > 100 {
		parsed.HealthScore = 100
	}
	if len(parsed.SignalDistribution) == 0 {
		parsed.SignalDistribution = distributionFor(entries)
	}
	if parsed.Metrics == nil {
		parsed.Metrics = map[string]any{}
	}
	return parsed, nil
}

func fallbackInsight(entries []insightEntry) insightResponse {
	errors := 0
	totalLatency := 0.0
	latencyCount := 0
	for _, entry := range entries {
		if entry.Type == string(TypeException) {
			errors++
		}
		if status, ok := entry.Content["status"].(float64); ok && status >= 500 {
			errors++
		}
		if duration, ok := entry.Content["duration_ms"].(float64); ok && duration > 0 {
			totalLatency += duration
			latencyCount++
		}
	}
	avg := 0.0
	if latencyCount > 0 {
		avg = totalLatency / float64(latencyCount)
	}
	score := 92
	if errors > 0 {
		score = 58
	} else if avg >= 500 {
		score = 72
	}
	findings := []insightFinding{
		{Title: "Microscope coverage", Detail: fmt.Sprintf("%d correlated records were included in this manual analysis window.", len(entries)), Severity: "info"},
	}
	if errors > 0 {
		findings = append(findings, insightFinding{
			Title: "Failure evidence detected", Detail: fmt.Sprintf("%d error-class records were present in the submitted window.", errors), Severity: "critical",
		})
	}
	if avg >= 200 {
		findings = append(findings, insightFinding{
			Title: "Latency pressure", Detail: fmt.Sprintf("Average recorded span cost is %.0fms across timed operations.", avg), Severity: "warning",
		})
	}
	recs := []string{"Inspect the dominant slow span first", "Compare this window against a healthy baseline trace"}
	if errors > 0 {
		recs = []string{"Open the earliest failure boundary", "Correlate exceptions with request and SQL evidence", "Validate downstream dependency health"}
	}
	return insightResponse{
		Summary:            fmt.Sprintf("Manual analysis across %d records shows %d error-class events and %.0fms average span cost.", len(entries), errors, avg),
		HealthScore:        score,
		Findings:           findings,
		Recommendations:    recs,
		Metrics:            map[string]any{"error_rate": fmt.Sprintf("%d", errors), "avg_latency_ms": avg, "dominant_signal": dominantType(entries), "risk_level": riskLevel(score)},
		SignalDistribution: distributionFor(entries),
	}
}

func distributionFor(entries []insightEntry) []signalDist {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Type]++
	}
	total := len(entries)
	if total == 0 {
		return nil
	}
	result := make([]signalDist, 0, len(counts))
	for typ, count := range counts {
		result = append(result, signalDist{Type: typ, Count: count, Pct: float64(count) / float64(total) * 100})
	}
	return result
}

func dominantType(entries []insightEntry) string {
	counts := map[string]int{}
	for _, entry := range entries {
		counts[entry.Type]++
	}
	best := ""
	bestCount := 0
	for typ, count := range counts {
		if count > bestCount {
			best = typ
			bestCount = count
		}
	}
	return best
}

func riskLevel(score int) string {
	if score >= 85 {
		return "low"
	}
	if score >= 65 {
		return "medium"
	}
	return "high"
}

func callLLM(ctx context.Context, provider, model, apiKey, prompt string) (string, error) {
	switch provider {
	case "openai", "cursor":
		return callOpenAICompatible(ctx, openAIEndpoint(provider), model, apiKey, prompt)
	case "anthropic":
		return callAnthropic(ctx, model, apiKey, prompt)
	case "gemini":
		return callGemini(ctx, model, apiKey, prompt)
	default:
		return "", fmt.Errorf("unsupported provider %q", provider)
	}
}

func openAIEndpoint(provider string) string {
	if provider == "cursor" {
		return "https://api.cursor.com/v1/chat/completions"
	}
	return "https://api.openai.com/v1/chat/completions"
}

func callOpenAICompatible(ctx context.Context, endpoint, model, apiKey, prompt string) (string, error) {
	payload := map[string]any{
		"model": model,
		"messages": []map[string]string{
			{"role": "system", "content": "Return only JSON. No markdown."},
			{"role": "user", "content": prompt},
		},
	}
	if modelSupportsCustomTemperature(model) {
		payload["temperature"] = 0.2
	}
	return postOpenAICompatible(ctx, endpoint, apiKey, payload)
}

func modelSupportsCustomTemperature(model string) bool {
	lower := strings.ToLower(model)
	// Reasoning and newer GPT families only accept the provider default temperature.
	restricted := []string{"o1", "o3", "o4", "gpt-5", "reasoning", "thinking"}
	for _, token := range restricted {
		if strings.Contains(lower, token) {
			return false
		}
	}
	return true
}

func postOpenAICompatible(ctx context.Context, endpoint, apiKey string, payload map[string]any) (string, error) {
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		bodyText := strings.TrimSpace(string(raw))
		if _, hasTemp := payload["temperature"]; hasTemp && strings.Contains(bodyText, "temperature") {
			delete(payload, "temperature")
			return postOpenAICompatible(ctx, endpoint, apiKey, payload)
		}
		return "", fmt.Errorf("provider returned %d: %s", res.StatusCode, bodyText)
	}
	var parsed struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Choices) == 0 {
		return "", fmt.Errorf("provider returned no choices")
	}
	return parsed.Choices[0].Message.Content, nil
}

func callAnthropic(ctx context.Context, model, apiKey, prompt string) (string, error) {
	payload := map[string]any{
		"model":      model,
		"max_tokens": 1800,
		"messages": []map[string]string{
			{"role": "user", "content": prompt},
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-api-key", apiKey)
	req.Header.Set("anthropic-version", "2023-06-01")
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("anthropic returned %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Content) == 0 {
		return "", fmt.Errorf("anthropic returned no content")
	}
	return parsed.Content[0].Text, nil
}

func callGemini(ctx context.Context, model, apiKey, prompt string) (string, error) {
	endpoint := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent", model)
	payload := map[string]any{
		"contents": []map[string]any{
			{"parts": []map[string]string{{"text": prompt}}},
		},
		"generationConfig": map[string]any{
			"responseMimeType": "application/json",
		},
	}
	body, _ := json.Marshal(payload)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("x-goog-api-key", apiKey)
	client := &http.Client{Timeout: 45 * time.Second}
	res, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	raw, _ := io.ReadAll(io.LimitReader(res.Body, 1<<20))
	if res.StatusCode >= 400 {
		return "", fmt.Errorf("gemini returned %d: %s", res.StatusCode, strings.TrimSpace(string(raw)))
	}
	var parsed struct {
		Candidates []struct {
			Content struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"content"`
		} `json:"candidates"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		return "", err
	}
	if len(parsed.Candidates) == 0 || len(parsed.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("gemini returned no candidates")
	}
	return parsed.Candidates[0].Content.Parts[0].Text, nil
}
