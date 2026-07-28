<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import type { Entry } from '../types'
import { entryDuration, entrySignalColor, isError, metricLanguageLabel, signalFor, summarize } from '../utils'
import SignalIcon from './SignalIcon.vue'

const props = defineProps<{ entry: Entry; batch: Entry[] }>()
const router = useRouter()

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
  if (entry.type === 'metric') return `${metricLanguageLabel(entry)} runtime metric ${content.name || ''} measured ${content.value ?? '—'} ${content.unit || ''}`.trim()
  return summarize(entry)
})
const diagnosis = computed(() => {
  if (errors.value.length) return `${errors.value.length} error record${errors.value.length === 1 ? '' : 's'} occurred in the same execution context. The first failure is the strongest starting point.`
  if (queries.value.length > 8) return `${queries.value.length} SQL operations belong to this execution. Repeated statements or N+1 access may be contributing cost.`
  if (entryDuration(slowest.value) >= 500) return `${signalFor(slowest.value.type).label} is the dominant recorded span at ${entryDuration(slowest.value)}ms.`
  return `No failure signature was detected across ${props.batch.length} linked operations. The trace is consistent with expected execution.`
})
const actions = computed(() => {
  if (errors.value.length) return ['Open the earliest crimson operation below', 'Inspect stack and payload at the failure boundary', 'Compare against a successful request from the same route']
  if (queries.value.length > 8) return ['Group repeated SQL statements', 'Inspect query plans and missing indexes', 'Compare database time with total request time']
  if (entryDuration(slowest.value) >= 500) return ['Inspect the longest operation', 'Compare with a faster trace', 'Verify downstream and infrastructure timing']
  return ['Validate the response and linked side effects', 'Bookmark this trace as a healthy baseline']
})
const impactMetrics = computed(() => [
  { key: 'errors', label: 'Errors', value: String(errors.value.length), hot: errors.value.length > 0, danger: errors.value.length > 0 },
  { key: 'sql', label: 'SQL', value: String(queries.value.length), hot: queries.value.length > 5 },
  { key: 'cost', label: 'Entry cost', value: entryDuration(props.entry) ? `${entryDuration(props.entry)}` : '—', suffix: entryDuration(props.entry) ? 'ms' : '' },
  { key: 'longest', label: 'Longest', value: slowest.value ? signalFor(slowest.value.type).shortLabel : '—' },
])
const evidenceDialOffset = computed(() => {
  const circumference = 138
  const ratio = Math.min(1, props.batch.length / 12)
  return circumference - ratio * circumference
})
const sequenceTypes = computed(() => {
  const seen = new Set<string>()
  return ordered.value
    .filter(item => {
      if (seen.has(item.type)) return false
      seen.add(item.type)
      return true
    })
    .map(item => ({ type: item.type, color: entrySignalColor(item), label: signalFor(item.type).shortLabel }))
})
</script>

