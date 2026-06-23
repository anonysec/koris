import { ref, computed, watchEffect } from 'vue'

export type ThemeMode = 'system' | 'dark' | 'light'
export type UITheme = 'default-light' | 'default-dark' | 'ocean' | 'forest' | 'sunset' | 'monochrome'

export interface ThemeColors {
  primary: string
  primaryHover: string
  secondary: string
  background: string
  surface: string
  surfaceHover: string
  text: string
  textMuted: string
  border: string
  success: string
  warning: string
  error: string
  info: string
  accent: string
}

export interface ThemeShadows {
  sm: string
  md: string
  lg: string
}

export interface ThemeConfig {
  colors: ThemeColors
  borderRadius: string
  shadows: ThemeShadows
}

export interface ThemeInfo {
  id: UITheme
  name: string
  description: string
  mode: 'light' | 'dark'
  config: ThemeConfig
  /** Shorthand accessors for backward compat */
  colors: { bg: string; surface: string; primary: string; accent: string }
  isDefault?: boolean
}

const MODE_KEY = 'koris-mode'
const THEME_KEY = 'koris-ui-theme'
const MODE_ATTRIBUTE = 'data-theme'
const THEME_ATTRIBUTE = 'data-ui-theme'

export const availableThemes: ThemeInfo[] = [
  {
    id: 'default-light',
    name: 'Default Light',
    description: 'Clean light theme with blue accents',
    mode: 'light',
    config: {
      colors: {
        primary: '#3b82f6',
        primaryHover: '#2563eb',
        secondary: '#64748b',
        background: '#ffffff',
        surface: '#f8fafc',
        surfaceHover: '#f1f5f9',
        text: '#1e293b',
        textMuted: '#64748b',
        border: '#e2e8f0',
        success: '#22c55e',
        warning: '#f59e0b',
        error: '#ef4444',
        info: '#3b82f6',
        accent: '#8b5cf6',
      },
      borderRadius: '8px',
      shadows: {
        sm: '0 1px 2px rgba(0,0,0,0.05)',
        md: '0 4px 6px rgba(0,0,0,0.07)',
        lg: '0 10px 15px rgba(0,0,0,0.1)',
      },
    },
    colors: { bg: '#ffffff', surface: '#f8fafc', primary: '#3b82f6', accent: '#8b5cf6' },
    isDefault: true,
  },
  {
    id: 'default-dark',
    name: 'Default Dark',
    description: 'Dark theme with soft blue tones',
    mode: 'dark',
    config: {
      colors: {
        primary: '#60a5fa',
        primaryHover: '#93c5fd',
        secondary: '#94a3b8',
        background: '#0f172a',
        surface: '#1e293b',
        surfaceHover: '#334155',
        text: '#f1f5f9',
        textMuted: '#94a3b8',
        border: '#334155',
        success: '#4ade80',
        warning: '#fbbf24',
        error: '#f87171',
        info: '#60a5fa',
        accent: '#a78bfa',
      },
      borderRadius: '8px',
      shadows: {
        sm: '0 1px 2px rgba(0,0,0,0.3)',
        md: '0 4px 6px rgba(0,0,0,0.4)',
        lg: '0 10px 15px rgba(0,0,0,0.5)',
      },
    },
    colors: { bg: '#0f172a', surface: '#1e293b', primary: '#60a5fa', accent: '#a78bfa' },
  },
  {
    id: 'ocean',
    name: 'Ocean',
    description: 'Deep blue-cyan palette for dark mode',
    mode: 'dark',
    config: {
      colors: {
        primary: '#06b6d4',
        primaryHover: '#22d3ee',
        secondary: '#7dd3fc',
        background: '#0c1222',
        surface: '#162032',
        surfaceHover: '#1e3048',
        text: '#e0f2fe',
        textMuted: '#7dd3fc',
        border: '#1e3a5f',
        success: '#34d399',
        warning: '#fbbf24',
        error: '#fb7185',
        info: '#06b6d4',
        accent: '#a78bfa',
      },
      borderRadius: '10px',
      shadows: {
        sm: '0 1px 3px rgba(6,182,212,0.1)',
        md: '0 4px 8px rgba(6,182,212,0.15)',
        lg: '0 10px 20px rgba(6,182,212,0.2)',
      },
    },
    colors: { bg: '#0c1222', surface: '#162032', primary: '#06b6d4', accent: '#a78bfa' },
  },
  {
    id: 'forest',
    name: 'Forest',
    description: 'Nature-inspired dark green theme',
    mode: 'dark',
    config: {
      colors: {
        primary: '#22c55e',
        primaryHover: '#4ade80',
        secondary: '#86efac',
        background: '#0a1a0f',
        surface: '#142b1a',
        surfaceHover: '#1e3d26',
        text: '#ecfdf5',
        textMuted: '#86efac',
        border: '#1e4d2b',
        success: '#22c55e',
        warning: '#eab308',
        error: '#f87171',
        info: '#38bdf8',
        accent: '#c084fc',
      },
      borderRadius: '6px',
      shadows: {
        sm: '0 1px 3px rgba(34,197,94,0.1)',
        md: '0 4px 8px rgba(34,197,94,0.15)',
        lg: '0 10px 20px rgba(34,197,94,0.2)',
      },
    },
    colors: { bg: '#0a1a0f', surface: '#142b1a', primary: '#22c55e', accent: '#c084fc' },
  },
  {
    id: 'sunset',
    name: 'Sunset',
    description: 'Warm orange light theme',
    mode: 'light',
    config: {
      colors: {
        primary: '#f97316',
        primaryHover: '#fb923c',
        secondary: '#f59e0b',
        background: '#fffbeb',
        surface: '#fef3c7',
        surfaceHover: '#fde68a',
        text: '#451a03',
        textMuted: '#92400e',
        border: '#fcd34d',
        success: '#16a34a',
        warning: '#f97316',
        error: '#dc2626',
        info: '#0891b2',
        accent: '#7c3aed',
      },
      borderRadius: '12px',
      shadows: {
        sm: '0 1px 3px rgba(249,115,22,0.1)',
        md: '0 4px 8px rgba(249,115,22,0.12)',
        lg: '0 10px 20px rgba(249,115,22,0.15)',
      },
    },
    colors: { bg: '#fffbeb', surface: '#fef3c7', primary: '#f97316', accent: '#7c3aed' },
  },
  {
    id: 'monochrome',
    name: 'Monochrome',
    description: 'Minimalist grayscale light theme',
    mode: 'light',
    config: {
      colors: {
        primary: '#374151',
        primaryHover: '#1f2937',
        secondary: '#6b7280',
        background: '#ffffff',
        surface: '#f9fafb',
        surfaceHover: '#f3f4f6',
        text: '#111827',
        textMuted: '#6b7280',
        border: '#d1d5db',
        success: '#059669',
        warning: '#d97706',
        error: '#dc2626',
        info: '#374151',
        accent: '#374151',
      },
      borderRadius: '4px',
      shadows: {
        sm: '0 1px 2px rgba(0,0,0,0.05)',
        md: '0 2px 4px rgba(0,0,0,0.06)',
        lg: '0 4px 8px rgba(0,0,0,0.08)',
      },
    },
    colors: { bg: '#ffffff', surface: '#f9fafb', primary: '#374151', accent: '#374151' },
  },
]

