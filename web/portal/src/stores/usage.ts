import { ref, computed } from 'vue'
import { defineStore } from 'pinia'
import { useApi } from '@koris/composables/useApi'
import { useWebSocket } from '@koris/composables/useWebSocket'

/**
 * A single usage session
 */
export interface UsageSession {
  id: number
  start_time: string
  stop_time: string
  session_seconds: number
  input_bytes: number
  output_bytes: number
  total_bytes: number
  framed_ip: string
  online: boolean
}

/**
 * Bandwidth usage summary from the backend
 */
export interface UsageSummary {
  online: boolean
  active_sessions: number
  connection_limit: number
  total_input_bytes: number
  total_output_bytes: number
  total_usage_bytes: number
  max_data_bytes: number
  remaining_bytes?: number
  last_connected_at: string
  last_disconnected_at: string
  sessions: UsageSession[]
}

/**
 * API response types
 */
interface UsageResponse {
  ok: boolean
  usage: UsageSummary
}

/**
 * Portal usage store (Pinia Composition API style)
 *
 * Manages bandwidth usage data and session history.
 * Uses useApi composable for all API interactions.
 *
 * Requirements: 3.2, 3.3, 3.4, 23.4
 */
export const useUsageStore = defineStore('portal-usage', () => {
  // ─── State ────────────────────────────────────────────────────────────────
  const usage = ref<UsageSummary | null>(null)
  const loading = ref(false)

  // ─── API composable ───────────────────────────────────────────────────────
  // No onUnauthorized handler — the portal auth store and router guard handle
  // auth redirects. This prevents race conditions where a 401 during initial
  // data load would cause a redirect loop after login.
  const { get, error } = useApi()

  // ─── Computed ─────────────────────────────────────────────────────────────
  const isOnline = computed(() => usage.value?.online ?? false)

  const activeSessions = computed(() => usage.value?.active_sessions ?? 0)

  const connectionLimit = computed(() => usage.value?.connection_limit ?? 0)

  const totalUsageBytes = computed(() => usage.value?.total_usage_bytes ?? 0)

  const maxDataBytes = computed(() => usage.value?.max_data_bytes ?? 0)

  const usagePercent = computed(() => {
    if (!usage.value?.max_data_bytes) return 0
    return Math.min(100, Math.round((usage.value.total_usage_bytes / usage.value.max_data_bytes) * 100))
  })

  const remainingBytes = computed(() => {
    if (!usage.value?.max_data_bytes) return Infinity
    return Math.max(0, usage.value.max_data_bytes - usage.value.total_usage_bytes)
  })

  const sessions = computed(() => usage.value?.sessions ?? [])

  /**
   * Chart data points derived from sessions for bandwidth visualization
   */
  const bandwidthChartData = computed(() => {
    if (!usage.value?.sessions?.length) return []
    return usage.value.sessions
      .slice()
      .reverse()
      .map((s) => ({
        label: new Date(s.start_time).toLocaleDateString('en', { month: 'short', day: 'numeric' }),
        value: Math.round(s.total_bytes / (1024 * 1024)), // MB
      }))
  })

  // ─── Live stream (WebSocket) ──────────────────────────────────────────────
  // Keep usage fresh without polling: the backend streams the same UsageSummary
  // the REST endpoint returns, every 5s. Same-origin, so the session cookie is
  // sent automatically. The store is a singleton, so this persists for the app
  // session (matching the admin realtime store pattern).
  function handleStreamMessage(data: any): void {
    if (data && data.type === 'usage' && data.data) {
      usage.value = data.data
    }
  }

  const wsUrl = typeof window !== 'undefined'
    ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/portal/usage/ws`
    : 'ws://localhost/api/portal/usage/ws'

  const { connected: streamConnected, connect: connectStream, disconnect: disconnectStream } = useWebSocket({
    url: wsUrl,
    autoConnect: true,
    reconnect: true,
    maxReconnectAttempts: 10,
    onMessage: handleStreamMessage,
  })

  // ─── Actions ──────────────────────────────────────────────────────────────

  /**
   * Load usage data from the backend.
   * GET /api/portal/usage → { ok, usage }
   */
  async function loadUsage(): Promise<void> {
    loading.value = true
    try {
      const res = await get<UsageResponse>('/api/portal/usage')
      usage.value = res.usage
    } catch {
      // Preserve existing data on error (Requirement 3.4)
    } finally {
      loading.value = false
    }
  }

  // ─── Expose ───────────────────────────────────────────────────────────────
  return {
    // State
    usage,
    loading,

    // API state
    error,

    // Live stream
    streamConnected,
    connectStream,
    disconnectStream,

    // Computed
    isOnline,
    activeSessions,
    connectionLimit,
    totalUsageBytes,
    maxDataBytes,
    usagePercent,
    remainingBytes,
    sessions,
    bandwidthChartData,

    // Actions
    loadUsage,
  }
})
