import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface XrayInbound {
  id: number
  customer_id: number
  node_id: number
  uuid: string
  protocol: string
  transport: string
  security: string
  port: number
  server_name?: string
  public_key?: string
  short_id?: string
  path?: string
  service_name?: string
  status: string
  rx_bytes: number
  tx_bytes: number
  core_name: string
  created_at: string
  updated_at: string
}

export interface InboundFilters {
  node_id?: number
  customer_id?: number
  protocol?: string
  status?: string
}

export interface CreateInboundData {
  customer_id: number
  node_id: number
  protocol: string
  transport: string
  security: string
  port: number
  server_name?: string
  path?: string
  service_name?: string
  core_name?: string
}

export interface UseXrayInboundsReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<XrayInbound[]>
  listInbounds: (filters?: InboundFilters) => Promise<XrayInbound[]>
  createInbound: (data: CreateInboundData) => Promise<XrayInbound>
  updateInbound: (id: number, data: Partial<CreateInboundData>) => Promise<XrayInbound>
  deleteInbound: (id: number) => Promise<void>
  getSubscription: () => Promise<string>
  getLinks: () => Promise<string[]>
}

export function useXrayInbounds(): UseXrayInboundsReturn {
  const { get, post, patch, del, loading, error } = useApi()
  const data = ref<XrayInbound[]>([]) as Ref<XrayInbound[]>

  async function listInbounds(filters?: InboundFilters): Promise<XrayInbound[]> {
    const params = new URLSearchParams()
    if (filters?.node_id) params.set('node_id', String(filters.node_id))
    if (filters?.customer_id) params.set('customer_id', String(filters.customer_id))
    if (filters?.protocol) params.set('protocol', filters.protocol)
    if (filters?.status) params.set('status', filters.status)
    const query = params.toString() ? `?${params.toString()}` : ''
    const result = await get<{ ok: boolean; inbounds: XrayInbound[] }>(`/api/xray/inbounds${query}`)
    data.value = result.inbounds
    return result.inbounds
  }

  async function createInbound(inboundData: CreateInboundData): Promise<XrayInbound> {
    const result = await post<{ ok: boolean; inbound: XrayInbound }>('/api/xray/inbounds', inboundData)
    return result.inbound
  }

  async function updateInbound(id: number, inboundData: Partial<CreateInboundData>): Promise<XrayInbound> {
    const result = await patch<{ ok: boolean; inbound: XrayInbound }>(`/api/xray/inbounds/${id}`, inboundData)
    return result.inbound
  }

  async function deleteInbound(id: number): Promise<void> {
    await del(`/api/xray/inbounds/${id}`)
  }

  async function getSubscription(): Promise<string> {
    const result = await get<{ ok: boolean; subscription: string }>('/api/portal/xray/subscription')
    return result.subscription
  }

  async function getLinks(): Promise<string[]> {
    const result = await get<{ ok: boolean; links: string[] }>('/api/portal/xray/links')
    return result.links
  }

  return {
    loading,
    error,
    data,
    listInbounds,
    createInbound,
    updateInbound,
    deleteInbound,
    getSubscription,
    getLinks,
  }
}
