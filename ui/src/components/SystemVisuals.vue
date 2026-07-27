<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Entry, EntryType } from '../types'
import { detectMetricLanguage, entryDuration, isError, signalFor } from '../utils'

const props = defineProps<{ entries: Entry[] }>()
const router = useRouter()
const tooltip = ref<{ x: number; y: number; title: string; value: string; detail: string } | null>(null)

function showTooltip(event: MouseEvent, title: string, value: string, detail: string) {
  tooltip.value = {
    x: Math.min(window.innerWidth - 198, Math.max(10, event.clientX + 12)),
    y: Math.max(8, event.clientY - 72),
    title,
    value,
    detail,
  }
}

function hideTooltip() {
  tooltip.value = null
}

const LANGUAGE_SHORT_CODES: Record<string, string> = {
  go: 'GO',
  python: 'PY',
  node: 'JS',
  ruby: 'RB',
  php: 'PHP',
  elixir: 'EX',
}

const serviceLanguageCode = computed(() => {
  const counts = new Map<string, number>()
  for (const entry of props.entries) {
    if (entry.type !== 'metric') continue
    const language = detectMetricLanguage(entry)
    if (language === 'unknown') continue
    counts.set(language, (counts.get(language) || 0) + 1)
  }
  const top = [...counts].sort((a, b) => b[1] - a[1])[0]
  return LANGUAGE_SHORT_CODES[top?.[0] || 'go'] || 'GO'
})

const chronological = computed(() => [...props.entries].sort((a, b) => +new Date(a.created_at) - +new Date(b.created_at)))
const activityBuckets = computed(() => {
  const buckets = Array.from({ length: 24 }, () => 0)
  if (!chronological.value.length) return buckets
  const first = +new Date(chronological.value[0].created_at)
  const last = Math.max(first + 1, +new Date(chronological.value.at(-1)!.created_at))
  for (const entry of chronological.value) {
    const index = Math.min(23, Math.floor((+new Date(entry.created_at) - first) / (last - first) * 24))
    buckets[index]++
  }
  return buckets
})
const maxActivity = computed(() => Math.max(1, ...activityBuckets.value))
const linePoints = computed(() => activityBuckets.value.map((count, index) => {
  const x = index / 23 * 700
  const y = 128 - count / maxActivity.value * 102
  return `${x.toFixed(1)},${y.toFixed(1)}`
}).join(' '))
const areaPoints = computed(() => `0,140 ${linePoints.value} 700,140`)

const latencyBuckets = computed(() => {
  const definitions = [
    { label: '<10', min: 0, max: 10, count: 0 },
    { label: '10–50', min: 10, max: 50, count: 0 },
    { label: '50–100', min: 50, max: 100, count: 0 },
    { label: '100–500', min: 100, max: 500, count: 0 },
    { label: '500+', min: 500, max: Infinity, count: 0 },
  ]
  props.entries.forEach((entry) => {
    const duration = entryDuration(entry)
    if (!duration) return
    definitions.find(bucket => duration >= bucket.min && duration < bucket.max)!.count++
  })
  return definitions
})
const maxLatencyBucket = computed(() => Math.max(1, ...latencyBuckets.value.map(bucket => bucket.count)))

const dependencyDefinitions: Array<{ type: EntryType; label: string; x: number; y: number }> = [
  { type: 'query', label: 'Postgres', x: 58, y: 45 },
  { type: 'redis', label: 'Redis', x: 58, y: 145 },
  { type: 'topic', label: 'Redpanda', x: 480, y: 45 },
  { type: 'http-client', label: 'External', x: 480, y: 145 },
  { type: 'job', label: 'Workers', x: 270, y: 196 },
]
const dependencies = computed(() => dependencyDefinitions.map(item => ({
  ...item,
  count: props.entries.filter(entry => entry.type === item.type).length,
  color: signalFor(item.type).color,
})))
const errors = computed(() => props.entries.filter(isError).length)
const timed = computed(() => props.entries.map(entryDuration).filter(Boolean))
const average = computed(() => timed.value.length ? Math.round(timed.value.reduce((sum, value) => sum + value, 0) / timed.value.length) : 0)
const busiest = computed(() => {
  const counts = new Map<string, number>()
  props.entries.forEach(entry => counts.set(entry.type, (counts.get(entry.type) || 0) + 1))
  const top = [...counts].sort((a, b) => b[1] - a[1])[0]
  return top ? { signal: signalFor(top[0]), count: top[1] } : null
})
</script>

