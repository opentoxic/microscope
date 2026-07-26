import { computed, reactive, watch } from 'vue'
import { listLLMModels } from './api/client'
import type { EntryType } from './types'
import { signals } from './utils'

export type LLMProvider = 'openai' | 'gemini' | 'cursor' | 'anthropic'
export type LLMMode = 'manual'
export type LLMPeriod = '15m' | '1h' | '6h' | '24h' | '7d'

export interface LLMProviderOption {
  id: LLMProvider
  label: string
  description: string
}

export interface LLMModelOption {
  id: string
  label: string
  created_at?: string
}

export interface LLMInsightFinding {
  title: string
  detail: string
  severity: 'info' | 'warning' | 'critical'
}

export interface LLMInsightResult {
  summary: string
  health_score: number
  findings: LLMInsightFinding[]
  recommendations: string[]
  metrics: Record<string, string | number>
  signal_distribution: Array<{ type: string; count: number; pct: number }>
}

export const llmProviders: LLMProviderOption[] = [
  { id: 'openai', label: 'OpenAI', description: 'Live models from api.openai.com/v1/models' },
  { id: 'gemini', label: 'Google Gemini', description: 'Live models from generativelanguage.googleapis.com' },
  { id: 'cursor', label: 'Cursor', description: 'Live models from api.cursor.com/v1/models' },
  { id: 'anthropic', label: 'Anthropic', description: 'Live models from api.anthropic.com/v1/models' },
]

export const llmPeriods: Array<{ id: LLMPeriod; label: string; minutes: number }> = [
  { id: '15m', label: 'Last 15 minutes', minutes: 15 },
  { id: '1h', label: 'Last hour', minutes: 60 },
  { id: '6h', label: 'Last 6 hours', minutes: 360 },
  { id: '24h', label: 'Last 24 hours', minutes: 1440 },
  { id: '7d', label: 'Last 7 days', minutes: 10080 },
]

const STORAGE_KEY = 'signal-llm-settings'

export interface LLMSettings {
  enabled: boolean
  provider: LLMProvider
  model: string
  apiKey: string
  mode: LLMMode
  period: LLMPeriod
  dataTypes: EntryType[]
}

export const providerModels = reactive({
  items: [] as LLMModelOption[],
  loading: false,
  loaded: false,
  error: '',
  provider: '' as LLMProvider | '',
})

function defaultDataTypes(): EntryType[] {
  return signals
    .filter(signal => signal.type)
    .map(signal => signal.type as EntryType)
}

function loadStored(): LLMSettings {
  try {
    const stored = JSON.parse(localStorage.getItem(STORAGE_KEY) || '{}') as Partial<LLMSettings>
    const provider = llmProviders.find(item => item.id === stored.provider)?.id || 'openai'
    return {
      enabled: stored.enabled ?? false,
      provider,
      model: stored.model || '',
      apiKey: stored.apiKey || '',
      mode: 'manual',
      period: stored.period || '1h',
      dataTypes: stored.dataTypes?.length ? stored.dataTypes : defaultDataTypes(),
    }
  } catch {
    return {
      enabled: false,
      provider: 'openai',
      model: '',
      apiKey: '',
      mode: 'manual',
      period: '1h',
      dataTypes: defaultDataTypes(),
    }
  }
}

export const llmSettings = reactive<LLMSettings>(loadStored())

watch(llmSettings, (value) => {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(value))
}, { deep: true })

export const llmConfigured = computed(() => llmSettings.enabled && llmSettings.apiKey.trim().length > 0 && llmSettings.model.trim().length > 0)

export const activeProvider = computed(() => llmProviders.find(item => item.id === llmSettings.provider) || llmProviders[0])

let pendingModels: Promise<void> | null = null
let modelsDebounce: ReturnType<typeof setTimeout> | null = null

export function scheduleProviderModels(force = false) {
  if (modelsDebounce) clearTimeout(modelsDebounce)
  modelsDebounce = setTimeout(() => {
    void loadProviderModels(force)
  }, 350)
}

export function loadProviderModels(force = false): Promise<void> {
  const provider = llmSettings.provider
  const apiKey = llmSettings.apiKey.trim()
  if (!apiKey) {
    providerModels.items = []
    providerModels.loaded = false
    providerModels.error = ''
    providerModels.provider = provider
    return Promise.resolve()
  }
  if (pendingModels && !force && providerModels.provider === provider) return pendingModels

  providerModels.loading = true
  providerModels.error = ''
  pendingModels = listLLMModels(provider, apiKey)
    .then(({ models }) => {
      providerModels.items = models
      providerModels.loaded = true
      providerModels.provider = provider
      if (!models.some(model => model.id === llmSettings.model)) {
        llmSettings.model = models[0]?.id || ''
      }
    })
    .catch((error: Error) => {
      providerModels.items = []
      providerModels.loaded = false
      providerModels.error = error.message
      providerModels.provider = provider
    })
    .finally(() => {
      providerModels.loading = false
      pendingModels = null
    })
  return pendingModels
}

export function setLLMProvider(provider: LLMProvider) {
  if (llmSettings.provider === provider) return
  llmSettings.provider = provider
  providerModels.items = []
  providerModels.loaded = false
  providerModels.error = ''
  scheduleProviderModels(true)
}

export function toggleLLMDataType(type: EntryType) {
  const index = llmSettings.dataTypes.indexOf(type)
  if (index >= 0) {
    if (llmSettings.dataTypes.length > 1) {
      llmSettings.dataTypes.splice(index, 1)
    }
  } else {
    llmSettings.dataTypes.push(type)
  }
}

export function periodMinutes(period: LLMPeriod): number {
  return llmPeriods.find(item => item.id === period)?.minutes || 60
}