function getPersistedMode(): ThemeMode {
  try {
    const stored = localStorage.getItem(MODE_KEY)
    if (stored === 'system' || stored === 'dark' || stored === 'light') {
      return stored
    }
  } catch {
    // localStorage unavailable
  }
  return 'system'
}

function getPersistedTheme(): UITheme {
  try {
    const stored = localStorage.getItem(THEME_KEY)
    if (stored && availableThemes.some((t) => t.id === stored)) {
      return stored as UITheme
    }
  } catch {
    // localStorage unavailable
  }
  return 'default-dark'
}

function getSystemPrefersDark(): boolean {
  if (typeof window === 'undefined') return true
  return window.matchMedia('(prefers-color-scheme: dark)').matches
}

function resolveEffectiveMode(mode: ThemeMode, theme: UITheme): 'dark' | 'light' {
  const themeInfo = availableThemes.find((t) => t.id === theme)
  // Theme's declared mode takes priority (each preset has a fixed mode)
  if (themeInfo) {
    return themeInfo.mode
  }
  if (mode === 'system') {
    return getSystemPrefersDark() ? 'dark' : 'light'
  }
  return mode
}

function applyToDocument(effectiveMode: 'dark' | 'light', theme: UITheme): void {
  document.documentElement.setAttribute(MODE_ATTRIBUTE, effectiveMode)
  document.documentElement.setAttribute(THEME_ATTRIBUTE, theme)
}

