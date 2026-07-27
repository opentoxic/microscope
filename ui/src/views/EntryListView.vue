<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getEntry, listEntries, pruneEntries, subscribeEntries } from '../api/client'
import type { Entry } from '../types'
import AppShell from '../components/AppShell.vue'
import EntryTable from '../components/EntryTable.vue'
import LLMInsightPanel from '../components/LLMInsightPanel.vue'
import SignalIcon from '../components/SignalIcon.vue'
import SystemVisuals from '../components/SystemVisuals.vue'
import { entryDuration, isError, signalFor } from '../utils'
import { enabledSignals, loadSignalSettings, signalEnabled } from '../settings'
import { llmSettings } from '../llmSettings'

const route = useRoute()
const router = useRouter()
const entries = ref<Entry[]>([])
const total = ref(0)
const status = ref('Opening live channel…')
const search = ref(String(route.query.search || ''))
const loading = ref(false)
const clearing = ref(false)
const actionNotice = ref<{ tone: 'success' | 'error'; text: string } | null>(null)
const autoRefresh = ref(true)
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
let timer: ReturnType<typeof setInterval> | null = null
let debounce: ReturnType<typeof setTimeout>
let stopStream: (() => void) | null = null

const visibleEntries = computed(() => {
  let result = entries.value
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

watch(() => [route.query.type, route.query.search, route.query.view, route.query.signals, route.query.tags], ([type, routeSearch, view, routeSignals, routeTags]) => {
  currentType.value = String(type || '')
  search.value = String(routeSearch || '')
  filter.value = view === 'errors' || view === 'slow' ? view : 'all'
  selectedTypes.value = String(routeSignals || '').split(',').filter(Boolean)
  selectedTags.value = String(routeTags || '').split(',').filter(Boolean)
  load()
})

async function load(silent = false) {
  if (loading.value && !silent) return
  if (!silent) loading.value = true
  const params = new URLSearchParams({ limit: '80' })
  if (currentType.value) params.set('type', currentType.value)
  if (search.value) params.set('search', search.value)
  try {
    if (route.query.bookmarked === '1') {
      let ids: string[] = []
      try { ids = JSON.parse(localStorage.getItem('signal-bookmarks') || '[]') } catch { ids = [] }
      const bookmarked = await Promise.all(ids.map(id => getEntry(id).then(result => result.entry).catch(() => null)))
      entries.value = bookmarked.filter((entry): entry is Entry => entry !== null)
      total.value = entries.value.length
      updateStatus()
      return
    }
    const data = await listEntries(params)
    entries.value = dedupeEntries(data.entries || [])
    total.value = data.total || 0
    updateStatus()
  } catch {
    status.value = 'Recorder offline'
  } finally {
    loading.value = false
  }
}

function onSearch() {
  clearTimeout(debounce)
  debounce = setTimeout(() => load(), 180)
}

function selectFilter(next: 'all' | 'errors' | 'slow') {
  filter.value = next
  if (route.query.bookmarked === '1') {
    const query: Record<string, string> = {}
    if (currentType.value) query.type = currentType.value
    if (next !== 'all') query.view = next
    router.push({ path: '/', query })
  }
}

function toggleType(type: string) {
  selectedTypes.value = selectedTypes.value.includes(type)
    ? selectedTypes.value.filter(item => item !== type)
    : [...selectedTypes.value, type]
}

function toggleTag(tag: string) {
  selectedTags.value = selectedTags.value.includes(tag)
    ? selectedTags.value.filter(item => item !== tag)
    : [...selectedTags.value, tag]
}

function clearLabels() {
  selectedTypes.value = []
  selectedTags.value = []
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

async function clearAll() {
  if (!confirm('Erase all recorded activity? This cannot be undone.')) return
  clearing.value = true
  actionNotice.value = null
  try {
    const result = await pruneEntries()
    entries.value = []
    total.value = 0
    updateStatus()
    actionNotice.value = { tone: 'success', text: `${result.deleted.toLocaleString()} records deleted in real time` }
  } catch (error) {
    actionNotice.value = { tone: 'error', text: error instanceof Error ? error.message : 'Deletion failed' }
  } finally {
    clearing.value = false
  }
}

function toggleStreaming() {
  autoRefresh.value = !autoRefresh.value
  updateStatus()
}

function updateStatus() {
  const state = !autoRefresh.value ? 'paused' : streamConnected.value ? 'live' : 'reconnecting'
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
  if (!autoRefresh.value) return
  if (route.query.bookmarked === '1') return
  if (currentType.value && entry.type !== currentType.value) return
  if (search.value) {
    const haystack = `${entry.request_id || ''} ${entry.correlation_id || ''} ${JSON.stringify(entry.content)}`.toLowerCase()
    if (!haystack.includes(search.value.toLowerCase())) return
  }
  entries.value = dedupeEntries([entry, ...entries.value]).slice(0, 80)
  total.value += 1
  updateStatus()
}

function onStreamControl(event: { action: string; type?: string; deleted: number }) {
  if (event.action === 'clear-all') {
    entries.value = []
    total.value = 0
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
  const params = new URLSearchParams({ limit: '80', offset: String(entries.value.length) })
  if (currentType.value) params.set('type', currentType.value)
  if (search.value) params.set('search', search.value)
  try {
    const data = await listEntries(params)
    entries.value = dedupeEntries([...entries.value, ...(data.entries || [])])
    total.value = data.total || total.value
  } finally {
    loading.value = false
  }
}

function startAutoRefresh() {
  timer = setInterval(() => {
    if (autoRefresh.value && !loading.value) load(true)
  }, 30000)
}

onMounted(() => {
  loadSignalSettings().then(() => {
    if (currentType.value && !signalEnabled(currentType.value)) router.replace('/')
    selectedTypes.value = selectedTypes.value.filter(signalEnabled)
  })
  load()
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
          <div class="runtime-spectrum" :style="{ background: signalGradient }">
            <span>{{ entries.length }}</span>
          </div>
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
      <section class="stream-pane">
        <header class="stream-toolbar">
          <div class="stream-title">
            <span class="stream-glyph" :style="{ '--signal': selectedSignal.color }"><SignalIcon :type="selectedSignal.type" size="md" /></span>
            <div>
              <strong>{{ currentType ? selectedSignal.label : 'Activity stream' }}</strong>
              <small>{{ visibleEntries.length }} visible records</small>
            </div>
          </div>
          <div class="stream-filters">
            <div class="filter-switch">
              <button :class="{ active: filter === 'all' && route.query.bookmarked !== '1' }" @click="selectFilter('all')">All</button>
              <button :class="{ active: filter === 'errors' && route.query.bookmarked !== '1' }" @click="selectFilter('errors')">Errors</button>
              <button :class="{ active: filter === 'slow' && route.query.bookmarked !== '1' }" @click="selectFilter('slow')">Slow</button>
              <button :class="{ active: route.query.bookmarked === '1' }" title="Bookmarked traces" @click="router.push({ path: '/', query: { bookmarked: '1' } })">Pinned</button>
            </div>
            <label class="inline-search">
              <svg viewBox="0 0 20 20"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
              <input v-model="search" placeholder="Filter stream…" @input="onSearch" />
              <span v-if="loading" class="action-spinner" />
              <kbd v-else>/</kbd>
            </label>
            <button class="tool-button" :class="{ 'is-live': autoRefresh }" :title="autoRefresh ? 'Pause live stream' : 'Resume live stream'" @click="toggleStreaming">
              <svg v-if="autoRefresh" viewBox="0 0 20 20"><path d="M7 5v10m6-10v10"/></svg>
              <svg v-else viewBox="0 0 20 20"><path d="m7 5 8 5-8 5V5Z"/></svg>
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

        <div class="label-filter-strip">
          <span class="label-filter-title">Recorders</span>
          <div class="label-filter-scroll">
            <button
              v-for="signal in filterSignals"
              :key="signal.type"
              :class="{ active: selectedTypes.includes(signal.type) }"
              :style="{ '--signal': signal.color }"
              @click="toggleType(signal.type)"
            >
              <SignalIcon :type="signal.type" size="sm" /><span>{{ signal.shortLabel }}</span>
            </button>
            <span v-if="availableTags.length" class="label-separator" />
            <button
              v-for="tag in availableTags"
              :key="tag"
              class="tag-label"
              :class="{ active: selectedTags.includes(tag) }"
              @click="toggleTag(tag)"
            >
              <svg viewBox="0 0 20 20"><path d="M3 9V4h5l8 8-5 5-8-8Z"/><circle cx="6.5" cy="6.5" r=".8"/></svg>
              <span>{{ tag }}</span>
            </button>
          </div>
          <button v-if="selectedTypes.length || selectedTags.length" class="clear-labels" @click="clearLabels">Clear {{ selectedTypes.length + selectedTags.length }}</button>
        </div>
        <Transition name="notice">
          <div v-if="actionNotice" class="action-notice" :class="`is-${actionNotice.tone}`">
            <span>{{ actionNotice.text }}</span>
            <button @click="actionNotice = null">×</button>
          </div>
        </Transition>

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
          <button v-if="search || filter !== 'all' || selectedTypes.length || selectedTags.length" @click="search = ''; filter = 'all'; clearLabels(); load()">Reset filters</button>
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
