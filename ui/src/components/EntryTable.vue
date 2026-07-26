<script setup lang="ts">
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import type { Entry } from '../types'
import Badge from './Badge.vue'
import SignalIcon from './SignalIcon.vue'
import { entryDuration, entryMeta, formatClock, isError, methodClass, signalFor, statusClass, summarize, timeAgo } from '../utils'

const props = defineProps<{ entries: Entry[]; currentType: string }>()
const router = useRouter()
const selected = ref<string[]>([])
const focused = ref(0)
const bookmarks = ref<string[]>(loadBookmarks())
const maxDuration = computed(() => Math.max(1, ...props.entries.map(entryDuration)))

function loadBookmarks(): string[] {
  try { return JSON.parse(localStorage.getItem('signal-bookmarks') || '[]') } catch { return [] }
}

function openEntry(id: string) {
  router.push(`/entries/${id}`)
}

function toggleBookmark(id: string) {
  bookmarks.value = bookmarks.value.includes(id)
    ? bookmarks.value.filter(item => item !== id)
    : [...bookmarks.value, id]
  localStorage.setItem('signal-bookmarks', JSON.stringify(bookmarks.value))
}

function toggleCompare(id: string) {
  selected.value = selected.value.includes(id)
    ? selected.value.filter(item => item !== id)
    : [...selected.value.slice(-1), id]
}

function compare() {
  if (selected.value.length !== 2) return
  router.push({ path: `/entries/${selected.value[0]}`, query: { compare: selected.value[1] } })
}

function onKey(event: KeyboardEvent) {
  if (event.key === 'ArrowDown') focused.value = Math.min(props.entries.length - 1, focused.value + 1)
  else if (event.key === 'ArrowUp') focused.value = Math.max(0, focused.value - 1)
  else if (event.key === 'Enter' && props.entries[focused.value]) openEntry(props.entries[focused.value].id)
  else return
  event.preventDefault()
  document.querySelector<HTMLElement>(`[data-entry-index="${focused.value}"]`)?.focus()
}
</script>

<template>
  <div class="activity-stream" role="list" @keydown="onKey">
    <article
      v-for="(entry, index) in entries"
      :key="entry.id"
      :data-entry-index="index"
      class="activity-row"
      :class="{ 'is-error': isError(entry), 'is-selected': selected.includes(entry.id) }"
      role="listitem"
      tabindex="0"
      :style="{ '--signal': signalFor(entry.type).color, '--duration': `${Math.max(3, entryDuration(entry) / maxDuration * 100)}%` }"
      @focus="focused = index"
      @dblclick="openEntry(entry.id)"
      @keydown.enter="openEntry(entry.id)"
    >
      <div class="activity-time">
        <strong>{{ formatClock(entry.created_at) }}</strong>
        <span>{{ timeAgo(entry.created_at) }}</span>
      </div>
      <div class="activity-thread">
        <span class="activity-node"><SignalIcon :type="entry.type" size="sm" /></span>
      </div>
      <div class="activity-main" @click="openEntry(entry.id)">
        <div class="activity-title">
          <span class="activity-kind">{{ signalFor(entry.type).shortLabel }}</span>
          <Badge v-if="entry.type === 'request'" :label="String(entry.content?.method || 'GET')" :class-name="methodClass(String(entry.content?.method || ''))" />
          <strong>{{ summarize(entry) }}</strong>
        </div>
        <div class="activity-context">
          <span>{{ entryMeta(entry) }}</span>
          <span v-if="entry.request_id">req·{{ entry.request_id.slice(0, 8) }}</span>
          <span v-if="entry.tags?.length">{{ entry.tags.slice(0, 2).join(' · ') }}</span>
        </div>
      </div>
      <div class="activity-metric">
        <template v-if="entryDuration(entry)">
          <span>{{ entryDuration(entry) }}ms</span>
          <i><b /></i>
        </template>
        <Badge v-else-if="entry.type === 'request'" :label="String(entry.content?.status || '—')" :class-name="statusClass(entry.content?.status)" />
        <span v-else class="activity-id">{{ entry.id.slice(0, 7) }}</span>
      </div>
      <div class="activity-tools">
        <button
          :class="{ 'is-active': selected.includes(entry.id) }"
          title="Select for comparison"
          aria-label="Select for comparison"
          @click.stop="toggleCompare(entry.id)"
        >
          <svg viewBox="0 0 20 20"><path d="M7 4H4v12h3m6-12h3v12h-3M8.5 7.5l3 2.5-3 2.5"/></svg>
        </button>
        <button
          :class="{ 'is-active': bookmarks.includes(entry.id) }"
          title="Bookmark entry"
          aria-label="Bookmark entry"
          @click.stop="toggleBookmark(entry.id)"
        >
          <svg viewBox="0 0 20 20"><path d="M6 3.5h8v13L10 14l-4 2.5v-13Z"/></svg>
        </button>
        <button title="Open inspector" aria-label="Open inspector" @click.stop="openEntry(entry.id)">
          <svg viewBox="0 0 20 20"><path d="m8 5 5 5-5 5"/></svg>
        </button>
      </div>
    </article>

    <Transition name="compare-tray">
      <div v-if="selected.length" class="compare-tray">
        <div class="compare-stack">
          <span v-for="(id, index) in selected" :key="id">{{ index + 1 }}</span>
        </div>
        <div>
          <strong>{{ selected.length === 2 ? 'Ready to compare' : 'Choose one more trace' }}</strong>
          <small>Side-by-side timing and payload inspection</small>
        </div>
        <button v-if="selected.length === 2" @click="compare">Compare traces <kbd>↵</kbd></button>
        <button class="compare-close" aria-label="Clear selection" @click="selected = []">×</button>
      </div>
    </Transition>
  </div>
</template>
