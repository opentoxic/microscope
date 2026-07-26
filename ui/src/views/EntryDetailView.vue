<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { getEntry } from '../api/client'
import type { Entry, EntryDetailResponse } from '../types'
import AppShell from '../components/AppShell.vue'
import Badge from '../components/Badge.vue'
import ContentTabs from '../components/ContentTabs.vue'
import RelatedTabs from '../components/RelatedTabs.vue'
import EntryInsight from '../components/EntryInsight.vue'
import LLMInsightPanel from '../components/LLMInsightPanel.vue'
import SignalIcon from '../components/SignalIcon.vue'
import { entryDuration, entryMeta, formatClock, formatTimeLong, metricLanguageLabel, metricUnitLabel, methodClass, signalFor, statusClass, summarize, timeAgo, typeBadgeClass } from '../utils'
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

onMounted(async () => {
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
      <span><i /></span><strong>Reconstructing trace</strong><small>Joining request context and related signals…</small>
    </div>
    <div v-else-if="error" class="inspector-error">
      <span>404 / trace</span><strong>{{ error }}</strong><p>It may have been pruned from the recorder.</p>
      <button @click="router.push('/')">Return to live activity</button>
    </div>

    <div v-else-if="data" class="trace-workspace">
      <header class="trace-heading" :style="{ '--signal': signalFor(data.entry.type).color }">
        <div class="trace-heading__identity">
          <span class="trace-symbol"><SignalIcon :type="data.entry.type" size="lg" /></span>
          <div>
            <div class="trace-badges">
              <Badge :label="data.entry.type" :class-name="typeBadgeClass(data.entry.type)" />
              <Badge v-if="data.entry.type === 'request'" :label="String(data.entry.content?.method || 'GET')" :class-name="methodClass(String(data.entry.content?.method || ''))" />
              <Badge v-if="data.entry.type === 'request'" :label="String(data.entry.content?.status || '—')" :class-name="statusClass(data.entry.content?.status)" />
              <span>{{ formatTimeLong(data.entry.created_at) }}</span>
            </div>
            <h1>{{ title }}</h1>
          </div>
        </div>
        <div class="trace-heading__tools">
          <button @click="copyID">
            <span>{{ copied ? 'Copied' : data.entry.id.slice(0, 8) }}</span>
            <svg viewBox="0 0 20 20"><rect x="7" y="7" width="9" height="9" rx="1"/><path d="M13 7V4H4v9h3"/></svg>
          </button>
        </div>
      </header>

      <section v-if="comparison" class="split-comparison">
        <header>
          <div><span>Comparison mode</span><strong>Two traces, one coordinate system</strong></div>
          <button @click="router.replace(`/entries/${props.id}`)">Close split view</button>
        </header>
        <div class="comparison-columns">
          <article v-for="(trace, index) in [data, comparison]" :key="trace.entry.id" :style="{ '--signal': signalFor(trace.entry.type).color }">
            <span class="comparison-index">0{{ index + 1 }}</span>
            <Badge :label="trace.entry.type" :class-name="typeBadgeClass(trace.entry.type)" />
            <h2>{{ summarize(trace.entry) }}</h2>
            <div class="comparison-vitals">
              <span><small>Duration</small><strong>{{ entryDuration(trace.entry) || '—' }}<i v-if="entryDuration(trace.entry)">ms</i></strong></span>
              <span><small>Signals</small><strong>{{ trace.batch.length }}</strong></span>
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
        <aside class="execution-map">
          <header>
            <div><span>Execution map</span><small>{{ timeline.length }} signals</small></div>
            <strong>{{ totalDuration }}ms</strong>
          </header>
          <div class="time-ruler"><span>0</span><span>25%</span><span>50%</span><span>75%</span><span>{{ totalDuration }}ms</span></div>
          <div class="timeline-list">
            <button
              v-for="entry in timeline"
              :key="entry.id"
              :class="{ active: entry.id === data.entry.id }"
              :style="{ '--signal': signalFor(entry.type).color }"
              @click="openTimelineEntry(entry)"
            >
              <span class="timeline-node"><SignalIcon :type="entry.type" size="sm" /></span>
              <span class="timeline-copy"><strong>{{ signalFor(entry.type).shortLabel }}</strong><small>{{ summarize(entry) }}</small></span>
              <span class="timeline-cost">{{ entryDuration(entry) ? `${entryDuration(entry)}ms` : formatClock(entry.created_at) }}</span>
              <span class="timeline-bar"><i :style="{ width: timelineWidth(entry) }" /></span>
            </button>
          </div>
          <footer>
            <span><i style="background: var(--blue)" /> HTTP</span>
            <span><i style="background: var(--green)" /> SQL</span>
            <span><i style="background: var(--crimson)" /> Error</span>
          </footer>
        </aside>

        <div class="inspection-stage">
          <section class="trace-vitals">
            <div><span>Wall time</span><strong>{{ entryDuration(data.entry) || '—' }}<small v-if="entryDuration(data.entry)">ms</small></strong><i><b :style="{ width: `${Math.min(100, entryDuration(data.entry) / 10)}%` }" /></i></div>
            <div><span>SQL queries</span><strong>{{ queryCount }}</strong><small>in this trace</small></div>
            <div v-if="signalEnabled('metric')"><span>Memory</span><strong>{{ data.entry.content?.memory_mb || '—' }}<small v-if="data.entry.content?.memory_mb">MB</small></strong><small>peak allocation</small></div>
            <div v-if="signalEnabled('metric')"><span>{{ data.entry.type === 'metric' ? metricUnitLabel(data.entry) : 'Concurrency' }}</span><strong>{{ data.entry.content?.value ?? '—' }}</strong><small v-if="data.entry.type === 'metric'">{{ metricLanguageLabel(data.entry) }}</small><small v-else>at completion</small></div>
            <div><span>Exceptions</span><strong :class="{ danger: errorCount }">{{ errorCount }}</strong><small>related signals</small></div>
          </section>

          <section class="trace-facts">
            <header><span>Trace context</span><small>immutable recording</small></header>
            <div class="fact-grid">
              <div><span>Entry</span><strong>{{ data.entry.id }}</strong></div>
              <div><span>Request</span><strong>{{ data.entry.request_id || 'not linked' }}</strong></div>
              <div><span>Correlation</span><strong>{{ data.entry.correlation_id || 'not linked' }}</strong></div>
              <div><span>Origin</span><strong>{{ data.entry.content?.ip || entryMeta(data.entry) }}</strong></div>
            </div>
          </section>

          <section v-if="data.entry.type === 'exception'" class="exception-focus">
            <header><span>Unhandled exception</span><strong>{{ data.entry.content?.kind || 'Application runtime' }}</strong></header>
            <h2>{{ data.entry.content?.message || title }}</h2>
            <p>Signal preserved the surrounding request and execution context for this failure.</p>
          </section>

          <section v-if="data.entry.type === 'query'" class="query-focus">
            <header><span>SQL inspector</span><strong>{{ entryDuration(data.entry) }}ms</strong></header>
            <pre>{{ data.entry.content?.sql }}</pre>
            <footer><span>Connection · {{ data.entry.content?.connection || 'default' }}</span><span>{{ data.entry.content?.bindings ? 'Bindings captured' : 'No bindings' }}</span></footer>
          </section>

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

      <RelatedTabs v-if="data.batch_groups?.length" :groups="data.batch_groups" :active-type="data.related_active_tab" />
    </div>
  </AppShell>
</template>
