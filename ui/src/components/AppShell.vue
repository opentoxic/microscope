<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useRoute, useRouter, RouterLink } from 'vue-router'
import CommandPalette from './CommandPalette.vue'
import ConfirmDialog from './ConfirmDialog.vue'
import SignalIcon from './SignalIcon.vue'
import ToastStack from './ToastStack.vue'
import { signalFor, signals, typeTitles } from '../utils'
import { enabledSignals, loadSignalSettings, signalEnabled } from '../settings'
import { demoMode } from '../api/client'

const route = useRoute()
const router = useRouter()
const paletteOpen = ref(false)
const chordOpen = ref(false)
let chordTimer: ReturnType<typeof setTimeout> | null = null
const compactSignals = computed(() => enabledSignals.value)
const currentType = computed(() => String(route.query.type || ''))
const currentSignal = computed(() => signalFor(currentType.value))
const title = computed(() => {
  if (route.name === 'detail') return 'Trace inspector'
  if (route.name === 'settings') return 'Settings'
  if (route.query.bookmarked === '1') return 'Bookmarks'
  return typeTitles[currentType.value] || 'All activity'
})

const sessionMode = computed(() => {
  if (demoMode) return 'demo'
  if (route.name === 'detail') return 'inspect'
  if (route.name === 'settings') return 'settings'
  if (route.query.bookmarked === '1') return 'bookmarks'
  return 'live'
})

const sessionEyebrow = computed(() => {
  if (demoMode) return 'Interactive demo'
  if (route.name === 'detail') return 'Single trace'
  if (route.name === 'settings') return 'Recording policy'
  if (route.query.bookmarked === '1') return 'Saved records'
  if (currentType.value) return `${currentSignal.value.shortLabel} stream`
  return 'Live session'
})

const sessionSignal = computed(() => {
  if (demoMode || route.name === 'settings') return 'var(--cyan)'
  if (route.query.bookmarked === '1') return 'var(--amber)'
  if (route.name === 'detail') return 'var(--purple)'
  return currentSignal.value.color
})

const sessionIconType = computed(() => {
  if (route.name === 'settings' || route.name === 'detail' || route.query.bookmarked === '1') return ''
  return currentType.value as typeof currentSignal.value.type
})

const sessionHomeLink = computed(() => {
  if (route.name !== 'list') return ''
  if (currentType.value || route.query.bookmarked === '1') return '/'
  return ''
})

const showSessionGlyph = computed(() => route.name === 'list' && Boolean(currentType.value))

function navigate(type: string) {
  router.push(type ? { path: '/', query: { type } } : '/')
}

function onKeydown(event: KeyboardEvent) {
  const target = event.target as HTMLElement
  const typing = target?.matches('input, textarea, [contenteditable="true"]')
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault()
    paletteOpen.value = !paletteOpen.value
    return
  }
  if (event.key === 'Escape') {
    chordOpen.value = false
    paletteOpen.value = false
    return
  }
  if (event.key === '/' && !typing) {
    event.preventDefault()
    paletteOpen.value = true
    return
  }
  if (!typing && event.key.toLowerCase() === 'g') {
    event.preventDefault()
    chordOpen.value = true
    if (chordTimer) clearTimeout(chordTimer)
    chordTimer = setTimeout(() => { chordOpen.value = false }, 1800)
    return
  }
  if (!typing && chordOpen.value) {
    event.preventDefault()
    const key = event.key.toUpperCase()
    chordOpen.value = false
    if (chordTimer) clearTimeout(chordTimer)
    if (key === 'B') {
      router.push({ path: '/', query: { bookmarked: '1' } })
      return
    }
    const signal = signals.find(item => item.key === key && signalEnabled(item.type))
    if (signal) navigate(signal.type)
    return
  }
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown)
  loadSignalSettings()
})
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  if (chordTimer) clearTimeout(chordTimer)
})
</script>

