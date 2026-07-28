<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { BatchTypeGroup } from '../types'
import { entryDuration, entrySignalColor, formatRecordDate, formatRecordTime, isError, signalFor, summarize } from '../utils'
import SignalIcon from './SignalIcon.vue'

const props = defineProps<{
  groups: BatchTypeGroup[]
  activeType: string
  currentId?: string
}>()
const router = useRouter()
const activeTab = ref(props.activeType || props.groups[0]?.type || '')

watch(() => props.activeType, (value) => {
  if (value) activeTab.value = value
})

const activeGroup = computed(() => props.groups.find(group => group.type === activeTab.value) || props.groups[0])
const activeSignal = computed(() => signalFor(activeGroup.value?.type || ''))

const ordered = computed(() => {
  if (!activeGroup.value) return []
  return [...activeGroup.value.entries].sort((a, b) => +new Date(a.created_at) - +new Date(b.created_at))
})

const maxDuration = computed(() => Math.max(1, ...ordered.value.map(entryDuration)))
const slowest = computed(() => [...ordered.value].sort((a, b) => entryDuration(b) - entryDuration(a))[0])
const groupCost = computed(() => ordered.value.reduce((sum, entry) => sum + entryDuration(entry), 0))
const totalEntries = computed(() => props.groups.reduce((sum, group) => sum + group.entries.length, 0))

function meterWidth(entry: typeof ordered.value[number]): string {
  const duration = entryDuration(entry)
  if (!duration) return '0%'
  return `${Math.max(8, duration / maxDuration.value * 100)}%`
}
</script>

<template>
  <section v-if="groups.length" class="relations-panel">
    <header class="relations-heading">
      <div class="relations-heading__lead">
        <span class="panel-label">
          <i class="panel-label__mark" aria-hidden="true" />
          Related signals
        </span>
        <small>same execution batch</small>
      </div>
      <div class="relation-tabs" role="tablist" aria-label="Filter related signals by type">
        <button
          v-for="group in groups"
          :key="group.type"
          role="tab"
          :aria-selected="activeTab === group.type"
          :class="{ active: activeTab === group.type }"
          :style="{ '--signal': signalFor(group.type).color }"
          @click="activeTab = group.type"
        >
          <i aria-hidden="true" />
          {{ group.label }}
          <span>{{ group.entries.length }}</span>
        </button>
      </div>
    </header>

    <div
      v-if="activeGroup"
      class="relations-body"
      :style="{ '--group-signal': activeSignal.color }"
    >
      <div class="relations-toolbar">
        <div class="relations-toolbar__lead">
          <span class="panel-label">
            <i class="panel-label__mark" aria-hidden="true" />
            {{ activeGroup.label }}
          </span>
          <small>{{ ordered.length }} linked {{ activeGroup.label.toLowerCase() }}</small>
        </div>
        <p class="relations-toolbar__hint">
          Chronological records from the same execution context. Select a row to open that trace.
        </p>
        <div class="relations-toolbar__stats">
          <span class="relations-stat">
            <small>Count</small>
            <strong>{{ ordered.length }}</strong>
          </span>
          <span v-if="groupCost" class="relations-stat">
            <small>Combined</small>
            <strong>{{ groupCost }}<i>ms</i></strong>
          </span>
          <span v-if="slowest && entryDuration(slowest)" class="relations-stat">
            <small>Longest</small>
            <strong>{{ entryDuration(slowest) }}<i>ms</i></strong>
          </span>
        </div>
      </div>

      <div class="relations-rows" role="list">
        <button
          v-for="(entry, index) in ordered"
          :key="entry.id"
          class="relation-row"
          role="listitem"
          :class="{ active: entry.id === currentId, error: isError(entry) }"
          :style="{
            '--signal': entrySignalColor(entry),
            '--row-delay': `${index * 30}ms`,
          }"
          :aria-label="`Inspect ${signalFor(entry.type).label}: ${summarize(entry)}`"
          @click="router.push(`/entries/${entry.id}`)"
        >
          <span class="relation-row__accent" aria-hidden="true" />
          <span class="relation-row__index">{{ String(index + 1).padStart(2, '0') }}</span>
          <span
            class="relation-row__time"
            :title="`${formatRecordDate(entry.created_at)} ${formatRecordTime(entry.created_at)}`"
          >
            <strong>{{ formatRecordTime(entry.created_at) }}</strong>
            <small>{{ formatRecordDate(entry.created_at) }}</small>
          </span>
          <span class="relation-row__icon"><SignalIcon :type="entry.type" size="sm" /></span>
          <span class="relation-row__type">{{ signalFor(entry.type).shortLabel }}</span>
          <span class="relation-row__summary">{{ summarize(entry) }}</span>
          <span class="relation-row__duration">
            <span class="relation-row__meter" aria-hidden="true">
              <i :style="{ width: meterWidth(entry) }" />
            </span>
            <strong>{{ entryDuration(entry) ? `${entryDuration(entry)}ms` : '—' }}</strong>
          </span>
          <svg class="relation-row__chevron" viewBox="0 0 20 20" aria-hidden="true">
            <path d="m8 5 5 5-5 5" />
          </svg>
        </button>
      </div>

      <footer class="relations-footer">
        <span class="relations-footer__context"><i aria-hidden="true" /> shared context</span>
        <div class="relations-footer__metrics">
          <span v-if="slowest && groupCost">Longest · {{ entryDuration(slowest) || 0 }}ms</span>
          <span v-if="groupCost">Combined · {{ groupCost }}ms</span>
        </div>
        <span class="relations-footer__total">{{ ordered.length }} of {{ totalEntries }} related</span>
      </footer>
    </div>
  </section>
</template>
