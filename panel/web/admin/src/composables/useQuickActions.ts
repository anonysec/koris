import { ref, type Ref } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useCustomersStore } from '../stores/customers'
import type { Customer } from '@koris/types'

/**
 * Quick action types supported on table rows
 */
export type QuickActionType = 'enable' | 'disable' | 'reset-traffic' | 'delete'

/**
 * Per-row loading state tracking
 */
export interface RowActionState {
  loading: boolean
  action: QuickActionType | null
}

/**
 * Composable for managing quick actions on Users table rows.
 *
 * Implements optimistic updates with rollback on failure:
 * 1. Immediately update the row's visual state
 * 2. Execute the API call
 * 3. On failure: revert the row to its pre-action state and show error toast
 * 4. On success: the optimistic state is already correct, no additional update needed
 *
 * Tracks per-row loading states and disables actions during pending calls.
 *
 * Requirements: 13.1, 13.2, 13.4, 13.5, 13.6
 */
export function useQuickActions() {
  const { patch, post, del } = useApi({ showErrorToast: false })
  const toast = useToast()
  const customersStore = useCustomersStore()

  /** Map of userId → loading state for per-row tracking */
  const rowStates: Ref<Map<number, RowActionState>> = ref(new Map())

  /**
   * Check if a row currently has an action in progress.
   */
  function isRowLoading(userId: number): boolean {
    return rowStates.value.get(userId)?.loading ?? false
  }

  /**
   * Get the current action type for a row (if loading).
   */
  function getRowAction(userId: number): QuickActionType | null {
    return rowStates.value.get(userId)?.action ?? null
  }

  /**
   * Set a row's loading state.
   */
  function setRowState(userId: number, loading: boolean, action: QuickActionType | null): void {
    const newMap = new Map(rowStates.value)
    if (loading) {
      newMap.set(userId, { loading, action })
    } else {
      newMap.delete(userId)
    }
    rowStates.value = newMap
  }

  /**
   * Apply an optimistic update to the customer list in the store.
   * Returns the previous state for rollback.
   */
  function applyOptimisticUpdate(
    userId: number,
    updater: (customer: Customer) => Partial<Customer>
  ): Customer | null {
    const index = customersStore.list.findIndex((c) => c.id === userId)
    if (index === -1) return null

    // Save previous state for rollback
    const previous = { ...customersStore.list[index] }

    // Apply optimistic change
    const updates = updater(previous)
    customersStore.list[index] = { ...previous, ...updates } as Customer

    return previous
  }

  /**
   * Revert a row to its previous state (rollback on failure).
   */
  function rollback(userId: number, previous: Customer): void {
    const index = customersStore.list.findIndex((c) => c.id === userId)
    if (index !== -1) {
      customersStore.list[index] = previous
    }
  }

  /**
   * Toggle a user's status between enabled (active) and disabled.
   * Uses PATCH /api/customers/:id with the new status.
   *
   * Requirements: 13.1, 13.2, 13.4, 13.5, 13.6
   */
  async function toggleStatus(userId: number, currentStatus: string): Promise<boolean> {
    if (isRowLoading(userId)) return false

    const newStatus = currentStatus === 'disabled' ? 'active' : 'disabled'
    const actionType: QuickActionType = newStatus === 'disabled' ? 'disable' : 'enable'

    // Set loading state
    setRowState(userId, true, actionType)

    // Apply optimistic update
    const previous = applyOptimisticUpdate(userId, () => ({
      status: newStatus as Customer['status'],
    }))

    try {
      await patch<{ ok: boolean }>(`/api/customers/${userId}`, { status: newStatus })
      return true
    } catch {
      // Rollback on failure
      if (previous) {
        rollback(userId, previous)
      }
      toast.error(`Failed to ${actionType} user`, 5000)
      return false
    } finally {
      setRowState(userId, false, null)
    }
  }

  /**
   * Reset traffic counters for a user.
   * Uses POST /api/customers/:id/traffic-reset
   *
   * Requirements: 13.1, 13.2, 13.4, 13.5, 13.6
   */
  async function resetTraffic(userId: number): Promise<boolean> {
    if (isRowLoading(userId)) return false

    // Set loading state
    setRowState(userId, true, 'reset-traffic')

    // For traffic reset, we don't have a visible field to optimistically update
    // in the table, but we still track loading state per-row
    try {
      await post<{ ok: boolean }>(`/api/customers/${userId}/traffic-reset`, {})
      // Refresh list to show updated usage data
      await customersStore.loadCustomers()
      return true
    } catch {
      toast.error('Failed to reset traffic', 5000)
      return false
    } finally {
      setRowState(userId, false, null)
    }
  }

  /**
   * Delete a user.
   * Uses DELETE /api/customers/:id
   *
   * Note: delete requires confirmation dialog (handled by calling component),
   * this function only executes the API call and handles optimistic removal.
   *
   * Requirements: 13.1, 13.4, 13.5, 13.6
   */
  async function deleteUser(userId: number): Promise<boolean> {
    if (isRowLoading(userId)) return false

    // Set loading state
    setRowState(userId, true, 'delete')

    // Save previous state for potential rollback
    const index = customersStore.list.findIndex((c) => c.id === userId)
    const previous = index !== -1 ? { ...customersStore.list[index] } : null
    const previousIndex = index

    // Optimistically remove from list
    if (index !== -1) {
      customersStore.list.splice(index, 1)
    }

    try {
      await del<{ ok: boolean }>(`/api/customers/${userId}`)
      return true
    } catch {
      // Rollback: re-insert at original position
      if (previous && previousIndex !== -1) {
        customersStore.list.splice(previousIndex, 0, previous as Customer)
      }
      toast.error('Failed to delete user', 5000)
      return false
    } finally {
      setRowState(userId, false, null)
    }
  }

  return {
    rowStates,
    isRowLoading,
    getRowAction,
    toggleStatus,
    resetTraffic,
    deleteUser,
  }
}
