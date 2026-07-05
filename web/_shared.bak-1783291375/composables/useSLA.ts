import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface SLAConfigEntry {
  id: number
  priority: string
  response_minutes: number
}

export interface SLAStats {
  total_tickets: number
  met_sla: number
  breached_sla: number
  compliance_percent: number
  avg_response_minutes: number
}

export interface UseSLAReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<SLAConfigEntry[]>
  getConfig: () => Promise<SLAConfigEntry[]>
  updateConfig: (entries: Array<{ priority: string; response_minutes: number }>) => Promise<SLAConfigEntry[]>
  getStats: () => Promise<SLAStats>
}

export function useSLA(): UseSLAReturn {
  const { get, patch, loading, error } = useApi()
  const data = ref<SLAConfigEntry[]>([]) as Ref<SLAConfigEntry[]>

  async function getConfig(): Promise<SLAConfigEntry[]> {
    const result = await get<{ ok: boolean; config: SLAConfigEntry[] }>('/api/sla/config')
    data.value = result.config
    return result.config
  }

  async function updateConfig(entries: Array<{ priority: string; response_minutes: number }>): Promise<SLAConfigEntry[]> {
    const result = await patch<{ ok: boolean; config: SLAConfigEntry[] }>('/api/sla/config', { entries })
    data.value = result.config
    return result.config
  }

  async function getStats(): Promise<SLAStats> {
    const result = await get<{ ok: boolean; stats: SLAStats }>('/api/sla/stats')
    return result.stats
  }

  return {
    loading,
    error,
    data,
    getConfig,
    updateConfig,
    getStats,
  }
}
