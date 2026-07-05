import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface FilterPreset {
  id: number
  admin_username: string
  name: string
  filters_json: string
  created_at: string
}

export interface UseFilterPresetsReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<FilterPreset[]>
  listPresets: () => Promise<FilterPreset[]>
  savePreset: (name: string, filters: Record<string, unknown>) => Promise<FilterPreset>
  deletePreset: (id: number) => Promise<void>
}

export function useFilterPresets(): UseFilterPresetsReturn {
  const { get, post, del, loading, error } = useApi()
  const data = ref<FilterPreset[]>([]) as Ref<FilterPreset[]>

  async function listPresets(): Promise<FilterPreset[]> {
    const result = await get<{ ok: boolean; presets: FilterPreset[] }>('/api/filter-presets')
    data.value = result.presets
    return result.presets
  }

  async function savePreset(name: string, filters: Record<string, unknown>): Promise<FilterPreset> {
    const result = await post<{ ok: boolean; preset: FilterPreset }>('/api/filter-presets', {
      name,
      filters_json: JSON.stringify(filters),
    })
    return result.preset
  }

  async function deletePreset(id: number): Promise<void> {
    await del(`/api/filter-presets/${id}`)
  }

  return {
    loading,
    error,
    data,
    listPresets,
    savePreset,
    deletePreset,
  }
}
