import type { Entry, EntryType } from './types'

export interface SignalDefinition {
  type: EntryType | ''
  label: string
  shortLabel: string
  key: string
  color: string
  group: 'flow' | 'runtime' | 'output'
  available: boolean
}

export const signals: SignalDefinition[] = [
  { type: '', label: 'All activity', shortLabel: 'Overview', key: 'A', color: '#c4ced0', group: 'flow', available: true },
  { type: 'request', label: 'HTTP requests', shortLabel: 'HTTP', key: 'R', color: '#4c9fff', group: 'flow', available: true },
  { type: 'query', label: 'SQL queries', shortLabel: 'SQL', key: 'Q', color: '#28e0a0', group: 'flow', available: true },
  { type: 'http-client', label: 'External calls', shortLabel: 'Calls', key: 'X', color: '#4c9fff', group: 'flow', available: true },
  { type: 'cache', label: 'Cache', shortLabel: 'Cache', key: 'C', color: '#ffb84d', group: 'runtime', available: true },
  { type: 'redis', label: 'Redis', shortLabel: 'Redis', key: 'D', color: '#ffb84d', group: 'runtime', available: true },
  { type: 'job', label: 'Queue jobs', shortLabel: 'Jobs', key: 'J', color: '#a879ff', group: 'runtime', available: true },
  { type: 'topic', label: 'Redpanda topics', shortLabel: 'Topics', key: 'K', color: '#ff7448', group: 'runtime', available: true },
  { type: 'schedule', label: 'Scheduled tasks', shortLabel: 'Tasks', key: 'T', color: '#a879ff', group: 'runtime', available: true },
  { type: 'event', label: 'Events', shortLabel: 'Events', key: 'E', color: '#a879ff', group: 'runtime', available: true },
  { type: 'websocket', label: 'WebSockets', shortLabel: 'Sockets', key: 'W', color: '#20d9ee', group: 'runtime', available: true },
  { type: 'log', label: 'Logs', shortLabel: 'Logs', key: 'L', color: '#c4ced0', group: 'output', available: true },
  { type: 'exception', label: 'Exceptions', shortLabel: 'Errors', key: '!', color: '#ff476f', group: 'output', available: true },
  { type: 'mail', label: 'Mail', shortLabel: 'Mail', key: 'M', color: '#ffb84d', group: 'output', available: true },
  { type: 'notification', label: 'Notifications', shortLabel: 'Notify', key: 'N', color: '#ffb84d', group: 'output', available: true },
  { type: 'performance', label: 'Performance', shortLabel: 'Perf', key: 'P', color: '#20d9ee', group: 'output', available: true },
  { type: 'metric', label: 'Metrics', shortLabel: 'Metrics', key: 'I', color: '#20d9ee', group: 'output', available: true },
  { type: 'custom', label: 'Custom events', shortLabel: 'Custom', key: 'U', color: '#c4ced0', group: 'output', available: true },
]

export const watchers = signals
export const typeTitles = Object.fromEntries(signals.map(signal => [signal.type, signal.label]))

export function signalFor(type: string): SignalDefinition {
  return signals.find(signal => signal.type === type) || signals[0]
}

export function timeAgo(iso: string): string {
  const seconds = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 1000))
  if (seconds < 5) return 'now'
  if (seconds < 60) return `${seconds}s`
  const minutes = Math.floor(seconds / 60)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}

export function formatTime(iso: string): string {
  return new Date(iso).toLocaleString()
}

export function formatClock(iso: string): string {
  return new Date(iso).toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
}

export function formatRecordDate(iso: string): string {
  return new Date(iso).toLocaleDateString('en-US', { month: 'short', day: 'numeric', year: 'numeric' })
}

export function formatRecordTime(iso: string): string {
  const date = new Date(iso)
  const clock = date.toLocaleTimeString('en-US', { hour: '2-digit', minute: '2-digit', second: '2-digit', hour12: false })
  const ms = date.getMilliseconds()
  return ms ? `${clock}.${String(ms).padStart(3, '0')}` : clock
}

