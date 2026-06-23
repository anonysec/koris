import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { useApi } from '@koris/composables/useApi'

/**
 * Ticket entity matching the v2 support system backend
 */
export interface Ticket {
  id: number
  customer_id: number
  username: string
  subject: string
  category: 'billing' | 'technical' | 'general'
  priority: 'low' | 'medium' | 'high'
  status: 'open' | 'in_progress' | 'waiting' | 'resolved' | 'closed'
  assigned_to: string
  satisfaction_rating?: number
  created_at: string
  updated_at: string
  resolved_at?: string
}

/**
 * A single message within a ticket conversation thread
 */
export interface TicketMessage {
  id: number
  ticket_id: number
  sender_type: 'admin' | 'customer'
  sender_id: string
  sender_name: string
  body: string
  message?: string // legacy field alias
  is_internal: boolean
  created_at: string
}

/**
 * Extended ticket with its message thread
 */
export interface TicketDetail extends Ticket {
  messages: TicketMessage[]
}

/**
 * Canned response for quick replies
 */
export interface CannedResponse {
  id: number
  title: string
  body: string
  category: string
  created_at: string
}

/**
 * Filters for ticket listing
 */
export interface TicketFilters {
  status?: string
  category?: string
  priority?: string
  assigned_to?: string
  page?: number
  limit?: number
}

/**
 * API response types
 */
interface TicketsListResponse {
  ok: boolean
  tickets: Ticket[]
  total?: number
}

interface TicketDetailResponse {
  ok: boolean
  ticket: TicketDetail
}

interface CannedResponsesResponse {
  ok: boolean
  canned_responses: CannedResponse[]
}

interface TicketCreateResponse {
  ok: boolean
  id: number
}

interface GenericResponse {
  ok: boolean
}

/**
 * Admin tickets store (Pinia Composition API style)
 *
 * Manages support ticket state including listing, filtering, detail view,
 * replies, internal notes, status changes, and canned responses.
 */
export const useTicketsStore = defineStore('tickets', () => {
  // ─── State ────────────────────────────────────────────────────────────────
  const list = ref<Ticket[]>([])
  const detail = ref<TicketDetail | null>(null)
  const cannedResponses = ref<CannedResponse[]>([])
  const loading = ref(false)
  const total = ref(0)
  const filters = ref<TicketFilters>({})

  // ─── API composable ───────────────────────────────────────────────────────
  const { get, post, put, error } = useApi()

  // ─── Computed ─────────────────────────────────────────────────────────────

  /** Tickets grouped by status for kanban view */
  const ticketsByStatus = computed(() => {
    const groups: Record<string, Ticket[]> = {
      open: [],
      in_progress: [],
      waiting: [],
      resolved: [],
      closed: [],
    }
    for (const ticket of list.value) {
      if (groups[ticket.status]) {
        groups[ticket.status].push(ticket)
      }
    }
    return groups
  })

  /** All tickets with open-like statuses (legacy compat) */
  const openTickets = computed(() =>
    list.value.filter((t) => t.status === 'open' || t.status === 'in_progress' || t.status === 'waiting')
  )

  /** All closed/resolved tickets (legacy compat) */
  const closedTickets = computed(() =>
    list.value.filter((t) => t.status === 'closed' || t.status === 'resolved')
  )

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Load tickets with optional filters from admin endpoint.
   * GET /api/admin/tickets?status=&category=&priority=&assigned_to=&page=&limit=
   */
  async function loadTickets(params?: TicketFilters): Promise<void> {
    loading.value = true
    try {
      const query = new URLSearchParams()
      const f = params || filters.value
      if (f.status) query.set('status', f.status)
      if (f.category) query.set('category', f.category)
      if (f.priority) query.set('priority', f.priority)
      if (f.assigned_to) query.set('assigned_to', f.assigned_to)
      if (f.page) query.set('page', String(f.page))
      if (f.limit) query.set('limit', String(f.limit))

      const qs = query.toString()
      const url = `/api/admin/tickets${qs ? '?' + qs : ''}`
      const res = await get<TicketsListResponse>(url)
      list.value = res.tickets || []
      total.value = res.total ?? list.value.length
    } catch {
      // Preserve existing data on error
    } finally {
      loading.value = false
    }
  }

  /**
   * Load a single ticket's detail including messages.
   * GET /api/admin/tickets/:id
   */
  async function loadTicketDetail(id: number): Promise<void> {
    loading.value = true
    try {
      const res = await get<TicketDetailResponse>(`/api/admin/tickets/${id}`)
      detail.value = res.ticket
    } catch {
      // Preserve existing detail on error
    } finally {
      loading.value = false
    }
  }

  /**
   * Reply to a ticket (public or internal note).
   * POST /api/admin/tickets/:id/reply { body, is_internal }
   */
  async function replyToTicket(id: number, body: string, isInternal: boolean = false): Promise<boolean> {
    try {
      await post<GenericResponse>(`/api/admin/tickets/${id}/reply`, {
        body,
        is_internal: isInternal,
      })
      await loadTicketDetail(id)
      return true
    } catch {
      return false
    }
  }

  /**
   * Update ticket fields (status, priority, category, assigned_to).
   * PUT /api/admin/tickets/:id
   */
  async function updateTicket(id: number, updates: Partial<Pick<Ticket, 'status' | 'priority' | 'category' | 'assigned_to'>>): Promise<boolean> {
    try {
      await put<GenericResponse>(`/api/admin/tickets/${id}`, updates)
      // Refresh detail if we're viewing this ticket
      if (detail.value?.id === id) {
        await loadTicketDetail(id)
      }
      return true
    } catch {
      return false
    }
  }

  /**
   * Close a ticket by setting status to 'closed'.
   */
  async function closeTicket(id: number): Promise<boolean> {
    return updateTicket(id, { status: 'closed' })
  }

  /**
   * Reopen a closed ticket by setting status to 'open'.
   */
  async function openTicket(id: number): Promise<boolean> {
    return updateTicket(id, { status: 'open' })
  }

  /**
   * Create an admin-initiated ticket.
   * POST /api/admin/tickets
   */
  async function createTicket(params: {
    username: string
    subject: string
    priority: string
    message: string
    category?: string
  }): Promise<number | null> {
    loading.value = true
    try {
      const res = await post<TicketCreateResponse>('/api/admin/tickets', params)
      await loadTickets()
      return res.id
    } catch {
      return null
    } finally {
      loading.value = false
    }
  }

  /**
   * Load canned responses.
   * GET /api/admin/canned-responses
   */
  async function loadCannedResponses(): Promise<void> {
    try {
      const res = await get<CannedResponsesResponse>('/api/admin/canned-responses')
      cannedResponses.value = res.canned_responses || []
    } catch {
      // Non-critical, ignore
    }
  }

  // ─── Expose ───────────────────────────────────────────────────────────────
  return {
    // State
    list,
    detail,
    cannedResponses,
    loading,
    total,
    filters,

    // API state
    error,

    // Computed
    ticketsByStatus,
    openTickets,
    closedTickets,

    // Actions
    loadTickets,
    loadTicketDetail,
    replyToTicket,
    updateTicket,
    closeTicket,
    openTicket,
    createTicket,
    loadCannedResponses,
  }
})
