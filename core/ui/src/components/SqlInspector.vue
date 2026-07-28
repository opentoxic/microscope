<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ sql: string; bindings?: unknown; duration?: number; connection?: string }>()
const formatted = ref(true)
const copied = ref(false)

const keywords = ['select', 'from', 'where', 'left join', 'right join', 'inner join', 'outer join', 'join', 'group by', 'order by', 'having', 'limit', 'offset', 'insert into', 'values', 'update', 'set', 'delete from', 'returning', 'and', 'or']

const displaySql = computed(() => {
  if (!formatted.value) return props.sql || ''
  let sql = (props.sql || '').replace(/\s+/g, ' ').trim()
  for (const keyword of keywords.slice(0, 18)) {
    sql = sql.replace(new RegExp(`\\s+(${keyword.replace(' ', '\\s+')})\\s+`, 'ig'), `\n$1 `)
  }
  return sql.replace(/^\n/, '')
})

const lines = computed(() => displaySql.value.split('\n').map(line => {
  const parts = line.split(/(\b(?:SELECT|FROM|WHERE|LEFT|RIGHT|INNER|OUTER|JOIN|GROUP|BY|ORDER|HAVING|LIMIT|OFFSET|INSERT|INTO|VALUES|UPDATE|SET|DELETE|RETURNING|AND|OR|AS|ON|NULL|IS|NOT)\b|'[^']*'|\b\d+(?:\.\d+)?\b)/gi)
  return parts.map(part => ({
    text: part,
    kind: /^(select|from|where|left|right|inner|outer|join|group|by|order|having|limit|offset|insert|into|values|update|set|delete|returning|and|or|as|on|null|is|not)$/i.test(part)
      ? 'keyword'
      : /^'/.test(part) ? 'string' : /^\d/.test(part) ? 'number' : 'plain',
  }))
}))

const bindings = computed(() => {
  if (Array.isArray(props.bindings)) return props.bindings
  if (props.bindings && typeof props.bindings === 'object') return Object.values(props.bindings as Record<string, unknown>)
  return []
})

async function copy() {
  await navigator.clipboard.writeText(props.sql || '')
  copied.value = true
  setTimeout(() => { copied.value = false }, 1400)
}
</script>

<template>
  <section class="sql-studio">
    <header class="data-toolbar">
      <div class="sql-identity">
        <span>SQL</span>
        <strong>{{ connection || 'default connection' }}</strong>
        <i v-if="duration != null" :class="{ slow: duration > 500 }">{{ duration }}ms</i>
      </div>
      <div class="sql-actions">
        <button class="data-action" :class="{ active: formatted }" @click="formatted = !formatted">{{ formatted ? 'Formatted' : 'Original' }}</button>
        <button class="data-action" @click="copy">{{ copied ? 'Copied' : 'Copy query' }}</button>
      </div>
    </header>
    <pre class="sql-code"><code><span v-for="(line, index) in lines" :key="index" class="sql-line"><i>{{ index + 1 }}</i><b><span v-for="(token, tokenIndex) in line" :key="tokenIndex" :class="`sql-${token.kind}`">{{ token.text }}</span></b></span></code></pre>
    <footer v-if="bindings.length" class="sql-bindings">
      <span>Bindings</span>
      <button v-for="(binding, index) in bindings" :key="index" :title="`Parameter ${index + 1}: ${String(binding)}`">
        <i>${{ index + 1 }}</i><strong>{{ String(binding) }}</strong>
      </button>
    </footer>
  </section>
</template>
