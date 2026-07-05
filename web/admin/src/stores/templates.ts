import { ref } from 'vue'
import { defineStore } from 'pinia'
import { useApi } from '@koris/composables/useApi'

/**
 * RADIUS attribute definition used in user templates
 */
export interface RadiusAttribute {
  attribute: string
  op: string
  value: string
}

/**
 * User template entity matching the backend user_templates table
 */
export interface UserTemplate {
  id: number
  name: string
  plan_id: number | null
  status: string
  connection_limit: number
  radius_checks: RadiusAttribute[]
  radius_replies: RadiusAttribute[]
  created_by: string
  deleted_at: string | null
  created_at: string
  updated_at: string
}

/**
 * Template creation payload matching POST /api/templates
 */
export interface CreateTemplatePayload {
  name: string
  plan_id?: number | null
  status?: string
  connection_limit?: number
  radius_checks?: RadiusAttribute[]
  radius_replies?: RadiusAttribute[]
}

/**
 * Template update payload matching PATCH /api/templates/{id}
 */
export interface UpdateTemplatePayload {
  name?: string
  plan_id?: number | null
  status?: string
  connection_limit?: number
  radius_checks?: RadiusAttribute[]
  radius_replies?: RadiusAttribute[]
}

/**
 * API response types matching backend endpoints
 */
interface TemplatesListResponse {
  ok: boolean
  templates: UserTemplate[]
}

interface TemplateMutationResponse {
  ok: boolean
  template?: UserTemplate
}

/**
 * Templates management store (Pinia Composition API style)
 *
 * Manages user template CRUD operations for the admin panel.
 * Uses useApi composable for all API interactions with loading state management.
 *
 * Requirements: 1.2, 1.3, 1.4
 */
export const useTemplatesStore = defineStore('templates', () => {
  // ─── State ────────────────────────────────────────────────────────────────
  const list = ref<UserTemplate[]>([])
  const loading = ref(false)

  // ─── API composable ───────────────────────────────────────────────────────
  // No onUnauthorized handler — the router guard handles auth redirects.
  // This prevents race conditions where a 401 during initial data load
  // would clear auth state and cause a redirect loop after login.
  const { get, post, patch, del, error } = useApi()

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Load all non-deleted templates from the API.
   * GET /api/templates → { ok: boolean, templates: UserTemplate[] }
   *
   * Sets loading = true before request, false after (success or failure).
   * On error, preserves existing data.
   */
  async function loadTemplates(): Promise<void> {
    loading.value = true
    try {
      const res = await get<TemplatesListResponse>('/api/templates')
      list.value = res.templates || []
    } catch {
      // Preserve existing data on error
    } finally {
      loading.value = false
    }
  }

  /**
   * Create a new user template.
   * POST /api/templates with { name, plan_id, status, connection_limit, radius_checks, radius_replies }
   *
   * On success, reloads the templates list (Requirement 1.2).
   * On error, preserves existing data.
   */
  async function createTemplate(payload: CreateTemplatePayload): Promise<boolean> {
    loading.value = true
    try {
      await post<TemplateMutationResponse>('/api/templates', payload)
      await loadTemplates()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Update an existing user template.
   * PATCH /api/templates/{id} with partial template fields
   *
   * On success, reloads the templates list (Requirement 1.3).
   * On error, preserves existing data.
   */
  async function updateTemplate(id: number, payload: UpdateTemplatePayload): Promise<boolean> {
    loading.value = true
    try {
      await patch<TemplateMutationResponse>(`/api/templates/${id}`, payload)
      await loadTemplates()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Soft-delete a user template.
   * DELETE /api/templates/{id}
   *
   * On success, reloads the templates list (Requirement 1.4).
   * Customers previously created from this template remain unaffected.
   * On error, preserves existing data.
   */
  async function deleteTemplate(id: number): Promise<boolean> {
    loading.value = true
    try {
      await del<TemplateMutationResponse>(`/api/templates/${id}`)
      await loadTemplates()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  // ─── Expose ───────────────────────────────────────────────────────────────
  return {
    // State
    list,
    loading,

    // API state (from useApi)
    error,

    // Actions
    loadTemplates,
    createTemplate,
    updateTemplate,
    deleteTemplate,
  }
})
