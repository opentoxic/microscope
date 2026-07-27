import { reactive } from 'vue'

export type ToastTone = 'success' | 'error'

export interface ToastOptions {
  text: string
  title?: string
  tone?: ToastTone
  duration?: number
}

export interface ToastItem {
  id: string
  text: string
  title: string
  tone: ToastTone
}

const toasts = reactive<ToastItem[]>([])

function createToastId() {
  if (typeof crypto !== 'undefined' && 'randomUUID' in crypto) return crypto.randomUUID()
  return `${Date.now()}-${Math.random().toString(36).slice(2)}`
}

export function useToasts() {
  return toasts
}

export function showToast(options: ToastOptions) {
  const tone = options.tone || 'success'
  const toast: ToastItem = {
    id: createToastId(),
    text: options.text,
    title: options.title || (tone === 'error' ? 'Action failed' : 'Done'),
    tone,
  }
  toasts.push(toast)
  const duration = options.duration ?? 5200
  window.setTimeout(() => dismissToast(toast.id), duration)
  return toast.id
}

export function dismissToast(id: string) {
  const index = toasts.findIndex(item => item.id === id)
  if (index >= 0) toasts.splice(index, 1)
}
