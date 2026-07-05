import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface Payout {
  id: number
  reseller_username: string
  amount: number
  status: string
  payment_details: string
  admin_note: string | null
  requested_at: string
  processed_at: string | null
  processed_by: string | null
}

export interface UseResellerPayoutsReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<Payout[]>
  listPayouts: (status?: string) => Promise<Payout[]>
  requestPayout: (amount: number, details: string) => Promise<Payout>
  approvePayout: (id: number, note?: string) => Promise<Payout>
  rejectPayout: (id: number, note?: string) => Promise<Payout>
}

export function useResellerPayouts(): UseResellerPayoutsReturn {
  const { get, post, patch, loading, error } = useApi()
  const data = ref<Payout[]>([]) as Ref<Payout[]>

  async function listPayouts(status?: string): Promise<Payout[]> {
    const query = status ? `?status=${encodeURIComponent(status)}` : ''
    const result = await get<{ ok: boolean; payouts: Payout[] }>(`/api/admin/payouts${query}`)
    data.value = result.payouts
    return result.payouts
  }

  async function requestPayout(amount: number, details: string): Promise<Payout> {
    const result = await post<{ ok: boolean; payout: Payout }>('/api/reseller/payouts', { amount, payment_details: details })
    return result.payout
  }

  async function approvePayout(id: number, note?: string): Promise<Payout> {
    const result = await patch<{ ok: boolean; payout: Payout }>(`/api/admin/payouts/${id}`, { action: 'approve', note })
    return result.payout
  }

  async function rejectPayout(id: number, note?: string): Promise<Payout> {
    const result = await patch<{ ok: boolean; payout: Payout }>(`/api/admin/payouts/${id}`, { action: 'reject', note })
    return result.payout
  }

  return {
    loading,
    error,
    data,
    listPayouts,
    requestPayout,
    approvePayout,
    rejectPayout,
  }
}
