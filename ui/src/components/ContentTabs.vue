<script setup lang="ts">
import { ref } from 'vue'
import type { ContentTab } from '../types'
import CodeBlock from './CodeBlock.vue'
import JsonExplorer from './JsonExplorer.vue'

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
    <Transition name="data-view" mode="out-in">
      <JsonExplorer v-if="tabs[active].json" :key="tabs[active].id" :body="tabs[active].body" />
      <CodeBlock v-else :key="tabs[active].id" :body="tabs[active].body" />
    </Transition>
  </section>
</template>
