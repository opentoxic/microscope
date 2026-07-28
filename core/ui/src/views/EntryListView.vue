<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getEntry, getRecordingState, getStorageUsage, listEntries, pruneEntries, setRecordingPaused, subscribeEntries } from '../api/client'
import type { Entry, EntryType, StorageUsage } from '../types'
import AppShell from '../components/AppShell.vue'
import EntryTable from '../components/EntryTable.vue'
import LLMInsightPanel from '../components/LLMInsightPanel.vue'
import SignalIcon from '../components/SignalIcon.vue'
import SystemVisuals from '../components/SystemVisuals.vue'
import { entryDuration, isError, signalFor } from '../utils'
import { enabledSignals, loadSignalSettings, signalEnabled } from '../settings'
import { askConfirm } from '../confirm'
import { showToast } from '../toast'
import { llmSettings } from '../llmSettings'

const route = useRoute()
const router = useRouter()
const entries = ref<Entry[]>([])
const total = ref(0)
const status = ref('Opening live channel…')
const search = ref(String(route.query.search || ''))
const loading = ref(false)
const clearing = ref(false)
const storagePruning = ref(false)
const recordingPaused = ref(false)
const recordingToggling = ref(false)
const streamConnected = ref(false)
const initialView = String(route.query.view || 'all')
const filter = ref<'all' | 'errors' | 'slow'>(initialView === 'errors' || initialView === 'slow' ? initialView : 'all')
interface SavedFilter {
  name: string
  type: string
  filter: 'all' | 'errors' | 'slow'
  search: string
  signals?: string[]
  tags?: string[]
}
const savedFilters = ref<SavedFilter[]>(loadSavedFilters())
const savedOpen = ref(false)
const selectedTypes = ref<string[]>(String(route.query.signals || '').split(',').filter(Boolean))
const selectedTags = ref<string[]>(String(route.query.tags || '').split(',').filter(Boolean))
const currentType = ref(String(route.query.type || ''))
const storageUsage = ref<StorageUsage | null>(null)
const filtersOpen = ref(false)
let timer: ReturnType<typeof setInterval> | null = null
let debounce: ReturnType<typeof setTimeout>
let stopStream: (() => void) | null = null
let loadSeq = 0

const visibleEntries = computed(() => {
  let result = entries.value
  if (currentType.value) result = result.filter(entry => entry.type === currentType.value)
  if (selectedTypes.value.length) result = result.filter(entry => selectedTypes.value.includes(entry.type))
  if (selectedTags.value.length) result = result.filter(entry => selectedTags.value.every(tag => entry.tags?.includes(tag)))
  if (filter.value === 'errors') result = result.filter(isError)
  if (filter.value === 'slow') result = result.filter(entry => entryDuration(entry) >= 500)
  return result
})
const availableTags = computed(() => Array.from(new Set(entries.value.flatMap(entry => entry.tags || []))).slice(0, 12))
const errorCount = computed(() => entries.value.filter(isError).length)
const slowEntries = computed(() => entries.value.filter(entry => entryDuration(entry) >= 500))
const durations = computed(() => entries.value.map(entryDuration).filter(Boolean))
const p95 = computed(() => {
  if (!durations.value.length) return '—'
  const sorted = [...durations.value].sort((a, b) => a - b)
  return `${sorted[Math.min(sorted.length - 1, Math.floor(sorted.length * .95))]}ms`
})
const averageDuration = computed(() => durations.value.length
  ? `${Math.round(durations.value.reduce((sum, duration) => sum + duration, 0) / durations.value.length)}ms`
  : '—')
const maximumDuration = computed(() => durations.value.length ? `${Math.max(...durations.value)}ms` : '—')
const batchCount = computed(() => new Set(entries.value.map(entry => entry.batch_id)).size)
const typeCounts = computed(() => enabledSignals.value.filter(signal => signal.available && signal.type).map(signal => ({
  ...signal,
  count: entries.value.filter(entry => entry.type === signal.type).length,
})).filter(item => item.count).sort((a, b) => b.count - a.count))
const filterSignals = computed(() => enabledSignals.value.filter(signal => signal.type))
const typeCountMap = computed(() => Object.fromEntries(typeCounts.value.map(item => [item.type, item.count])))
const recorderGroups = computed(() => ([
  { id: 'flow', label: 'Flow', signals: filterSignals.value.filter(signal => signal.group === 'flow') },
  { id: 'runtime', label: 'Runtime', signals: filterSignals.value.filter(signal => signal.group === 'runtime') },
  { id: 'output', label: 'Output', signals: filterSignals.value.filter(signal => signal.group === 'output') },
] as const).filter(group => group.signals.length))
const hasActiveFilters = computed(() => filter.value !== 'all'
  || route.query.bookmarked === '1'
  || selectedTypes.value.length > 0
  || selectedTags.value.length > 0
  || !!search.value)
