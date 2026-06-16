import { ref, shallowRef, triggerRef, computed } from 'vue'
import { defineStore } from 'pinia'
import { useWebSocket } from '@koris/composables/useWebSocket'

export interface Stats {
  customers: number
  active_customers: number
  nodes: number
  open_tickets: number
  approved_payments: number
  pending_payments: number
  total_rx_bps: number
  total_tx_bps: number
}

export interface LiveSession {
  id: number
  username: string
  framed_ip: string
  calling_station_id: string
  node_name: string
  session_seconds: number
  input_bytes: number
  output_bytes: number
}

export interface Notification {
  id: string
  type: string
  message: string
  timestamp: string
  read: boolean
}

const HISTORY_MAX = 60

export const useRealtimeStore = defineStore('realtime', () => {
  const stats = ref<Stats>({
    customers: 0,
    active_customers: 0,
    nodes: 0,
    open_tickets: 0,
    approved_payments: 0,
    pending_payments: 0,
    total_rx_bps: 0,
    total_tx_bps: 0,
  })

  const liveSessions = shallowRef<LiveSession[]>([])
  const rxHistory = shallowRef<number[]>([])
  const txHistory = shallowRef<number[]>([])
  const notifications = ref<Notification[]>([])

  let rafId: number | null = null
  let pendingRx: number | null = null
  let pendingTx: number | null = null

  function pushHistory() {
    if (pendingRx !== null && pendingTx !== null) {
      const newRx = [...rxHistory.value, pendingRx].slice(-HISTORY_MAX)
      const newTx = [...txHistory.value, pendingTx].slice(-HISTORY_MAX)
      rxHistory.value = newRx
      txHistory.value = newTx
      triggerRef(rxHistory)
      triggerRef(txHistory)
      pendingRx = null
      pendingTx = null
    }
    rafId = null
  }

  function scheduleHistoryUpdate(rx: number, tx: number) {
    pendingRx = rx
    pendingTx = tx
    if (rafId === null) {
      rafId = requestAnimationFrame(pushHistory)
    }
  }

  function handleMessage(data: any) {
    if (!data || typeof data !== 'object') return

    if (data.type === 'stats' && data.data) {
      const d = data.data
      stats.value = {
        customers: d.customers ?? stats.value.customers,
        active_customers: d.active_customers ?? stats.value.active_customers,
        nodes: d.nodes ?? stats.value.nodes,
        open_tickets: d.open_tickets ?? stats.value.open_tickets,
        approved_payments: d.approved_payments ?? stats.value.approved_payments,
        pending_payments: d.pending_payments ?? stats.value.pending_payments,
        total_rx_bps: d.total_rx_bps ?? stats.value.total_rx_bps,
        total_tx_bps: d.total_tx_bps ?? stats.value.total_tx_bps,
      }
      scheduleHistoryUpdate(d.total_rx_bps || 0, d.total_tx_bps || 0)
    }

    if (data.type === 'sessions' && Array.isArray(data.data)) {
      liveSessions.value = data.data
      triggerRef(liveSessions)
    }

    if (data.type === 'notification' && data.data) {
      notifications.value = [data.data, ...notifications.value].slice(0, 50)
    }
  }

  const wsUrl = typeof window !== 'undefined'
    ? `${window.location.protocol === 'https:' ? 'wss:' : 'ws:'}//${window.location.host}/api/realtime`
    : 'ws://localhost/api/realtime'

  const { connected, connect, disconnect } = useWebSocket({
    url: wsUrl,
    autoConnect: true,
    reconnect: true,
    maxReconnectAttempts: 10,
    onMessage: handleMessage,
  })

  const notificationCount = computed(() => notifications.value.filter(n => !n.read).length)

  function markAllRead() {
    notifications.value = notifications.value.map(n => ({ ...n, read: true }))
  }

  return {
    stats,
    liveSessions,
    rxHistory,
    txHistory,
    notifications,
    connected,
    notificationCount,
    connect,
    disconnect,
    markAllRead,
  }
})
