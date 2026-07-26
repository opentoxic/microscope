<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import AppShell from '../components/AppShell.vue'
import LLMInsightPanel from '../components/LLMInsightPanel.vue'
import SignalIcon from '../components/SignalIcon.vue'
import { listEntries } from '../api/client'
import { loadSignalSettings, setSignalEnabled, signalSettings } from '../settings'
import {
  activeProvider,
  llmConfigured,
  llmPeriods,
  llmProviders,
  llmSettings,
  loadProviderModels,
  providerModels,
  scheduleProviderModels,
  setLLMProvider,
  toggleLLMDataType,
} from '../llmSettings'
import { signals, type SignalDefinition } from '../utils'
import type { Entry, EntryType } from '../types'

const activeTab = ref<'recording' | 'llm'>('recording')
const pending = ref<Record<string, boolean>>({})
const notice = ref<{ tone: 'success' | 'error'; text: string } | null>(null)
const analysisEntries = ref<Entry[]>([])
const analysisLoading = ref(false)

const groups = computed(() => ([
  { id: 'flow', title: 'Request flow', description: 'Network and database activity that explains how work moves through the service.' },
  { id: 'runtime', title: 'Runtime systems', description: 'Infrastructure, brokers, workers, and internal application dispatch.' },
  { id: 'output', title: 'Evidence & health', description: 'Diagnostics, errors, delivery records, performance, and custom evidence.' },
].map(group => ({ ...group, signals: signals.filter(signal => signal.type && signal.group === group.id) }))))

const dataTypeSignals = computed(() => signals.filter(signal => signal.type))

function settingFor(signal: SignalDefinition) {
  return signalSettings.values[signal.type]
}

async function toggle(signal: SignalDefinition) {
  const setting = settingFor(signal)
  const next = !(setting?.enabled !== false)
  if (!next) {
    const count = setting?.count || 0
    if (!confirm(`Disable ${signal.label}? This immediately deletes ${count.toLocaleString()} retained records and stops future recording.`)) return
  }
  pending.value[signal.type] = true
  notice.value = null
  try {
    const result = await setSignalEnabled(signal.type as EntryType, next)
    notice.value = next
      ? { tone: 'success', text: `${signal.label} is live. New records will appear immediately.` }
      : { tone: 'success', text: `${signal.label} disabled and ${result.deleted.toLocaleString()} records permanently removed.` }
  } catch (error) {
    notice.value = { tone: 'error', text: error instanceof Error ? error.message : 'Setting could not be saved' }
  } finally {
    pending.value[signal.type] = false
  }
}

watch(() => llmSettings.apiKey, () => scheduleProviderModels())
watch(() => activeTab.value, (tab) => {
  if (tab === 'llm') {
    if (llmSettings.apiKey.trim()) scheduleProviderModels()
    loadAnalysisEntries()
  }
})

async function loadAnalysisEntries() {
  analysisLoading.value = true
  try {
    const data = await listEntries(new URLSearchParams({ limit: '80' }))
    analysisEntries.value = data.entries || []
  } catch {
    analysisEntries.value = []
  } finally {
    analysisLoading.value = false
  }
}

onMounted(() => {
  loadSignalSettings(true)
  if (llmSettings.apiKey.trim()) scheduleProviderModels()
  if (activeTab.value === 'llm') loadAnalysisEntries()
})
</script>

