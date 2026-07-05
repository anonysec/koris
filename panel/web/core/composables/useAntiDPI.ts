import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface AntiDPIConfig {
  id: number
  node_id: number
  technique: string
  config_json: string
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface UseAntiDPIReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<AntiDPIConfig[]>
  listConfigs: (nodeId: number) => Promise<AntiDPIConfig[]>
  upsertConfig: (nodeId: number, technique: string, config: Record<string, unknown>) => Promise<AntiDPIConfig>
  deleteConfig: (nodeId: number, technique: string) => Promise<void>
}

export function useAntiDPI(): UseAntiDPIReturn {
  const { get, post, del, loading, error } = useApi()
  const data = ref<AntiDPIConfig[]>([]) as Ref<AntiDPIConfig[]>

  async function listConfigs(nodeId: number): Promise<AntiDPIConfig[]> {
    const result = await get<{ ok: boolean; configs: AntiDPIConfig[] }>(`/api/nodes/${nodeId}/antidpi`)
    data.value = result.configs
    return result.configs
  }

  async function upsertConfig(nodeId: number, technique: string, config: Record<string, unknown>): Promise<AntiDPIConfig> {
    const result = await post<{ ok: boolean; config: AntiDPIConfig }>(`/api/nodes/${nodeId}/antidpi`, {
      technique,
      config_json: JSON.stringify(config),
    })
    return result.config
  }

  async function deleteConfig(nodeId: number, technique: string): Promise<void> {
    await del(`/api/nodes/${nodeId}/antidpi/${encodeURIComponent(technique)}`)
  }

  return {
    loading,
    error,
    data,
    listConfigs,
    upsertConfig,
    deleteConfig,
  }
}
