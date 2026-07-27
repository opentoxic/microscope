<script setup lang="ts">
import { computed, ref, watch } from 'vue'

const props = withDefaults(defineProps<{
  body: string
  variant?: string
}>(), {
  variant: 'default',
})

const mode = ref<'tree' | 'raw'>('tree')
const query = ref('')
const expanded = ref(new Set<string>())
const expandedStrings = ref(new Set<string>())
const copied = ref(false)
const copiedPath = ref('')
const selectedPath = ref('')

const SENSITIVE_KEYS = new Set([
  'authorization', 'cookie', 'set-cookie', 'x-api-key', 'x-auth-token',
  'x-csrf-token', 'proxy-authorization', 'x-amz-security-token',
])

const VARIANT_LABELS: Record<string, string> = {
  payload: 'Request body',
  headers: 'Header map',
  response: 'Response body',
}

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
const needle = computed(() => query.value.trim().toLowerCase())

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

function countNodes(value: unknown): number {
  const childList = children(value)
  if (!childList.length) return 1
  return 1 + childList.reduce((sum, [, child]) => sum + countNodes(child), 0)
}

function maxDepth(value: unknown, depth = 0): number {
  const childList = children(value)
  if (!childList.length) return depth
  return Math.max(...childList.map(([, child]) => maxDepth(child, depth + 1)))
}

const stats = computed(() => {
  const rootChildren = children(parsed.value.value)
  return {
    bytes: new TextEncoder().encode(props.body).length,
    keys: rootChildren.length,
    nodes: isValid.value ? countNodes(parsed.value.value) : 0,
    depth: isValid.value ? maxDepth(parsed.value.value) : 0,
  }
})

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
    rootChildren.forEach(([key, value]) => walk(key, value, pathFor('', key), 0, !needle.value))
  } else {
    walk('value', parsed.value.value, '/value', 0, !needle.value)
  }
  return result
}

function rowMatches(row: Row): boolean {
  if (!needle.value) return false
  return `${row.key} ${row.path} ${displayValue(row.value)}`.toLowerCase().includes(needle.value)
}

const rows = computed(() => {
  const result = allRows()
  if (!needle.value) return result
  return result.filter(row => rowMatches(row))
})

const matchCount = computed(() => {
  if (!needle.value) return 0
  return allRows().filter(row => rowMatches(row)).length
})

watch(() => props.body, () => {
  query.value = ''
  selectedPath.value = ''
  expandedStrings.value = new Set()
  mode.value = parseBody(props.body).error ? 'raw' : 'tree'
  const next = new Set<string>()
  children(parseBody(props.body).value).forEach(([key, value]) => {
    if (children(value).length) next.add(pathFor('', key))
  })
  expanded.value = next
}, { immediate: true })

watch(needle, (value) => {
  if (!value) return
  const next = new Set(expanded.value)
  const walk = (node: unknown, path: string) => {
    const childList = children(node)
    if (!childList.length) return
    next.add(path)
    childList.forEach(([key, child]) => walk(child, pathFor(path, key)))
  }
  children(parsed.value.value).forEach(([key, child]) => walk(child, pathFor('', key)))
  expanded.value = next
})

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function displayValue(value: unknown, path = '') {
  if (Array.isArray(value)) return `Array(${value.length})`
  if (value !== null && typeof value === 'object') return `Object(${Object.keys(value).length})`
  if (typeof value === 'string') {
    const expanded = expandedStrings.value.has(path)
    const quoted = `"${value}"`
    if (!expanded && value.length > 96) return `"${value.slice(0, 96)}…"`
    return quoted
  }
  if (value === null) return 'null'
  return String(value)
}

function isLongString(value: unknown, path: string): boolean {
  return typeof value === 'string' && value.length > 96 && !expandedStrings.value.has(path)
}

function valueType(value: unknown) {
  if (value === null) return 'null'
  if (Array.isArray(value)) return 'array'
  return typeof value
}

function isSensitive(key: string): boolean {
  return props.variant === 'headers' && SENSITIVE_KEYS.has(key.toLowerCase())
}

function escapeHtml(text: string): string {
  return text
    .replaceAll('&', '&amp;')
    .replaceAll('<', '&lt;')
    .replaceAll('>', '&gt;')
    .replaceAll('"', '&quot;')
}

function escapeRegex(text: string): string {
  return text.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')
}

function highlightText(text: string): string {
  const escaped = escapeHtml(text)
  if (!needle.value) return escaped
  const re = new RegExp(`(${escapeRegex(needle.value)})`, 'gi')
  return escaped.replace(re, '<mark class="json-hit">$1</mark>')
}

function toggle(path: string) {
  const next = new Set(expanded.value)
  next.has(path) ? next.delete(path) : next.add(path)
  expanded.value = next
}

function toggleString(path: string) {
  const next = new Set(expandedStrings.value)
  next.has(path) ? next.delete(path) : next.add(path)
  expandedStrings.value = next
}

function selectRow(path: string) {
  selectedPath.value = path
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
  expandedStrings.value = new Set()
}

function rawValue(value: unknown): string {
  if (typeof value === 'string') return value
  if (value === null) return 'null'
  return JSON.stringify(value)
}

