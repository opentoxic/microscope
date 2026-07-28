<script setup lang="ts">
import { computed, nextTick, ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import { createCustomEntry } from '../api/client'
import { enabledSignals, loadSignalSettings, signalEnabled } from '../settings'
import SignalIcon from './SignalIcon.vue'

const props = defineProps<{ open: boolean }>()
const emit = defineEmits<{ close: [] }>()
const router = useRouter()
const query = ref('')
const active = ref(0)
const input = ref<HTMLInputElement | null>(null)
const runningId = ref('')
const actionError = ref('')

const commands = computed(() => {
  const needle = query.value.trim().toLowerCase()
  const items = enabledSignals.value.map(signal => ({
    id: `signal-${signal.type || 'all'}`,
    label: signal.label,
    detail: signal.available ? 'Open recorder' : 'Awaiting recorder',
    key: `G ${signal.key}`,
    color: signal.color,
    disabled: false,
    run: () => router.push(signal.type ? { path: '/', query: { type: signal.type } } : '/'),
  }))
  const actions = [
    { id: 'search', label: 'Search recorded activity', detail: 'Find paths, IDs, and messages', key: '/', color: '#a9b0b2', disabled: !needle, run: () => router.push({ path: '/', query: { search: query.value } }) },
    { id: 'bookmarks', label: 'Open bookmarked traces', detail: 'Return to pinned investigations', key: 'G B', color: '#e9ad58', disabled: false, run: () => router.push({ path: '/', query: { bookmarked: '1' } }) },
    { id: 'settings', label: 'Configure Microscope recording', detail: 'Enable, purge, or restore recorders', key: 'G ,', color: '#20d9ee', disabled: false, run: () => router.push('/settings') },
    { id: 'marker', label: 'Add timeline marker', detail: 'Record a custom event at this moment', key: '+', color: '#55cfe1', disabled: false, run: addMarker },
    { id: 'portal', label: 'Return to activity overview', detail: 'Open the live observability workspace', key: '↗', color: '#a9b0b2', disabled: false, run: () => router.push('/') },
  ].filter(action => action.id !== 'marker' || signalEnabled('custom'))
  const all = [...items, ...actions]
  if (!needle) return all
  return all
    .map(item => ({ item, score: item.id === 'search' ? 1 : fuzzyScore(`${item.label} ${item.detail}`.toLowerCase(), needle) }))
    .filter(result => result.score >= 0)
    .sort((a, b) => b.score - a.score)
    .map(result => result.item)
})

function fuzzyScore(haystack: string, needle: string): number {
  let cursor = 0
  let score = 0
  let streak = 0
  for (const character of needle) {
    const match = haystack.indexOf(character, cursor)
    if (match < 0) return -1
    streak = match === cursor ? streak + 1 : 0
    score += 2 + streak * 3 - Math.min(2, match - cursor)
    cursor = match + 1
  }
  if (haystack.startsWith(needle)) score += 24
  return score
}

async function addMarker() {
  const name = prompt('Marker name')
  if (!name?.trim()) return
  await createCustomEntry(name.trim(), { source: 'dashboard' })
  await router.push({ path: '/', query: { type: 'custom' } })
}

watch(() => props.open, async (value) => {
  if (!value) return
  query.value = ''
  actionError.value = ''
  active.value = 0
  await nextTick()
  input.value?.focus()
})

watch(query, () => { active.value = 0 })

function move(delta: number) {
  const count = commands.value.length
  if (!count) return
  active.value = (active.value + delta + count) % count
}

async function choose(index = active.value) {
  const command = commands.value[index]
  if (!command || command.disabled || runningId.value) return
  runningId.value = command.id
  actionError.value = ''
  try {
    await command.run()
    emit('close')
  } catch (error) {
    actionError.value = error instanceof Error ? error.message : 'Action failed'
  } finally {
    runningId.value = ''
  }
}

loadSignalSettings()
</script>

<template>
  <Teleport to="body">
    <Transition name="palette">
      <div v-if="open" class="command-veil" @mousedown.self="emit('close')">
        <div class="command-palette" role="dialog" aria-modal="true" aria-label="Command palette">
          <div class="command-input">
            <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
            <input
              ref="input"
              v-model="query"
              placeholder="Go to a recorder or search activity…"
              @keydown.down.prevent="move(1)"
              @keydown.up.prevent="move(-1)"
              @keydown.enter.prevent="choose()"
              @keydown.esc="emit('close')"
            />
            <kbd>esc</kbd>
          </div>
          <div class="command-list">
            <div class="command-heading">{{ query ? 'Best matches' : 'Recorders & actions' }}</div>
            <button
              v-for="(command, index) in commands"
              :key="command.id"
              class="command-item"
              :class="{ 'is-active': active === index, 'is-disabled': command.disabled || !!runningId }"
              @mouseenter="active = index"
              @click="choose(index)"
            >
              <SignalIcon v-if="command.id.startsWith('signal-')" :type="command.id.replace('signal-', '') as any" size="sm" class="command-signal-icon" :style="{ '--signal': command.color }" />
              <span v-else class="command-signal" :style="{ '--signal': command.color }" />
              <span class="command-copy">
                <strong>{{ command.label }}</strong>
                <small>{{ command.detail }}</small>
              </span>
              <span v-if="runningId === command.id" class="action-spinner" />
              <kbd v-else>{{ command.key }}</kbd>
            </button>
            <div v-if="actionError" class="command-error">{{ actionError }}</div>
            <div v-if="!commands.length" class="command-empty">No command matches “{{ query }}”</div>
          </div>
          <div class="command-footer">
            <span><kbd>↑</kbd><kbd>↓</kbd> navigate</span>
            <span><kbd>↵</kbd> open</span>
            <span class="command-brand">Opentoxic Microscope</span>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
