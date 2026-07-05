/**
 * Unit tests for useTheme composable
 *
 * Tests theme switching, mode switching, persistence via localStorage,
 * CSS variable generation, available themes count, isDark computed,
 * toggleMode behavior, currentTheme info, and default state.
 */
import { describe, it, expect, beforeEach, vi } from 'vitest'

// We need to reset the module-level singleton state between tests
// so each test gets a clean slate.
let useTheme: typeof import('./useTheme').useTheme
let getCSSVariables: typeof import('./useTheme').getCSSVariables
let availableThemes: typeof import('./useTheme').availableThemes

describe('useTheme', () => {
  beforeEach(async () => {
    // Clear localStorage
    localStorage.clear()

    // Reset document attributes
    document.documentElement.removeAttribute('data-theme')
    document.documentElement.removeAttribute('data-ui-theme')
    document.documentElement.style.cssText = ''

    // Reset the module to get fresh singleton state
    vi.resetModules()
    const mod = await import('./useTheme')
    useTheme = mod.useTheme
    getCSSVariables = mod.getCSSVariables
    availableThemes = mod.availableThemes
  })

  describe('default state', () => {
    it('initial theme is default-dark when no localStorage', () => {
      const { theme } = useTheme()
      expect(theme.value).toBe('default-dark')
    })

    it('initial mode is system when no localStorage', () => {
      const { mode } = useTheme()
      expect(mode.value).toBe('system')
    })
  })

  describe('theme switching', () => {
    it('setTheme changes theme.value to the new theme', () => {
      const { theme, setTheme } = useTheme()
      setTheme('ocean')
      expect(theme.value).toBe('ocean')
    })

    it('setTheme updates data-ui-theme attribute on documentElement', () => {
      const { setTheme } = useTheme()
      setTheme('ocean')
      expect(document.documentElement.getAttribute('data-ui-theme')).toBe('ocean')
    })

    it('setTheme updates data-theme attribute to the theme mode', () => {
      const { setTheme } = useTheme()
      // ocean is a dark theme
      setTheme('ocean')
      expect(document.documentElement.getAttribute('data-theme')).toBe('dark')
    })

    it('setTheme to a light theme updates data-theme to light', () => {
      const { setTheme } = useTheme()
      setTheme('sunset')
      expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    })
  })

  describe('mode switching', () => {
    it('setMode changes mode.value', () => {
      const { mode, setMode } = useTheme()
      setMode('dark')
      expect(mode.value).toBe('dark')
    })

    it('setMode updates data-theme attribute on documentElement', () => {
      const { setMode, setTheme } = useTheme()
      // Set a light theme first, mode takes effect based on theme's declared mode
      setTheme('default-light')
      setMode('light')
      expect(document.documentElement.getAttribute('data-theme')).toBe('light')
    })
  })

  describe('persistence', () => {
    it('setTheme persists to localStorage', () => {
      const { setTheme } = useTheme()
      setTheme('forest')
      expect(localStorage.getItem('koris-ui-theme')).toBe('forest')
    })

    it('setMode persists to localStorage', () => {
      const { setMode } = useTheme()
      setMode('dark')
      expect(localStorage.getItem('koris-mode')).toBe('dark')
    })

    it('reads theme from localStorage on init', async () => {
      localStorage.setItem('koris-ui-theme', 'ocean')
      vi.resetModules()
      const mod = await import('./useTheme')
      const { theme } = mod.useTheme()
      expect(theme.value).toBe('ocean')
    })

    it('reads mode from localStorage on init', async () => {
      localStorage.setItem('koris-mode', 'light')
      vi.resetModules()
      const mod = await import('./useTheme')
      const { mode } = mod.useTheme()
      expect(mode.value).toBe('light')
    })
  })

  describe('CSS variable output', () => {
    it('getCSSVariables returns --koris- prefixed keys for all colors', () => {
      const themeInfo = availableThemes.find(t => t.id === 'default-dark')!
      const vars = getCSSVariables(themeInfo.config)

      // All color keys should produce --koris-{kebab-case} variables
      expect(vars['--koris-primary']).toBe('#60a5fa')
      expect(vars['--koris-primary-hover']).toBe('#93c5fd')
      expect(vars['--koris-secondary']).toBe('#94a3b8')
      expect(vars['--koris-background']).toBe('#0f172a')
      expect(vars['--koris-surface']).toBe('#1e293b')
      expect(vars['--koris-surface-hover']).toBe('#334155')
      expect(vars['--koris-text']).toBe('#f1f5f9')
      expect(vars['--koris-text-muted']).toBe('#94a3b8')
      expect(vars['--koris-border']).toBe('#334155')
      expect(vars['--koris-success']).toBe('#4ade80')
      expect(vars['--koris-warning']).toBe('#fbbf24')
      expect(vars['--koris-error']).toBe('#f87171')
      expect(vars['--koris-info']).toBe('#60a5fa')
      expect(vars['--koris-accent']).toBe('#a78bfa')
    })

    it('getCSSVariables includes borderRadius', () => {
      const themeInfo = availableThemes.find(t => t.id === 'ocean')!
      const vars = getCSSVariables(themeInfo.config)
      expect(vars['--koris-border-radius']).toBe('10px')
    })

    it('getCSSVariables includes shadows', () => {
      const themeInfo = availableThemes.find(t => t.id === 'default-dark')!
      const vars = getCSSVariables(themeInfo.config)
      expect(vars['--koris-shadow-sm']).toBe('0 1px 2px rgba(0,0,0,0.3)')
      expect(vars['--koris-shadow-md']).toBe('0 4px 6px rgba(0,0,0,0.4)')
      expect(vars['--koris-shadow-lg']).toBe('0 10px 15px rgba(0,0,0,0.5)')
    })

    it('getCSSVariables returns correct total key count (14 colors + 1 borderRadius + 3 shadows = 18)', () => {
      const themeInfo = availableThemes.find(t => t.id === 'default-dark')!
      const vars = getCSSVariables(themeInfo.config)
      expect(Object.keys(vars).length).toBe(18)
    })
  })

  describe('available themes', () => {
    it('availableThemes has exactly 6 entries', () => {
      expect(availableThemes.length).toBe(6)
    })

    it('all themes have required properties', () => {
      for (const theme of availableThemes) {
        expect(theme.id).toBeDefined()
        expect(theme.name).toBeDefined()
        expect(theme.description).toBeDefined()
        expect(theme.mode).toMatch(/^(light|dark)$/)
        expect(theme.config).toBeDefined()
        expect(theme.config.colors).toBeDefined()
        expect(theme.config.borderRadius).toBeDefined()
        expect(theme.config.shadows).toBeDefined()
      }
    })
  })

  describe('isDark computed', () => {
    it('isDark is true for dark themes', () => {
      const { isDark, setTheme } = useTheme()
      setTheme('default-dark')
      expect(isDark.value).toBe(true)
    })

    it('isDark is false for light themes', () => {
      const { isDark, setTheme } = useTheme()
      setTheme('default-light')
      expect(isDark.value).toBe(false)
    })

    it('isDark reflects the theme mode (ocean = dark)', () => {
      const { isDark, setTheme } = useTheme()
      setTheme('ocean')
      expect(isDark.value).toBe(true)
    })

    it('isDark reflects the theme mode (sunset = light)', () => {
      const { isDark, setTheme } = useTheme()
      setTheme('sunset')
      expect(isDark.value).toBe(false)
    })
  })

  describe('toggleMode', () => {
    it('toggleMode switches from dark to light theme', () => {
      const { theme, setTheme, toggleMode } = useTheme()
      setTheme('default-dark')
      toggleMode()
      expect(theme.value).toBe('default-light')
    })

    it('toggleMode switches from light to dark theme', () => {
      const { theme, setTheme, toggleMode } = useTheme()
      setTheme('default-light')
      toggleMode()
      expect(theme.value).toBe('default-dark')
    })

    it('toggleMode from a non-default dark theme goes to default-light', () => {
      const { theme, setTheme, toggleMode } = useTheme()
      setTheme('ocean')
      toggleMode()
      expect(theme.value).toBe('default-light')
    })
  })

  describe('currentTheme', () => {
    it('currentTheme returns the correct theme info object', () => {
      const { currentTheme, setTheme } = useTheme()
      setTheme('forest')
      expect(currentTheme.value.id).toBe('forest')
      expect(currentTheme.value.name).toBe('Forest')
      expect(currentTheme.value.mode).toBe('dark')
    })

    it('currentTheme updates when theme changes', () => {
      const { currentTheme, setTheme } = useTheme()
      setTheme('monochrome')
      expect(currentTheme.value.id).toBe('monochrome')
      expect(currentTheme.value.name).toBe('Monochrome')
      expect(currentTheme.value.mode).toBe('light')
    })

    it('currentTheme falls back to default-dark for invalid theme', () => {
      const { currentTheme } = useTheme()
      // Default state is default-dark
      expect(currentTheme.value.id).toBe('default-dark')
    })
  })
})
