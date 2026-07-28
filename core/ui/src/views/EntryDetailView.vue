<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getEntry } from '../api/client'
import type { Entry, EntryDetailResponse } from '../types'
import { activeDetailEntryType } from '../detailContext'
import AppShell from '../components/AppShell.vue'
import Badge from '../components/Badge.vue'
import ContentTabs from '../components/ContentTabs.vue'
import RelatedTabs from '../components/RelatedTabs.vue'
import EntryInsight from '../components/EntryInsight.vue'
import LLMInsightPanel from '../components/LLMInsightPanel.vue'
import SignalIcon from '../components/SignalIcon.vue'
import SqlInspector from '../components/SqlInspector.vue'
import { entryDuration, entryMeta, entrySignalColor, formatRecordDate, formatRecordTime, formatTimeLong, isError, metricLanguageLabel, metricUnitLabel, methodClass, signalFor, statusClass, summarize, timeAgo, typeBadgeClass } from '../utils'
import { llmSettings } from '../llmSettings'
import { signalEnabled } from '../settings'

const props = defineProps<{ id: string }>()
const route = useRoute()
const router = useRouter()
const data = ref<EntryDetailResponse | null>(null)
const comparison = ref<EntryDetailResponse | null>(null)
const loading = ref(true)
const error = ref('')
const copied = ref(false)
const copiedField = ref('')
const bookmarked = ref(false)

const title = computed(() => data.value ? summarize(data.value.entry) : '')
const timeline = computed(() => {
  if (!data.value) return []
  return [...data.value.batch].sort((a, b) => new Date(a.created_at).getTime() - new Date(b.created_at).getTime())
})
const totalDuration = computed(() => Math.max(
  Number(data.value?.entry.content?.duration_ms || 0),
  timeline.value.reduce((sum, entry) => sum + entryDuration(entry), 0),
  1,
))
const queryCount = computed(() => timeline.value.filter(entry => entry.type === 'query').length)
const errorCount = computed(() => timeline.value.filter(entry => entry.type === 'exception').length)
const timelineLegend = computed(() => {
  const counts = new Map<string, number>()
  for (const entry of timeline.value) counts.set(entry.type, (counts.get(entry.type) || 0) + 1)
  return [...counts.entries()]
    .map(([type, count]) => ({
      type,
      count,
      color: signalFor(type).color,
      label: signalFor(type).shortLabel,
    }))
    .sort((a, b) => b.count - a.count)
})

onMounted(async () => {
  activeDetailEntryType.value = ''
  try {
    const compareID = String(route.query.compare || '')
    const [primary, secondary] = await Promise.all([
      getEntry(props.id),
      compareID ? getEntry(compareID) : Promise.resolve(null),
    ])
    data.value = primary
    comparison.value = secondary
    bookmarked.value = loadBookmarks().includes(props.id)
  } catch {
    error.value = 'This trace is no longer available'
  } finally {
    loading.value = false
  }
})

watch(data, (val) => {
  activeDetailEntryType.value = val?.entry?.type || ''
})

watch(() => props.id, () => {
  activeDetailEntryType.value = ''
})

onBeforeUnmount(() => {
  activeDetailEntryType.value = ''
})

function loadBookmarks(): string[] {
  try { return JSON.parse(localStorage.getItem('signal-bookmarks') || '[]') } catch { return [] }
}

function toggleBookmark() {
  const current = loadBookmarks()
  const next = current.includes(props.id) ? current.filter(id => id !== props.id) : [...current, props.id]
  localStorage.setItem('signal-bookmarks', JSON.stringify(next))
  bookmarked.value = next.includes(props.id)
}

async function copyID() {
  await navigator.clipboard.writeText(props.id)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1400)
}

async function copyField(label: string, value: string) {
  if (!value || value === 'not linked') return
  await navigator.clipboard.writeText(value)
  copiedField.value = label
  setTimeout(() => { if (copiedField.value === label) copiedField.value = '' }, 1400)
}

async function fullscreen() {
  if (document.fullscreenElement) await document.exitFullscreen()
  else await document.documentElement.requestFullscreen()
}

function timelineWidth(entry: Entry): string {
  return `${Math.max(3, entryDuration(entry) / totalDuration.value * 100)}%`
}

function openTimelineEntry(entry: Entry) {
  if (entry.id !== props.id) router.push(`/entries/${entry.id}`)
}
</script>