/**
 * Get CSS custom properties for a theme config.
 * Returns an object of `--koris-*` variable names to values.
 */
export function getCSSVariables(themeConfig: ThemeConfig): Record<string, string> {
  const vars: Record<string, string> = {}
  // Colors
  for (const [key, value] of Object.entries(themeConfig.colors)) {
    const cssKey = key.replace(/([A-Z])/g, '-$1').toLowerCase()
    vars[`--koris-${cssKey}`] = value
  }
  // Border radius
  vars['--koris-border-radius'] = themeConfig.borderRadius
  // Shadows
  vars['--koris-shadow-sm'] = themeConfig.shadows.sm
  vars['--koris-shadow-md'] = themeConfig.shadows.md
  vars['--koris-shadow-lg'] = themeConfig.shadows.lg
  return vars
}

// Module-level singleton state
const mode = ref<ThemeMode>(getPersistedMode())
const theme = ref<UITheme>(getPersistedTheme())

// Apply immediately on load
applyToDocument(resolveEffectiveMode(mode.value, theme.value), theme.value)

// Listen for system preference changes
let mediaQuery: MediaQueryList | null = null
if (typeof window !== 'undefined') {
  mediaQuery = window.matchMedia('(prefers-color-scheme: dark)')
  mediaQuery.addEventListener('change', () => {
    if (mode.value === 'system') {
      applyToDocument(resolveEffectiveMode(mode.value, theme.value), theme.value)
    }
  })
}

// Apply CSS custom properties to :root when theme changes
if (typeof window !== 'undefined') {
  watchEffect(() => {
    const themeInfo = availableThemes.find(t => t.id === theme.value)
    if (!themeInfo) return
    const vars = getCSSVariables(themeInfo.config)
    const root = document.documentElement
    for (const [key, value] of Object.entries(vars)) {
      root.style.setProperty(key, value)
    }
  })
}

/**
 * useTheme composable
 *
 * Two-level theme system:
 * - mode: controls dark/light/system preference
 * - theme: controls the full UI color palette restyle
 *
 * The admin saves these to the server via /api/admin/theme.
 * Both admin and portal read from server on startup to apply the admin-chosen theme.
 */
export function useTheme() {
  const isDark = computed(() => {
    return resolveEffectiveMode(mode.value, theme.value) === 'dark'
  })

  const currentTheme = computed(() => {
    return availableThemes.find((t) => t.id === theme.value) || availableThemes[1]
  })

  const cssVariables = computed(() => {
    const t = currentTheme.value
    return getCSSVariables(t.config)
  })

  function setMode(newMode: ThemeMode): void {
    mode.value = newMode
    try {
      localStorage.setItem(MODE_KEY, newMode)
    } catch {
      // silent
    }
    applyToDocument(resolveEffectiveMode(newMode, theme.value), theme.value)
  }

  function setTheme(newTheme: UITheme): void {
    theme.value = newTheme
    try {
      localStorage.setItem(THEME_KEY, newTheme)
    } catch {
      // silent
    }
    applyToDocument(resolveEffectiveMode(mode.value, newTheme), newTheme)
  }

  /** Legacy toggle for backward compatibility */
  function toggle(): void {
    // Toggle between light and dark preset equivalents
    if (isDark.value) {
      setTheme('default-light')
    } else {
      setTheme('default-dark')
    }
  }

  /** Alias for toggle() */
  function toggleMode(): void {
    toggle()
  }

  return {
    /** Current mode setting */
    mode,
    /** Current UI theme */
    theme,
    /** Current theme info object */
    currentTheme,
    /** Whether the resolved mode is dark */
    isDark,
    /** CSS variables for the current theme */
    cssVariables,
    /** List of available themes with metadata */
    availableThemes,
    /** Set the dark/light/system mode */
    setMode,
    /** Set the UI theme */
    setTheme,
    /** Toggle between dark and light (legacy) */
    toggle,
    /** Toggle between dark and light mode */
    toggleMode,
    /** Get CSS variables for any theme config */
    getCSSVariables,
  }
}
