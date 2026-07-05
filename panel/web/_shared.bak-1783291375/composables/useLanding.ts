import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface LandingSettings {
  id: number
  enabled: boolean
  title: string
  description: string | null
  logo_url: string | null
  hero_content: string | null
  updated_at: string
}

export interface UpdateLandingData {
  enabled?: boolean
  title?: string
  description?: string
  logo_url?: string
  hero_content?: string
}

export interface UseLandingReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<LandingSettings | null>
  getSettings: () => Promise<LandingSettings>
  updateSettings: (data: UpdateLandingData) => Promise<LandingSettings>
}

export function useLanding(): UseLandingReturn {
  const { get, patch, loading, error } = useApi()
  const data = ref<LandingSettings | null>(null) as Ref<LandingSettings | null>

  async function getSettings(): Promise<LandingSettings> {
    const result = await get<{ ok: boolean; settings: LandingSettings }>('/api/admin/landing')
    data.value = result.settings
    return result.settings
  }

  async function updateSettings(settingsData: UpdateLandingData): Promise<LandingSettings> {
    const result = await patch<{ ok: boolean; settings: LandingSettings }>('/api/admin/landing', settingsData)
    data.value = result.settings
    return result.settings
  }

  return {
    loading,
    error,
    data,
    getSettings,
    updateSettings,
  }
}
