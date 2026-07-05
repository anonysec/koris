import { ref } from 'vue'
import { useApi } from '@koris/composables/useApi'

export type StatMetric =
  | 'bandwidth'
  | 'user_growth'
  | 'revenue'
  | 'node_performance'
  | 'protocol_usage'

export interface StatisticsQuery {
  metric: StatMetric
  period?: 'hour' | 'day' | 'week' | 'month'
  from?: string // ISO date YYYY-MM-DD
  to?: string
  nodeId?: number
}

export interface StatisticsResponse {
  ok: boolean
  metric: string
  series?: any[]
  summary?: Record<string, any>
  protocols?: any[]
  nodes?: any[]
}

export function useStatistics() {
  const loading = ref(false)
  const error = ref<string | null>(null)

  const { get } = useApi()

  async function fetchMetric(query: StatisticsQuery): Promise<StatisticsResponse | null> {
    loading.value = true
    error.value = null

    try {
      const params = new URLSearchParams()
      params.set('metric', query.metric)
      if (query.period) params.set('period', query.period)
      if (query.from) params.set('from', query.from)
      if (query.to) params.set('to', query.to)
      if (query.nodeId) params.set('nodeId', String(query.nodeId))

      const data = await get(`/api/admin/statistics?${params.toString()}`)
      return data as StatisticsResponse
    } catch (e: any) {
      error.value = e.message || 'Failed to fetch statistics'
      return null
    } finally {
      loading.value = false
    }
  }

  async function fetchMultiple(queries: StatisticsQuery[]): Promise<StatisticsResponse[]> {
    const results = await Promise.all(queries.map(q => fetchMetric(q)))
    return results.filter(Boolean) as StatisticsResponse[]
  }

  function exportCSV(data: any[], filename: string): void {
    if (!data || data.length === 0) return

    const headers = Object.keys(data[0])
    const csvRows = [
      headers.join(','),
      ...data.map(row =>
        headers.map(h => {
          const val = row[h]
          // Escape commas and quotes in values
          if (typeof val === 'string' && (val.includes(',') || val.includes('"'))) {
            return `"${val.replace(/"/g, '""')}"`
          }
          return String(val ?? '')
        }).join(',')
      )
    ]

    const blob = new Blob([csvRows.join('\n')], { type: 'text/csv;charset=utf-8;' })
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = `${filename}.csv`
    link.click()
    URL.revokeObjectURL(url)
  }

  return {
    loading,
    error,
    fetchMetric,
    fetchMultiple,
    exportCSV,
  }
}