const activeFilterCount = computed(() => {
  let count = selectedTypes.value.length + selectedTags.value.length
  if (filter.value !== 'all') count += 1
  if (route.query.bookmarked === '1') count += 1
  if (search.value) count += 1
  return count
})
const maxTypeCount = computed(() => Math.max(1, ...typeCounts.value.map(item => item.count)))
const signalGradient = computed(() => {
  const totalSignals = typeCounts.value.reduce((sum, item) => sum + item.count, 0)
  if (!totalSignals) return 'conic-gradient(#233137 0 100%)'
  let cursor = 0
  const stops = typeCounts.value.map((item) => {
    const start = cursor
    cursor += item.count / totalSignals * 100
    return `${item.color} ${start}% ${cursor}%`
  })
  return `conic-gradient(${stops.join(', ')})`
})
const errorRate = computed(() => entries.value.length ? Math.round(errorCount.value / entries.value.length * 100) : 0)
const typeMixTotal = computed(() => typeCounts.value.reduce((sum, item) => sum + item.count, 0))
const attentionCount = computed(() => errorCount.value + slowEntries.value.length)
const attentionItems = computed(() => [...entries.value.filter(isError), ...slowEntries.value].slice(0, 4))
const pulseHealth = computed(() => {
  if (errorCount.value > 5 || errorRate.value > 15) return { label: 'Critical', tone: 'critical' as const }
  if (errorCount.value > 0 || slowEntries.value.length > 3 || errorRate.value > 5) return { label: 'Elevated', tone: 'warning' as const }
  if (!entries.value.length) return { label: 'Idle', tone: 'idle' as const }
  return { label: 'Healthy', tone: 'healthy' as const }
})
const pulseMessage = computed(() => {
  if (!entries.value.length) return 'Waiting for the first records from your application.'
  if (pulseHealth.value.tone === 'critical') return 'Error volume is elevated. Review failing operations in the attention queue.'
  if (pulseHealth.value.tone === 'warning') return 'Some operations need review. Monitor errors and slow paths.'
  return 'Recorder is receiving records normally. No sustained pressure detected.'
})
const sparkBars = computed(() => {
  const recent = [...entries.value]
    .sort((a, b) => +new Date(a.created_at) - +new Date(b.created_at))
    .slice(-32)
  const maxDuration = Math.max(1, ...recent.map(entryDuration))
  const bars = recent.map((entry) => ({
    height: 6 + Math.round((entryDuration(entry) / maxDuration) * 40),
    color: isError(entry) ? '#ff5f7e' : signalFor(entry.type).color,
  }))
  const padding = Array.from({ length: Math.max(0, 32 - bars.length) }, () => ({
    height: 4,
    color: 'rgba(37, 50, 57, .7)',
    empty: true,
  }))
  return [...padding, ...bars.map((bar) => ({ ...bar, empty: false }))]
})
const selectedSignal = computed(() => signalFor(currentType.value))

function formatStorageMB(mb: number) {
  if (mb >= 1) return `${mb.toFixed(2)} MB`
  return `${Math.max(1, Math.round(mb * 1024))} KB`
}

const storageTotalLabel = computed(() => storageUsage.value ? formatStorageMB(storageUsage.value.total_mb) : '—')
const storageEntriesLabel = computed(() => storageUsage.value ? formatStorageMB(storageUsage.value.entries_mb) : '—')
const storageEntryCountLabel = computed(() => storageUsage.value ? storageUsage.value.entry_count.toLocaleString() : '—')
const storageEntriesBreakdown = computed(() => {
  if (!storageUsage.value) return 'heap and indexes'
  return `${formatStorageMB(storageUsage.value.entries_data_mb)} data · ${formatStorageMB(storageUsage.value.entries_indexes_mb)} indexes`
})
const storageHasRecords = computed(() => (storageUsage.value?.entry_count ?? 0) > 0)

watch(() => [route.query.type, route.query.search, route.query.view, route.query.signals, route.query.tags], ([type, routeSearch, view, routeSignals, routeTags]) => {
  currentType.value = String(type || '')
  search.value = String(routeSearch || '')
  filter.value = view === 'errors' || view === 'slow' ? view : 'all'
  if (currentType.value) {
    selectedTypes.value = []
  } else {
    selectedTypes.value = String(routeSignals || '').split(',').filter(Boolean)
  }
  selectedTags.value = String(routeTags || '').split(',').filter(Boolean)
  load()
}, { immediate: true })

function buildStreamQuery(overrides?: {
  type?: string
  search?: string
  view?: string
  signals?: string[]
  tags?: string[]
  bookmarked?: string
}): Record<string, string> {
  const query: Record<string, string> = {}
  const type = overrides?.type ?? currentType.value
  const qSearch = overrides?.search ?? search.value
  const qView = overrides?.view ?? filter.value
  const qSignals = overrides?.signals ?? selectedTypes.value
  const qTags = overrides?.tags ?? selectedTags.value

  if (type) query.type = type
  if (qSearch) query.search = qSearch
  if (qView && qView !== 'all') query.view = qView
  if (!type && qSignals.length) query.signals = qSignals.join(',')
  if (qTags.length) query.tags = qTags.join(',')
  if (overrides?.bookmarked) query.bookmarked = overrides.bookmarked
  return query
}

function replaceStreamQuery(overrides?: Parameters<typeof buildStreamQuery>[0]) {
  router.replace({ path: '/', query: buildStreamQuery(overrides) })
}

function resolveFetchTypes(): string[] {
  if (currentType.value) return [currentType.value]
  if (selectedTypes.value.length === 1) return [selectedTypes.value[0]]
  if (selectedTypes.value.length > 1) return selectedTypes.value
  return []
}

