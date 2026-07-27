import { reactive, watch } from 'vue'

export type Accent = 'cyan' | 'violet' | 'lime' | 'amber' | 'rose'
export type Density = 'compact' | 'comfortable' | 'spacious'

interface AppearanceSettings {
  accent: Accent
  density: Density
  glow: boolean
  scanlines: boolean
  grid: boolean
  motion: boolean
  highContrast: boolean
  monospaceData: boolean
}

const defaults: AppearanceSettings = {
  accent: 'cyan',
  density: 'comfortable',
  glow: true,
  scanlines: true,
  grid: true,
  motion: true,
  highContrast: false,
  monospaceData: true,
}

function load(): AppearanceSettings {
  try {
    return { ...defaults, ...JSON.parse(localStorage.getItem('microscope-appearance') || '{}') }
  } catch {
    return { ...defaults }
  }
}

export const appearance = reactive<AppearanceSettings>(load())

export const accentOptions: Array<{ id: Accent; label: string; color: string }> = [
  { id: 'cyan', label: 'Ion cyan', color: '#20d9ee' },
  { id: 'violet', label: 'Plasma violet', color: '#a879ff' },
  { id: 'lime', label: 'Terminal lime', color: '#56f39a' },
  { id: 'amber', label: 'Reactor amber', color: '#ffb84d' },
  { id: 'rose', label: 'Alert rose', color: '#ff5f8d' },
]

export function applyAppearance() {
  const root = document.documentElement
  root.dataset.accent = appearance.accent
  root.dataset.density = appearance.density
  root.classList.toggle('no-glow', !appearance.glow)
  root.classList.toggle('has-scanlines', appearance.scanlines)
  root.classList.toggle('has-grid', appearance.grid)
  root.classList.toggle('reduce-motion', !appearance.motion)
  root.classList.toggle('high-contrast', appearance.highContrast)
  root.classList.toggle('mono-data', appearance.monospaceData)
}

export function resetAppearance() {
  Object.assign(appearance, defaults)
}

watch(appearance, () => {
  localStorage.setItem('microscope-appearance', JSON.stringify(appearance))
  applyAppearance()
}, { deep: true })

