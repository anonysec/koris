/**
 * @koris/core — framework layer for Koris apps.
 *
 * This package provides:
 *   - Composables (data fetching, state, i18n, theming logic, formatting)
 *   - Type definitions (API contracts, entities, component props)
 *   - Base CSS (reset, tokens, utilities, transitions, RTL)
 *
 * @koris/core is REQUIRED and NEVER swapped. Visual layer lives in @koris/theme.
 */

// Composables — deep imports also allowed via '@koris/core/composables/<name>'.
export * from './composables/useApi'
export * from './composables/useToast'
export * from './composables/useConfirm'
export * from './composables/useI18n'
export * from './composables/useTheme'
export * from './composables/useDirection'
export * from './composables/useFormatDate'
export * from './composables/useFormValidation'
export * from './composables/useClipboard'
export * from './composables/useFreshData'
export * from './composables/useWebSocket'
export * from './composables/formatBytes'

// Types
export type * from './types'