<template>
  <AppShell>
    <template #status>
      <span v-if="data">{{ data.entry.id.slice(0, 12) }} · {{ timeAgo(data.entry.created_at) }}</span>
    </template>
    <template #actions>
      <button class="inspector-action" :class="{ active: bookmarked }" title="Bookmark trace" @click="toggleBookmark">
        <svg viewBox="0 0 20 20"><path d="M6 3.5h8v13L10 14l-4 2.5v-13Z"/></svg>
      </button>
      <button class="inspector-action" title="Fullscreen inspector" @click="fullscreen">
        <svg viewBox="0 0 20 20"><path d="M4 8V4h4m4 0h4v4m0 4v4h-4m-4 0H4v-4"/></svg>
      </button>
    </template>

    <div v-if="loading" class="inspector-loading">
      <span><i /></span><strong>Reconstructing trace</strong><small>Joining request context and related records…</small>
    </div>
    <div v-else-if="error" class="inspector-error">
      <span>404 / trace</span><strong>{{ error }}</strong><p>It may have been pruned from the recorder.</p>
      <button @click="router.push('/')">Return to live activity</button>
    </div>

    <div v-else-if="data" class="trace-workspace">
      <header class="trace-hero" :style="{ '--signal': entrySignalColor(data.entry) }">
        <div class="trace-hero__sheen" aria-hidden="true" />
        <div class="trace-hero__main">
          <button type="button" class="trace-back" @click="router.push('/')">
            <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M12 5 7 10l5 5"/></svg>
            <span>Live activity</span>
          </button>
          <div class="trace-heading__identity">
            <span class="trace-symbol"><SignalIcon :type="data.entry.type" size="lg" /></span>
            <div class="trace-heading__copy">
              <div class="trace-badges">
                <Badge :label="data.entry.type" :class-name="typeBadgeClass(data.entry.type)" />
                <Badge v-if="data.entry.type === 'request'" :label="String(data.entry.content?.method || 'GET')" :class-name="methodClass(String(data.entry.content?.method || ''))" />
                <Badge v-if="data.entry.type === 'request'" :label="String(data.entry.content?.status || '—')" :class-name="statusClass(data.entry.content?.status)" />
                <span class="trace-timestamp">{{ formatTimeLong(data.entry.created_at) }}</span>
              </div>
              <h1>{{ title }}</h1>
            </div>
          </div>
        </div>
        <div class="trace-hero__aside">
          <div class="trace-hero-stats">
            <span class="trace-hero-stat">
              <small>Duration</small>
              <strong>{{ entryDuration(data.entry) || '—' }}<i v-if="entryDuration(data.entry)">ms</i></strong>
            </span>
            <span class="trace-hero-stat">
              <small>Records</small>
              <strong>{{ timeline.length }}</strong>
            </span>
            <span class="trace-hero-stat" :class="{ 'is-hot': queryCount > 0 }">
              <small>SQL</small>
              <strong>{{ queryCount }}</strong>
            </span>
            <span class="trace-hero-stat" :class="{ 'is-danger': errorCount > 0 }">
              <small>Errors</small>
              <strong>{{ errorCount }}</strong>
            </span>
          </div>
          <div class="trace-heading__tools">
            <button type="button" class="trace-id-btn" :class="{ copied: copied }" @click="copyID">
              <span>{{ copied ? 'Copied' : data.entry.id.slice(0, 8) }}</span>
              <svg viewBox="0 0 20 20" aria-hidden="true"><rect x="7" y="7" width="9" height="9" rx="1"/><path d="M13 7V4H4v9h3"/></svg>
            </button>
          </div>
        </div>
      </header>

      <section v-if="comparison" class="split-comparison">
        <header>
          <div><span>Comparison mode</span><strong>Two traces, one coordinate system</strong></div>
          <button @click="router.replace(`/entries/${props.id}`)">Close split view</button>
        </header>
        <div class="comparison-columns">
          <article v-for="(trace, index) in [data, comparison]" :key="trace.entry.id" :style="{ '--signal': entrySignalColor(trace.entry) }">
            <span class="comparison-index">0{{ index + 1 }}</span>
            <Badge :label="trace.entry.type" :class-name="typeBadgeClass(trace.entry.type)" />
            <h2>{{ summarize(trace.entry) }}</h2>
            <div class="comparison-vitals">
              <span><small>Duration</small><strong>{{ entryDuration(trace.entry) || '—' }}<i v-if="entryDuration(trace.entry)">ms</i></strong></span>
              <span><small>Records</small><strong>{{ trace.batch.length }}</strong></span>
              <span><small>SQL</small><strong>{{ trace.batch.filter(item => item.type === 'query').length }}</strong></span>
            </div>
            <div class="comparison-bar"><i :style="{ width: `${Math.max(4, entryDuration(trace.entry) / Math.max(entryDuration(data.entry), entryDuration(comparison.entry), 1) * 100)}%` }" /></div>
          </article>
        </div>
      </section>

      <EntryInsight :entry="data.entry" :batch="data.batch" />

      <LLMInsightPanel
        v-if="llmSettings.enabled"
        :entries="data.batch"
        :context="`Trace ${data.entry.id} · ${data.entry.type}`"
        title="Trace intelligence"
      />

      <div class="inspector-grid">
        <aside class="execution-map" :style="{ '--map-signal': entrySignalColor(data.entry) }">
          <header class="execution-map__header">
            <div class="execution-map__title">
              <span class="panel-label">
                <i class="panel-label__mark" aria-hidden="true" />
                Execution map
              </span>
              <small>{{ timeline.length }} records · chronological</small>
            </div>
            <div class="execution-map__duration">
              <small>wall time</small>
              <strong>{{ totalDuration }}<i>ms</i></strong>
            </div>
          </header>

          <div class="time-ruler" aria-hidden="true">
            <div class="time-ruler__track">
              <span class="time-ruler__fill" :style="{ width: '100%' }" />
              <span v-for="tick in [0, 25, 50, 75, 100]" :key="tick" class="time-ruler__tick" :style="{ left: `${tick}%` }" />
            </div>
            <div class="time-ruler__labels">
              <span>0</span>
              <span>25%</span>
              <span>50%</span>
              <span>75%</span>
              <span>{{ totalDuration }}ms</span>
            </div>
          </div>

          <div class="timeline-list">
            <button
              v-for="(entry, index) in timeline"
              :key="entry.id"
              :class="{ active: entry.id === data.entry.id, error: isError(entry) }"
              :style="{
                '--signal': entrySignalColor(entry),
                '--row-delay': `${index * 35}ms`,
              }"
              @click="openTimelineEntry(entry)"
            >
              <span class="timeline-signal-stripe" aria-hidden="true" />
              <span class="timeline-index">{{ String(index + 1).padStart(2, '0') }}</span>
              <span class="timeline-moment" :title="formatTimeLong(entry.created_at)">
                <strong>{{ formatRecordTime(entry.created_at) }}</strong>
                <small>{{ formatRecordDate(entry.created_at) }}</small>
              </span>
              <span class="timeline-node"><SignalIcon :type="entry.type" size="sm" /></span>
              <span class="timeline-copy">
                <strong>{{ signalFor(entry.type).shortLabel }}</strong>
                <small>{{ summarize(entry) }}</small>
              </span>
              <span class="timeline-cost">{{ entryDuration(entry) ? `${entryDuration(entry)}ms` : '—' }}</span>
              <span class="timeline-bar" aria-hidden="true"><i :style="{ width: timelineWidth(entry) }" /></span>
            </button>
          </div>

          <footer class="execution-map__legend">
            <span
              v-for="item in timelineLegend"
              :key="item.type"
              class="execution-map__legend-item"
              :style="{ '--legend': item.color }"
            >
              <i aria-hidden="true" />
              {{ item.label }}
              <b>{{ item.count }}</b>
            </span>
          </footer>
        </aside>

        <div class="inspection-stage">
          <section class="trace-vitals">
            <div class="trace-vital trace-vital--primary">
              <span>Wall time</span>
              <strong>{{ entryDuration(data.entry) || '—' }}<small v-if="entryDuration(data.entry)">ms</small></strong>
              <i class="trace-vital-bar"><b :style="{ width: `${Math.min(100, entryDuration(data.entry) / 10)}%` }" /></i>
              <small>recorded span</small>
            </div>
            <div class="trace-vital" :class="{ 'is-hot': queryCount > 5 }">
              <span>SQL queries</span>
              <strong>{{ queryCount }}</strong>
              <small>in this trace</small>
            </div>
            <div v-if="signalEnabled('metric')" class="trace-vital">
              <span>Memory</span>
              <strong>{{ data.entry.content?.memory_mb || '—' }}<small v-if="data.entry.content?.memory_mb">MB</small></strong>
              <small>peak allocation</small>
            </div>
            <div v-if="signalEnabled('metric')" class="trace-vital">
              <span>{{ data.entry.type === 'metric' ? metricUnitLabel(data.entry) : 'Concurrency' }}</span>
              <strong>{{ data.entry.content?.value ?? '—' }}</strong>
              <small v-if="data.entry.type === 'metric'">{{ metricLanguageLabel(data.entry) }}</small>
              <small v-else>at completion</small>
            </div>
            <div class="trace-vital" :class="{ 'is-danger': errorCount > 0 }">
              <span>Exceptions</span>
              <strong>{{ errorCount }}</strong>
              <small>related records</small>
            </div>
          </section>

          <section class="trace-facts">
            <header>
              <span>Trace context</span>
              <small>immutable recording · click to copy IDs</small>
            </header>
            <div class="fact-grid">
              <button
                type="button"
                class="fact-cell"
                :class="{ copied: copiedField === 'entry' }"
                @click="copyField('entry', data.entry.id)"
              >
                <span>Entry</span>
                <strong>{{ data.entry.id }}</strong>
              </button>
              <button
                type="button"
                class="fact-cell"
                :class="{ copied: copiedField === 'request', muted: !data.entry.request_id }"
                :disabled="!data.entry.request_id"
                @click="copyField('request', data.entry.request_id || '')"
              >
                <span>Request</span>
                <strong>{{ data.entry.request_id || 'not linked' }}</strong>
              </button>
              <button
                type="button"
                class="fact-cell"
                :class="{ copied: copiedField === 'correlation', muted: !data.entry.correlation_id }"
                :disabled="!data.entry.correlation_id"
                @click="copyField('correlation', data.entry.correlation_id || '')"
              >
                <span>Correlation</span>
                <strong>{{ data.entry.correlation_id || 'not linked' }}</strong>
              </button>
              <div class="fact-cell is-static">
                <span>Origin</span>
                <strong>{{ data.entry.content?.ip || entryMeta(data.entry) }}</strong>
              </div>
            </div>
          </section>

          <section v-if="data.entry.type === 'exception'" class="exception-focus">
            <header><span>Unhandled exception</span><strong>{{ data.entry.content?.kind || 'Application runtime' }}</strong></header>
            <h2>{{ data.entry.content?.message || title }}</h2>
            <p>Microscope preserved the surrounding request and execution context for this failure.</p>
          </section>

          <SqlInspector
            v-if="data.entry.type === 'query'"
            :sql="String(data.entry.content?.sql || '')"
            :bindings="data.entry.content?.bindings"
            :duration="entryDuration(data.entry)"
            :connection="String(data.entry.content?.connection || 'default')"
          />

          <section v-if="data.entry.type === 'topic'" class="topic-focus">
            <header>
              <span>Redpanda topic activity</span>
              <strong>{{ String(data.entry.content?.action || 'observed').toUpperCase() }}</strong>
            </header>
            <div class="topic-route">
              <span class="topic-cluster">RP</span>
              <i />
              <strong>{{ data.entry.content?.topic || 'unnamed-topic' }}</strong>
              <i />
              <span>{{ data.entry.content?.partition != null ? `partition ${data.entry.content.partition}` : 'broker' }}</span>
            </div>
            <div class="topic-vitals">
              <span><small>Messages</small><strong>{{ data.entry.content?.message_count || 1 }}</strong></span>
              <span><small>Bytes</small><strong>{{ data.entry.content?.size_bytes || '—' }}</strong></span>
              <span><small>Offset</small><strong>{{ data.entry.content?.offset ?? '—' }}</strong></span>
              <span><small>Duration</small><strong>{{ entryDuration(data.entry) || '—' }}ms</strong></span>
            </div>
          </section>

          <ContentTabs v-if="data.content_tabs?.length" :tabs="data.content_tabs" />
        </div>
      </div>

      <RelatedTabs
        v-if="data.batch_groups?.length"
        :groups="data.batch_groups"
        :active-type="data.related_active_tab"
        :current-id="data.entry.id"
      />
    </div>
  </AppShell>
</template>