async function fetchEntries(offset = 0): Promise<{ entries: Entry[]; total: number }> {
  const types = resolveFetchTypes()
  const limit = 80

  if (types.length === 0) {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset) })
    if (search.value) params.set('search', search.value)
    const data = await listEntries(params)
    return { entries: data.entries || [], total: data.total || 0 }
  }

  if (types.length === 1) {
    const params = new URLSearchParams({ limit: String(limit), offset: String(offset), type: types[0] })
    if (search.value) params.set('search', search.value)
    const data = await listEntries(params)
    return { entries: data.entries || [], total: data.total || 0 }
  }

  const results = await Promise.all(
    types.map((type) => {
      const typeOffset = offset > 0 ? entries.value.filter(entry => entry.type === type).length : 0
      const params = new URLSearchParams({ limit: String(limit), offset: String(typeOffset), type })
      if (search.value) params.set('search', search.value)
      return listEntries(params)
    }),
  )
  const merged = dedupeEntries(results.flatMap(result => result.entries || []))
    .sort((a, b) => +new Date(b.created_at) - +new Date(a.created_at))
    .slice(0, limit)
  const totalCount = results.reduce((sum, result) => sum + (result.total || 0), 0)
  return { entries: merged, total: totalCount }
}

async function load(silent = false) {
  const seq = ++loadSeq
  if (!silent) loading.value = true
  try {
    if (route.query.bookmarked === '1') {
      let ids: string[] = []
      try { ids = JSON.parse(localStorage.getItem('signal-bookmarks') || '[]') } catch { ids = [] }
      const bookmarked = await Promise.all(ids.map(id => getEntry(id).then(result => result.entry).catch(() => null)))
      if (seq !== loadSeq) return
      entries.value = bookmarked.filter((entry): entry is Entry => entry !== null)
      total.value = entries.value.length
      updateStatus()
      return
    }
    const data = await fetchEntries()
    if (seq !== loadSeq) return
    entries.value = dedupeEntries(data.entries)
    total.value = data.total
    updateStatus()
  } catch {
    if (seq !== loadSeq) return
    status.value = 'Recorder offline'
  } finally {
    if (!silent && seq === loadSeq) loading.value = false
  }
}

function onSearch() {
  clearTimeout(debounce)
  debounce = setTimeout(() => replaceStreamQuery({ search: search.value }), 180)
}

function selectFilter(next: 'all' | 'errors' | 'slow') {
  if (route.query.bookmarked === '1') {
    const query: Record<string, string> = {}
    if (currentType.value) query.type = currentType.value
    if (next !== 'all') query.view = next
    router.push({ path: '/', query })
    return
  }
  replaceStreamQuery({ view: next })
}

function toggleType(type: string) {
  const next = selectedTypes.value.includes(type)
    ? selectedTypes.value.filter(item => item !== type)
    : [...selectedTypes.value, type]
  replaceStreamQuery({ signals: next })
}

function toggleTag(tag: string) {
  const next = selectedTags.value.includes(tag)
    ? selectedTags.value.filter(item => item !== tag)
    : [...selectedTags.value, tag]
  replaceStreamQuery({ tags: next })
}

function toggleFilters() {
  filtersOpen.value = !filtersOpen.value
}

function resetFilters() {
  router.replace({ path: '/' })
}

function clearSearch() {
  search.value = ''
  replaceStreamQuery({ search: '' })
}

function removeActiveType(type: string) {
  replaceStreamQuery({ signals: selectedTypes.value.filter(item => item !== type) })
}

function removeActiveTag(tag: string) {
  replaceStreamQuery({ tags: selectedTags.value.filter(item => item !== tag) })
}

function loadSavedFilters(): SavedFilter[] {
  try {
    const stored = JSON.parse(localStorage.getItem('signal-filters') || '[]') as Array<SavedFilter | string>
    return stored.map(item => typeof item === 'string'
      ? { name: item, type: '', filter: 'all', search: '' }
      : { ...item, signals: item.signals || [], tags: item.tags || [] })
  } catch {
    return []
  }
}

function saveFilter() {
  const fallback = [currentType.value || 'all', filter.value !== 'all' ? filter.value : '', search.value].filter(Boolean).join(' · ')
  const name = prompt('Name this filter', fallback)
  if (!name) return
  savedFilters.value = [
    ...savedFilters.value.filter(item => item.name !== name),
    { name, type: currentType.value, filter: filter.value, search: search.value, signals: [...selectedTypes.value], tags: [...selectedTags.value] },
  ]
  localStorage.setItem('signal-filters', JSON.stringify(savedFilters.value))
  savedOpen.value = true
}

function applySavedFilter(saved: SavedFilter) {
  savedOpen.value = false
  const query: Record<string, string> = {}
  if (saved.type) query.type = saved.type
  if (saved.search) query.search = saved.search
  if (saved.filter !== 'all') query.view = saved.filter
  if (saved.signals?.length) query.signals = saved.signals.join(',')
  if (saved.tags?.length) query.tags = saved.tags.join(',')
  router.push({ path: '/', query })
}