<template>
  <AppShell>
    <template #status>{{ signalSettings.loading ? 'Synchronizing policy…' : activeTab === 'llm' ? 'LLM manual mode' : 'Policy enforced at ingestion' }}</template>

    <section class="settings-intro">
      <div>
        <span>Recorder control plane</span>
        <h2>Configure what Signal records and how it interprets it.</h2>
        <p>Manage signal retention policies and connect an LLM provider for manual, on-demand runtime intelligence.</p>
      </div>
      <div class="settings-enforcement">
        <i /><span><strong>Backend enforced</strong><small>UI visibility and database ingestion share one policy</small></span>
      </div>
    </section>

    <div class="settings-tabs">
      <button class="settings-tab" :class="{ 'is-active': activeTab === 'recording' }" @click="activeTab = 'recording'">Signal recording</button>
      <button class="settings-tab" :class="{ 'is-active': activeTab === 'llm' }" @click="activeTab = 'llm'">LLM provider</button>
    </div>

    <Transition name="notice">
      <div v-if="notice" class="action-notice settings-notice" :class="`is-${notice.tone}`">
        <span>{{ notice.text }}</span><button @click="notice = null">×</button>
      </div>
    </Transition>

    <template v-if="activeTab === 'recording'">
      <div v-if="signalSettings.loading && !signalSettings.loaded" class="settings-loading">
        <div v-for="index in 9" :key="index"><i /><span /><b /></div>
      </div>
      <div v-else class="settings-groups">
        <section v-for="group in groups" :key="group.id" class="settings-group">
          <header><span>{{ group.title }}</span><p>{{ group.description }}</p></header>
          <div class="settings-signals">
            <button
              v-for="signal in group.signals"
              :key="signal.type"
              class="setting-row"
              :class="{ 'is-disabled': settingFor(signal)?.enabled === false, 'is-pending': pending[signal.type] }"
              :style="{ '--signal': signal.color }"
              :disabled="pending[signal.type]"
              @click="toggle(signal)"
            >
              <SignalIcon :type="signal.type" size="md" />
              <span class="setting-copy">
                <strong>{{ signal.label }}</strong>
                <small v-if="signal.type === 'metric'">Runtime metrics from Go, Python, Node.js, Ruby, PHP, or Elixir services, plus custom metrics</small>
                <small v-else>{{ settingFor(signal)?.count || 0 }} retained records</small>
              </span>
              <span class="setting-count">{{ (settingFor(signal)?.count || 0).toLocaleString() }}</span>
              <span v-if="pending[signal.type]" class="action-spinner" />
              <span v-else class="setting-switch" :class="{ on: settingFor(signal)?.enabled !== false }"><i /></span>
            </button>
          </div>
        </section>
      </div>
    </template>

    <section v-else class="llm-settings">
      <header>
        <div>
          <span>LLM intelligence</span>
          <p>Connect a model provider to generate deep insights, health scoring, and investigation guidance. Analysis runs only when you trigger it manually.</p>
        </div>
        <div class="llm-status" :class="{ 'is-ready': llmConfigured }">
          <i />
          <div>
            <strong>{{ llmConfigured ? 'Ready for manual analysis' : 'Not configured' }}</strong>
            <small>{{ llmConfigured ? `${activeProvider.label} · ${llmSettings.model}` : 'Add an API key to enable' }}</small>
          </div>
        </div>
      </header>

      <div class="llm-grid">
        <div class="llm-field llm-field--full">
          <label>Provider</label>
          <div class="provider-cards">
            <button
              v-for="provider in llmProviders"
              :key="provider.id"
              class="provider-card"
              :class="{ 'is-active': llmSettings.provider === provider.id }"
              @click="setLLMProvider(provider.id)"
            >
              <strong>{{ provider.label }}</strong>
              <small>{{ provider.description }}</small>
            </button>
          </div>
        </div>

        <div class="llm-field">
          <label>Model</label>
          <div class="llm-model-row">
            <select v-model="llmSettings.model" :disabled="providerModels.loading || !providerModels.items.length">
              <option v-if="providerModels.loading" value="">Loading live models…</option>
              <option v-else-if="!llmSettings.apiKey.trim()" value="">Enter an API key first</option>
              <option v-else-if="!providerModels.items.length" value="">No models available</option>
              <option v-for="model in providerModels.items" :key="model.id" :value="model.id">
                {{ model.label }}{{ model.id !== model.label ? ` · ${model.id}` : '' }}
              </option>
            </select>
            <button
              class="llm-refresh-btn"
              type="button"
              :disabled="!llmSettings.apiKey.trim() || providerModels.loading"
              title="Refresh models from provider"
              @click="loadProviderModels(true)"
            >
              <span v-if="providerModels.loading" class="action-spinner" />
              <span v-else>Refresh</span>
            </button>
          </div>
          <small v-if="providerModels.error" class="llm-field-error">{{ providerModels.error }}</small>
          <small v-else-if="providerModels.loaded">
            {{ providerModels.items.length }} live models from {{ activeProvider.label }}
          </small>
          <small v-else>Models are fetched from the provider's list endpoint using your API key.</small>
        </div>

        <div class="llm-field">
          <label>API key</label>
          <input v-model="llmSettings.apiKey" type="password" placeholder="sk-… or provider key" autocomplete="off" />
          <small>Stored locally in this browser. Sent only when you run analysis.</small>
        </div>

        <div class="llm-field">
          <label>Analysis mode</label>
          <div class="llm-mode-badge"><i /> Manual — run on demand</div>
          <small>Automatic and scheduled modes will be available later.</small>
        </div>

        <div class="llm-field">
          <label>Time window</label>
          <div class="llm-periods">
            <button
              v-for="period in llmPeriods"
              :key="period.id"
              class="llm-period"
              :class="{ 'is-active': llmSettings.period === period.id }"
              @click="llmSettings.period = period.id"
            >
              {{ period.label }}
            </button>
          </div>
          <small>How far back to include signals when you run analysis.</small>
        </div>

        <div class="llm-field llm-field--full">
          <label>Data types to send</label>
          <div class="llm-data-types">
            <button
              v-for="signal in dataTypeSignals"
              :key="signal.type"
              class="llm-data-chip"
              :class="{ 'is-on': llmSettings.dataTypes.includes(signal.type as EntryType) }"
              :style="{ '--chip-signal': signal.color }"
              @click="toggleLLMDataType(signal.type as EntryType)"
            >
              <SignalIcon :type="signal.type" size="sm" />
              <span>{{ signal.shortLabel }}</span>
            </button>
          </div>
          <small>Select which signal types are included in LLM context payloads.</small>
        </div>

        <div class="llm-field">
          <label>Enable LLM insights</label>
          <button
            class="setting-row"
            style="min-height: 52px; grid-template-columns: minmax(0, 1fr) 42px; padding: 0 14px; border: 1px solid #2a383d;"
            @click="llmSettings.enabled = !llmSettings.enabled"
          >
            <span class="setting-copy">
              <strong>{{ llmSettings.enabled ? 'Insights enabled' : 'Insights disabled' }}</strong>
              <small>Show LLM panels in explorer and trace views</small>
            </span>
            <span class="setting-switch" :class="{ on: llmSettings.enabled }"><i /></span>
          </button>
        </div>
      </div>

      <LLMInsightPanel
        v-if="llmSettings.apiKey.trim() && llmSettings.model"
        :entries="analysisEntries"
        context="Settings configuration test"
        title="Test your LLM configuration"
        setup-mode
      />
      <div v-else-if="analysisLoading" class="llm-insight-empty">Loading recent signals for analysis…</div>
    </section>
  </AppShell>
</template>
