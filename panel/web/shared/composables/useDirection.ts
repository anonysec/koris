import { computed, watchEffect } from 'vue'
import { useI18n } from './useI18n'
import type { Locale } from './useI18n'

/** Locales that require right-to-left layout */
const RTL_LOCALES: Locale[] = ['fa']

/**
 * Composable for automatic RTL/LTR direction handling.
 * Auto-detects direction from the current locale (FA = RTL, others = LTR).
 * Sets `dir` and `lang` attributes on the `<html>` element reactively.
 *
 * @example
 * ```ts
 * const { direction, isRTL, locale } = useDirection()
 * // When locale is 'fa', isRTL = true, direction = 'rtl'
 * // HTML element gets dir="rtl" lang="fa"
 * ```
 */
export function useDirection() {
  const { locale } = useI18n()

  const isRTL = computed(() => RTL_LOCALES.includes(locale.value))
  const direction = computed(() => (isRTL.value ? 'rtl' : 'ltr'))

  // Update DOM attributes reactively when locale changes
  watchEffect(() => {
    document.documentElement.dir = direction.value
    document.documentElement.lang = locale.value
  })

  return { direction, isRTL, locale }
}
