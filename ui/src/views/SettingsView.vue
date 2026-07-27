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
import { accentOptions, appearance, resetAppearance, type Density } from '../appearance'

const activeTab = ref<'recording' | 'appearance' | 'integrations' | 'llm'>('recording')
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
const densities: Array<{ id: Density; label: string; detail: string }> = [
  { id: 'compact', label: 'Compact', detail: 'Maximum signal density' },
  { id: 'comfortable', label: 'Comfortable', detail: 'Balanced for daily use' },
  { id: 'spacious', label: 'Spacious', detail: 'More breathing room' },
]
const ecosystems = [
  { language: 'Go', code: 'GO', frameworks: ['net/http', 'Gin', 'Echo', 'Fiber'], status: 'Core + HTTP API' },
  { language: 'Python', code: 'PY', frameworks: ['Django', 'FastAPI', 'Flask', 'Starlette'], status: 'SDK included' },
  { language: 'TypeScript', code: 'TS', frameworks: ['Express', 'NestJS', 'Fastify', 'Next.js'], status: 'SDK included' },
  { language: 'PHP', code: 'PHP', frameworks: ['Laravel', 'Symfony', 'Slim'], status: 'SDK + Laravel' },
  { language: 'Ruby', code: 'RB', frameworks: ['Rails', 'Sinatra', 'Hanami'], status: 'SDK included' },
  { language: 'Elixir', code: 'EX', frameworks: ['Phoenix', 'Plug', 'Oban'], status: 'SDK included' },
]

function toggleAppearance(key: 'glow' | 'scanlines' | 'grid' | 'motion' | 'highContrast' | 'monospaceData') {
  appearance[key] = !appearance[key]
}

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
    <template #status>{{ signalSettings.loading ? 'Synchronizing policy…' : activeTab === 'llm' ? 'LLM manual mode' : activeTab === 'appearance' ? 'Personalized locally' : activeTab === 'integrations' ? 'Six ecosystems ready' : 'Policy enforced at ingestion' }}</template>

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
      <button class="settings-tab" :class="{ 'is-active': activeTab === 'appearance' }" @click="activeTab = 'appearance'">Appearance</button>
      <button class="settings-tab" :class="{ 'is-active': activeTab === 'integrations' }" @click="activeTab = 'integrations'">SDKs & frameworks</button>
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

    <section v-else-if="activeTab === 'appearance'" class="appearance-settings">
      <header class="settings-section-heading">
        <div><span>Visual system</span><strong>Make the console yours.</strong><p>All appearance preferences apply instantly and stay in this browser.</p></div>
        <button class="settings-reset" @click="resetAppearance">Reset defaults</button>
      </header>

      <div class="appearance-preview" :style="{ '--preview-accent': accentOptions.find(item => item.id === appearance.accent)?.color }">
        <div class="preview-chrome"><i /><i /><i /><span>MICROSCOPE / LIVE RUNTIME</span></div>
        <div class="preview-grid">
          <aside><i v-for="index in 6" :key="index" /></aside>
          <main>
            <span>OBSERVABILITY FEED</span>
            <strong>Every signal. One timeline.</strong>
            <div><i v-for="index in 18" :key="index" :style="{ height: `${12 + (index * 13) % 38}px` }" /></div>
          </main>
        </div>
      </div>

      <div class="preference-grid">
        <section class="preference-panel preference-panel--wide">
          <header><strong>Accent energy</strong><small>Used for active navigation, focus, and live-state feedback.</small></header>
          <div class="accent-picker">
            <button v-for="option in accentOptions" :key="option.id" :class="{ active: appearance.accent === option.id }" @click="appearance.accent = option.id">
              <i :style="{ background: option.color, boxShadow: `0 0 16px ${option.color}` }" /><span>{{ option.label }}</span><b />
            </button>
          </div>
        </section>

        <section class="preference-panel preference-panel--wide">
          <header><strong>Interface density</strong><small>Controls spacing without hiding information.</small></header>
          <div class="density-picker">
            <button v-for="density in densities" :key="density.id" :class="{ active: appearance.density === density.id }" @click="appearance.density = density.id">
              <span>{{ density.label }}</span><small>{{ density.detail }}</small>
            </button>
          </div>
        </section>

        <button
          v-for="option in [
            { key: 'motion', label: 'Interface motion', detail: 'Page transitions and animated state changes' },
            { key: 'glow', label: 'Neon glow', detail: 'Luminous edges and signal halos' },
            { key: 'scanlines', label: 'Scanline texture', detail: 'Subtle cyberdeck display finish' },
            { key: 'grid', label: 'Ambient grid', detail: 'Faint coordinate grid behind workspaces' },
            { key: 'highContrast', label: 'High contrast', detail: 'Brighter text and stronger boundaries' },
            { key: 'monospaceData', label: 'Technical typography', detail: 'Monospace IDs, metrics, and values' },
          ]"
          :key="option.key"
          class="preference-toggle"
          @click="toggleAppearance(option.key as 'glow' | 'scanlines' | 'grid' | 'motion' | 'highContrast' | 'monospaceData')"
        >
          <span><strong>{{ option.label }}</strong><small>{{ option.detail }}</small></span>
          <i class="setting-switch" :class="{ on: appearance[option.key as keyof typeof appearance] }"><b /></i>
        </button>
      </div>
    </section>

    <section v-else-if="activeTab === 'integrations'" class="integration-settings">
      <header class="settings-section-heading">
        <div><span>Polyglot telemetry</span><strong>One microscope for every service.</strong><p>SDKs emit the same normalized entry format to the standalone Go collector.</p></div>
        <span class="integration-ready"><i /> HTTP ingestion ready</span>
      </header>
      <div class="ecosystem-grid">
        <article v-for="ecosystem in ecosystems" :key="ecosystem.language" class="ecosystem-card">
          <header><i>{{ ecosystem.code }}</i><div><strong>{{ ecosystem.language }}</strong><small>{{ ecosystem.status }}</small></div><span>READY</span></header>
          <div>
            <span v-for="framework in ecosystem.frameworks" :key="framework">{{ framework }}</span>
          </div>
          <footer><i /><span>Runtime metrics</span><i /><span>Custom events</span><i /><span>HTTP transport</span></footer>
        </article>
      </div>
      <div class="integration-contract">
        <div><span>Universal contract</span><strong>POST /microscope/api/entries</strong><small>Any framework can integrate without a native SDK.</small></div>
        <code>{ "name": "checkout.completed", "content": { "duration_ms": 84 } }</code>
      </div>
    </section>

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
