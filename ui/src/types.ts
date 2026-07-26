export type EntryType =
  | 'request'
  | 'query'
  | 'cache'
  | 'redis'
  | 'job'
  | 'event'
  | 'log'
  | 'exception'
  | 'schedule'
  | 'mail'
  | 'notification'
  | 'http-client'
  | 'websocket'
  | 'performance'
  | 'metric'
  | 'custom'
  | 'topic'

export interface Entry {
  id: string
  batch_id: string
  type: EntryType
  request_id?: string
  correlation_id?: string
  tags?: string[]
  content: Record<string, unknown>
  created_at: string
}

export interface ListResult {
  entries: Entry[]
  total: number
}

export interface SignalSetting {
  type: EntryType
  enabled: boolean
  count: number
}

export interface LLMInsightFinding {
  title: string
  detail: string
  severity: 'info' | 'warning' | 'critical'
}

export interface LLMInsightResponse {
  summary: string
  health_score: number
  findings: LLMInsightFinding[]
  recommendations: string[]
  metrics: Record<string, string | number>
  signal_distribution: Array<{ type: string; count: number; pct: number }>
}

export interface ContentTab {
  id: string
  label: string
  body: string
  json: boolean
}

export interface BatchTypeGroup {
  type: EntryType
  label: string
  entries: Entry[]
}

export interface EntryDetailResponse {
  entry: Entry
  batch: Entry[]
  batch_groups: BatchTypeGroup[]
  content_tabs: ContentTab[]
  related_active_tab: string
}
