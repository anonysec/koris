import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { useApi } from '@koris/composables/useApi'
import { useAuthStore } from '@/stores/auth'
import router from '@/router'
import type { Plan } from '@koris/types'

/**
 * Plan creation payload matching POST /api/plans
 */
export interface CreatePlanPayload {
  name: string
  data_gb: number
  speed_mbps: number
  duration_days: number
  price: number
  is_active: boolean
  sort_order: number
}

/**
 * Plan update payload matching PATCH /api/plans/:id
 */
export interface UpdatePlanPayload {
  name?: string
  data_gb?: number
  speed_mbps?: number
  duration_days?: number
  price?: number
  is_active?: boolean
  sort_order?: number
}

/**
 * API response types matching backend endpoints
 */
interface PlansListResponse {
  ok: boolean
  plans: Plan[]
}

interface PlanMutationResponse {
  ok: boolean
  id?: number
}

/**
 * Plans management store (Pinia Composition API style)
 *
 * Manages subscription plan CRUD operations for the admin panel.
 * Uses useApi composable for all API interactions with loading state management.
 *
 * Requirements: 3.1, 3.3, 22.4
 */
export const usePlansStore = defineStore('plans', () => {
  // ─── State ────────────────────────────────────────────────────────────────
  const list = ref<Plan[]>([])
  const loading = ref(false)

  // ─── API composable ───────────────────────────────────────────────────────
  const { get, post, patch, del, error } = useApi({
    onUnauthorized: () => {
      // Only redirect if auth was fully initialized and user WAS authenticated
      // (i.e. the session actually expired). This prevents race conditions on
      // initial page load where stores fire before auth state is propagated.
      const auth = useAuthStore()
      if (auth.initialized && auth.isAuthenticated) {
        auth.user = null
        auth.isAuthenticated = false
        router.push({ name: 'login' })
      }
    },
  })

  // ─── Computed ─────────────────────────────────────────────────────────────

  /**
   * Active plans filtered by is_active === true
   */
  const activePlans = computed(() => list.value.filter((plan) => plan.is_active === true))

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Load all plans from the API.
   * GET /api/plans → { ok: boolean, plans: Plan[] }
   *
   * Sets loading = true before request, false after (success or failure).
   * On error, preserves existing data (Requirement 3.4).
   */
  async function loadPlans(): Promise<void> {
    loading.value = true
    try {
      const res = await get<PlansListResponse>('/api/plans')
      list.value = res.plans || []
    } catch {
      // Preserve existing data on error (Requirement 3.4)
    } finally {
      loading.value = false
    }
  }

  /**
   * Create a new plan.
   * POST /api/plans with { name, data_gb, speed_mbps, duration_days, price, is_active, sort_order }
   *
   * On success, reloads the plans list.
   * On error, preserves existing data.
   */
  async function createPlan(payload: CreatePlanPayload): Promise<boolean> {
    loading.value = true
    try {
      await post<PlanMutationResponse>('/api/plans', payload)
      await loadPlans()
      return true
    } catch {
      // Preserve existing data on error (Requirement 3.4)
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Update an existing plan.
   * PATCH /api/plans/:id with partial plan fields
   *
   * On success, reloads the plans list.
   * On error, preserves existing data.
   */
  async function updatePlan(id: number, payload: UpdatePlanPayload): Promise<boolean> {
    loading.value = true
    try {
      await patch<PlanMutationResponse>(`/api/plans/${id}`, payload)
      await loadPlans()
      return true
    } catch {
      // Preserve existing data on error (Requirement 3.4)
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Delete (archive/deactivate) a plan.
   * DELETE /api/plans/:id
   *
   * On success, reloads the plans list.
   * On error, preserves existing data.
   */
  async function deletePlan(id: number): Promise<boolean> {
    loading.value = true
    try {
      await del<PlanMutationResponse>(`/api/plans/${id}`)
      await loadPlans()
      return true
    } catch {
      // Preserve existing data on error (Requirement 3.4)
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

    // Computed
    activePlans,

    // API state (from useApi)
    error,

    // Actions
    loadPlans,
    createPlan,
    updatePlan,
    deletePlan,
  }
})
