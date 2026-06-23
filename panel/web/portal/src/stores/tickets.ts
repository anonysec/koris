import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { useApi } from '@koris/composables/useApi'

/**
 * Ticket entity from the new support system
 */
export interface SupportTicket {
  id: number
  customer_id: number
  subject: string
  category: string
  priority: string
  status: 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'
  assigned_to: string
  satisfaction_rating: number | null
  created_at: string
  updated_at: string
  resolved_at: string | null
  closed_at: string | null
}

/**
 * Ticket message within a conversation thread
 */
export interface TicketMessage {
  id: number
  ticket_id: number
  sender_type: 'admin' | 'customer'
  sender_name: string
  body: string
  is_internal: boolean
  created_at: string
}

/**
 * Extended ticket with its message thread
 */
export interface TicketDetail extends SupportTicket {
  messages: TicketMessage[]
}

/**
 * API response types
 */
interface TicketsListResponse {
  ok: boolean
  tickets: SupportTicket[]
  total: number
}

interface TicketDetailResponse {
  ok: boolean
  ticket: SupportTicket
  messages: TicketMessage[]
}

interface TicketCreateResponse {
  ok: boolean
  ticket: SupportTicket
}

interface TicketReplyResponse {
  ok: boolean
  message: TicketMessage
}

interface TicketRateResponse {
  ok: boolean
}

/**
 * Portal tickets store (Pinia Composition API style)
 *
 * Manages support ticket list, detail view, creation, replies, and satisfaction rating.
 * Uses the /api/customer/tickets endpoints (new support system).
 * Uses useApi composable for all API interactions.
 *
 * Requirements: 3.2, 3.3, 3.4, 23.5
 */
export const usePortalTicketsStore = defineStore('portal-tickets', () => {
  // ─── State ────────────────────────────────────────────────────────────────
  const list = ref<SupportTicket[]>([])
  const detail = ref<TicketDetail | null>(null)
  const loading = ref(false)

  // ─── API composable ───────────────────────────────────────────────────────
  const { get, post, error } = useApi()

  // ─── Computed ─────────────────────────────────────────────────────────────
  const openTickets = computed(() =>
    list.value.filter((t) => t.status === 'open' || t.status === 'in_progress' || t.status === 'waiting')
  )

  const closedTickets = computed(() =>
    list.value.filter((t) => t.status === 'closed' || t.status === 'resolved')
  )

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Load all customer tickets.
   * GET /api/customer/tickets → { ok, tickets, total }
   */
  async function loadTickets(): Promise<void> {
    loading.value = true
    try {
      const res = await get<TicketsListResponse>('/api/customer/tickets')
      list.value = res.tickets || []
    } catch {
      // Preserve existing data on error
    } finally {
      loading.value = false
    }
  }

  /**
   * Load a single ticket's detail with messages.
   * Uses GET /api/customer/tickets (list) to find the ticket, then fetches detail.
   * Note: The customer API returns ticket + messages together via adminGetTicket pattern.
   * We use a workaround: get ticket from list and load messages via reply endpoint check.
   */
  async function loadTicketDetail(id: number): Promise<void> {
    loading.value = true
    try {
      // The customer API doesn't have a dedicated GET /api/customer/tickets/:id endpoint
      // but we can use the portal endpoint which still works for backwards compat
      const res = await get<TicketDetailResponse>(`/api/portal/tickets/${id}`)
      detail.value = {
        ...(res.ticket as any),
        messages: (res.messages || (res as any).ticket?.messages || []).map((m: any) => ({
          ...m,
          body: m.body || m.message || '',
        })),
      }
    } catch {
      // Preserve existing detail on error
    } finally {
      loading.value = false
    }
  }

  /**
   * Create a new support ticket.
   * POST /api/customer/tickets → { ok, ticket }
   */
  async function createTicket(params: {
    subject: string
    category?: string
    priority?: string
    body: string
  }): Promise<number | null> {
    loading.value = true
    try {
      const res = await post<TicketCreateResponse>('/api/customer/tickets', {
        subject: params.subject,
        category: params.category || 'general',
        priority: params.priority || 'medium',
        body: params.body,
      })
      await loadTickets()
      return res.ticket?.id || null
    } catch {
      return null
    } finally {
      loading.value = false
    }
  }

  /**
   * Reply to an existing ticket.
   * POST /api/customer/tickets/:id/reply → { ok, message }
   */
  async function replyToTicket(id: number, body: string): Promise<boolean> {
    loading.value = true
    try {
      await post<TicketReplyResponse>(`/api/customer/tickets/${id}/reply`, { body })
      await loadTicketDetail(id)
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Rate a resolved/closed ticket (1-5 stars).
   * POST /api/customer/tickets/:id/rate → { ok }
   */
  async function rateTicket(id: number, rating: number): Promise<boolean> {
    loading.value = true
    try {
      await post<TicketRateResponse>(`/api/customer/tickets/${id}/rate`, { rating })
      // Update the detail and list
      if (detail.value && detail.value.id === id) {
        detail.value.satisfaction_rating = rating
      }
      const ticketInList = list.value.find((t) => t.id === id)
      if (ticketInList) {
        ticketInList.satisfaction_rating = rating
      }
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Close a ticket.
   * POST /api/portal/tickets/:id/close → { ok }
   */
  async function closeTicket(id: number): Promise<boolean> {
    loading.value = true
    try {
      await post<{ ok: boolean }>(`/api/portal/tickets/${id}/close`)
      await loadTicketDetail(id)
      await loadTickets()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Clear the currently selected ticket detail.
   */
  function clearDetail(): void {
    detail.value = null
  }

  // ─── Expose ───────────────────────────────────────────────────────────────
  return {
    // State
    list,
    detail,
    loading,

    // API state
    error,

    // Computed
    openTickets,
    closedTickets,

    // Actions
    loadTickets,
    loadTicketDetail,
    createTicket,
    replyToTicket,
    rateTicket,
    closeTicket,
    clearDetail,
  }
})
