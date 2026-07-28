export function durationCell(ms: unknown): string {
  if (ms == null || ms === '') return '<span class="text-muted-foreground">-</span>'
  const val = Number(ms)
  const slow = val > 100
  const cls = slow ? 'text-destructive font-semibold' : 'text-muted-foreground'
  return `<span class="${cls}">${val}ms</span>`
}

export { summarize, timeAgo, methodClass, statusClass, levelClass } from './utils'
