import { reactive } from 'vue'

export type ConfirmTone = 'danger' | 'default'

export interface ConfirmOptions {
  title: string
  message: string
  detail?: string
  confirmLabel?: string
  cancelLabel?: string
  tone?: ConfirmTone
}

interface ConfirmState {
  open: boolean
  title: string
  message: string
  detail: string
  confirmLabel: string
  cancelLabel: string
  tone: ConfirmTone
}

const state = reactive<ConfirmState>({
  open: false,
  title: '',
  message: '',
  detail: '',
  confirmLabel: 'Confirm',
  cancelLabel: 'Cancel',
  tone: 'default',
})

let resolvePromise: ((value: boolean) => void) | null = null

export function useConfirmState() {
  return state
}

export function askConfirm(options: ConfirmOptions): Promise<boolean> {
  if (resolvePromise) resolvePromise(false)
  state.title = options.title
  state.message = options.message
  state.detail = options.detail || ''
  state.confirmLabel = options.confirmLabel || 'Confirm'
  state.cancelLabel = options.cancelLabel || 'Cancel'
  state.tone = options.tone || 'default'
  state.open = true
  return new Promise(resolve => {
    resolvePromise = resolve
  })
}

export function closeConfirm(accepted: boolean) {
  state.open = false
  resolvePromise?.(accepted)
  resolvePromise = null
}
