import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface CorePlugin {
  id: number
  name: string
  version: string
  download_url: string
  checksum_sha256: string
  protocols_json: string
  config_template: string | null
  created_at: string
  updated_at: string
}

export interface RegisterCoreData {
  name: string
  version: string
  download_url: string
  checksum_sha256: string
  protocols_json: string
  config_template?: string
}

export interface UseCorePluginsReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<CorePlugin[]>
  listCores: () => Promise<CorePlugin[]>
  registerCore: (data: RegisterCoreData) => Promise<CorePlugin>
  installCore: (nodeId: number, name: string, version: string) => Promise<void>
  updateCore: (nodeId: number, name: string, version: string) => Promise<void>
  removeCore: (nodeId: number, name: string) => Promise<void>
}

export function useCorePlugins(): UseCorePluginsReturn {
  const { get, post, del, loading, error } = useApi()
  const data = ref<CorePlugin[]>([]) as Ref<CorePlugin[]>

  async function listCores(): Promise<CorePlugin[]> {
    const result = await get<{ ok: boolean; cores: CorePlugin[] }>('/api/cores')
    data.value = result.cores
    return result.cores
  }

  async function registerCore(coreData: RegisterCoreData): Promise<CorePlugin> {
    const result = await post<{ ok: boolean; core: CorePlugin }>('/api/cores', coreData)
    return result.core
  }

  async function installCore(nodeId: number, name: string, version: string): Promise<void> {
    await post(`/api/nodes/${nodeId}/cores/install`, { name, version })
  }

  async function updateCore(nodeId: number, name: string, version: string): Promise<void> {
    await post(`/api/nodes/${nodeId}/cores/update`, { name, version })
  }

  async function removeCore(nodeId: number, name: string): Promise<void> {
    await del(`/api/nodes/${nodeId}/cores/${encodeURIComponent(name)}`)
  }

  return {
    loading,
    error,
    data,
    listCores,
    registerCore,
    installCore,
    updateCore,
    removeCore,
  }
}
