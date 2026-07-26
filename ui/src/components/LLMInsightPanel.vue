<script setup lang="ts">
import { computed, ref } from 'vue'
import { analyzeInsights } from '../api/client'
import type { Entry } from '../types'
import { llmConfigured, llmSettings, periodMinutes } from '../llmSettings'
import { signalFor } from '../utils'
import SignalIcon from './SignalIcon.vue'

const props = defineProps<{
  entries: Entry[]
  context?: string
  title?: string
  compact?: boolean
  setupMode?: boolean
}>()

const loading = ref(false)
const error = ref('')
const result = ref<Awaited<ReturnType<typeof analyzeInsights>> | null>(null)

const canRun = computed(() => {
  const hasCredentials = llmSettings.apiKey.trim().length > 0 && llmSettings.model.trim().length > 0
  if (props.setupMode) return hasCredentials
  return llmConfigured.value
})

const filteredEntries = computed(() => {
  const cutoff = Date.now() - periodMinutes(llmSettings.period) * 60_000
  return props.entries.filter(entry => {
    if (!llmSettings.dataTypes.includes(entry.type)) return false
    return +new Date(entry.created_at) >= cutoff
  })
})

const chartBars = computed(() => {
  const dist = result.value?.signal_distribution || []
  const max = Math.max(1, ...dist.map(item => item.count))
  return dist.slice(0, 12).map(item => ({
    ...item,
    height: Math.max(8, item.count / max * 100),
    color: signalFor(item.type).color,
  }))
})

const scoreOffset = computed(() => {
  const score = result.value?.health_score || 0
  const circumference = 2 * Math.PI * 42
  return circumference - (score / 100) * circumference
})

async function runAnalysis() {
  if (!canRun.value || loading.value) return
  if (!filteredEntries.value.length) {
    error.value = 'No signals match your selected data types and time window.'
    return
  }
  loading.value = true
  error.value = ''
  try {
    result.value = await analyzeInsights({
      provider: llmSettings.provider,
      model: llmSettings.model,
      api_key: llmSettings.apiKey,
      period: llmSettings.period,
      context: props.context || props.title || 'Signal runtime analysis',
      entries: filteredEntries.value.map(entry => ({
        id: entry.id,
        type: entry.type,
        created_at: entry.created_at,
        content: entry.content,
      })),
    })
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Analysis failed'
    result.value = null
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <section class="llm-insight">
    <header>
      <div>
        <span>LLM intelligence</span>
        <strong>{{ title || 'Manual runtime analysis' }}</strong>
        <p v-if="canRun">
          {{ filteredEntries.length }} signals · {{ llmSettings.period }} window · {{ llmSettings.provider }} / {{ llmSettings.model }}
        </p>
        <p v-else>Configure a provider, API key, and model to run analysis.</p>
      </div>
      <button class="llm-run-btn" :disabled="!canRun || loading" @click="runAnalysis">
        <span v-if="loading" class="action-spinner" />
        <span>{{ loading ? 'Analyzing…' : 'Run analysis' }}</span>
      </button>
    </header>

    <div v-if="error" class="action-notice is-error" style="margin: 0; border-radius: 0; border-left: 0; border-right: 0;">
      <span>{{ error }}</span>
      <button @click="error = ''">×</button>
    </div>

    <div v-if="loading" class="llm-loading">
      <span class="action-spinner" />
      <span>Correlating {{ filteredEntries.length }} signals with {{ llmSettings.provider }}…</span>
    </div>

    <div v-else-if="!result" class="llm-insight-empty">
      <template v-if="canRun">
        Manual mode is active. Select your data types and period{{ setupMode ? ' above' : ' in settings' }}, then run analysis when you want deeper interpretation.
      </template>
      <template v-else>
        Open <RouterLink to="/settings">Signal settings</RouterLink> and connect an LLM provider to enable insights.
      </template>
    </div>

    <div v-else class="llm-insight-body">
      <div class="llm-insight-grid" :class="{ 'llm-insight-grid--compact': compact }">
        <div class="llm-insight-main">
          <article class="llm-summary">
            <p>{{ result.summary }}</p>
          </article>

          <div class="llm-findings">
            <article
              v-for="(finding, index) in result.findings"
              :key="finding.title"
              class="llm-finding"
              :class="`is-${finding.severity}`"
            >
              <i>{{ String(index + 1).padStart(2, '0') }}</i>
              <div>
                <strong>{{ finding.title }}</strong>
                <small>{{ finding.detail }}</small>
              </div>
            </article>
          </div>

          <div v-if="chartBars.length" class="llm-chart-strip" aria-label="Signal distribution">
            <i
              v-for="bar in chartBars"
              :key="bar.type"
              :style="{ height: `${bar.height}%`, background: bar.color }"
              :title="`${signalFor(bar.type).label}: ${bar.count}`"
            />
          </div>
        </div>

        <aside class="llm-side-panel">
          <div class="llm-score-ring">
            <svg viewBox="0 0 100 100" aria-hidden="true">
              <circle class="track" cx="50" cy="50" r="42" />
              <circle
                class="fill"
                cx="50"
                cy="50"
                r="42"
                :stroke-dasharray="`${2 * Math.PI * 42}`"
                :stroke-dashoffset="scoreOffset"
              />
            </svg>
            <div class="llm-score-value">
              <strong>{{ result.health_score }}</strong>
              <small>Health</small>
            </div>
          </div>

          <div class="llm-metrics">
            <div v-for="(value, key) in result.metrics" :key="key" class="llm-metric">
              <small>{{ String(key).replace(/_/g, ' ') }}</small>
              <strong>{{ value }}</strong>
            </div>
          </div>

          <div class="llm-actions-list">
            <div v-for="(action, index) in result.recommendations" :key="action" class="llm-action">
              <i>{{ index + 1 }}</i>
              <span>{{ action }}</span>
            </div>
          </div>

          <div v-if="result.signal_distribution?.length" class="llm-findings">
            <article
              v-for="item in result.signal_distribution.slice(0, 6)"
              :key="item.type"
              class="llm-finding"
            >
              <SignalIcon :type="item.type as any" size="sm" :style="{ '--signal': signalFor(item.type).color }" />
              <div>
                <strong>{{ signalFor(item.type).label }}</strong>
                <small>{{ item.count }} signals · {{ Math.round(item.pct) }}%</small>
              </div>
            </article>
          </div>
        </aside>
      </div>
    </div>
  </section>
</template>