export function formatTimeLong(iso: string): string {
  return new Date(iso).toLocaleString('en-US', {
    month: 'short',
    day: 'numeric',
    year: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}

export function typeBadgeClass(type: EntryType): string {
  const signal = signalFor(type)
  return 'signal-badge'
    + (signal.type === 'exception' ? ' signal-badge--error' : '')
}

export function methodClass(method: string): string {
  const value = (method || '').toUpperCase()
  if (value === 'DELETE') return 'signal-badge signal-badge--error'
  if (value === 'POST' || value === 'PUT' || value === 'PATCH') return 'signal-badge signal-badge--blue'
  return 'signal-badge'
}

export function statusClass(status: unknown): string {
  const code = Number(status)
  if (code >= 500 || !code) return 'signal-badge signal-badge--error'
  if (code >= 400) return 'signal-badge signal-badge--amber'
  if (code >= 300) return 'signal-badge signal-badge--blue'
  return 'signal-badge signal-badge--success'
}

export function levelClass(level: string): string {
  const value = (level || 'info').toLowerCase()
  if (value === 'error' || value === 'fatal') return 'signal-badge signal-badge--error'
  if (value === 'warn' || value === 'warning') return 'signal-badge signal-badge--amber'
  return 'signal-badge'
}

export function summarize(entry: Entry): string {
  const content = entry.content || {}
  switch (entry.type) {
    case 'request': return String(content.path || content.uri || 'Incoming request')
    case 'query': return truncate(String(content.sql || 'SQL query'), 110)
    case 'log': return String(content.message || 'Log entry')
    case 'event': return String(content.event_type || content.name || 'Application event')
    case 'notification': return [content.kind, content.email].filter(Boolean).join(' → ') || 'Notification'
    case 'exception': return String(content.message || 'Unhandled exception')
    case 'cache': return String(content.key || content.operation || 'Cache operation')
    case 'redis': return String(content.command || 'Redis command')
    case 'job': return String(content.name || content.job || 'Queue job')
    case 'http-client': return String(content.url || 'External request')
    case 'schedule': return String(content.name || 'Scheduled task')
    case 'mail': return String(content.subject || 'Mail delivery')
    case 'websocket': return `${content.event || 'message'} · ${content.channel || 'socket'}`
    case 'performance': return String(content.name || 'Performance span')
    case 'metric': return `${metricLanguageLabel(entry)} · ${content.name || 'Metric'} ${content.value ?? ''} ${content.unit || ''}`.trim()
    case 'custom': return String(content.name || 'Custom event')
    case 'topic': return `${content.action || 'activity'} · ${content.topic || 'topic'}`
    default: return String(content.name || entry.type)
  }
}

export function entryMeta(entry: Entry): string {
  const content = entry.content || {}
  switch (entry.type) {
    case 'request': return `${String(content.method || 'GET').toUpperCase()} · ${content.status || '—'}`
    case 'query': return String(content.connection || 'database')
    case 'log': return String(content.level || 'info')
    case 'exception': return String(content.kind || content.type || 'runtime')
    case 'notification': return String(content.kind || 'notification')
    case 'cache': return `${content.operation || 'operation'} · ${content.hit === true ? 'hit' : content.hit === false ? 'miss' : 'cache'}`
    case 'redis': return String(content.command || 'command')
    case 'job': return `${content.queue || 'default'} · ${content.state || 'observed'}`
    case 'schedule': return String(content.state || 'executed')
    case 'mail': return String(content.state || 'sent')
    case 'http-client': return `${content.method || 'GET'} · ${content.status || '—'}`
    case 'websocket': return String(content.direction || 'activity')
    case 'performance': return 'performance span'
    case 'metric': return `${metricLanguageLabel(entry)} · ${content.unit || 'sample'}`
    case 'custom': return 'custom record'
    case 'topic': return `${content.partition != null ? `partition ${content.partition}` : 'Redpanda'} · ${content.message_count || 1} msg`
    default: return entry.type
  }
}

export const KNOWN_METRIC_LANGUAGES = ['go', 'python', 'node', 'ruby', 'php', 'elixir'] as const
export type MetricLanguage = (typeof KNOWN_METRIC_LANGUAGES)[number] | 'unknown'

const METRIC_LANGUAGE_LABELS: Record<string, string> = {
  go: 'Go',
  python: 'Python',
  node: 'TypeScript / Node.js',
  ruby: 'Ruby',
  php: 'PHP',
  elixir: 'Elixir',
}

export const METRIC_LANGUAGE_COLORS: Record<MetricLanguage, string> = {
  go: '#00add8',
  python: '#ffd343',
  node: '#4f8cff',
  ruby: '#ff5a58',
  php: '#9b9de2',
  elixir: '#b56cff',
  unknown: '#62d8c6',
}

function isKnownMetricLanguage(value: string): value is (typeof KNOWN_METRIC_LANGUAGES)[number] {
  return (KNOWN_METRIC_LANGUAGES as readonly string[]).includes(value)
}

function normalizeMetricLanguage(value: string): string {
  const normalized = value.trim().toLowerCase()
  if (['typescript', 'javascript', 'nodejs', 'node.js', 'ts', 'js'].includes(normalized)) return 'node'
  if (normalized === 'golang') return 'go'
  if (normalized === 'py') return 'python'
  if (normalized === 'rb') return 'ruby'
  if (normalized === 'ex' || normalized === 'beam') return 'elixir'
  return normalized
}

/**
 * Detects which language reported a metric entry. Prefers the explicit
 * `language` field SDKs set, and falls back to the "<language>.runtime"
 * naming convention for older entries that predate it.
 */
export function detectMetricLanguage(entry: Entry): MetricLanguage {
  const content = entry.content || {}
  const explicit = normalizeMetricLanguage(String(content.language || ''))
  if (isKnownMetricLanguage(explicit)) return explicit

  const name = String(content.name || '')
  const prefix = normalizeMetricLanguage(name.split('.')[0] || '')
  if (isKnownMetricLanguage(prefix)) return prefix

  return 'unknown'
}

export function metricLanguageLabel(entry: Entry): string {
  return METRIC_LANGUAGE_LABELS[detectMetricLanguage(entry)] || 'Unknown runtime'
}

export function metricLanguageColor(entry: Entry): string {
  return METRIC_LANGUAGE_COLORS[detectMetricLanguage(entry)]
}

export function entrySignalColor(entry: Entry): string {
  return entry.type === 'metric' ? metricLanguageColor(entry) : signalFor(entry.type).color
}

/** A human label for a metric entry's concurrency unit, e.g. "Goroutines", "Threads". */
export function metricUnitLabel(entry: Entry): string {
  const unit = String(entry.content?.unit || '')
  if (!unit) return 'Value'
  return unit.charAt(0).toUpperCase() + unit.slice(1)
}

export function entryDuration(entry: Entry): number {
  return Number(entry.content?.duration_ms || 0)
}

export function isError(entry: Entry): boolean {
  const status = Number(entry.content?.status || 0)
  const level = String(entry.content?.level || '').toLowerCase()
  return entry.type === 'exception' || status >= 500 || level === 'error' || level === 'fatal'
}

export function truncate(value: string, length: number): string {
  return value.length > length ? `${value.slice(0, length)}…` : value
}