<template>
  <section class="finding-console" :class="`finding-console--${status.tone}`" :style="{ '--signal': entrySignalColor(entry) }">
    <div class="finding-console__sheen" aria-hidden="true" />

    <header class="finding-header">
      <div class="finding-status" :class="`is-${status.tone}`">
        <span class="finding-status__icon" aria-hidden="true">
          <!-- NOMINAL: steady check -->
          <svg v-if="status.tone === 'healthy'" class="finding-status__glyph" viewBox="0 0 16 16">
            <circle cx="8" cy="8" r="5.5" fill="none" stroke="currentColor" stroke-width="1.15" />
            <path d="M5.4 8.1 7.1 9.8 10.8 6.1" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round" />
          </svg>
          <!-- SLOW: stopwatch -->
          <svg v-else-if="status.tone === 'warning'" class="finding-status__glyph" viewBox="0 0 16 16">
            <circle cx="8.5" cy="9" r="4.5" fill="none" stroke="currentColor" stroke-width="1.1" />
            <path d="M8.5 9V6.2M6.5 3.5h4" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" />
            <path d="M7 3.5V2.5h3v1" fill="none" stroke="currentColor" stroke-width="1" />
            <path d="M11.5 11.5l1.5 1.5" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round" />
          </svg>
          <!-- FAILURE: alert -->
          <svg v-else class="finding-status__glyph" viewBox="0 0 16 16">
            <path d="M8 2.5 14 13H2L8 2.5z" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linejoin="round" />
            <path d="M8 6.5v3.5M8 11.5v.5" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" />
          </svg>
        </span>
        <div class="finding-status__copy">
          <span class="finding-status__code">{{ status.code }}</span>
          <strong>{{ status.label }}</strong>
        </div>
      </div>

      <div class="finding-title">
        <div class="finding-title__eyebrow">
          <span class="finding-title__icon"><SignalIcon :type="entry.type" size="sm" /></span>
          <small>Primary finding</small>
        </div>
        <h2>{{ finding }}</h2>
      </div>

      <div class="finding-evidence-count">
        <div class="evidence-dial" :aria-label="`${batch.length} linked operations`">
          <svg viewBox="0 0 52 52" aria-hidden="true">
            <circle cx="26" cy="26" r="22" class="dial-track" />
            <circle cx="26" cy="26" r="22" class="dial-fill" :style="{ strokeDashoffset: evidenceDialOffset }" />
          </svg>
          <strong>{{ batch.length }}</strong>
        </div>
        <div class="evidence-copy">
          <span class="evidence-copy__label">linked operations</span>
          <small>in this execution batch</small>
        </div>
      </div>
    </header>

    <div class="finding-body">
      <article class="diagnosis-panel">
        <header>
          <span class="panel-label">
            <i class="panel-label__mark" aria-hidden="true" />
            Diagnosis
          </span>
          <small>deterministic analysis</small>
        </header>
        <blockquote class="diagnosis-quote">{{ diagnosis }}</blockquote>
        <div class="impact-grid">
          <div
            v-for="metric in impactMetrics"
            :key="metric.key"
            class="impact-tile"
            :class="{ 'is-hot': metric.hot, 'is-danger': metric.danger }"
          >
            <small>{{ metric.label }}</small>
            <strong>{{ metric.value }}<i v-if="metric.suffix">{{ metric.suffix }}</i></strong>
          </div>
        </div>
      </article>

      <article class="investigation-panel">
        <header>
          <span class="panel-label">
            <i class="panel-label__mark" aria-hidden="true" />
            Investigation path
          </span>
          <small>recommended order</small>
        </header>
        <ol class="investigation-steps">
          <li v-for="(action, index) in actions" :key="action">
            <span class="step-index">{{ String(index + 1).padStart(2, '0') }}</span>
            <span class="step-copy">{{ action }}</span>
            <svg class="step-chevron" viewBox="0 0 20 20" aria-hidden="true"><path d="m8 5 5 5-5 5" /></svg>
          </li>
        </ol>
      </article>
    </div>

    <div class="trace-sequence">
      <header>
        <div class="trace-sequence__lead">
          <span class="panel-label">
            <i class="panel-label__mark" aria-hidden="true" />
            Trace sequence
          </span>
          <small>Why these operations are grouped</small>
        </div>
        <p class="trace-sequence__hint">They share the same request or batch context. Read left to right to reconstruct the execution, then select any card to inspect its payload.</p>
        <div class="sequence-legend" aria-label="Card legend">
          <span class="sequence-legend__item">
            <i class="sequence-legend__stripe" aria-hidden="true" />
            Top stripe · signal type
          </span>
          <span class="sequence-legend__item">
            <i class="sequence-legend__bar" aria-hidden="true" /><i class="sequence-legend__bar is-short" aria-hidden="true" />
            Bottom bar · relative duration
          </span>
        </div>
      </header>

      <div class="sequence-scroller">
        <div class="sequence-scroller__fade sequence-scroller__fade--left" aria-hidden="true" />
        <div class="sequence-scroller__fade sequence-scroller__fade--right" aria-hidden="true" />
        <div class="sequence-rail">
          <button
            v-for="(item, index) in ordered.slice(0, 18)"
            :key="item.id"
            :class="{ active: item.id === entry.id, error: isError(item) }"
            :style="{
              '--item': entrySignalColor(item),
              '--cost': `${Math.max(4, entryDuration(item) / maxDuration * 100)}%`,
              '--sequence-delay': `${index * 45}ms`,
            }"
            :aria-label="`Inspect ${signalFor(item.type).label}: ${summarize(item)}`"
            @click="router.push(`/entries/${item.id}`)"
          >
            <span class="sequence-signal-stripe" aria-hidden="true" :title="`${signalFor(item.type).label} signal`" />
            <i class="sequence-index">{{ String(index + 1).padStart(2, '0') }}</i>
            <span class="sequence-icon"><SignalIcon :type="item.type" size="sm" /></span>
            <span class="sequence-copy">
              <small>{{ signalFor(item.type).shortLabel }}</small>
              <strong>{{ summarize(item) }}</strong>
            </span>
            <span class="sequence-cost">{{ entryDuration(item) ? `${entryDuration(item)}ms` : 'event' }}</span>
            <span class="sequence-bar" aria-hidden="true"><i /></span>
          </button>
        </div>
      </div>

      <footer>
        <span class="sequence-footer__context"><i /> shared context</span>
        <div class="sequence-footer__types" aria-label="Signal types in sequence">
          <span
            v-for="item in sequenceTypes"
            :key="item.type"
            class="sequence-type-chip"
            :style="{ '--chip': item.color }"
          >{{ item.label }}</span>
        </div>
        <strong v-if="slowest">Longest: {{ signalFor(slowest.type).label }} · {{ entryDuration(slowest) || 0 }}ms</strong>
        <span class="sequence-footer__total">{{ ordered.length }} total</span>
      </footer>
    </div>
  </section>
</template>
