<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Entry, EntryType } from '../types'
import { detectMetricLanguage, entryDuration, entrySignalColor, isError, METRIC_LANGUAGE_COLORS, signalFor } from '../utils'

const props = defineProps<{ entries: Entry[] }>()
const router = useRouter()

type ChartKind = 'activity' | 'topology' | 'latency' | 'evidence'

const CHART_LABELS: Record<ChartKind, string> = {
  activity: 'Activity current',
  topology: 'Service interactions',
  latency: 'Latency distribution',
  evidence: 'Evidence matrix',
}

const tooltip = ref<{
  x: number
  y: number
  placement: 'above' | 'below'
  chart: ChartKind
  chartLabel: string
  position: string
  title: string
  value: string
  detail: string
} | null>(null)

const TOOLTIP_WIDTH = 212
const TOOLTIP_HEIGHT = 128
const TOOLTIP_GAP = 10

function screenAnchor(target: Element, event: MouseEvent) {
  if (target instanceof SVGCircleElement) {
    const svg = target.ownerSVGElement
    const matrix = target.getScreenCTM()
    if (svg && matrix) {
      const point = svg.createSVGPoint()
      point.x = target.cx.baseVal.value
      point.y = target.cy.baseVal.value
      const mapped = point.matrixTransform(matrix)
      return { x: mapped.x, y: mapped.y }
    }
  }

  if (target instanceof SVGGraphicsElement) {
    const svg = target.ownerSVGElement
    const matrix = target.getScreenCTM()
    if (svg && matrix) {
      const box = target.getBBox()
      const point = svg.createSVGPoint()
      point.x = box.x + box.width / 2
      point.y = box.y + box.height / 2
      const mapped = point.matrixTransform(matrix)
      return { x: mapped.x, y: mapped.y }
    }
  }

  if (target instanceof Element) {
    const rect = target.getBoundingClientRect()
    if (rect.width > 0 || rect.height > 0) {
      return { x: rect.left + rect.width / 2, y: rect.top + rect.height / 2 }
    }
  }

  return { x: event.clientX, y: event.clientY }
}

function showChartTooltip(
  event: MouseEvent,
  chart: ChartKind,
  position: string,
  title: string,
  value: string,
  detail: string,
) {
  const target = event.currentTarget
  if (!(target instanceof Element)) return

  const anchor = screenAnchor(target, event)
  let placement: 'above' | 'below' = 'above'
  let x = anchor.x - TOOLTIP_WIDTH / 2
  let y = anchor.y - TOOLTIP_HEIGHT - TOOLTIP_GAP

  if (y < 8) {
    placement = 'below'
    y = anchor.y + TOOLTIP_GAP
  }

  x = Math.min(window.innerWidth - TOOLTIP_WIDTH - 10, Math.max(10, x))
  y = Math.min(window.innerHeight - TOOLTIP_HEIGHT - 10, Math.max(8, y))

  tooltip.value = {
    x,
    y,
    placement,
    chart,
    chartLabel: CHART_LABELS[chart],
    position,
    title,
    value,
    detail,
  }
}

function hideTooltip() {
  tooltip.value = null
}

function activityPosition(index: number) {
  const total = 24
  const pct = Math.round((index / Math.max(1, total - 1)) * 100)
  const slice = index === 0 ? 'earliest slice' : index === total - 1 ? 'latest slice' : `${pct}% through window`
  return `Interval ${index + 1}/${total} · ${slice}`
}

const SERVICE_CENTER = { x: 300, y: 118 }
const SERVICE_RADIUS = 44
const NODE_CARD = { width: 68, height: 38 }

function topologyPosition(node: { x: number; y: number; label: string }) {
  const horizontal = node.x < 280 ? 'West' : node.x > 320 ? 'East' : 'Center'
  const vertical = node.y < 100 ? 'North' : node.y > 124 ? 'South' : 'Mid'
  return `${vertical} ${horizontal} · ${node.label}`
}

function topologyEdge(node: { x: number; y: number }) {
  const nx = node.x + NODE_CARD.width / 2
  const ny = node.y + NODE_CARD.height / 2
  const dx = nx - SERVICE_CENTER.x
  const dy = ny - SERVICE_CENTER.y
  const dist = Math.hypot(dx, dy) || 1
  const x1 = SERVICE_CENTER.x + (dx / dist) * SERVICE_RADIUS
  const y1 = SERVICE_CENTER.y + (dy / dist) * SERVICE_RADIUS

  let x2 = nx
  let y2 = ny
  if (Math.abs(dx) > Math.abs(dy)) {
    x2 = dx > 0 ? node.x : node.x + NODE_CARD.width
    y2 = ny
  } else {
    x2 = nx
    y2 = dy > 0 ? node.y : node.y + NODE_CARD.height
  }

  return { x1, y1, x2, y2 }
}

