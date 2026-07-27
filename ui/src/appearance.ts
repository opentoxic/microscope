import { reactive, watch } from 'vue'

export type Accent = 'cyan' | 'violet' | 'lime' | 'amber' | 'rose'
export type Density = 'compact' | 'comfortable' | 'spacious'
export type ThemePreset = 'terminal' | 'neural' | 'black-ice' | 'reactor'
export type MotionLevel = 'calm' | 'fluid' | 'cinematic'
export type CornerStyle = 'cut' | 'rounded' | 'square'

export interface AppearanceSettings {
  preset: ThemePreset
  accent: Accent
  density: Density
  motionLevel: MotionLevel
  cornerStyle: CornerStyle
  fontScale: number
  glassOpacity: number
  glow: boolean
  scanlines: boolean
  grid: boolean
  motion: boolean
  highContrast: boolean
  monospaceData: boolean
  glass: boolean
  noise: boolean
  ambientLight: boolean
  dataPulse: boolean
}

export const presetOptions: Array<{
  id: ThemePreset
  label: string
  code: string
  description: string
  color: string
}> = [
  { id: 'terminal', label: 'Terminal', code: 'T-01', description: 'Phosphor green, compact data, deep glass', color: '#56f39a' },
  { id: 'neural', label: 'Neural glass', code: 'N-72', description: 'Cyan light, fluid motion, translucent layers', color: '#20d9ee' },
  { id: 'black-ice', label: 'Black ice', code: 'B-09', description: 'Violet energy, hard contrast, quiet surfaces', color: '#a879ff' },
  { id: 'reactor', label: 'Reactor', code: 'R-88', description: 'Amber telemetry, cinematic motion, warm alerts', color: '#ffb84d' },
]

const presetValues: Record<ThemePreset, Partial<AppearanceSettings>> = {
  terminal: {
    preset: 'terminal', accent: 'lime', density: 'compact', motionLevel: 'fluid', cornerStyle: 'cut',
    fontScale: 100, glassOpacity: 72, glow: true, scanlines: true, grid: true, motion: true,
    highContrast: true, monospaceData: true, glass: true, noise: true, ambientLight: true, dataPulse: true,
  },
  neural: {
    preset: 'neural', accent: 'cyan', density: 'comfortable', motionLevel: 'fluid', cornerStyle: 'rounded',
    fontScale: 100, glassOpacity: 64, glow: true, scanlines: false, grid: true, motion: true,
    highContrast: false, monospaceData: true, glass: true, noise: false, ambientLight: true, dataPulse: true,
  },
  'black-ice': {
    preset: 'black-ice', accent: 'violet', density: 'compact', motionLevel: 'calm', cornerStyle: 'square',
    fontScale: 98, glassOpacity: 82, glow: false, scanlines: false, grid: true, motion: true,
    highContrast: true, monospaceData: true, glass: true, noise: true, ambientLight: false, dataPulse: false,
  },
  reactor: {
    preset: 'reactor', accent: 'amber', density: 'spacious', motionLevel: 'cinematic', cornerStyle: 'cut',
    fontScale: 103, glassOpacity: 58, glow: true, scanlines: true, grid: false, motion: true,
    highContrast: false, monospaceData: false, glass: true, noise: true, ambientLight: true, dataPulse: true,
  },
}

const defaults = presetValues.terminal as AppearanceSettings

function load(): AppearanceSettings {
  try {
    const saved = JSON.parse(localStorage.getItem('microscope-appearance') || '{}') as Partial<AppearanceSettings>
    // Settings from the older appearance model intentionally migrate to the new terminal default.
    if (!saved.preset) return { ...defaults }
    return { ...defaults, ...saved }
  } catch {
    return { ...defaults }
  }
}

export const appearance = reactive<AppearanceSettings>(load())

export const accentOptions: Array<{ id: Accent; label: string; color: string }> = [
  { id: 'lime', label: 'Terminal lime', color: '#56f39a' },
  { id: 'cyan', label: 'Ion cyan', color: '#20d9ee' },
  { id: 'violet', label: 'Plasma violet', color: '#a879ff' },
  { id: 'amber', label: 'Reactor amber', color: '#ffb84d' },
  { id: 'rose', label: 'Alert rose', color: '#ff5f8d' },
]

export function applyPreset(preset: ThemePreset) {
  Object.assign(appearance, presetValues[preset])
}

export function applyAppearance() {
  const root = document.documentElement
  root.dataset.preset = appearance.preset
  root.dataset.accent = appearance.accent
  root.dataset.density = appearance.density
  root.dataset.motion = appearance.motionLevel
  root.dataset.corners = appearance.cornerStyle
  root.style.setProperty('--font-scale', String(appearance.fontScale / 100))
  root.style.setProperty('--glass-opacity', String(appearance.glassOpacity / 100))
  root.classList.toggle('no-glow', !appearance.glow)
  root.classList.toggle('has-scanlines', appearance.scanlines)
  root.classList.toggle('has-grid', appearance.grid)
  root.classList.toggle('reduce-motion', !appearance.motion)
  root.classList.toggle('high-contrast', appearance.highContrast)
  root.classList.toggle('mono-data', appearance.monospaceData)
  root.classList.toggle('has-glass', appearance.glass)
  root.classList.toggle('has-noise', appearance.noise)
  root.classList.toggle('has-ambient', appearance.ambientLight)
  root.classList.toggle('has-data-pulse', appearance.dataPulse)
}

export function resetAppearance() {
  applyPreset('terminal')
}

watch(appearance, () => {
  localStorage.setItem('microscope-appearance', JSON.stringify(appearance))
  applyAppearance()
}, { deep: true })
