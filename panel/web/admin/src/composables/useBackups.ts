import { ref } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'

export interface SkippedNode {
  name: string
  reason: string
}

export interface BackupRecord {
  id: number
  filename: string
  status: 'in_progress' | 'completed' | 'failed'
  type: 'manual' | 'scheduled' | 'pre_restore'
  size_bytes: number | null
  checksum: string | null
  nodes_included: string[] | null
  nodes_skipped: SkippedNode[] | null
  error_message: string | null
  started_at: string
  completed_at: string | null
}

export interface ManifestData {
  version: string
  timestamp: string
  panel_version: string
  database: string
  table_count: number
  total_row_count: number
  nodes_included: string[]
  nodes_skipped: SkippedNode[]
  files: Record<string, { size: number }>
  checksum_algorithm: string
  checksum?: string
}

export interface BackupSettings {
  schedule: string
  retention_count: number
}

export function useBackups() {
  const { get, post, del, put } = useApi()
  const toast = useToast()
  const backups = ref<BackupRecord[]>([])
  const loading = ref(false)
  const settings = ref<BackupSettings>({ schedule: 'daily:02', retention_count: 7 })

  async function fetchBackups() {
    loading.value = true
    try {
      const res = await get<{ ok: boolean; backups: BackupRecord[] }>('/api/admin/backups')
      backups.value = res.backups || []
    } finally {
      loading.value = false
    }
  }

  async function createBackup(): Promise<number | null> {
    const res = await post<{ ok: boolean; backup_id: number }>('/api/admin/backups', {})
    return res.ok ? res.backup_id : null
  }

  function downloadBackup(id: number) {
    window.open(`/api/admin/backups/${id}/download`, '_blank')
  }

  async function verifyBackup(id: number): Promise<boolean> {
    const res = await post<{ ok: boolean; valid: boolean }>(`/api/admin/backups/${id}/verify`, {})
    return res.valid
  }

  async function deleteBackup(id: number): Promise<boolean> {
    const res = await del<{ ok: boolean }>(`/api/admin/backups/${id}`)
    return res.ok
  }

  async function restoreBackup(file: File): Promise<boolean> {
    const formData = new FormData()
    formData.append('file', file)
    const res = await fetch('/api/admin/backups/restore', {
      method: 'POST',
      credentials: 'same-origin',
      body: formData,
    })
    const data = await res.json()
    return data.ok
  }

  async function fetchSettings() {
    const res = await get<{ ok: boolean; schedule: string; retention_count: number }>('/api/admin/backups/settings')
    settings.value = { schedule: res.schedule, retention_count: res.retention_count }
  }

  async function updateSettings(schedule: string, retentionCount: number): Promise<boolean> {
    const res = await put<{ ok: boolean }>('/api/admin/backups/settings', {
      schedule,
      retention_count: retentionCount,
    })
    return res.ok
  }

  async function previewBackup(id: number): Promise<ManifestData | null> {
    try {
      const res = await get<{ ok: boolean; manifest: ManifestData }>(`/api/admin/backups/${id}/preview`)
      if (res.ok && res.manifest) {
        return res.manifest
      }
      return null
    } catch {
      toast.error('Failed to load backup preview')
      return null
    }
  }

  return {
    backups,
    loading,
    settings,
    fetchBackups,
    createBackup,
    downloadBackup,
    verifyBackup,
    deleteBackup,
    restoreBackup,
    fetchSettings,
    updateSettings,
    previewBackup,
  }
}