function removeSavedFilter(name: string) {
  savedFilters.value = savedFilters.value.filter(item => item.name !== name)
  localStorage.setItem('signal-filters', JSON.stringify(savedFilters.value))
}

async function pruneDatabaseRecords() {
  const accepted = await askConfirm({
    title: 'Prune database records',
    message: 'Delete all microscope records from the database?',
    detail: 'This permanently removes every row from microscope_entries and cannot be undone.',
    confirmLabel: 'Prune records',
    cancelLabel: 'Cancel',
    tone: 'danger',
  })
  if (!accepted) return
  storagePruning.value = true
  try {
    const result = await pruneEntries()
    entries.value = []
    total.value = 0
    updateStatus()
    await loadStorageUsage()
    showToast({
      tone: 'success',
      title: 'Records pruned',
      text: `${result.deleted.toLocaleString()} rows removed and disk space reclaimed.`,
    })
  } catch (error) {
    showToast({
      tone: 'error',
      title: 'Prune failed',
      text: error instanceof Error ? error.message : 'Database records could not be pruned.',
    })
  } finally {
    storagePruning.value = false
  }
}

async function clearAll() {
  const accepted = await askConfirm({
    title: 'Clear recorded activity',
    message: 'Erase all recorded activity from the live stream?',
    detail: 'This deletes every retained microscope entry and cannot be undone.',
    confirmLabel: 'Clear all',
    cancelLabel: 'Cancel',
    tone: 'danger',
  })
  if (!accepted) return
  clearing.value = true
  try {
    const result = await pruneEntries()
    entries.value = []
    total.value = 0
    updateStatus()
    await loadStorageUsage()
    showToast({
      tone: 'success',
      title: 'Activity cleared',
      text: `${result.deleted.toLocaleString()} records deleted from the live stream.`,
    })
  } catch (error) {
    showToast({
      tone: 'error',
      title: 'Clear failed',
      text: error instanceof Error ? error.message : 'Recorded activity could not be cleared.',
    })
  } finally {
    clearing.value = false
  }
}

async function toggleRecording() {
  if (recordingToggling.value) return
  const next = !recordingPaused.value
  recordingToggling.value = true
  try {
    const result = await setRecordingPaused(next)
    recordingPaused.value = result.paused
    updateStatus()
    showToast({
      tone: 'success',
      title: result.paused ? 'Recording paused' : 'Recording resumed',
      text: result.paused
        ? 'New microscope records are not being written until you resume.'
        : 'Live recording and stream updates are active again.',
    })
  } catch (error) {
    showToast({
      tone: 'error',
      title: 'Recording state not saved',
      text: error instanceof Error ? error.message : 'Could not update recording state.',
    })
  } finally {
    recordingToggling.value = false
  }
}

function updateStatus() {
  const state = recordingPaused.value ? 'paused' : streamConnected.value ? 'live' : 'reconnecting'
  status.value = `${total.value} retained · ${state}`
}

function dedupeEntries(items: Entry[]): Entry[] {
  const seen = new Set<string>()
  const requestBatches = new Set(items.filter(entry => entry.type === 'request').map(entry => entry.batch_id))
  return items.filter(entry => {
    if (seen.has(entry.id)) return false
    if (!currentType.value && entry.type === 'log' && entry.content?.message === 'request' && requestBatches.has(entry.batch_id)) return false
    seen.add(entry.id)
    return true
  })
}

function onStreamEntry(entry: Entry) {
  if (recordingPaused.value) return
  if (route.query.bookmarked === '1') return
  if (currentType.value && entry.type !== currentType.value) return
  if (!currentType.value && selectedTypes.value.length && !selectedTypes.value.includes(entry.type)) return
  if (search.value) {
    const haystack = `${entry.request_id || ''} ${entry.correlation_id || ''} ${JSON.stringify(entry.content)}`.toLowerCase()
    if (!haystack.includes(search.value.toLowerCase())) return
  }
  entries.value = dedupeEntries([entry, ...entries.value]).slice(0, 80)
  total.value += 1
  updateStatus()
}

function onStreamControl(event: { action: string; type?: string; deleted: number; paused?: boolean }) {
  if (event.action === 'recording-paused' && typeof event.paused === 'boolean') {
    recordingPaused.value = event.paused
  } else if (event.action === 'clear-all') {
    entries.value = []
    total.value = 0
    void loadStorageUsage()
  } else if (event.action === 'signal-setting' && event.type) {
    const removed = entries.value.filter(entry => entry.type === event.type).length
    entries.value = entries.value.filter(entry => entry.type !== event.type)
    total.value = Math.max(0, total.value - Math.max(removed, event.deleted))
    selectedTypes.value = selectedTypes.value.filter(type => type !== event.type)
    if (currentType.value === event.type) router.replace('/')
    loadSignalSettings(true)
  }
  updateStatus()
}

async function loadMore() {
  if (loading.value || entries.value.length >= total.value) return
  loading.value = true
  try {
    const data = await fetchEntries(entries.value.length)
    entries.value = dedupeEntries([...entries.value, ...data.entries])
    total.value = data.total || total.value
  } finally {
    loading.value = false
  }
}

function startAutoRefresh() {
  timer = setInterval(() => {
    if (!recordingPaused.value && !loading.value) load(true)
    if (!currentType.value) void loadStorageUsage(true)
  }, 30000)
}

