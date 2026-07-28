import { computed, reactive } from 'vue'
import { getSignalSettings, getRedactionSetting, updateRedactionSetting, updateSignalSetting } from './api/client'
import type { EntryType, SignalSetting } from './types'
import { signals } from './utils'

export const signalSettings = reactive({
  loaded: false,
  loading: false,
  values: {} as Record<string, SignalSetting>,
  error: '',
})

export const redactionSettings = reactive({
  loaded: false,
  loading: false,
  enabled: false,
  error: '',
  pending: false,
})

let pendingLoad: Promise<void> | null = null
let pendingRedactionLoad: Promise<void> | null = null

export function signalEnabled(type: string): boolean {
  if (!type) return true
  return signalSettings.values[type]?.enabled !== false
}

export const enabledSignals = computed(() => signals.filter(signal => signalEnabled(signal.type)))

export function loadSignalSettings(force = false): Promise<void> {
  if (pendingLoad && !force) return pendingLoad
  signalSettings.loading = true
  signalSettings.error = ''
  pendingLoad = getSignalSettings()
    .then(({ settings }) => {
      signalSettings.values = Object.fromEntries(settings.map(setting => [setting.type, setting]))
      signalSettings.loaded = true
    })
    .catch((error: Error) => {
      signalSettings.error = error.message
    })
    .finally(() => {
      signalSettings.loading = false
      pendingLoad = null
    })
  return pendingLoad
}

export async function setSignalEnabled(type: EntryType, enabled: boolean) {
  const result = await updateSignalSetting(type, enabled)
  const current = signalSettings.values[type]
  signalSettings.values[type] = {
    type,
    enabled: result.enabled,
    count: result.enabled ? (current?.count || 0) : 0,
  }
  return result
}

export function loadRedactionSettings(force = false): Promise<void> {
  if (pendingRedactionLoad && !force) return pendingRedactionLoad
  redactionSettings.loading = true
  redactionSettings.error = ''
  pendingRedactionLoad = getRedactionSetting()
    .then(({ enabled }) => {
      redactionSettings.enabled = enabled
      redactionSettings.loaded = true
    })
    .catch((error: Error) => {
      redactionSettings.error = error.message
    })
    .finally(() => {
      redactionSettings.loading = false
      pendingRedactionLoad = null
    })
  return pendingRedactionLoad
}

export async function setRedactionEnabled(enabled: boolean) {
  const result = await updateRedactionSetting(enabled)
  redactionSettings.enabled = result.enabled
  return result
}
