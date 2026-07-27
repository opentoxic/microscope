<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = defineProps<{ body: string }>()
const mode = ref<'tree' | 'raw'>('tree')
const query = ref('')
const expanded = ref(new Set<string>())
const copied = ref(false)

interface ParseResult {
  value: unknown
  error: string
  decodedLayers: number
}

function parseBody(body: string): ParseResult {
  let value: unknown = body
  let layers = 0
  try {
    value = JSON.parse(body.replace(/^\uFEFF/, '').trim())
    layers++
    // Some SDKs deliver a JSON document inside a JSON string. Decode those layers too.
    while (typeof value === 'string' && layers < 4) {
      const candidate = value.trim()
      if (!candidate.startsWith('{') && !candidate.startsWith('[') && !candidate.startsWith('"')) break
      value = JSON.parse(candidate)
      layers++
    }
    return { value, error: '', decodedLayers: layers }
  } catch (error) {
    return {
      value: body,
      error: error instanceof Error ? error.message : 'Invalid JSON document',
      decodedLayers: layers,
    }
  }
}

const parsed = computed(() => parseBody(props.body))
const isValid = computed(() => !parsed.value.error)

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
  if (value !== null && typeof value === 'object') return Object.entries(value as Record<string, unknown>)
  return []
}

function pathFor(parent: string, key: string) {
  return `${parent}/${key.replaceAll('~', '~0').replaceAll('/', '~1')}`
}

function allRows(): Row[] {
  const result: Row[] = []
  const walk = (key: string, value: unknown, path: string, depth: number, respectExpansion: boolean) => {
    const expandable = children(value).length > 0
    const isExpanded = expanded.value.has(path)
    result.push({ path, key, value, depth, expandable, expanded: isExpanded })
    if (expandable && (!respectExpansion || isExpanded)) {
      children(value).forEach(([childKey, child]) => walk(childKey, child, pathFor(path, childKey), depth + 1, respectExpansion))
    }
  }

  const rootChildren = children(parsed.value.value)
  if (rootChildren.length) {
    rootChildren.forEach(([key, value]) => walk(key, value, pathFor('', key), 0, !query.value.trim()))
  } else {
    walk('value', parsed.value.value, '/value', 0, !query.value.trim())
  }
  return result
}

const rows = computed(() => {
  const result = allRows()
  const needle = query.value.trim().toLowerCase()
  if (!needle) return result
  return result.filter(row => `${row.key} ${row.path} ${displayValue(row.value)}`.toLowerCase().includes(needle))
})

watch(() => props.body, () => {
  query.value = ''
  mode.value = parseBody(props.body).error ? 'raw' : 'tree'
  const next = new Set<string>()
  children(parseBody(props.body).value).forEach(([key, value]) => {
    if (children(value).length) next.add(pathFor('', key))
  })
  expanded.value = next
}, { immediate: true })

function displayValue(value: unknown) {
  if (Array.isArray(value)) return `Array(${value.length})`
  if (value !== null && typeof value === 'object') return `Object(${Object.keys(value).length})`
  if (typeof value === 'string') return `"${value}"`
  if (value === null) return 'null'
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
    children(value).forEach(([key, child]) => walk(child, pathFor(path, key)))
  }
  children(parsed.value.value).forEach(([key, child]) => walk(child, pathFor('', key)))
  expanded.value = next
}

function collapseAll() {
  expanded.value = new Set()
}

async function copy() {
  const formatted = isValid.value ? JSON.stringify(parsed.value.value, null, 2) : props.body
  await navigator.clipboard.writeText(formatted)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1400)
}
</script>

<template>
  <div class="json-explorer">
    <div class="data-toolbar">
      <div class="json-document-state" :class="{ invalid: !isValid }">
        <i />
        <span>{{ isValid ? 'Parsed JSON' : 'Invalid JSON' }}</span>
        <small v-if="parsed.decodedLayers > 1">{{ parsed.decodedLayers }} layers decoded</small>
      </div>
      <div class="view-switch">
        <button :class="{ active: mode === 'tree' }" :disabled="!isValid" @click="mode = 'tree'">Tree</button>
        <button :class="{ active: mode === 'raw' }" @click="mode = 'raw'">Source</button>
      </div>
      <label v-if="mode === 'tree'" class="data-search">
        <svg viewBox="0 0 20 20"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
        <input v-model="query" placeholder="Search every key and value" />
      </label>
      <button v-if="mode === 'tree'" class="data-action" @click="expandAll">Expand</button>
      <button v-if="mode === 'tree'" class="data-action" @click="collapseAll">Collapse</button>
      <button class="data-action" @click="copy">{{ copied ? 'Copied' : 'Copy' }}</button>
    </div>

    <div v-if="mode === 'tree' && isValid" class="json-tree">
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
      <div v-if="!rows.length" class="data-empty">No keys or values match “{{ query }}”.</div>
    </div>
    <div v-else class="json-source">
      <p v-if="parsed.error"><strong>Parser message</strong>{{ parsed.error }}</p>
      <pre class="json-raw"><code>{{ isValid ? JSON.stringify(parsed.value, null, 2) : body }}</code></pre>
    </div>
  </div>
</template>