function latencyPosition(index: number, label: string) {
  return `Bucket ${index + 1}/5 · ${label} ms`
}

function evidencePosition(index: number) {
  const col = (index % 14) + 1
  const row = Math.floor(index / 14) + 1
  return `Row ${row} · Col ${col} · cell ${index + 1}/56`
}

const LANGUAGE_SHORT_CODES: Record<string, string> = {
  go: 'GO',
  python: 'PY',
  node: 'JS',
  ruby: 'RB',
  php: 'PHP',
  elixir: 'EX',
}

const serviceLanguage = computed(() => {
  const counts = new Map<string, number>()
  for (const entry of props.entries) {
    if (entry.type !== 'metric') continue
    const language = detectMetricLanguage(entry)
    if (language === 'unknown') continue
    counts.set(language, (counts.get(language) || 0) + 1)
  }
  const top = [...counts].sort((a, b) => b[1] - a[1])[0]
  return top?.[0] || 'unknown'
})
const serviceLanguageCode = computed(() => {
  if (serviceLanguage.value === 'unknown') return 'APP'
  return LANGUAGE_SHORT_CODES[serviceLanguage.value] || 'APP'
})
const serviceLanguageColor = computed(() => METRIC_LANGUAGE_COLORS[serviceLanguage.value as keyof typeof METRIC_LANGUAGE_COLORS] || METRIC_LANGUAGE_COLORS.unknown)

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
const ACTIVITY_CHART = { width: 700, height: 220, baseline: 205, plotHeight: 168 }
const activityY = (count: number) => ACTIVITY_CHART.baseline - 7 - count / maxActivity.value * ACTIVITY_CHART.plotHeight

const maxActivity = computed(() => Math.max(1, ...activityBuckets.value))
const linePoints = computed(() => activityBuckets.value.map((count, index) => {
  const x = index / 23 * ACTIVITY_CHART.width
  const y = activityY(count)
  return `${x.toFixed(1)},${y.toFixed(1)}`
}).join(' '))
const areaPoints = computed(() => `0,${ACTIVITY_CHART.baseline} ${linePoints.value} ${ACTIVITY_CHART.width},${ACTIVITY_CHART.baseline}`)

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
const peakLatencyBucket = computed(() => {
  const top = [...latencyBuckets.value].sort((a, b) => b.count - a.count)[0]
  return top?.count ? top : null
})

