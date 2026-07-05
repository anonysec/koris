import { ref } from 'vue'
import { useApi } from '@koris/composables/useApi'

export interface WireGuardPeer {
  id: number
  customer_id: number | null
  customer_username?: string
  node_id: number
  node_name?: string
  public_key: string
  allowed_ips: string
  status: 'active' | 'disabled' | 'revoked'
  last_handshake_at: string | null
  rx_bytes: number
  tx_bytes: number
  created_at: string
}

export interface WireGuardPeerFilters {
  node_id?: number
  status?: string
  customer_id?: number
}

export interface CreatePeerPayload {
  node_id: number
  customer_id?: number
}

export interface WireGuardNodeConfig {
  port: number
  network: string
  dns_1: string
  dns_2: string
  mtu: number
  gaming_optimize: boolean
  enabled: boolean
}

export function useWireGuard() {
  const { get, post, del } = useApi()
  const peers = ref<WireGuardPeer[]>([])
  const loading = ref(false)

  async function fetchPeers(filters?: WireGuardPeerFilters) {
    loading.value = true
    try {
      const params = new URLSearchParams()
      if (filters?.node_id) params.set('node_id', String(filters.node_id))
      if (filters?.status) params.set('status', filters.status)
      if (filters?.customer_id) params.set('customer_id', String(filters.customer_id))
      const qs = params.toString()
      const url = `/api/wireguard/peers${qs ? '?' + qs : ''}`
      const res = await get<{ ok: boolean; peers: WireGuardPeer[] }>(url)
      peers.value = res.peers || []
    } finally {
      loading.value = false
    }
  }

  async function createPeer(data: CreatePeerPayload): Promise<WireGuardPeer | null> {
    const res = await post<{ ok: boolean; peer: WireGuardPeer }>('/api/wireguard/peers', data)
    if (res.ok && res.peer) {
      peers.value.push(res.peer)
      return res.peer
    }
    return null
  }

  async function deletePeer(id: number): Promise<boolean> {
    const res = await del<{ ok: boolean }>(`/api/wireguard/peers/${id}`)
    if (res.ok) {
      peers.value = peers.value.filter(p => p.id !== id)
    }
    return res.ok
  }

  function downloadConfig(id: number) {
    window.open(`/api/wireguard/peers/${id}/config`, '_blank')
  }

  async function saveNodeWireGuardConfig(nodeId: number, config: WireGuardNodeConfig): Promise<boolean> {
    const payload = {
      protocol: 'wireguard',
      port: config.port,
      network: config.network,
      enabled: config.enabled,
      mtu: config.mtu,
      extra_json: {
        dns_1: config.dns_1,
        dns_2: config.dns_2,
        gaming_optimize: config.gaming_optimize,
      },
    }
    const res = await post<{ ok: boolean }>(`/api/nodes/vpn-config/${nodeId}`, payload)
    return res.ok
  }

  return {
    peers,
    loading,
    fetchPeers,
    createPeer,
    deletePeer,
    downloadConfig,
    saveNodeWireGuardConfig,
  }
}
