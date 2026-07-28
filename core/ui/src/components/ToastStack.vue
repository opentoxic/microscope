<script setup lang="ts">
import { dismissToast, useToasts } from '../toast'

const toasts = useToasts()
</script>

<template>
  <Teleport to="body">
    <div class="toast-stack" aria-live="polite" aria-relevant="additions text">
      <TransitionGroup name="toast">
        <article
          v-for="toast in toasts"
          :key="toast.id"
          class="toast"
          :class="`is-${toast.tone}`"
        >
          <span class="toast__icon" aria-hidden="true" />
          <div class="toast__copy">
            <strong>{{ toast.title }}</strong>
            <p>{{ toast.text }}</p>
          </div>
          <button type="button" class="toast__dismiss" aria-label="Dismiss notification" @click="dismissToast(toast.id)">
            ×
          </button>
        </article>
      </TransitionGroup>
    </div>
  </Teleport>
</template>
