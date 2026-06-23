import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface MTProtoProxy {
  id: number
  node_id: number
  port: number
  secret: string
  status: string
  connections: number
  rx_bytes: number
  tx_bytes: number
  created_at: string
  updated_at: string
}

export interface ShareLink {
  link: string
}

export interface UseMTProtoReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<MTProtoProxy[]>
  listProxies: () => Promise<MTProtoProxy[]>
  enableProxy: (nodeId: number, port: number) => Promise<MTProtoProxy>
  disableProxy: (id: number) => Promise<void>
  rotateSecret: (id: number) => Promise<MTProtoProxy>
  getShareLink: (id: number) => Promise<string>
}

export function useMTProto(): UseMTProtoReturn {
  const { get, post, del, loading, error } = useApi()
  const data = ref<MTProtoProxy[]>([]) as Ref<MTProtoProxy[]>

  async function listProxies(): Promise<MTProtoProxy[]> {
    const result = await get<{ ok: boolean; proxies: MTProtoProxy[] }>('/api/mtproto')
    data.value = result.proxies
    return result.proxies
  }

  async function enableProxy(nodeId: number, port: number): Promise<MTProtoProxy> {
    const result = await post<{ ok: boolean; proxy: MTProtoProxy }>('/api/mtproto', { node_id: nodeId, port })
    return result.proxy
  }

  async function disableProxy(id: number): Promise<void> {
    await del(`/api/mtproto/${id}`)
  }

  async function rotateSecret(id: number): Promise<MTProtoProxy> {
    const result = await post<{ ok: boolean; proxy: MTProtoProxy }>(`/api/mtproto/${id}/rotate`)
    return result.proxy
  }

  async function getShareLink(id: number): Promise<string> {
    const result = await get<{ ok: boolean; link: string }>(`/api/mtproto/${id}/link`)
    return result.link
  }

  return {
    loading,
    error,
    data,
    listProxies,
    enableProxy,
    disableProxy,
    rotateSecret,
    getShareLink,
  }
}
