import { ref, type Ref } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'

export interface WalletAdjustResponse {
  ok: boolean
  balance?: number
  error?: string
}

/**
 * Composable for wallet top-up and deduction actions.
 * Calls POST /api/wallets/:username/adjust with positive (top-up) or negative (deduct) amounts.
 *
 * @param username - Reactive ref containing the target user's username
 * @returns loading state, error ref, topUp and deduct functions
 */
export function useWalletActions(username: Ref<string>) {
  const { post } = useApi({ showErrorToast: false })
  const toast = useToast()
  const loading = ref(false)
  const error = ref<string | null>(null)

  /**
   * Top up the user's wallet with a positive amount.
   * @param amount - Positive numeric amount to add
   * @param description - Description for the transaction
   * @returns true on success, false on failure
   */
  async function topUp(amount: number, description: string): Promise<boolean> {
    loading.value = true
    error.value = null

    try {
      const res = await post<WalletAdjustResponse>(
        `/api/wallets/${encodeURIComponent(username.value)}/adjust`,
        { amount, description }
      )

      if (res.ok) {
        toast.success('Wallet topped up successfully')
        return true
      }

      error.value = res.error || 'Top-up failed'
      toast.error(error.value)
      return false
    } catch (e: any) {
      error.value = e.message || 'Failed to top up wallet'
      toast.error(error.value!)
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Deduct from the user's wallet with a negative amount.
   * @param amount - Positive numeric amount to deduct (will be sent as negative)
   * @param description - Description for the transaction
   * @returns true on success, false on failure
   */
  async function deduct(amount: number, description: string): Promise<boolean> {
    loading.value = true
    error.value = null

    try {
      const res = await post<WalletAdjustResponse>(
        `/api/wallets/${encodeURIComponent(username.value)}/adjust`,
        { amount: -amount, description }
      )

      if (res.ok) {
        toast.success('Wallet deduction applied successfully')
        return true
      }

      error.value = res.error || 'Deduction failed'
      toast.error(error.value)
      return false
    } catch (e: any) {
      error.value = e.message || 'Failed to deduct from wallet'
      toast.error(error.value!)
      return false
    } finally {
      loading.value = false
    }
  }

  return {
    loading,
    error,
    topUp,
    deduct,
  }
}
