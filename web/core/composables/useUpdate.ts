import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface ReleaseInfo {
  version: string
  changelog: string
  url: string
  checksum_sha256: string
}

export interface UpdateResult {
  ok: boolean
  message?: string
}

export interface UseUpdateReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<ReleaseInfo | null>
  checkUpdate: () => Promise<ReleaseInfo>
  applyUpdate: (version: string) => Promise<UpdateResult>
  rollbackUpdate: () => Promise<UpdateResult>
  bulkUpdateNodes: (nodeIds: number[]) => Promise<UpdateResult>
}

export function useUpdate(): UseUpdateReturn {
  const { get, post, loading, error } = useApi({ baseUrl: '/api/admin' })
  const data = ref<ReleaseInfo | null>(null) as Ref<ReleaseInfo | null>

  async function checkUpdate(): Promise<ReleaseInfo> {
    const result = await get<ReleaseInfo>('/update/check')
    data.value = result
    return result
  }

  async function applyUpdate(version: string): Promise<UpdateResult> {
    return post<UpdateResult>('/update/apply', { version })
  }

  async function rollbackUpdate(): Promise<UpdateResult> {
    return post<UpdateResult>('/update/rollback')
  }

  async function bulkUpdateNodes(nodeIds: number[]): Promise<UpdateResult> {
    return post<UpdateResult>('/nodes/update/bulk', { node_ids: nodeIds })
  }

  return {
    loading,
    error,
    data,
    checkUpdate,
    applyUpdate,
    rollbackUpdate,
    bulkUpdateNodes,
  }
}
