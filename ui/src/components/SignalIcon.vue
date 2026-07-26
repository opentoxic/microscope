<script setup lang="ts">
import type { EntryType } from '../types'

defineProps<{
  type: EntryType | ''
  size?: 'sm' | 'md' | 'lg'
}>()
</script>

<template>
  <span class="signal-icon" :class="`signal-icon--${size || 'md'}`" aria-hidden="true">
    <!-- All activity: scope sweep -->
    <svg v-if="type === ''" viewBox="0 0 16 16"><path d="M2 12c2-4 4-6 6-6s4 2 6 6" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/><path d="M8 10v-3M6.5 8.5 8 10l1.5-1.5" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><circle cx="8" cy="8" r="1.2" fill="currentColor" opacity=".9"/></svg>

    <!-- HTTP requests: inbound arrow -->
    <svg v-else-if="type === 'request'" viewBox="0 0 16 16"><path d="M3 4h10M3 8h7M3 12h4" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><path d="M11 10l3-2-3-2v4z" fill="currentColor"/></svg>

    <!-- SQL queries: table grid -->
    <svg v-else-if="type === 'query'" viewBox="0 0 16 16"><rect x="3" y="3" width="10" height="10" rx="1" fill="none" stroke="currentColor" stroke-width="1.1"/><path d="M3 7h10M3 10h10M7 3v10" fill="none" stroke="currentColor" stroke-width="1"/></svg>

    <!-- External calls: outbound -->
    <svg v-else-if="type === 'http-client'" viewBox="0 0 16 16"><path d="M2 8h9M9 5l3 3-3 3" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/><path d="M3 4.5c1.5-1 3-1.5 5-1.5M3 11.5c1.5 1 3 1.5 5 1.5" fill="none" stroke="currentColor" stroke-width=".9" opacity=".55" stroke-linecap="round"/></svg>

    <!-- Cache: layered stack -->
    <svg v-else-if="type === 'cache'" viewBox="0 0 16 16"><path d="M3 5.5 8 3l5 2.5L8 8 3 5.5z" fill="none" stroke="currentColor" stroke-width="1"/><path d="M3 8.5 8 11l5-2.5" fill="none" stroke="currentColor" stroke-width="1" opacity=".7"/><path d="M3 11.5 8 14l5-2.5" fill="none" stroke="currentColor" stroke-width="1" opacity=".45"/></svg>

    <!-- Redis: key cylinder -->
    <svg v-else-if="type === 'redis'" viewBox="0 0 16 16"><ellipse cx="8" cy="5" rx="4.5" ry="1.8" fill="none" stroke="currentColor" stroke-width="1"/><path d="M3.5 5v6c0 1 2 1.8 4.5 1.8S12.5 12 12.5 11V5" fill="none" stroke="currentColor" stroke-width="1"/><ellipse cx="8" cy="11" rx="4.5" ry="1.8" fill="none" stroke="currentColor" stroke-width="1" opacity=".6"/><circle cx="10.5" cy="3.5" r="1.2" fill="currentColor"/></svg>

    <!-- Queue jobs: stacked bars -->
    <svg v-else-if="type === 'job'" viewBox="0 0 16 16"><rect x="3" y="3" width="10" height="2.2" rx=".5" fill="currentColor" opacity=".35"/><rect x="3" y="6.5" width="10" height="2.2" rx=".5" fill="currentColor" opacity=".6"/><rect x="3" y="10" width="10" height="2.2" rx=".5" fill="currentColor"/><path d="M12 4.2v7.6" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" opacity=".5"/></svg>

    <!-- Redpanda topics: partition stream -->
    <svg v-else-if="type === 'topic'" viewBox="0 0 16 16"><path d="M2 5h12M2 8h12M2 11h12" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round"/><path d="M5 5v6M10 5v6" fill="none" stroke="currentColor" stroke-width="1" opacity=".5"/><circle cx="12.5" cy="5" r="1" fill="currentColor"/></svg>

    <!-- Scheduled tasks: clock -->
    <svg v-else-if="type === 'schedule'" viewBox="0 0 16 16"><circle cx="8" cy="8" r="5" fill="none" stroke="currentColor" stroke-width="1.1"/><path d="M8 5v3.5l2.5 1.5" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><path d="M8 2.5v1M8 12.5v1M2.5 8h1M12.5 8h1" fill="none" stroke="currentColor" stroke-width=".8" opacity=".45" stroke-linecap="round"/></svg>

    <!-- Events: broadcast pulse -->
    <svg v-else-if="type === 'event'" viewBox="0 0 16 16"><circle cx="8" cy="8" r="1.5" fill="currentColor"/><path d="M8 3.5v1.5M8 11v1.5M3.5 8h1.5M11 8h1.5" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round"/><path d="M5.2 5.2l1 1M9.8 9.8l1 1M5.2 10.8l1-1M9.8 6.2l1-1" fill="none" stroke="currentColor" stroke-width=".9" opacity=".55" stroke-linecap="round"/></svg>

    <!-- WebSockets: bidirectional -->
    <svg v-else-if="type === 'websocket'" viewBox="0 0 16 16"><path d="M3 6l3 2-3 2V6zM13 6l-3 2 3 2V6z" fill="currentColor" opacity=".7"/><path d="M6.5 8h3" fill="none" stroke="currentColor" stroke-width="1" stroke-linecap="round" stroke-dasharray="1.5 1.5"/></svg>

    <!-- Logs: text lines -->
    <svg v-else-if="type === 'log'" viewBox="0 0 16 16"><path d="M3 4.5h10M3 8h7M3 11.5h9" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><circle cx="2" cy="4.5" r=".7" fill="currentColor"/><circle cx="2" cy="8" r=".7" fill="currentColor" opacity=".7"/><circle cx="2" cy="11.5" r=".7" fill="currentColor" opacity=".45"/></svg>

    <!-- Exceptions: alert -->
    <svg v-else-if="type === 'exception'" viewBox="0 0 16 16"><path d="M8 2.5 14 13H2L8 2.5z" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linejoin="round"/><path d="M8 6.5v3.5M8 11.5v.5" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round"/></svg>

    <!-- Mail: envelope -->
    <svg v-else-if="type === 'mail'" viewBox="0 0 16 16"><rect x="2.5" y="4.5" width="11" height="7" rx="1" fill="none" stroke="currentColor" stroke-width="1.1"/><path d="M2.5 5.5 8 9.5l5.5-4" fill="none" stroke="currentColor" stroke-width="1"/></svg>

    <!-- Notifications: bell -->
    <svg v-else-if="type === 'notification'" viewBox="0 0 16 16"><path d="M8 2.5c-2 0-3.5 1.5-3.5 4v2.5L3 11h10l-1.5-2V6.5c0-2.5-1.5-4-3.5-4z" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linejoin="round"/><path d="M6.5 11.5a1.5 1.5 0 0 0 3 0" fill="none" stroke="currentColor" stroke-width="1"/></svg>

    <!-- Performance: stopwatch -->
    <svg v-else-if="type === 'performance'" viewBox="0 0 16 16"><circle cx="8.5" cy="9" r="4.5" fill="none" stroke="currentColor" stroke-width="1.1"/><path d="M8.5 9V6.5M6.5 3.5h4" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><path d="M7 3.5V2.5h3v1" fill="none" stroke="currentColor" stroke-width="1"/></svg>

    <!-- Metrics: gauge -->
    <svg v-else-if="type === 'metric'" viewBox="0 0 16 16"><path d="M3 12a5 5 0 0 1 10 0" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><path d="M8 12V7.5" fill="none" stroke="currentColor" stroke-width="1.1" stroke-linecap="round"/><circle cx="8" cy="12" r="1" fill="currentColor"/></svg>

    <!-- Custom: marker pin -->
    <svg v-else-if="type === 'custom'" viewBox="0 0 16 16"><path d="M8 2.5c-2 0-3.5 1.5-3.5 3.5C4.5 9 8 13.5 8 13.5S11.5 9 11.5 6c0-2-1.5-3.5-3.5-3.5z" fill="none" stroke="currentColor" stroke-width="1.1"/><circle cx="8" cy="6" r="1.3" fill="currentColor"/></svg>

    <!-- Fallback: waveform -->
    <svg v-else viewBox="0 0 16 16"><path d="M2 10 4 6 6 9 8 4 10 8 12 5 14 10" fill="none" stroke="currentColor" stroke-width="1.2" stroke-linecap="round" stroke-linejoin="round"/></svg>
  </span>
</template>