<template>
  <div class="app-frame">
    <div class="ambient-field" aria-hidden="true"><i /><i /><i /></div>
    <header class="instrument-bar">
      <div class="brand-lockup">
        <RouterLink to="/" class="brand-mark" aria-label="Opentoxic Microscope home">
          <span /><span /><span />
        </RouterLink>
        <div class="brand-name">
          <strong>Microscope</strong>
          <span>Runtime observatory</span>
        </div>
      </div>

      <component
        :is="sessionHomeLink ? RouterLink : 'div'"
        :to="sessionHomeLink || undefined"
        class="session-identity"
        :class="[
          `session-identity--${sessionMode}`,
          { 'session-identity--linked': Boolean(sessionHomeLink) },
        ]"
        :style="{ '--session-signal': sessionSignal }"
        :aria-label="sessionHomeLink ? 'Return to all activity' : undefined"
      >
        <span class="session-identity__sheen" aria-hidden="true" />
        <span class="session-identity__beacon" aria-hidden="true">
          <i /><i class="is-core" /><i />
        </span>
        <span v-if="showSessionGlyph" class="session-identity__glyph" aria-hidden="true">
          <SignalIcon :type="sessionIconType" size="sm" />
        </span>
        <span class="session-identity__copy">
          <span class="session-identity__eyebrow"><i />{{ sessionEyebrow }}</span>
          <strong class="session-identity__title">{{ title }}</strong>
        </span>
        <span v-if="demoMode" class="demo-chip session-identity__chip">Demo</span>
      </component>

      <div class="instrument-actions">
        <button class="command-trigger" @click="paletteOpen = true">
          <svg viewBox="0 0 20 20" aria-hidden="true"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
          <span>Find anything</span>
          <kbd>⌘ K</kbd>
        </button>
        <slot name="actions" />
        <a
          href="https://github.com/opentoxic/microscope"
          class="portal-exit"
          target="_blank"
          rel="noopener noreferrer"
          aria-label="OpenToxic Microscope on GitHub"
        >
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M8 4H4v12h12v-4M10 10l6-6m-4 0h4v4"/></svg>
        </a>
      </div>
    </header>

    <div class="app-body">
      <nav class="signal-sidebar" aria-label="Activity recorders">
        <div class="signal-sidebar__scroll">
          <button
            v-for="(signal, index) in compactSignals"
            :key="signal.type"
            class="signal-tab"
            :class="{ 'is-active': currentType === signal.type && route.name === 'list', 'is-dormant': !signal.available }"
            :style="{ '--signal': signal.color, '--nav-delay': `${index * 32}ms` }"
            @click="navigate(signal.type)"
          >
            <span class="signal-tab__wave"><SignalIcon :type="signal.type" size="sm" /></span>
            <span>{{ signal.shortLabel }}</span>
            <span v-if="index === 2 || index === 6" class="signal-divider" />
          </button>
        </div>
        <RouterLink class="settings-launch" :class="{ 'is-active': route.name === 'settings' }" to="/settings" aria-label="Microscope settings">
          <svg viewBox="0 0 20 20" aria-hidden="true"><path d="M10 6.7a3.3 3.3 0 1 0 0 6.6 3.3 3.3 0 0 0 0-6.6Z"/><path d="m15.5 11.4 1.2 1.1-1.8 3-1.6-.5a6 6 0 0 1-1.9 1.1L11 17.8H7.5L7.1 16a6 6 0 0 1-1.8-1l-1.7.5-1.7-3 1.3-1.2a6 6 0 0 1 0-2.2L2 8l1.8-3 1.6.5a6 6 0 0 1 1.9-1.1L7.7 2h3.5l.4 2.3a6 6 0 0 1 1.8 1l1.7-.5 1.7 3-1.3 1.1a6 6 0 0 1 0 2.5Z"/></svg>
        </RouterLink>
      </nav>

      <main class="workspace">
      <div class="workspace-heading">
        <div>
          <button v-if="route.name === 'detail'" class="back-link" @click="router.back()">
            <svg viewBox="0 0 20 20" aria-hidden="true"><path d="m12.5 5-5 5 5 5"/></svg>
            Back to activity
          </button>
          <template v-else-if="route.name !== 'settings'">
            <span class="workspace-kicker" :style="{ '--signal': currentSignal.color }">
              <i /> {{ currentType ? 'Microscope explorer' : 'Runtime now' }}
            </span>
            <h1>{{ title }}</h1>
          </template>
          <template v-else>
            <span class="workspace-kicker" style="--signal: #20d9ee"><SignalIcon type="" size="sm" /> Recording policy</span>
            <h1>Settings</h1>
          </template>
        </div>
        <div class="workspace-status">
          <slot name="status" />
        </div>
      </div>
      <slot />
      </main>
    </div>

    <CommandPalette :open="paletteOpen" @close="paletteOpen = false" />
    <ConfirmDialog />
    <ToastStack />
    <Transition name="shortcut">
      <div v-if="chordOpen" class="shortcut-hud"><kbd>G</kbd><span>then choose a recorder key</span></div>
    </Transition>
  </div>
</template>
