<script setup lang="ts">
import { computed } from 'vue'
import type { Entry } from '../types'
import { entryDuration, isError, signalFor, summarize } from '../utils'
import SignalIcon from './SignalIcon.vue'

const props = defineProps<{ entry: Entry; batch: Entry[] }>()

const ordered = computed(() => [...props.batch].sort((a, b) => +new Date(a.created_at) - +new Date(b.created_at)))
const errors = computed(() => ordered.value.filter(isError))
const queries = computed(() => ordered.value.filter(entry => entry.type === 'query'))
const slowest = computed(() => [...ordered.value].sort((a, b) => entryDuration(b) - entryDuration(a))[0])
const maxDuration = computed(() => Math.max(1, ...ordered.value.map(entryDuration)))
const status = computed(() => {
  if (isError(props.entry) || errors.value.length) return { label: 'Action required', tone: 'critical', code: 'FAILURE' }
  if (entryDuration(props.entry) >= 500 || entryDuration(slowest.value) >= 500) return { label: 'Performance regression', tone: 'warning', code: 'SLOW' }
  return { label: 'Operating normally', tone: 'healthy', code: 'NOMINAL' }
})
const finding = computed(() => {
  const entry = props.entry
  const content = entry.content || {}
  if (entry.type === 'exception') return String(content.message || 'An unhandled exception interrupted application execution')
  if (entry.type === 'request' && Number(content.status) >= 500) return `${String(content.method || 'HTTP').toUpperCase()} ${content.path || '/'} returned ${content.status}`
  if (entry.type === 'query' && entryDuration(entry) >= 100) return `SQL execution occupied ${entryDuration(entry)}ms of this trace`
  if (entry.type === 'topic') return `${String(content.action || 'Topic activity')} on ${content.topic || 'an unnamed Redpanda topic'}`
  if (entry.type === 'metric') return `${content.name || 'Runtime metric'} measured ${content.value ?? '—'} ${content.unit || ''}`.trim()
  return summarize(entry)
})
const diagnosis = computed(() => {
  if (errors.value.length) return `${errors.value.length} error signal${errors.value.length === 1 ? '' : 's'} occurred in the same execution context. The first failure is the strongest starting point.`
  if (queries.value.length > 8) return `${queries.value.length} SQL operations were correlated with this execution. Repeated statements or N+1 access may be contributing cost.`
  if (entryDuration(slowest.value) >= 500) return `${signalFor(slowest.value.type).label} is the dominant recorded span at ${entryDuration(slowest.value)}ms.`
  return `No failure signature was detected across ${props.batch.length} correlated signals. The evidence is consistent with expected execution.`
})
const actions = computed(() => {
  if (errors.value.length) return ['Open the earliest crimson evidence below', 'Inspect stack and payload at the failure boundary', 'Compare against a successful request from the same route']
  if (queries.value.length > 8) return ['Group repeated SQL statements', 'Inspect query plans and missing indexes', 'Compare database time with total request time']
  if (entryDuration(slowest.value) >= 500) return ['Inspect the dominant span', 'Compare with a faster trace', 'Verify downstream and infrastructure timing']
  return ['Validate the response and correlated side effects', 'Bookmark this trace as a healthy baseline']
})
const relativePosition = (entry: Entry) => {
  const first = +new Date(ordered.value[0]?.created_at || 0)
  const last = Math.max(first + 1, +new Date(ordered.value.at(-1)?.created_at || 0))
  return Math.max(1, Math.min(97, (+new Date(entry.created_at) - first) / (last - first) * 96))
}
</script>

<template>
  <section class="finding-console" :style="{ '--signal': signalFor(entry.type).color }">
    <header class="finding-header">
      <div class="finding-status" :class="`is-${status.tone}`">
        <span><i />{{ status.code }}</span>
        <strong>{{ status.label }}</strong>
      </div>
      <div class="finding-title">
        <small>Primary finding</small>
        <h2>{{ finding }}</h2>
      </div>
      <div class="finding-evidence-count">
        <strong>{{ batch.length }}</strong><span>correlated<br>signals</span>
      </div>
    </header>

    <div class="finding-body">
      <article class="diagnosis-panel">
        <header><span>Diagnosis</span><small>deterministic evidence</small></header>
        <p>{{ diagnosis }}</p>
        <div class="impact-strip">
          <span><small>Errors</small><strong :class="{ danger: errors.length }">{{ errors.length }}</strong></span>
          <span><small>SQL</small><strong>{{ queries.length }}</strong></span>
          <span><small>Entry cost</small><strong>{{ entryDuration(entry) || '—' }}<i v-if="entryDuration(entry)">ms</i></strong></span>
          <span><small>Dominant</small><strong>{{ slowest ? signalFor(slowest.type).shortLabel : '—' }}</strong></span>
        </div>
      </article>

      <article class="investigation-panel">
        <header><span>Investigation path</span><small>recommended order</small></header>
        <ol>
          <li v-for="(action, index) in actions" :key="action"><i>0{{ index + 1 }}</i><span>{{ action }}</span></li>
        </ol>
      </article>
    </div>

    <div class="evidence-timeline">
      <header><span>Correlated evidence</span><small>execution order · marker width indicates relative cost</small></header>
      <div class="evidence-track">
        <span class="track-line" />
        <button
          v-for="item in ordered.slice(0, 18)"
          :key="item.id"
          :class="{ active: item.id === entry.id, error: isError(item) }"
          :style="{ '--item': signalFor(item.type).color, left: `${relativePosition(item)}%`, '--cost': `${Math.max(5, entryDuration(item) / maxDuration * 100)}%` }"
          :title="`${summarize(item)} · ${entryDuration(item) || 0}ms`"
        >
          <SignalIcon :type="item.type" size="sm" />
          <b /><span>{{ signalFor(item.type).shortLabel }}</span>
        </button>
      </div>
      <footer>
        <span>trace start</span>
        <strong v-if="slowest">Dominant evidence · {{ signalFor(slowest.type).label }} · {{ entryDuration(slowest) || 0 }}ms</strong>
        <span>trace end</span>
      </footer>
    </div>
  </section>
</template>
