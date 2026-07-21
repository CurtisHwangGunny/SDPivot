import { computed, ref } from 'vue'
import { STORAGE_KEYS } from '../utils/storage'

export type ThemeMode = 'light' | 'dark' | 'system'

type AppliedTheme = 'light' | 'dark'

const STORAGE_KEY = STORAGE_KEYS.themeMode
const themeMode = ref<ThemeMode>('system')
const appliedTheme = ref<AppliedTheme>('light')
let initialized = false
let mediaQuery: MediaQueryList | null = null

function getSystemTheme(): AppliedTheme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function resolveTheme(mode: ThemeMode): AppliedTheme {
  return mode === 'system' ? getSystemTheme() : mode
}

function applyThemeToDom(theme: AppliedTheme) {
  document.documentElement.setAttribute('data-theme', theme)
  document.documentElement.style.colorScheme = theme
  appliedTheme.value = theme
}

function handleSystemThemeChange() {
  if (themeMode.value === 'system') {
    applyThemeToDom(getSystemTheme())
  }
}

export function initTheme() {
  if (typeof window === 'undefined' || initialized) return
  initialized = true

  const saved = window.localStorage.getItem(STORAGE_KEY) as ThemeMode | null
  if (saved === 'light' || saved === 'dark' || saved === 'system') {
    themeMode.value = saved
  }

  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  if (typeof mediaQuery.addEventListener === 'function') {
    mediaQuery.addEventListener('change', handleSystemThemeChange)
  } else if (typeof mediaQuery.addListener === 'function') {
    mediaQuery.addListener(handleSystemThemeChange)
  }

  applyThemeToDom(resolveTheme(themeMode.value))
}

export function useTheme() {
  const currentThemeLabel = computed(() => {
    if (themeMode.value === 'system') {
      return `跟随系统 · ${appliedTheme.value === 'dark' ? '深色' : '浅色'}`
    }
    return themeMode.value === 'dark' ? '深色模式' : '浅色模式'
  })

  function setTheme(mode: ThemeMode) {
    themeMode.value = mode
    if (typeof window !== 'undefined') {
      window.localStorage.setItem(STORAGE_KEY, mode)
    }
    applyThemeToDom(resolveTheme(mode))
  }

  return {
    themeMode,
    appliedTheme,
    currentThemeLabel,
    setTheme,
  }
}