async function copy() {
  const formatted = isValid.value ? JSON.stringify(parsed.value.value, null, 2) : props.body
  await navigator.clipboard.writeText(formatted)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1400)
}

async function copyRowValue(row: Row) {
  await navigator.clipboard.writeText(rawValue(row.value))
  copiedPath.value = row.path
  setTimeout(() => { if (copiedPath.value === row.path) copiedPath.value = '' }, 1400)
}

async function copyPath(row: Row) {
  await navigator.clipboard.writeText(row.path)
  copiedPath.value = `${row.path}:path`
  setTimeout(() => { if (copiedPath.value === `${row.path}:path`) copiedPath.value = '' }, 1400)
}
</script>

<template>
  <div class="json-explorer" :class="`json-explorer--${variant}`">
    <div class="data-toolbar">
      <div class="json-document-state" :class="{ invalid: !isValid }">
        <i />
        <span>{{ isValid ? 'Parsed JSON' : 'Invalid JSON' }}</span>
        <small v-if="VARIANT_LABELS[variant]">{{ VARIANT_LABELS[variant] }}</small>
        <small v-if="parsed.decodedLayers > 1">{{ parsed.decodedLayers }} layers decoded</small>
      </div>
      <div v-if="isValid" class="json-stats">
        <span>{{ formatBytes(stats.bytes) }}</span>
        <span v-if="stats.keys">{{ stats.keys }} keys</span>
        <span>{{ stats.nodes }} nodes</span>
        <span v-if="stats.depth">depth {{ stats.depth }}</span>
      </div>
      <div class="view-switch">
        <button :class="{ active: mode === 'tree' }" :disabled="!isValid" @click="mode = 'tree'">Tree</button>
        <button :class="{ active: mode === 'raw' }" @click="mode = 'raw'">Source</button>
      </div>
      <label v-if="mode === 'tree'" class="data-search">
        <svg viewBox="0 0 20 20"><circle cx="8.5" cy="8.5" r="4.5"/><path d="m12 12 4 4"/></svg>
        <input v-model="query" placeholder="Search keys and values" />
        <span v-if="needle" class="json-match-count">{{ matchCount }}</span>
      </label>
      <button v-if="mode === 'tree'" class="data-action" @click="expandAll">Expand</button>
      <button v-if="mode === 'tree'" class="data-action" @click="collapseAll">Collapse</button>
      <button class="data-action" @click="copy">{{ copied ? 'Copied' : 'Copy' }}</button>
    </div>

    <div v-if="mode === 'tree' && isValid" class="json-tree">
      <div
        v-for="row in rows"
        :key="row.path"
        class="json-row"
        :class="{
          selected: selectedPath === row.path,
          'is-match': rowMatches(row),
          'is-sensitive': isSensitive(row.key),
        }"
        :style="{ '--depth': row.depth }"
        @click="selectRow(row.path)"
      >
        <button
          type="button"
          class="json-chevron"
          :class="{ open: row.expanded, hidden: !row.expandable }"
          :disabled="!row.expandable"
          :aria-label="row.expanded ? 'Collapse' : 'Expand'"
          @click.stop="row.expandable && toggle(row.path)"
        >›</button>
        <div class="json-line">
          <span class="json-key-wrap">
            <span class="json-key" v-html="highlightText(row.key)" />
            <span v-if="isSensitive(row.key)" class="json-sensitive-badge" title="Sensitive header">secret</span>
          </span>
          <span class="json-separator">:</span>
          <span
            class="json-value"
            :class="[`is-${valueType(row.value)}`, { 'is-truncated': isLongString(row.value, row.path), 'is-expanded': expandedStrings.has(row.path) }]"
            :title="typeof row.value === 'string' ? row.value : undefined"
            v-html="highlightText(displayValue(row.value, row.path))"
          />
          <button
            v-if="typeof row.value === 'string' && row.value.length > 96"
            type="button"
            class="json-expand-string"
            @click.stop="toggleString(row.path)"
          >{{ isLongString(row.value, row.path) ? 'more' : 'less' }}</button>
        </div>
        <div class="json-row-meta">
          <span class="json-path" :title="row.path">{{ row.path }}</span>
          <span class="json-row-actions">
            <button
              type="button"
              class="json-row-action"
              :class="{ copied: copiedPath === row.path }"
              title="Copy value"
              @click.stop="copyRowValue(row)"
            >{{ copiedPath === row.path ? '✓' : 'val' }}</button>
            <button
              type="button"
              class="json-row-action"
              :class="{ copied: copiedPath === `${row.path}:path` }"
              title="Copy JSON path"
              @click.stop="copyPath(row)"
            >path</button>
          </span>
        </div>
      </div>
      <div v-if="!rows.length" class="data-empty">No keys or values match “{{ query }}”.</div>
    </div>
    <div v-else class="json-source">
      <p v-if="parsed.error"><strong>Parser message</strong>{{ parsed.error }}</p>
      <pre class="json-raw"><code>{{ isValid ? JSON.stringify(parsed.value, null, 2) : body }}</code></pre>
    </div>
  </div>
</template>
