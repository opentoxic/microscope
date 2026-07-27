package microscope

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// ContentTab is a tab panel on the entry detail page.
type ContentTab struct {
	ID    string `json:"id"`
	Label string `json:"label"`
	Body  string `json:"body"`
	JSON  bool   `json:"json"`
}

// EntryDetailResponse is the enriched detail API payload.
type EntryDetailResponse struct {
	Entry            Entry            `json:"entry"`
	Batch            []Entry          `json:"batch"`
	BatchGroups      []BatchTypeGroup `json:"batch_groups"`
	ContentTabs      []ContentTab     `json:"content_tabs"`
	RelatedActiveTab string           `json:"related_active_tab"`
}

func buildEntryDetail(entry *Entry, batch []Entry) EntryDetailResponse {
	batchGroups := groupBatchRelated(batch, entry.ID)
	return EntryDetailResponse{
		Entry:            *entry,
		Batch:            batch,
		BatchGroups:      batchGroups,
		ContentTabs:      entryContentTabs(*entry),
		RelatedActiveTab: firstRelatedTabType(batchGroups),
	}
}

func entryContentTabs(e Entry) []ContentTab {
	c := e.Content
	if c == nil {
		c = map[string]any{}
	}

	var tabs []ContentTab
	switch e.Type {
	case TypeRequest:
		if v := contentString(c["request_body"]); v != "" && v != "null" && v != "{}" {
			tabs = append(tabs, ContentTab{ID: "payload", Label: "Payload", Body: prettyContent(v), JSON: looksLikeJSON(v)})
		}
		if h, ok := c["headers"]; ok && h != nil {
			tabs = append(tabs, ContentTab{ID: "headers", Label: "Headers", Body: jsonPrettyAny(h), JSON: true})
		}
		if v := contentString(c["response_body"]); v != "" && v != "null" {
			tabs = append(tabs, ContentTab{ID: "response", Label: "Response", Body: prettyContent(v), JSON: looksLikeJSON(v)})
		}
	case TypeQuery:
		if v := contentString(c["sql"]); v != "" {
			tabs = append(tabs, ContentTab{ID: "query", Label: "Query", Body: v})
		}
	case TypeException:
		if v := contentString(c["stack"]); v != "" {
			tabs = append(tabs, ContentTab{ID: "stack", Label: "Stack Trace", Body: v})
		}
	case TypeLog, TypeEvent, TypeNotification, TypeCache, TypeRedis, TypeJob,
		TypeSchedule, TypeMail, TypeHTTPClient, TypeWebSocket, TypePerformance,
		TypeMetric, TypeCustom:
	case TypeTopic:
		tabs = append(tabs, ContentTab{ID: "message", Label: "Message metadata", Body: jsonPretty(c), JSON: true})
	default:
		tabs = append(tabs, ContentTab{ID: "payload", Label: "Payload", Body: jsonPretty(c), JSON: true})
	}

	if len(tabs) == 0 {
		tabs = append(tabs, ContentTab{ID: "payload", Label: "Payload", Body: jsonPretty(c), JSON: true})
	}
	return tabs
}

func groupBatchRelated(batch []Entry, currentID string) []BatchTypeGroup {
	filtered := make([]Entry, 0, len(batch))
	for _, e := range batch {
		if e.ID != currentID {
			filtered = append(filtered, e)
		}
	}
	return groupBatchByType(filtered)
}

func firstRelatedTabType(groups []BatchTypeGroup) string {
	if len(groups) == 0 {
		return ""
	}
	return string(groups[0].Type)
}

func batchGroupSummary(g BatchTypeGroup) string {
	n := len(g.Entries)
	switch g.Type {
	case TypeQuery:
		dup := countDuplicateQueries(g.Entries)
		word := "queries"
		if n == 1 {
			word = "query"
		}
		return fmt.Sprintf("%d %s, %d of which are duplicated.", n, word, dup)
	case TypeLog:
		word := "logs"
		if n == 1 {
			word = "log"
		}
		return fmt.Sprintf("%d %s", n, word)
	case TypeRequest:
		word := "requests"
		if n == 1 {
			word = "request"
		}
		return fmt.Sprintf("%d %s", n, word)
	default:
		word := strings.ToLower(g.Label)
		return fmt.Sprintf("%d %s", n, word)
	}
}

func batchGroupTotalDuration(g BatchTypeGroup) string {
	if g.Type != TypeQuery {
		return ""
	}
	var total float64
	for _, e := range g.Entries {
		total += contentFloat(e.Content["duration_ms"])
	}
	if total < 1 {
		return fmt.Sprintf("%.2fms", total)
	}
	return fmt.Sprintf("%.0fms", total)
}

func countDuplicateQueries(entries []Entry) int {
	seen := make(map[string]int)
	dup := 0
	for _, e := range entries {
		sql := contentString(e.Content["sql"])
		seen[sql]++
		if seen[sql] == 2 {
			dup++
		}
	}
	return dup
}

func formatTimeLong(t time.Time) string {
	day := t.Day()
	suffix := "th"
	switch day % 10 {
	case 1:
		if day != 11 {
			suffix = "st"
		}
	case 2:
		if day != 12 {
			suffix = "nd"
		}
	case 3:
		if day != 13 {
			suffix = "rd"
		}
	}
	return t.Local().Format("January ") + fmt.Sprintf("%d%s", day, suffix) + t.Format(", 2006 3:04:05 PM")
}

func prettyContent(v string) string {
	if looksLikeJSON(v) {
		return prettyJSONString(v)
	}
	return v
}

func looksLikeJSON(v string) bool {
	v = strings.TrimSpace(v)
	return json.Valid([]byte(v))
}

func prettyJSONString(v string) string {
	v = strings.TrimSpace(v)
	var parsed any
	if err := json.Unmarshal([]byte(v), &parsed); err != nil {
		return v
	}
	b, err := json.MarshalIndent(parsed, "", "  ")
	if err != nil {
		return v
	}
	return string(b)
}

func jsonPrettyAny(v any) string {
	switch val := v.(type) {
	case map[string]any:
		return jsonPretty(val)
	case map[string][]string:
		m := make(map[string]any, len(val))
		for k, vv := range val {
			m[k] = vv
		}
		return jsonPretty(m)
	default:
		b, err := json.MarshalIndent(v, "", "  ")
		if err != nil {
			if s := contentString(v); s != "" {
				return s
			}
			return "{}"
		}
		return string(b)
	}
}
