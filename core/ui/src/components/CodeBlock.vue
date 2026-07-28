<script setup lang="ts">
import { computed, ref } from 'vue'

const props = defineProps<{ body: string; json?: boolean }>()
const copied = ref(false)
const lines = computed(() => props.body.split('\n'))

async function copy() {
  await navigator.clipboard.writeText(props.body)
  copied.value = true
  setTimeout(() => { copied.value = false }, 1500)
}
</script>

<template>
  <div class="code-workspace">
    <div class="code-gutter-head">
      <span>{{ lines.length }} lines</span>
      <button @click="copy">
        <svg viewBox="0 0 20 20"><rect x="7" y="7" width="9" height="9" rx="1"/><path d="M13 7V4H4v9h3"/></svg>
        {{ copied ? 'Copied' : 'Copy' }}
      </button>
    </div>
    <pre :class="{ 'is-json': json }"><code><span v-for="(line, index) in lines" :key="index" class="code-line"><i>{{ index + 1 }}</i><b>{{ line || ' ' }}</b></span></code></pre>
  </div>
</template>
