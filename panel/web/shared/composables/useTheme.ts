import { ref, computed } from 'vue'

export type Theme = 'dark' | 'light'

const STORAGE_KEY = 'koris-theme'
const ATTRIBUTE = 'data-theme'

/**
 * Reads persisted theme from localStorage.
 * Defaults to 'dark' if nothing is stored or value is invalid.
 */
function getPersistedTheme(): Theme {
  try {
    const stored = localStorage.getItem(STORAGE_KEY)
    if (stored === 'light' || stored === 'dark') {
      return stored
    }
  } catch {
    // localStorage may be unavailable (e.g. private browsing in some browsers)
  }
  return 'dark'
}

/**
 * Applies theme to the document root via the data-theme attribute.
 * CSS custom properties in tokens.css respond to [data-theme="light"] overrides.
 */
function applyTheme(theme: Theme): void {
  document.documentElement.setAttribute(ATTRIBUTE, theme)
}

// Initialize before first paint — read persisted value and apply immediately.
// This runs at module load time so the correct theme is set before any component renders.
const initialTheme = getPersistedTheme()
applyTheme(initialTheme)

// Module-level singleton reactive state so all consumers share the same theme
const current = ref<Theme>(initialTheme)

/**
 * useTheme composable
 *
 * Provides reactive theme state with persistence to localStorage
 * and automatic application via the data-theme attribute on document.documentElement.
 *
 * Requirements: 14.1, 14.2, 14.3, 26.2
 */
export function useTheme() {
  /** Whether the current theme is dark (reactive computed) */
  const isDark = computed(() => current.value === 'dark')

  /**
   * Toggles between dark and light themes.
   * Persists the new value to localStorage and applies it to the document root.
   */
  function toggle(): void {
    const next: Theme = current.value === 'dark' ? 'light' : 'dark'
    current.value = next
    applyTheme(next)
    try {
      localStorage.setItem(STORAGE_KEY, next)
    } catch {
      // Silently handle localStorage write failures
    }
  }

  /**
   * Sets a specific theme directly.
   * Persists the value to localStorage and applies it to the document root.
   */
  function setTheme(theme: Theme): void {
    current.value = theme
    applyTheme(theme)
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      // Silently handle localStorage write failures
    }
  }

  return {
    /** Current active theme (reactive ref) */
    current,
    /** Whether the current theme is dark (reactive computed) */
    isDark,
    /** Toggle between dark and light */
    toggle,
    /** Set a specific theme */
    setTheme,
  }
}
