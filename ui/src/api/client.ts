import type { Entry, EntryDetailResponse, EntryType, ListResult, LLMInsightResponse, SignalSetting, StorageUsage } from '../types'
import { demoDetail, demoEntries, demoList, demoSettings, demoStorageUsage } from '../demo'

const API = '/microscope/api'
export const demoMode = import.meta.env.VITE_DEMO_MODE === 'true'
let demoRecordingPaused = false
let demoRedactionEnabled = false

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const res = await fetch(API + path, init)
  if (!res.ok) {
    const payload = await res.json().catch(() => null) as { error?: string } | null
    throw new Error(payload?.error || `API error ${res.status}`)
  }
  return res.json() as Promise<T>
}

export function listEntries(params: URLSearchParams) {
  if (demoMode) return Promise.resolve(demoList(params))
  return request<ListResult>(`/entries?${params}`)
}

export function getEntry(id: string) {
  if (demoMode) return Promise.resolve(demoDetail(id))
  return request<EntryDetailResponse>(`/entries/${encodeURIComponent(id)}`)
}

export function pruneEntries() {
  if (demoMode) return Promise.resolve({ deleted: demoEntries.length })
  return request<{ deleted: number }>('/prune', { method: 'POST' })
}

export function getSignalSettings() {
  if (demoMode) return Promise.resolve({ settings: demoSettings() })
  return request<{ settings: SignalSetting[] }>('/settings')
}

export function getStorageUsage() {
  if (demoMode) return Promise.resolve(demoStorageUsage())
  return request<StorageUsage>('/storage')
}

export function getRecordingState() {
  if (demoMode) return Promise.resolve({ paused: demoRecordingPaused })
  return request<{ paused: boolean }>('/recording')
}

export function setRecordingPaused(paused: boolean) {
  if (demoMode) {
    demoRecordingPaused = paused
    return Promise.resolve({ paused: demoRecordingPaused })
  }
  return request<{ paused: boolean }>('/recording', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ paused }),
  })
}

export function getRedactionSetting() {
  if (demoMode) return Promise.resolve({ enabled: demoRedactionEnabled })
  return request<{ enabled: boolean }>('/redaction')
}

export function updateRedactionSetting(enabled: boolean) {
  if (demoMode) {
    demoRedactionEnabled = enabled
    return Promise.resolve({ enabled: demoRedactionEnabled })
  }
  return request<{ enabled: boolean }>('/redaction', {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  })
}

export function updateSignalSetting(type: EntryType, enabled: boolean) {
  if (demoMode) return Promise.resolve({ type, enabled, deleted: enabled ? 0 : demoEntries.filter(entry => entry.type === type).length })
  return request<{ type: EntryType; enabled: boolean; deleted: number }>(`/settings/${encodeURIComponent(type)}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ enabled }),
  })
}

export function analyzeInsights(payload: {
  provider: string
  model: string
  api_key: string
  period: string
  context?: string
  entries: Array<{ id: string; type: string; created_at: string; content: Record<string, unknown> }>
}) {
  return request<LLMInsightResponse>('/insights/analyze', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(payload),
  })
}

export function listLLMModels(provider: string, apiKey: string) {
  return request<{ models: Array<{ id: string; label: string; created_at?: string }> }>('/insights/models', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ provider, api_key: apiKey }),
  })
}

export function createCustomEntry(name: string, content: Record<string, unknown> = {}) {
  if (demoMode) return Promise.resolve({ id: `demo-marker-${Date.now()}` })
  return request<{ id: string }>('/entries', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, content }),
  })
}

export function subscribeEntries(
  onEntry: (entry: Entry) => void,
  onState?: (connected: boolean) => void,
  onControl?: (event: { action: string; type?: EntryType; deleted: number; paused?: boolean; redact_sensitive?: boolean }) => void,
) {
  if (demoMode) {
    onState?.(true)
    let index = 0
    const timer = window.setInterval(() => {
      if (demoRecordingPaused) return
      const source = demoEntries[index++ % demoEntries.length]
      onEntry({ ...source, id: `${source.id}-live-${index}`, created_at: new Date().toISOString() })
    }, 6000)
    return () => window.clearInterval(timer)
  }
  const source = new EventSource(`${API}/stream`)
  source.onopen = () => onState?.(true)
  source.onerror = () => onState?.(false)
  source.addEventListener('entry', (event) => {
    try {
      onEntry(JSON.parse((event as MessageEvent).data) as Entry)
    } catch {
      // Ignore a malformed event and keep the live channel open.
    }
  })
  source.addEventListener('control', (event) => {
    try {
      onControl?.(JSON.parse((event as MessageEvent).data))
    } catch {
      // Keep listening if a control message is malformed.
    }
  })
  return () => source.close()
}
