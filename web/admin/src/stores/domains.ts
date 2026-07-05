import { ref } from 'vue'
import { defineStore } from 'pinia'
import { useApi } from '@koris/composables/useApi'

// ─── TypeScript Interfaces ──────────────────────────────────────────────────

export interface VpnDomain {
  id: number
  name: string
  ip_address: string
  status: 'active' | 'blocked' | 'retired'
  created_at: string
  updated_at: string
  binding_count: number
  cert_status: 'valid' | 'expiring_soon' | 'expired' | 'none'
}

export interface ProtocolBinding {
  id: number
  node_id: number
  protocol: string
  domain_id: number
  position: number
  domain_name: string
  domain_ip: string
  domain_status: string
  warning?: boolean
}

export interface DomainIPHistory {
  id: number
  domain_id: number
  previous_ip: string
  new_ip: string
  admin_username: string
  rotated_at: string
}

export interface MTProtoSecretInfo {
  secret: string
  enabled: boolean
  connections: number
  connection_limit: number
}

// ─── API Response Types ─────────────────────────────────────────────────────

interface DomainsListResponse {
  ok: boolean
  domains: VpnDomain[]
}

interface DomainResponse {
  ok: boolean
  domain: VpnDomain
}

interface DomainMutationResponse {
  ok: boolean
}

interface HistoryListResponse {
  ok: boolean
  history: DomainIPHistory[]
}

interface BindingsListResponse {
  ok: boolean
  bindings: ProtocolBinding[]
}

interface BindingMutationResponse {
  ok: boolean
}

interface MTProtoSecretResponse {
  ok: boolean
  secret: string
  enabled: boolean
  connections: number
  connection_limit: number
}

// ─── Payload Types ──────────────────────────────────────────────────────────

export interface CreateDomainPayload {
  name: string
  ip_address: string
}

export interface UpdateDomainPayload {
  ip_address?: string
  status?: 'active' | 'blocked' | 'retired'
}

export interface RotateIPPayload {
  new_ip: string
}

export interface CreateBindingPayload {
  protocol: string
  domain_id: number
  position: number
}

export interface ReorderBindingsPayload {
  binding_ids: number[]
}

/**
 * Domains management store — handles domain CRUD, IP rotation,
 * protocol bindings, and MTProto secret management.
 *
 * All API calls go through the useApi composable.
 *
 * Requirements: 7.1, 7.2, 7.3, 7.4, 7.5, 7.6
 */
