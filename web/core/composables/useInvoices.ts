import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface Invoice {
  id: number
  invoice_number: string
  customer_id: number
  transaction_id: number | null
  amount: number
  tax: number
  total: number
  currency: string
  plan_name: string | null
  payment_method: string | null
  status: string
  refunded_amount: number
  created_at: string
}

export interface InvoiceFilters {
  status?: string
  customer_id?: number
  from_date?: string
  to_date?: string
}

export interface UseInvoicesReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<Invoice[]>
  listInvoices: (filters?: InvoiceFilters) => Promise<Invoice[]>
  getInvoice: (id: number) => Promise<Invoice>
  refundInvoice: (id: number, amount?: number) => Promise<Invoice>
  downloadInvoice: (id: number) => Promise<Blob>
}

export function useInvoices(): UseInvoicesReturn {
  const { get, post, loading, error } = useApi()
  const data = ref<Invoice[]>([]) as Ref<Invoice[]>

  async function listInvoices(filters?: InvoiceFilters): Promise<Invoice[]> {
    const params = new URLSearchParams()
    if (filters?.status) params.set('status', filters.status)
    if (filters?.customer_id) params.set('customer_id', String(filters.customer_id))
    if (filters?.from_date) params.set('from_date', filters.from_date)
    if (filters?.to_date) params.set('to_date', filters.to_date)
    const query = params.toString() ? `?${params.toString()}` : ''
    const result = await get<{ ok: boolean; invoices: Invoice[] }>(`/api/invoices${query}`)
    data.value = result.invoices
    return result.invoices
  }

  async function getInvoice(id: number): Promise<Invoice> {
    const result = await get<{ ok: boolean; invoice: Invoice }>(`/api/invoices/${id}`)
    return result.invoice
  }

  async function refundInvoice(id: number, amount?: number): Promise<Invoice> {
    const body = amount !== undefined ? { amount } : undefined
    const result = await post<{ ok: boolean; invoice: Invoice }>(`/api/invoices/${id}/refund`, body)
    return result.invoice
  }

  async function downloadInvoice(id: number): Promise<Blob> {
    const response = await fetch(`/api/invoices/${id}/download`, { credentials: 'same-origin' })
    if (!response.ok) throw new Error('Download failed')
    return response.blob()
  }

  return {
    loading,
    error,
    data,
    listInvoices,
    getInvoice,
    refundInvoice,
    downloadInvoice,
  }
}