<template>
  <section class="system-visuals">
    <header class="visuals-heading">
      <div><span>System topology</span><strong>Your runtime as a connected system</strong></div>
      <p>Every shape below is calculated from the current recording window.</p>
    </header>

    <div class="visuals-grid">
      <article class="flow-visual" @mouseleave="hideTooltip">
        <header><span>Activity current</span><small>24 adaptive intervals</small></header>
        <svg viewBox="0 0 700 150" preserveAspectRatio="none" role="img" aria-label="Activity over time">
          <defs><linearGradient id="activity-fill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#20d9ee" stop-opacity=".28"/><stop offset="1" stop-color="#20d9ee" stop-opacity="0"/></linearGradient></defs>
          <path class="chart-grid" d="M0 25H700M0 60H700M0 95H700M0 130H700"/>
          <polygon :points="areaPoints" fill="url(#activity-fill)" />
          <polyline :points="linePoints" />
          <circle
            v-for="(count, index) in activityBuckets"
            :key="index"
            :cx="index / 23 * 700"
            :cy="128 - count / maxActivity * 102"
            r="4"
            tabindex="0"
            @mousemove="showTooltip($event, `Interval ${index + 1}`, `${count} signals`, `${Math.round(count / maxActivity * 100)}% of peak activity`)"
          />
        </svg>
        <footer><span>earliest</span><strong>{{ entries.length }} total signals</strong><span>now</span></footer>
      </article>

      <article class="topology-visual" @mouseleave="hideTooltip">
        <header><span>Service interactions</span><small>edge strength = activity</small></header>
        <svg viewBox="0 0 600 250" role="img" aria-label="Service dependency topology">
          <g v-for="node in dependencies" :key="node.type">
            <line :x1="300" :y1="112" :x2="node.x + 32" :y2="node.y + 20" :class="{ active: node.count }" :style="{ '--edge': node.color, '--weight': Math.min(5, 1 + node.count / 5) }" />
          </g>
          <g class="service-core"><circle cx="300" cy="112" r="42"/><circle cx="300" cy="112" r="31"/><text x="300" y="109">{{ serviceLanguageCode }}</text><text x="300" y="126">SERVICE</text></g>
          <g v-for="node in dependencies" :key="`node-${node.type}`" class="dependency-node" :class="{ active: node.count }" :transform="`translate(${node.x} ${node.y})`" :style="{ '--node': node.color }" tabindex="0" @mousemove="showTooltip($event, node.label, `${node.count} operations`, `${signalFor(node.type).label} in this window`)">
            <rect width="66" height="42"/><circle cx="12" cy="13" r="3"/><text x="33" y="21">{{ node.label }}</text><text x="33" y="34">{{ node.count }} ops</text>
          </g>
        </svg>
      </article>

      <article class="latency-visual" @mouseleave="hideTooltip">
        <header><span>Latency distribution</span><small>milliseconds</small></header>
        <div class="histogram">
          <span v-for="bucket in latencyBuckets" :key="bucket.label" @mousemove="showTooltip($event, `${bucket.label} ms`, `${bucket.count} operations`, `${Math.round(bucket.count / Math.max(1, timed.length) * 100)}% of timed signals`)">
            <b>{{ bucket.count }}</b>
            <i><em :style="{ height: `${Math.max(bucket.count ? 8 : 1, bucket.count / maxLatencyBucket * 100)}%` }" /></i>
            <small>{{ bucket.label }}</small>
          </span>
        </div>
      </article>

      <article class="evidence-visual" @mouseleave="hideTooltip">
        <header><span>Evidence matrix</span><small>latest {{ Math.min(entries.length, 56) }}</small></header>
        <div class="evidence-matrix">
          <button
            v-for="entry in entries.slice(0, 56)"
            :key="entry.id"
            :class="{ error: isError(entry) }"
            :style="{ '--cell': signalFor(entry.type).color, '--intensity': Math.min(1, .35 + entryDuration(entry) / 800) }"
            :aria-label="`Open ${signalFor(entry.type).label} entry`"
            @mousemove="showTooltip($event, signalFor(entry.type).label, entryDuration(entry) ? `${entryDuration(entry)}ms` : 'Untimed', String(entry.content?.path || entry.content?.name || entry.id.slice(0, 12)))"
            @click="router.push(`/entries/${entry.id}`)"
          />
          <i v-for="index in Math.max(0, 56 - entries.length)" :key="`empty-${index}`" class="empty" />
        </div>
        <div class="evidence-readout">
          <span><small>Busiest</small><strong :style="{ color: busiest?.signal.color }">{{ busiest?.signal.shortLabel || '—' }} <i>{{ busiest?.count || 0 }}</i></strong></span>
          <span><small>Average cost</small><strong>{{ average || '—' }}<i v-if="average">ms</i></strong></span>
          <span><small>Error evidence</small><strong :class="{ danger: errors }">{{ errors }}</strong></span>
        </div>
      </article>
    </div>
    <div v-if="tooltip" class="chart-tooltip" :style="{ left: `${tooltip.x}px`, top: `${tooltip.y}px` }"><small>{{ tooltip.title }}</small><strong>{{ tooltip.value }}</strong><span>{{ tooltip.detail }}</span></div>
  </section>
</template>
