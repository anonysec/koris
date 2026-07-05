import { computed, ref, onMounted, onUnmounted, type ComputedRef } from 'vue'

/**
 * Composable that detects the user's `prefers-reduced-motion` system preference.
 * Returns a reactive computed boolean that updates when the preference changes.
 *
 * Usage:
 *   const reducedMotion = useReducedMotion()
 *   const duration = computed(() => reducedMotion.value ? 0 : 280)
 *
 * All animation durations should be multiplied by `reducedMotion.value ? 0 : 1`
 * to instantly apply state changes when reduced motion is enabled (Requirement 8.6).
 */
export function useReducedMotion(): ComputedRef<boolean> {
  const matches = ref(false)
  let mediaQuery: MediaQueryList | null = null

  function handleChange(e: MediaQueryListEvent) {
    matches.value = e.matches
  }

  onMounted(() => {
    mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
    matches.value = mediaQuery.matches
    mediaQuery.addEventListener('change', handleChange)
  })

  onUnmounted(() => {
    if (mediaQuery) {
      mediaQuery.removeEventListener('change', handleChange)
    }
  })

  return computed(() => matches.value)
}
