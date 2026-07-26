package microscope

// BatchTypeGroup groups related batch entries by watcher type.
type BatchTypeGroup struct {
	Type    EntryType `json:"type"`
	Label   string    `json:"label"`
	Entries []Entry   `json:"entries"`
}

var batchTypeOrder = []EntryType{
	TypeRequest,
	TypeQuery,
	TypeLog,
	TypeEvent,
	TypeNotification,
	TypeException,
	TypeCache,
	TypeRedis,
	TypeJob,
	TypeSchedule,
	TypeMail,
	TypeHTTPClient,
	TypeWebSocket,
	TypePerformance,
	TypeMetric,
	TypeCustom,
	TypeTopic,
}

func typeLabel(t EntryType) string {
	switch t {
	case TypeRequest:
		return "Requests"
	case TypeQuery:
		return "Queries"
	case TypeLog:
		return "Logs"
	case TypeEvent:
		return "Events"
	case TypeNotification:
		return "Notifications"
	case TypeException:
		return "Exceptions"
	case TypeCache:
		return "Cache"
	case TypeRedis:
		return "Redis"
	case TypeJob:
		return "Queue Jobs"
	case TypeSchedule:
		return "Scheduled Tasks"
	case TypeMail:
		return "Mail"
	case TypeHTTPClient:
		return "External Calls"
	case TypeWebSocket:
		return "WebSockets"
	case TypePerformance:
		return "Performance"
	case TypeMetric:
		return "Metrics"
	case TypeCustom:
		return "Custom Events"
	case TypeTopic:
		return "Redpanda Topics"
	default:
		return string(t)
	}
}

func groupBatchByType(batch []Entry) []BatchTypeGroup {
	byType := make(map[EntryType][]Entry, len(batchTypeOrder))
	for _, e := range batch {
		byType[e.Type] = append(byType[e.Type], e)
	}
	groups := make([]BatchTypeGroup, 0, len(batchTypeOrder))
	for _, t := range batchTypeOrder {
		entries := byType[t]
		if len(entries) == 0 {
			continue
		}
		groups = append(groups, BatchTypeGroup{
			Type:    t,
			Label:   typeLabel(t),
			Entries: entries,
		})
	}
	return groups
}
