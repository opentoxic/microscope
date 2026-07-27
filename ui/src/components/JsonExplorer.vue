<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ body: string }>()
const mode = ref<'tree' | 'raw'>('tree')
const query = ref('')
const expanded = ref(new Set<string>(['root']))
const copied = ref(false)

const parsed = computed<unknown>(() => {
  try { return JSON.parse(props.body) } catch { return props.body }
})

interface Row {
  path: string
  key: string
  value: unknown
  depth: number
  expandable: boolean
  expanded: boolean
}

function children(value: unknown): Array<[string, unknown]> {
  if (Array.isArray(value)) return value.map((item, index) => [String(index), item])
  if (value && typeof value === 'object') return Object.entries(value as Record<string, unknown>)
  return []
}

const rows = computed(() => {
  const result: Row[] = []
  const walk = (key: string, value: unknown, path: string, depth: number) => {
    const expandable = children(value).length > 0
    const isExpanded = expanded.value.has(path)
    result.push({ path, key, value, depth, expandable, expanded: isExpanded })
    if (expandable && isExpanded) children(value).forEach(([childKey, child]) => walk(childKey, child, `${path}.${childKey}`, depth + 1))
  }
  walk('root', parsed.value, 'root', 0)
  const needle = query.value.trim().toLowerCase()
  return needle
    ? result.filter(row => `${row.path} ${displayValue(row.value)}`.toLowerCase().includes(needle))
    : result
})

function displayValue(value: unknown) {
  if (Array.isArray(value)) return `Array(${value.length})`
  if (value && typeof value === 'object') return `Object(${Object.keys(value).length})`
  if (typeof value === 'string') return `"${value}"`
  return String(value)
}

function valueType(value: unknown) {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

function toggle(path: string) {
  const next = new Set(expanded.value)
  next.has(path) ? next.delete(path) : next.add(path)
  expanded.value = next
}

function expandAll() {
  const next = new Set<string>()
  const walk = (value: unknown, path: string) => {
    if (!children(value).length) return
    next.add(path)
    children(value).forEach(([key, child]) => walk(child, `${path}.${key}`))
  }
  walk(parsed.value, 'root')
  expanded.value = next
}

async function copy() {
  await navigator.clipboard.writeText(props.body)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1400)
}
</script>

<template>
  <div class="json-explorer">
    <div class="data-toolbar">
      <div class="view-switch">
        <button :class="{ active: mode === 'tree' }" @click="mode = 'tree'">Tree</button>
        <button :class="{ active: mode === 'raw' }" @click="mode = 'raw'">Raw</button>
      </div>
      <label v-if="mode === 'tree'" class="data-search">
        <svg viewBox="0 0 20 20"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
        <input v-model="query" placeholder="Search keys or values" />
      </label>
      <button v-if="mode === 'tree'" class="data-action" @click="expandAll">Expand all</button>
      <button class="data-action" @click="copy">{{ copied ? 'Copied' : 'Copy JSON' }}</button>
    </div>

    <div v-if="mode === 'tree'" class="json-tree">
      <button
        v-for="row in rows"
        :key="row.path"
        class="json-row"
        :style="{ '--depth': row.depth }"
        :title="row.path"
        @click="row.expandable && toggle(row.path)"
      >
        <span class="json-chevron" :class="{ open: row.expanded, hidden: !row.expandable }">›</span>
        <span class="json-key">{{ row.key }}</span>
        <span class="json-separator">:</span>
        <span class="json-value" :class="`is-${valueType(row.value)}`">{{ displayValue(row.value) }}</span>
        <span class="json-path">{{ row.path }}</span>
      </button>
      <div v-if="!rows.length" class="data-empty">No matching keys or values.</div>
    </div>
    <pre v-else class="json-raw"><code>{{ body }}</code></pre>
  </div>
</template>

