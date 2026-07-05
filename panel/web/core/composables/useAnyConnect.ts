import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface AnyConnectNode {
  id: number
  node_id: number
  port: number
  cert_path: string | null
  status: string
  created_at: string
  updated_at: string
}

export interface UseAnyConnectReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<AnyConnectNode[]>
  listNodes: () => Promise<AnyConnectNode[]>
  enableAnyConnect: (nodeId: number, port: number) => Promise<AnyConnectNode>
  disableAnyConnect: (id: number) => Promise<void>
  uploadCert: (id: number, cert: string, key: string) => Promise<AnyConnectNode>
}

export function useAnyConnect(): UseAnyConnectReturn {
  const { get, post, del, loading, error } = useApi()
  const data = ref<AnyConnectNode[]>([]) as Ref<AnyConnectNode[]>

  async function listNodes(): Promise<AnyConnectNode[]> {
    const result = await get<{ ok: boolean; nodes: AnyConnectNode[] }>('/api/anyconnect')
    data.value = result.nodes
    return result.nodes
  }

  async function enableAnyConnect(nodeId: number, port: number): Promise<AnyConnectNode> {
    const result = await post<{ ok: boolean; node: AnyConnectNode }>('/api/anyconnect', { node_id: nodeId, port })
    return result.node
  }

  async function disableAnyConnect(id: number): Promise<void> {
    await del(`/api/anyconnect/${id}`)
  }

  async function uploadCert(id: number, cert: string, key: string): Promise<AnyConnectNode> {
    const result = await post<{ ok: boolean; node: AnyConnectNode }>(`/api/anyconnect/${id}/cert`, { cert, key })
    return result.node
  }

  return {
    loading,
    error,
    data,
    listNodes,
    enableAnyConnect,
    disableAnyConnect,
    uploadCert,
  }
}
