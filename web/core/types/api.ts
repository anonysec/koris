/**
 * Shared API response types for KorisPanel
 * Used by both Admin Panel and Customer Portal
 */

export interface ApiResponse<T = unknown> {
  ok: boolean
  error?: string
  data?: T
}

export interface PaginatedResponse<T> extends ApiResponse<T> {
  items: T[]
  total: number
  page: number
  pageSize: number
}