const dependencyDefinitions: Array<{ type: EntryType; label: string; zone: string; x: number; y: number }> = [
  { type: 'query', label: 'Postgres', zone: 'Storage', x: 46, y: 36 },
  { type: 'redis', label: 'Redis', zone: 'Storage', x: 46, y: 146 },
  { type: 'topic', label: 'Redpanda', zone: 'Messaging', x: 482, y: 36 },
  { type: 'http-client', label: 'External', zone: 'Outbound', x: 482, y: 146 },
  { type: 'job', label: 'Workers', zone: 'Async workers', x: 264, y: 188 },
]
const dependencies = computed(() => dependencyDefinitions.map(item => ({
  ...item,
  count: props.entries.filter(entry => entry.type === item.type).length,
  color: signalFor(item.type).color,
})))
const maxDependencyCount = computed(() => Math.max(1, ...dependencies.value.map(node => node.count)))
const topologyStats = computed(() => {
  const active = dependencies.value.filter(node => node.count > 0)
  const totalOps = dependencies.value.reduce((sum, node) => sum + node.count, 0)
  const top = [...dependencies.value].sort((a, b) => b.count - a.count)[0]
  return {
    activeLinks: active.length,
    totalOps,
    top: top?.count ? top : null,
  }
})
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
        <svg :viewBox="`0 0 ${ACTIVITY_CHART.width} ${ACTIVITY_CHART.height}`" preserveAspectRatio="none" role="img" aria-label="Activity over time">
          <defs><linearGradient id="activity-fill" x1="0" y1="0" x2="0" y2="1"><stop offset="0" stop-color="#20d9ee" stop-opacity=".28"/><stop offset="1" stop-color="#20d9ee" stop-opacity="0"/></linearGradient></defs>
          <path class="chart-grid" d="M0 30H700M0 72H700M0 114H700M0 156H700M0 198H700"/>
          <polygon :points="areaPoints" fill="url(#activity-fill)" />
          <polyline :points="linePoints" />
          <circle
            v-for="(count, index) in activityBuckets"
            :key="index"
            :cx="index / 23 * ACTIVITY_CHART.width"
            :cy="activityY(count)"
            r="4"
            tabindex="0"
            @mousemove="showChartTooltip($event, 'activity', activityPosition(index), `Interval ${index + 1}`, `${count} records`, `${Math.round(count / maxActivity * 100)}% of peak activity`)"
          />
        </svg>
        <footer><span>earliest</span><strong>{{ entries.length }} total signals</strong><span>now</span></footer>
      </article>

      <article class="topology-visual" @mouseleave="hideTooltip">
        <header>
          <span>Service interactions</span>
          <small>{{ topologyStats.activeLinks }} active links · edge strength = activity</small>
        </header>
        <svg viewBox="0 0 600 248" role="img" aria-label="Service dependency topology">
          <defs>
            <radialGradient id="topology-vignette" cx="50%" cy="48%" r="58%">
              <stop offset="0%" stop-color="#142027" stop-opacity=".55" />
              <stop offset="100%" stop-color="#080d10" stop-opacity="0" />
            </radialGradient>
          </defs>

          <rect class="topology-backdrop" x="0" y="0" width="600" height="248" fill="url(#topology-vignette)" />
          <circle class="topology-ring" :cx="SERVICE_CENTER.x" :cy="SERVICE_CENTER.y" r="74" />
          <circle class="topology-ring topology-ring--outer" :cx="SERVICE_CENTER.x" :cy="SERVICE_CENTER.y" r="102" />

          <g v-for="node in dependencies" :key="`edge-${node.type}`" class="topology-edge-group">
            <line
              class="topology-edge"
              :class="{ active: node.count }"
              v-bind="topologyEdge(node)"
              :style="{ '--edge': node.color, '--weight': Math.min(4.5, 1.2 + node.count / maxDependencyCount * 3.3) }"
            />
            <circle
              v-if="node.count"
              class="topology-edge-dot"
              :cx="topologyEdge(node).x2"
              :cy="topologyEdge(node).y2"
              r="2.5"
              :style="{ fill: node.color }"
            />
          </g>

          <g
            class="service-core"
            :class="{ active: topologyStats.totalOps > 0 }"
            :style="{ '--runtime-color': serviceLanguageColor }"
          >
            <circle class="service-core__orbit" :cx="SERVICE_CENTER.x" :cy="SERVICE_CENTER.y" r="58" />
            <circle class="service-core__pulse" :cx="SERVICE_CENTER.x" :cy="SERVICE_CENTER.y" r="48" />
            <circle class="service-core__halo" :cx="SERVICE_CENTER.x" :cy="SERVICE_CENTER.y" r="44" />
            <circle class="service-core__body" :cx="SERVICE_CENTER.x" :cy="SERVICE_CENTER.y" r="32" />
            <text class="service-core__code" :x="SERVICE_CENTER.x" :y="SERVICE_CENTER.y - 2">{{ serviceLanguageCode }}</text>
            <text class="service-core__label" :x="SERVICE_CENTER.x" :y="SERVICE_CENTER.y + 12">SERVICE</text>
          </g>

          <g
            v-for="node in dependencies"
            :key="`node-${node.type}`"
            class="dependency-node"
            :class="{
              active: node.count,
              'is-hot': node.count > 0 && node.count === topologyStats.top?.count,
            }"
            :transform="`translate(${node.x} ${node.y})`"
            :style="{ '--node': node.color, '--heat': node.count / maxDependencyCount }"
            tabindex="0"
            @mousemove="showChartTooltip($event, 'topology', topologyPosition(node), node.label, `${node.count} operations`, `${signalFor(node.type).label} · ${node.zone}`)"
          >
            <rect class="dependency-node__glow" :width="NODE_CARD.width" :height="NODE_CARD.height" rx="8" />
            <rect class="dependency-node__card" :width="NODE_CARD.width" :height="NODE_CARD.height" rx="8" />
            <rect class="dependency-node__stripe" width="3" :height="NODE_CARD.height" rx="1.5" />
            <text class="dependency-node__label" :x="NODE_CARD.width / 2" :y="NODE_CARD.height / 2 + 3">{{ node.label }}</text>
            <rect class="dependency-node__meter" x="8" :y="NODE_CARD.height - 6" width="52" height="2" rx="1" />
            <rect
              class="dependency-node__meter-fill"
              x="8"
              :y="NODE_CARD.height - 6"
              :width="52 * (node.count / maxDependencyCount)"
              height="2"
              rx="1"
            />
            <line
              class="dependency-node__zone-mark"
              :x1="NODE_CARD.width / 2"
              :y1="NODE_CARD.height + 2"
              :x2="NODE_CARD.width / 2"
              :y2="NODE_CARD.height + 6"
            />
            <text class="dependency-node__zone" :x="NODE_CARD.width / 2" :y="NODE_CARD.height + 15">{{ node.zone }}</text>
          </g>
        </svg>
        <footer class="topology-readout">
          <span>
            <small>Active links</small>
            <strong>{{ topologyStats.activeLinks }}<i>/{{ dependencies.length }}</i></strong>
          </span>
          <span>
            <small>Dependency ops</small>
            <strong>{{ topologyStats.totalOps }}</strong>
          </span>
          <span>
            <small>Busiest link</small>
            <strong :style="{ color: topologyStats.top?.color }">
              {{ topologyStats.top?.label || '—' }}
              <i v-if="topologyStats.top">{{ topologyStats.top.count }}</i>
            </strong>
          </span>
        </footer>
      </article>

      <article class="latency-visual" @mouseleave="hideTooltip">
        <header><span>Latency distribution</span><small>milliseconds</small></header>
        <div class="histogram">
          <span v-for="(bucket, index) in latencyBuckets" :key="bucket.label" @mousemove="showChartTooltip($event, 'latency', latencyPosition(index, bucket.label), `${bucket.label} ms`, `${bucket.count} operations`, `${Math.round(bucket.count / Math.max(1, timed.length) * 100)}% of timed records`)">
            <b>{{ bucket.count }}</b>
            <i><em :style="{ height: `${Math.max(bucket.count ? 8 : 1, bucket.count / maxLatencyBucket * 100)}%` }" /></i>
            <small>{{ bucket.label }}</small>
          </span>
        </div>
        <footer class="latency-readout">
          <span>
            <small>Timed records</small>
            <strong>{{ timed.length }}</strong>
          </span>
          <span>
            <small>Average</small>
            <strong>{{ average || '—' }}<i v-if="average">ms</i></strong>
          </span>
          <span>
            <small>Peak bucket</small>
            <strong>{{ peakLatencyBucket?.label || '—' }}<i v-if="peakLatencyBucket">ms</i></strong>
          </span>
        </footer>
      </article>

      <article class="evidence-visual" @mouseleave="hideTooltip">
        <header><span>Evidence matrix</span><small>latest {{ Math.min(entries.length, 56) }}</small></header>
        <div class="evidence-matrix">
          <button
            v-for="(entry, index) in entries.slice(0, 56)"
            :key="entry.id"
            :class="{ error: isError(entry) }"
            :style="{ '--cell': entrySignalColor(entry), '--intensity': Math.min(1, .35 + entryDuration(entry) / 800) }"
            :aria-label="`Open ${signalFor(entry.type).label} entry`"
            @mousemove="showChartTooltip($event, 'evidence', evidencePosition(index), signalFor(entry.type).label, entryDuration(entry) ? `${entryDuration(entry)}ms` : 'Untimed', String(entry.content?.path || entry.content?.name || entry.id.slice(0, 12)))"
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
    <Teleport to="body">
      <div
        v-if="tooltip"
        class="chart-tooltip"
        :class="[`chart-tooltip--${tooltip.chart}`, `chart-tooltip--${tooltip.placement}`]"
        :style="{ left: `${tooltip.x}px`, top: `${tooltip.y}px` }"
      >
        <div class="chart-tooltip__head">
          <small>{{ tooltip.chartLabel }}</small>
          <i>{{ tooltip.position }}</i>
        </div>
        <strong>{{ tooltip.value }}</strong>
        <span class="chart-tooltip__title">{{ tooltip.title }}</span>
        <span class="chart-tooltip__detail">{{ tooltip.detail }}</span>
      </div>
    </Teleport>
  </section>
</template>