async function loadRecordingState() {
  try {
    const result = await getRecordingState()
    recordingPaused.value = result.paused
    updateStatus()
  } catch {
    // Keep default live state if recording status cannot be loaded.
  }
}

async function loadStorageUsage(silent = false) {
  try {
    storageUsage.value = await getStorageUsage()
  } catch {
    if (!silent) storageUsage.value = null
  }
}

onMounted(() => {
  loadSignalSettings().then(() => {
    if (currentType.value && !signalEnabled(currentType.value)) router.replace('/')
    selectedTypes.value = selectedTypes.value.filter(signalEnabled)
  })
  void loadStorageUsage()
  void loadRecordingState()
  stopStream = subscribeEntries(onStreamEntry, (connected) => {
    streamConnected.value = connected
    updateStatus()
  }, onStreamControl)
  startAutoRefresh()
})
onUnmounted(() => {
  stopStream?.()
  if (timer) clearInterval(timer)
})
</script>

<template>
  <AppShell>
    <template #status>{{ status }}</template>
    <template #actions>
      <button
        type="button"
        class="stream-control stream-control--instrument"
        :class="{ 'is-paused': recordingPaused }"
        :disabled="recordingToggling"
        :aria-pressed="recordingPaused"
        :title="recordingPaused ? 'Resume recording' : 'Pause recording'"
        @click="toggleRecording"
      >
        <span><i /></span>
        {{ recordingPaused ? 'Paused' : 'Live' }}
      </button>
    </template>

    <section v-if="!currentType" class="storage-readout runtime-readout" aria-label="Database storage usage">
      <div class="runtime-hero">
        <div class="runtime-number storage-number">
          <span class="runtime-number__label">Database storage</span>
          <strong>{{ storageTotalLabel }}</strong>
          <p class="runtime-number__copy">allocated table files across microscope tables</p>
        </div>
      </div>
      <div class="runtime-vitals storage-vitals--pair">
        <div class="runtime-vital storage-vital-entries" :class="{ 'is-hot': storageHasRecords }">
          <span>Retained entries</span>
          <strong>{{ storageEntryCountLabel }}</strong>
          <small>microscope_entries rows</small>
        </div>
        <div class="runtime-vital storage-vital-disk" :class="{ 'is-hot': storageHasRecords }">
          <span>Entries table</span>
          <strong>{{ storageEntriesLabel }}</strong>
          <small>{{ storageEntriesBreakdown }}</small>
        </div>
      </div>
      <div class="storage-actions runtime-wave-panel">
        <header>
          <span>Maintenance</span>
          <small>microscope database</small>
        </header>
        <button
          type="button"
          class="storage-prune-btn"
          :disabled="storagePruning || clearing || !storageHasRecords"
          :title="storagePruning ? 'Pruning database records…' : 'Delete all microscope database records'"
          @click="pruneDatabaseRecords"
        >
          {{ storagePruning ? 'Pruning…' : 'Prune records' }}
        </button>
        <p class="storage-actions__hint">Deletes all rows, reclaims disk with VACUUM FULL, then live recorders may add new entries.</p>
      </div>
    </section>

    <section v-if="!currentType" class="runtime-readout" aria-label="Captured now summary">
      <div class="runtime-hero">
        <div class="runtime-number">
          <span class="runtime-number__label">Captured now</span>
          <strong>{{ entries.length.toLocaleString() }}</strong>
          <p class="runtime-number__copy">
            signals in the current window
            <span v-if="typeCounts.length">{{ typeCounts.length }} active recorder types</span>
          </p>
        </div>
        <div class="runtime-spectrum-wrap" aria-hidden="true">
          <div class="runtime-spectrum" :style="{ background: signalGradient }" />
          <span class="runtime-spectrum__count">{{ entries.length.toLocaleString() }}</span>
        </div>
      </div>
      <div class="runtime-vitals">
        <div class="runtime-vital vital-error" :class="{ 'is-hot': errorCount > 0 }">
          <span>Error pressure</span>
          <strong>{{ errorRate }}%</strong>
          <small>{{ errorCount }} errors</small>
        </div>
        <div class="runtime-vital vital-slow" :class="{ 'is-hot': slowEntries.length > 0 }">
          <span>Slow paths</span>
          <strong>{{ slowEntries.length }}</strong>
          <small>over 500ms</small>
        </div>
        <div class="runtime-vital vital-latency">
          <span>p95 latency</span>
          <strong>{{ p95 }}</strong>
          <small>recorded spans</small>
        </div>
      </div>
      <div class="runtime-wave-panel">
        <header>
          <span>Recent intensity</span>
          <small>last 32 records</small>
        </header>
        <div class="runtime-wave-chart">
          <div class="runtime-wave" aria-hidden="true">
            <i
              v-for="(bar, index) in sparkBars"
              :key="index"
              :class="{ 'is-empty': bar.empty }"
              :style="{ height: `${bar.height}px`, background: bar.color }"
            />
          </div>
        </div>
      </div>
    </section>

    <section v-else class="channel-readout" :style="{ '--signal': selectedSignal.color }">
      <div class="channel-identity">
        <span><SignalIcon :type="selectedSignal.type" size="md" /></span>
        <div><small>Active recorder</small><strong>{{ selectedSignal.label }}</strong></div>
      </div>
      <div class="channel-stat"><small>Retained</small><strong>{{ total }}</strong><span>records</span></div>
      <div class="channel-stat"><small>Average cost</small><strong>{{ averageDuration }}</strong><span>timed operations</span></div>
      <div class="channel-stat"><small>Maximum cost</small><strong>{{ maximumDuration }}</strong><span>current window</span></div>
      <div class="channel-stat"><small>Execution batches</small><strong>{{ batchCount }}</strong><span>linked contexts</span></div>
      <div class="channel-state"><i :class="{ connected: streamConnected }" /><span>{{ streamConnected ? 'Receiving live records' : 'Reconnecting stream' }}</span></div>
    </section>

    <SystemVisuals v-if="!currentType" :entries="entries" />

    <LLMInsightPanel
      v-if="llmSettings.enabled && !currentType"
      :entries="entries"
      title="Runtime intelligence"
      compact
    />

    <div class="workbench" :class="{ 'workbench--focused': currentType }">
      <section class="stream-pane" :class="{ 'stream-pane--paused': recordingPaused }">
        <div class="stream-filter-panel" :class="{ 'has-active-filters': hasActiveFilters, 'is-open': filtersOpen }">
        <header class="stream-toolbar">
          <div class="stream-title">
            <span class="stream-glyph" :style="{ '--signal': selectedSignal.color }"><SignalIcon :type="selectedSignal.type" size="md" /></span>
            <div>
              <strong>{{ currentType ? selectedSignal.label : 'Activity stream' }}</strong>
              <small>
                {{ visibleEntries.length }} visible
                <template v-if="recordingPaused"> · recording paused</template>
                <template v-else-if="hasActiveFilters"> · {{ activeFilterCount }} filter{{ activeFilterCount === 1 ? '' : 's' }}</template>
              </small>
            </div>
          </div>
          <label class="inline-search stream-toolbar__search">
            <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
            <input v-model="search" placeholder="Search records…" @input="onSearch" />
            <button v-if="search" type="button" class="search-clear" title="Clear search" @click="clearSearch">×</button>
            <span v-else-if="loading" class="action-spinner" />
            <kbd v-else>/</kbd>
          </label>
          <div class="stream-toolbar__actions">
            <button
              type="button"
              class="filter-toggle"
              :class="{ 'is-open': filtersOpen, 'has-active': hasActiveFilters }"
              :aria-expanded="filtersOpen"
              @click="toggleFilters"
            >
              <svg class="filter-toggle__chevron" viewBox="0 0 20 20" aria-hidden="true"><path d="m6 8 4 4 4-4"/></svg>
              <span>Filters</span>
              <em v-if="hasActiveFilters">{{ activeFilterCount }}</em>
            </button>
            <div class="saved-filter-menu">
              <button class="tool-button" :title="`${savedFilters.length} saved filters`" @click="savedOpen = !savedOpen">
              <svg viewBox="0 0 20 20"><path d="M6 3.5h8v13L10 14l-4 2.5v-13Z"/></svg>
              </button>
              <Transition name="filter-pop">
                <div v-if="savedOpen" class="saved-filter-popover">
                  <header><span>Saved views</span><button @click="saveFilter">Save current</button></header>
                  <button v-for="saved in savedFilters.filter(item => !item.type || signalEnabled(item.type))" :key="saved.name" class="saved-filter-row" @click="applySavedFilter(saved)">
                    <span><strong>{{ saved.name }}</strong><small>{{ saved.type || 'all activity' }} · {{ saved.filter }}</small></span>
                    <i title="Remove saved view" @click.stop="removeSavedFilter(saved.name)">×</i>
                  </button>
                  <div v-if="!savedFilters.length" class="saved-filter-empty">Save a search or filter combination for instant recall.</div>
                </div>
              </Transition>
            </div>
            <button class="tool-button danger-tool" :title="clearing ? 'Deleting recorded activity…' : 'Clear recorded activity'" :disabled="clearing" @click="clearAll">
              <span v-if="clearing" class="action-spinner" />
              <svg v-else viewBox="0 0 20 20"><path d="M4 6h12M8 6V4h4v2m-6 0 .6 10h6.8L14 6M8.5 9v4m3-4v4"/></svg>
            </button>
          </div>
        </header>

        <div v-if="!filtersOpen && hasActiveFilters" class="active-filter-bar active-filter-bar--compact" aria-label="Active filters">
          <div class="active-filter-bar__chips">
            <button v-if="filter === 'errors'" type="button" class="filter-chip is-view is-error" @click="selectFilter('all')">Errors <span aria-hidden="true">×</span></button>
            <button v-if="filter === 'slow'" type="button" class="filter-chip is-view is-slow" @click="selectFilter('all')">Slow <span aria-hidden="true">×</span></button>
            <button v-if="route.query.bookmarked === '1'" type="button" class="filter-chip is-view" @click="router.replace({ path: '/' })">Pinned <span aria-hidden="true">×</span></button>
            <button v-for="type in selectedTypes" :key="`compact-type-${type}`" type="button" class="filter-chip is-signal" :style="{ '--signal': signalFor(type).color }" @click="removeActiveType(type)">
              <SignalIcon :type="type as EntryType" size="sm" />{{ signalFor(type).shortLabel }}<span aria-hidden="true">×</span>
            </button>
            <button v-for="tag in selectedTags" :key="`compact-tag-${tag}`" type="button" class="filter-chip is-tag" @click="removeActiveTag(tag)">{{ tag }} <span aria-hidden="true">×</span></button>
            <button v-if="search" type="button" class="filter-chip is-search" @click="clearSearch">"{{ search }}" <span aria-hidden="true">×</span></button>
          </div>
          <button type="button" class="filter-chip filter-chip--reset" @click="resetFilters">Clear all</button>
        </div>

        <Transition name="filter-collapse">
        <div v-if="filtersOpen" class="stream-filters-body">
          <div class="stream-filters">
            <div class="filter-switch" role="group" aria-label="Stream view">
              <button type="button" :class="{ active: filter === 'all' && route.query.bookmarked !== '1' }" @click="selectFilter('all')">All</button>
              <button type="button" class="filter-switch__errors" :class="{ active: filter === 'errors' && route.query.bookmarked !== '1' }" @click="selectFilter('errors')">Errors</button>
              <button type="button" class="filter-switch__slow" :class="{ active: filter === 'slow' && route.query.bookmarked !== '1' }" @click="selectFilter('slow')">Slow</button>
              <button type="button" :class="{ active: route.query.bookmarked === '1' }" title="Bookmarked traces" @click="router.push({ path: '/', query: { bookmarked: '1' } })">Pinned</button>
            </div>
          </div>

        <div v-if="hasActiveFilters" class="active-filter-bar" aria-label="Active filters">
          <span class="active-filter-bar__label">Active</span>
          <div class="active-filter-bar__chips">
            <button v-if="filter === 'errors'" type="button" class="filter-chip is-view is-error" @click="selectFilter('all')">
              Errors <span aria-hidden="true">×</span>
            </button>
            <button v-if="filter === 'slow'" type="button" class="filter-chip is-view is-slow" @click="selectFilter('all')">
              Slow <span aria-hidden="true">×</span>
            </button>
            <button v-if="route.query.bookmarked === '1'" type="button" class="filter-chip is-view" @click="router.replace({ path: '/' })">
              Pinned <span aria-hidden="true">×</span>
            </button>
            <button
              v-for="type in selectedTypes"
              :key="`type-${type}`"
              type="button"
              class="filter-chip is-signal"
              :style="{ '--signal': signalFor(type).color }"
              @click="removeActiveType(type)"
            >
              <SignalIcon :type="type as EntryType" size="sm" />
              {{ signalFor(type).shortLabel }}
              <span aria-hidden="true">×</span>
            </button>
            <button
              v-for="tag in selectedTags"
              :key="`tag-${tag}`"
              type="button"
              class="filter-chip is-tag"
              @click="removeActiveTag(tag)"
            >
              {{ tag }} <span aria-hidden="true">×</span>
            </button>
            <button v-if="search" type="button" class="filter-chip is-search" @click="clearSearch">
              "{{ search }}" <span aria-hidden="true">×</span>
            </button>
          </div>
          <button type="button" class="filter-chip filter-chip--reset" @click="resetFilters">Clear all</button>
        </div>

        <div v-if="!currentType" class="label-filter-strip">
          <div class="label-filter-groups">
            <section v-for="group in recorderGroups" :key="group.id" class="label-filter-group">
              <span class="label-filter-group__title">{{ group.label }}</span>
              <div class="label-filter-scroll-wrap">
                <div class="label-filter-scroll">
                  <button
                    v-for="signal in group.signals"
                    :key="signal.type"
                    type="button"
                    class="recorder-chip"
                    :class="{ active: selectedTypes.includes(signal.type), 'has-count': typeCountMap[signal.type] }"
                    :style="{ '--signal': signal.color }"
                    :title="`Filter ${signal.label}`"
                    @click="toggleType(signal.type)"
                  >
                    <SignalIcon :type="signal.type" size="sm" />
                    <span>{{ signal.shortLabel }}</span>
                    <em v-if="typeCountMap[signal.type]">{{ typeCountMap[signal.type] }}</em>
                  </button>
                </div>
              </div>
            </section>
          </div>
          <section v-if="availableTags.length" class="label-filter-tags">
            <span class="label-filter-title">Tags</span>
            <div class="label-filter-scroll-wrap">
              <div class="label-filter-scroll">
                <button
                  v-for="tag in availableTags"
                  :key="tag"
                  type="button"
                  class="tag-label"
                  :class="{ active: selectedTags.includes(tag) }"
                  @click="toggleTag(tag)"
                >
                  <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 9V4h5l8 8-5 5-8-8Z"/><circle cx="6.5" cy="6.5" r=".8"/></svg>
                  <span>{{ tag }}</span>
                </button>
              </div>
            </div>
          </section>
        </div>
        <div v-else-if="availableTags.length" class="label-filter-strip label-filter-strip--tags-only">
          <span class="label-filter-title">Tags</span>
          <div class="label-filter-scroll-wrap">
            <div class="label-filter-scroll">
              <button
                v-for="tag in availableTags"
                :key="tag"
                type="button"
                class="tag-label"
                :class="{ active: selectedTags.includes(tag) }"
                @click="toggleTag(tag)"
              >
                <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M3 9V4h5l8 8-5 5-8-8Z"/><circle cx="6.5" cy="6.5" r=".8"/></svg>
                <span>{{ tag }}</span>
              </button>
            </div>
          </div>
        </div>
        </div>
        </Transition>
        </div>

        <div class="stream-columns">
          <span>Timestamp</span>
          <span class="stream-col-pipeline" aria-hidden="true" />
          <span>Type / context</span>
          <span>Cost</span>
          <span class="stream-col-tools" aria-hidden="true" />
        </div>
        <div v-if="loading && !entries.length" class="stream-loading" aria-label="Loading activity">
          <div v-for="index in 8" :key="index"><span /><i /><b :style="{ width: `${38 + (index * 13) % 42}%` }" /></div>
        </div>
        <div v-else-if="!visibleEntries.length" class="signal-empty">
          <div class="empty-orbit" :style="{ '--signal': selectedSignal.color }"><span /><i /><b /></div>
          <strong>The stream is quiet</strong>
          <p>Activity will surface here the instant your application emits it. This recorder is fully backed and ready.</p>
          <button v-if="search || filter !== 'all' || selectedTypes.length || selectedTags.length" @click="resetFilters">Reset filters</button>
        </div>
        <EntryTable v-else :entries="visibleEntries" :current-type="currentType" />
        <footer v-if="visibleEntries.length" class="stream-footer">
          <span><kbd>↑</kbd><kbd>↓</kbd> move</span>
          <span><kbd>↵</kbd> inspect</span>
          <span>double-click opens instantly</span>
          <button v-if="entries.length < total" :disabled="loading" @click="loadMore">{{ loading ? 'Loading…' : `Load more · ${total - entries.length}` }}</button>
        </footer>
      </section>

      <aside v-if="!currentType" class="pulse-pane">
        <section class="pulse-section pulse-health" :class="`is-${pulseHealth.tone}`">
          <header>
            <span>Application pulse</span>
            <strong class="pulse-status" :class="`is-${pulseHealth.tone}`"><i />{{ pulseHealth.label }}</strong>
          </header>
          <div class="pulse-orbit">
            <div class="pulse-spectrum" :style="{ background: signalGradient }" />
            <div class="orbit-core">
              <span>{{ entries.length }}</span>
              <small>events</small>
            </div>
            <i v-for="index in 12" :key="index" :style="{ '--i': index }" />
          </div>
          <div class="pulse-metrics">
            <span><small>Error rate</small><strong>{{ errorRate }}<i>%</i></strong></span>
            <span><small>P95 cost</small><strong>{{ p95 }}</strong></span>
            <span><small>Types active</small><strong>{{ typeCounts.length }}</strong></span>
          </div>
          <p>{{ pulseMessage }}</p>
        </section>

        <section class="pulse-section signal-mix">
          <header>
            <span>Type mix</span>
            <small>{{ typeMixTotal }} records · current window</small>
          </header>
          <div v-if="typeCounts.length" class="mix-stack">
            <div
              v-for="(item, index) in typeCounts"
              :key="item.type"
              class="mix-row"
              :style="{ '--signal': item.color, '--width': `${item.count / maxTypeCount * 100}%` }"
            >
              <span class="mix-rank">{{ index + 1 }}</span>
              <span class="mix-label"><SignalIcon :type="item.type" size="sm" />{{ item.shortLabel }}</span>
              <i><b /></i>
              <strong>{{ item.count }}</strong>
              <em>{{ Math.round(item.count / Math.max(1, typeMixTotal) * 100) }}%</em>
            </div>
          </div>
          <div v-else class="pulse-muted">Waiting for a recorder profile…</div>
        </section>

        <section class="pulse-section attention-list" :class="{ 'has-items': attentionCount }">
          <header>
            <span>Needs attention</span>
            <small class="attention-badge" :class="{ active: attentionCount }">{{ attentionCount }}</small>
          </header>
          <div v-if="attentionCount" class="attention-stack">
            <button
              v-for="entry in attentionItems"
              :key="entry.id"
              :class="{ error: isError(entry) }"
              @click="$router.push(`/entries/${entry.id}`)"
            >
              <span class="attention-icon" :style="{ '--signal': signalFor(entry.type).color }">
                <SignalIcon :type="entry.type" size="sm" />
              </span>
              <span class="attention-copy">
                <strong>{{ signalFor(entry.type).shortLabel }}</strong>
                <small>{{ isError(entry) ? 'Error signal' : 'Slow path' }}</small>
              </span>
              <b>{{ entryDuration(entry) ? `${entryDuration(entry)}ms` : 'error' }}</b>
              <svg viewBox="0 0 20 20"><path d="m8 5 5 5-5 5"/></svg>
            </button>
          </div>
          <div v-else class="pulse-muted pulse-calm">
            <span class="pulse-calm__ring" aria-hidden="true"><i /></span>
            <strong>All clear</strong>
            <p>Nothing is asking for attention.</p>
          </div>
        </section>
      </aside>
    </div>
  </AppShell>
</template>
