// TODO: Admin views (Customers, ResellersView, Dashboard, CustomerDetail,
// Payments) still use .slice(0, 10) for date display. They could benefit from
// migrating to formatDate/formatDateTime for null-safety and locale consistency.

/**
 * Shared date formatting utilities.
 *
 * NOTE: These are plain utility functions, not Vue composables. They do not use
 * reactive state, refs, lifecycle hooks, or any Vue-specific APIs. The `use` prefix
 * in the filename is retained for consistency with the existing import pattern
 * already in place across portal views.
 *
 * formatDate  - date only (year, short month, 2-digit day)
 * formatDateTime - date + time (short month, 2-digit day, hour, minute)
 */

export function formatDate(value: string | null | undefined, fallback = '--'): string {
  if (!value) return fallback
  return new Intl.DateTimeFormat('en', {
    year: 'numeric',
    month: 'short',
    day: '2-digit',
  }).format(new Date(value))
}

export function formatDateTime(value: string | null | undefined, fallback = '--'): string {
  if (!value) return fallback
  return new Intl.DateTimeFormat('en', {
    month: 'short',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  }).format(new Date(value))
}
