<script setup lang="ts">
import { RouterView } from 'vue-router'
</script>

<template>
  <RouterView v-slot="{ Component, route }">
    <div :key="route.fullPath" class="route-progress" aria-hidden="true" />
    <Transition name="page-shift" mode="out-in">
      <component :is="Component" :key="`${String(route.name)}:${String(route.params.id || '')}`" />
    </Transition>
  </RouterView>
</template>
