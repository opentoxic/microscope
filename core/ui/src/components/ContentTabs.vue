<script setup lang="ts">
import { computed, ref } from 'vue'
import type { ContentTab } from '../types'
import CodeBlock from './CodeBlock.vue'
import JsonExplorer from './JsonExplorer.vue'

const props = defineProps<{ tabs: ContentTab[] }>()
const active = ref(0)

const activeTab = computed(() => props.tabs[active.value])

function tabSize(tab: ContentTab): string {
  const bytes = new TextEncoder().encode(tab.body).length
  if (bytes < 1024) return `${bytes} B`
  return `${(bytes / 1024).toFixed(1)} KB`
}
</script>

<template>
  <section v-if="tabs.length" class="editor-panel">
    <header class="editor-tabs">
      <div>
        <button
          v-for="(tab, index) in tabs"
          :key="tab.id"
          :class="{ active: active === index, [`is-${tab.id}`]: true }"
          @click="active = index"
        >
          <span>{{ tab.label }}</span>
          <i v-if="tab.json">JSON</i>
          <b v-if="tab.body">{{ tabSize(tab) }}</b>
        </button>
      </div>
      <span class="editor-language">{{ activeTab.json ? 'application/json' : 'text/plain' }}</span>
    </header>
    <Transition name="data-view" mode="out-in">
      <JsonExplorer
        v-if="activeTab.json"
        :key="activeTab.id"
        :body="activeTab.body"
        :variant="activeTab.id"
      />
      <CodeBlock v-else :key="activeTab.id" :body="activeTab.body" />
    </Transition>
  </section>
</template>
