<script setup lang="ts">
import { ref, watch } from 'vue'
import { useRouter } from 'vue-router'
import type { BatchTypeGroup } from '../types'
import { entryDuration, formatRecordDate, formatRecordTime, signalFor, summarize, timeAgo } from '../utils'

const props = defineProps<{ groups: BatchTypeGroup[]; activeType: string }>()
const router = useRouter()
const activeTab = ref(props.activeType || props.groups[0]?.type || '')

watch(() => props.activeType, (value) => {
  if (value) activeTab.value = value
})

function groupDuration(group: BatchTypeGroup): number {
  return group.entries.reduce((sum, entry) => sum + entryDuration(entry), 0)
}
</script>

<template>
  <section v-if="groups.length" class="relations-panel">
    <header class="relations-heading">
      <div><span>Related signals</span><small>same execution batch</small></div>
      <div class="relation-tabs">
        <button
          v-for="group in groups"
          :key="group.type"
          :class="{ active: activeTab === group.type }"
          :style="{ '--signal': signalFor(group.type).color }"
          @click="activeTab = group.type"
        >
          <i />{{ group.label }} <span>{{ group.entries.length }}</span>
        </button>
      </div>
    </header>

    <template v-for="group in groups" :key="group.type">
      <div v-show="activeTab === group.type" class="relation-body">
        <div class="relation-summary">
          <span>{{ group.entries.length }} {{ group.label.toLowerCase() }}</span>
          <span v-if="groupDuration(group)">Combined cost · {{ groupDuration(group) }}ms</span>
        </div>
        <button v-for="entry in group.entries" :key="entry.id" class="relation-row" :style="{ '--signal': signalFor(entry.type).color }" @click="router.push(`/entries/${entry.id}`)">
          <span class="relation-moment" :title="`${formatRecordDate(entry.created_at)} ${formatRecordTime(entry.created_at)}`">
            <strong>{{ formatRecordTime(entry.created_at) }}</strong>
            <small>{{ formatRecordDate(entry.created_at) }}</small>
          </span>
          <span class="relation-node"><i /></span>
          <span class="relation-type">{{ signalFor(entry.type).shortLabel }}</span>
          <strong>{{ summarize(entry) }}</strong>
          <small>{{ entryDuration(entry) ? `${entryDuration(entry)}ms` : timeAgo(entry.created_at) }}</small>
          <svg viewBox="0 0 20 20"><path d="m8 5 5 5-5 5"/></svg>
        </button>
      </div>
    </template>
  </section>
</template>
