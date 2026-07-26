import { computed, reactive } from 'vue'
import { getSignalSettings, updateSignalSetting } from './api/client'
import type { EntryType, SignalSetting } from './types'
import { signals } from './utils'

export const signalSettings = reactive({
  loaded: false,
  loading: false,
  values: {} as Record<string, SignalSetting>,
  error: '',
})

let pendingLoad: Promise<void> | null = null

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
