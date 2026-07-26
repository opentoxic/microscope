<script setup lang="ts">
import { ref } from 'vue'
import type { ContentTab } from '../types'
import CodeBlock from './CodeBlock.vue'

defineProps<{ tabs: ContentTab[] }>()
const active = ref(0)
</script>

<template>
  <section v-if="tabs.length" class="editor-panel">
    <header class="editor-tabs">
      <div>
        <button
          v-for="(tab, index) in tabs"
          :key="tab.id"
          :class="{ active: active === index }"
          @click="active = index"
        >
          <span>{{ tab.label }}</span>
          <i v-if="tab.json">JSON</i>
        </button>
      </div>
      <span class="editor-language">{{ tabs[active].json ? 'application/json' : 'text/plain' }}</span>
    </header>
    <CodeBlock :body="tabs[active].body" :json="tabs[active].json" />
  </section>
</template>
