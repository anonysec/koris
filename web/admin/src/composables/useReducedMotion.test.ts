import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import { defineComponent, type ComputedRef } from 'vue'
import { useReducedMotion } from './useReducedMotion'

/**
 * Unit tests for useReducedMotion composable.
 *
 * **Validates: Requirements 8.6**
 */

function createMockMatchMedia(matches: boolean) {
  const listeners: Array<(e: MediaQueryListEvent) => void> = []

  const mql = {
    matches,
    media: '(prefers-reduced-motion: reduce)',
    addEventListener: vi.fn((event: string, handler: (e: MediaQueryListEvent) => void) => {
      listeners.push(handler)
    }),
    removeEventListener: vi.fn((event: string, handler: (e: MediaQueryListEvent) => void) => {
      const idx = listeners.indexOf(handler)
      if (idx !== -1) listeners.splice(idx, 1)
    }),
    dispatchChange(newMatches: boolean) {
      mql.matches = newMatches
      for (const listener of listeners) {
        listener({ matches: newMatches } as MediaQueryListEvent)
      }
    },
  }

  return mql
}

describe('useReducedMotion', () => {
  let originalMatchMedia: typeof window.matchMedia

  beforeEach(() => {
    originalMatchMedia = window.matchMedia
  })

  afterEach(() => {
    window.matchMedia = originalMatchMedia
  })

  it('returns false when prefers-reduced-motion does not match', () => {
    const mql = createMockMatchMedia(false)
    window.matchMedia = vi.fn(() => mql as unknown as MediaQueryList)

    let result: ComputedRef<boolean> | undefined

    const wrapper = mount(defineComponent({
      setup() {
        result = useReducedMotion()
        return { result }
      },
      template: '<div>{{ result }}</div>',
    }))

    expect(result!.value).toBe(false)
    expect(window.matchMedia).toHaveBeenCalledWith('(prefers-reduced-motion: reduce)')

    wrapper.unmount()
  })

  it('returns true when prefers-reduced-motion matches', () => {
    const mql = createMockMatchMedia(true)
    window.matchMedia = vi.fn(() => mql as unknown as MediaQueryList)

    let result: ComputedRef<boolean> | undefined

    const wrapper = mount(defineComponent({
      setup() {
        result = useReducedMotion()
        return { result }
      },
      template: '<div>{{ result }}</div>',
    }))

    expect(result!.value).toBe(true)

    wrapper.unmount()
  })

  it('reacts to media query changes', async () => {
    const mql = createMockMatchMedia(false)
    window.matchMedia = vi.fn(() => mql as unknown as MediaQueryList)

    let result: ComputedRef<boolean> | undefined

    const wrapper = mount(defineComponent({
      setup() {
        result = useReducedMotion()
        return { result }
      },
      template: '<div>{{ result }}</div>',
    }))

    expect(result!.value).toBe(false)

    // Simulate preference change
    mql.dispatchChange(true)
    expect(result!.value).toBe(true)

    // Simulate changing back
    mql.dispatchChange(false)
    expect(result!.value).toBe(false)

    wrapper.unmount()
  })

  it('cleans up event listener on unmount', () => {
    const mql = createMockMatchMedia(false)
    window.matchMedia = vi.fn(() => mql as unknown as MediaQueryList)

    const wrapper = mount(defineComponent({
      setup() {
        const result = useReducedMotion()
        return { result }
      },
      template: '<div>{{ result }}</div>',
    }))

    expect(mql.addEventListener).toHaveBeenCalledTimes(1)

    wrapper.unmount()

    expect(mql.removeEventListener).toHaveBeenCalledTimes(1)
  })
})
