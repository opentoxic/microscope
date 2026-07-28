<script setup lang="ts">
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import { closeConfirm, useConfirmState } from '../confirm'

const state = useConfirmState()
const cancelButton = ref<HTMLButtonElement | null>(null)
const titleId = 'confirm-dialog-title'
const messageId = 'confirm-dialog-message'

function accept() {
  closeConfirm(true)
}

function cancel() {
  closeConfirm(false)
}

function onKeydown(event: KeyboardEvent) {
  if (!state.open) return
  if (event.key === 'Escape') {
    event.preventDefault()
    cancel()
    return
  }
  if (event.key === 'Enter' && state.tone !== 'danger') {
    event.preventDefault()
    accept()
  }
}

watch(() => state.open, async (open) => {
  if (!open) return
  await nextTick()
  cancelButton.value?.focus()
})

onMounted(() => window.addEventListener('keydown', onKeydown))
onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  if (state.open) closeConfirm(false)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="confirm">
      <div v-if="state.open" class="confirm-veil" @click.self="cancel">
        <div
          class="confirm-dialog"
          role="alertdialog"
          aria-modal="true"
          :aria-labelledby="titleId"
          :aria-describedby="messageId"
        >
          <header class="confirm-dialog__header">
            <span class="confirm-dialog__eyebrow" :class="`is-${state.tone}`">
              {{ state.tone === 'danger' ? 'Destructive action' : 'Confirmation' }}
            </span>
            <h2 :id="titleId">{{ state.title }}</h2>
          </header>
          <div class="confirm-dialog__body">
            <p :id="messageId">{{ state.message }}</p>
            <p v-if="state.detail" class="confirm-dialog__detail">{{ state.detail }}</p>
          </div>
          <footer class="confirm-dialog__actions">
            <button ref="cancelButton" type="button" class="confirm-dialog__cancel" @click="cancel">
              {{ state.cancelLabel }}
            </button>
            <button type="button" class="confirm-dialog__confirm" :class="`is-${state.tone}`" @click="accept">
              {{ state.confirmLabel }}
            </button>
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>