export const useDomainsStore = defineStore('domains', () => {
  // ─── State ────────────────────────────────────────────────────────────────
  const domains = ref<VpnDomain[]>([])
  const bindings = ref<ProtocolBinding[]>([])
  const history = ref<DomainIPHistory[]>([])
  const loading = ref(false)

  // ─── API composable ───────────────────────────────────────────────────────
  const { get, post, patch, del, error } = useApi()

  // ─── Domain CRUD ──────────────────────────────────────────────────────────

  /**
   * Fetch all domains with binding count and cert status.
   * GET /api/admin/domains → { ok, domains }
   */
  async function fetchDomains(): Promise<void> {
    loading.value = true
    try {
      const res = await get<DomainsListResponse>('/api/admin/domains')
      domains.value = res.domains || []
    } catch {
      // Preserve existing data on error
    } finally {
      loading.value = false
    }
  }

  /**
   * Create a new domain.
   * POST /api/admin/domains → { ok, domain }
   */
  async function createDomain(payload: CreateDomainPayload): Promise<VpnDomain | null> {
    loading.value = true
    try {
      const res = await post<DomainResponse>('/api/admin/domains', payload)
      await fetchDomains()
      return res.domain
    } catch {
      return null
    } finally {
      loading.value = false
    }
  }

  /**
   * Update an existing domain (IP or status).
   * PATCH /api/admin/domains/{id} → { ok }
   */
  async function updateDomain(id: number, payload: UpdateDomainPayload): Promise<boolean> {
    loading.value = true
    try {
      await patch<DomainMutationResponse>(`/api/admin/domains/${id}`, payload)
      await fetchDomains()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Delete a domain (fails if active bindings exist).
   * DELETE /api/admin/domains/{id} → { ok }
   */
  async function deleteDomain(id: number): Promise<boolean> {
    loading.value = true
    try {
      await del<DomainMutationResponse>(`/api/admin/domains/${id}`)
      await fetchDomains()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  // ─── IP Rotation ──────────────────────────────────────────────────────────

  /**
   * Rotate a domain's IP address with audit trail.
   * POST /api/admin/domains/{id}/rotate-ip → { ok }
   */
  async function rotateIP(id: number, payload: RotateIPPayload): Promise<boolean> {
    loading.value = true
    try {
      await post<DomainMutationResponse>(`/api/admin/domains/${id}/rotate-ip`, payload)
      await fetchDomains()
      return true
    } catch {
      return false
    } finally {
      loading.value = false
    }
  }

  /**
   * Fetch IP rotation history for a domain.
   * GET /api/admin/domains/{id}/history → { ok, history }
   */
  async function fetchHistory(domainId: number): Promise<DomainIPHistory[]> {
    try {
      const res = await get<HistoryListResponse>(`/api/admin/domains/${domainId}/history`)
      history.value = res.history || []
      return history.value
    } catch {
      return []
    }
  }

  // ─── Protocol Bindings ────────────────────────────────────────────────────

  /**
   * Fetch all protocol bindings for a node.
   * GET /api/admin/nodes/{nodeId}/bindings → { ok, bindings }
   */
  async function fetchBindings(nodeId: number): Promise<ProtocolBinding[]> {
    try {
      const res = await get<BindingsListResponse>(`/api/admin/nodes/${nodeId}/bindings`)
      bindings.value = res.bindings || []
      return bindings.value
    } catch {
      return []
    }
  }

  /**
   * Create a protocol binding for a node.
   * POST /api/admin/nodes/{nodeId}/bindings → { ok }
   */
  async function createBinding(nodeId: number, payload: CreateBindingPayload): Promise<boolean> {
    try {
      await post<BindingMutationResponse>(`/api/admin/nodes/${nodeId}/bindings`, payload)
      await fetchBindings(nodeId)
      return true
    } catch {
      return false
    }
  }

  /**
   * Delete a protocol binding from a node.
   * DELETE /api/admin/nodes/{nodeId}/bindings/{id} → { ok }
   */
  async function deleteBinding(nodeId: number, bindingId: number): Promise<boolean> {
    try {
      await del<BindingMutationResponse>(`/api/admin/nodes/${nodeId}/bindings/${bindingId}`)
      await fetchBindings(nodeId)
      return true
    } catch {
      return false
    }
  }

  /**
   * Reorder protocol bindings for a node (drag-and-drop).
   * PATCH /api/admin/nodes/{nodeId}/bindings/reorder → { ok }
   */
  async function reorderBindings(nodeId: number, payload: ReorderBindingsPayload): Promise<boolean> {
    try {
      await patch<BindingMutationResponse>(`/api/admin/nodes/${nodeId}/bindings/reorder`, payload)
      await fetchBindings(nodeId)
      return true
    } catch {
      return false
    }
  }

  // ─── MTProto Secrets ──────────────────────────────────────────────────────

  /**
   * Fetch a customer's MTProto secret and connection info.
   * GET /api/admin/customers/{id}/mtproto-secret → { ok, secret, enabled, connections, connection_limit }
   */
  async function fetchMTProtoSecret(customerId: number): Promise<MTProtoSecretInfo | null> {
    try {
      const res = await get<MTProtoSecretResponse>(`/api/admin/customers/${customerId}/mtproto-secret`)
      return {
        secret: res.secret,
        enabled: res.enabled,
        connections: res.connections,
        connection_limit: res.connection_limit,
      }
    } catch {
      return null
    }
  }

  /**
   * Regenerate a customer's MTProto secret and push to knode.
   * POST /api/admin/customers/{id}/mtproto-secret/regenerate → { ok, secret, ... }
   */
  async function regenerateSecret(customerId: number): Promise<MTProtoSecretInfo | null> {
    try {
      const res = await post<MTProtoSecretResponse>(`/api/admin/customers/${customerId}/mtproto-secret/regenerate`)
      return {
        secret: res.secret,
        enabled: res.enabled,
        connections: res.connections,
        connection_limit: res.connection_limit,
      }
    } catch {
      return null
    }
  }

  // ─── Expose ───────────────────────────────────────────────────────────────
  return {
    // State
    domains,
    bindings,
    history,
    loading,
    error,

    // Domain CRUD
    fetchDomains,
    createDomain,
    updateDomain,
    deleteDomain,

    // IP Rotation
    rotateIP,
    fetchHistory,

    // Protocol Bindings
    fetchBindings,
    createBinding,
    deleteBinding,
    reorderBindings,

    // MTProto Secrets
    fetchMTProtoSecret,
    regenerateSecret,
  }
})
