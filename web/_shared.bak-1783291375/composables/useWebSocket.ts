import { ref, onUnmounted } from 'vue'
import type { Ref } from 'vue'

/**
 * Options for configuring the useWebSocket composable.
 */
export interface UseWebSocketOptions {
  /** WebSocket URL to connect to (ws:// or wss://) */
  url: string
  /** Automatically connect on composable initialization (default: false) */
  autoConnect?: boolean
  /** Enable automatic reconnection on disconnect (default: true) */
  reconnect?: boolean
  /** Maximum number of reconnection attempts before giving up (default: 10) */
  maxReconnectAttempts?: number
  /** Callback invoked when a message is received */
  onMessage?: (data: any) => void
  /** Callback invoked when connection is established */
  onConnect?: () => void
  /** Callback invoked when connection is lost */
  onDisconnect?: () => void
}

/**
 * Return type for the useWebSocket composable.
 */
export interface UseWebSocketReturn {
  /** Reactive ref indicating whether the WebSocket is currently connected */
  connected: Ref<boolean>
  /** Manually establish the WebSocket connection */
  connect(): void
  /** Manually disconnect and cancel any pending reconnection */
  disconnect(): void
  /** Send data through the WebSocket (JSON-serialized) */
  send(data: any): void
}

/**
 * Calculates the reconnection delay using exponential backoff with jitter.
 *
 * Formula: min(baseDelay * 2^attempt + random(0, baseDelay), maxDelay)
 *
 * @param attempt - Current reconnection attempt number (0-indexed)
 * @param baseDelay - Base delay in milliseconds (default: 1000)
 * @param maxDelay - Maximum delay cap in milliseconds (default: 30000)
 * @returns Delay in milliseconds, or null if max attempts exceeded
 */
export function calculateReconnectDelay(
  attempt: number,
  maxAttempts: number,
  baseDelay: number = 1000,
  maxDelay: number = 30000
): number | null {
  if (attempt >= maxAttempts) {
    return null
  }

  const exponentialDelay = baseDelay * Math.pow(2, attempt)
  const jitter = Math.random() * baseDelay
  const delay = Math.min(exponentialDelay + jitter, maxDelay)

  return delay
}

/**
 * Vue composable for WebSocket connections with auto-connect and exponential backoff reconnection.
 *
 * Features:
 * - Auto-connect on initialization when `autoConnect: true`
 * - Exponential backoff reconnection with jitter (baseDelay * 2^attempt + jitter, capped at maxDelay)
 * - Stops reconnection when maxReconnectAttempts exceeded
 * - Cleans up connection and timers on component unmount
 *
 * @param options - Configuration options for the WebSocket connection
 * @returns Object with connected state, connect/disconnect/send methods
 *
 * Validates: Requirements 12.1, 12.2, 12.3, 12.4, 12.5
 */
export function useWebSocket(options: UseWebSocketOptions): UseWebSocketReturn {
  const {
    url,
    autoConnect = false,
    reconnect = true,
    maxReconnectAttempts = 10,
    onMessage,
    onConnect,
    onDisconnect,
  } = options

  const connected = ref(false)

  const BASE_DELAY = 1000
  const MAX_DELAY = 30000

  let socket: WebSocket | null = null
  let reconnectAttempt = 0
  let reconnectTimer: ReturnType<typeof setTimeout> | null = null
  let intentionalDisconnect = false

  /**
   * Clears any pending reconnection timer.
   */
  function clearReconnectTimer(): void {
    if (reconnectTimer !== null) {
      clearTimeout(reconnectTimer)
      reconnectTimer = null
    }
  }

  /**
   * Schedules a reconnection attempt using exponential backoff.
   */
  function scheduleReconnect(): void {
    if (!reconnect || intentionalDisconnect) {
      return
    }

    const delay = calculateReconnectDelay(
      reconnectAttempt,
      maxReconnectAttempts,
      BASE_DELAY,
      MAX_DELAY
    )

    if (delay === null) {
      // Max attempts exceeded — stop trying
      return
    }

    reconnectTimer = setTimeout(() => {
      reconnectTimer = null
      reconnectAttempt++
      connect()
    }, delay)
  }

  /**
   * Establishes a WebSocket connection.
   * If already connected, closes the existing connection first.
   */
  function connect(): void {
    // Clean up any existing connection
    if (socket) {
      socket.onopen = null
      socket.onclose = null
      socket.onmessage = null
      socket.onerror = null
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
        socket.close()
      }
    }

    intentionalDisconnect = false

    socket = new WebSocket(url)

    socket.onopen = () => {
      connected.value = true
      reconnectAttempt = 0
      clearReconnectTimer()
      onConnect?.()
    }

    socket.onclose = () => {
      connected.value = false
      onDisconnect?.()

      if (!intentionalDisconnect) {
        scheduleReconnect()
      }
    }

    socket.onerror = () => {
      // Error will be followed by onclose, which handles reconnection
    }

    socket.onmessage = (event: MessageEvent) => {
      if (onMessage) {
        try {
          const data = JSON.parse(event.data)
          onMessage(data)
        } catch {
          // If not JSON, pass raw data
          onMessage(event.data)
        }
      }
    }
  }

  /**
   * Disconnects the WebSocket, cancels all pending reconnection timers,
   * and sets connected to false.
   */
  function disconnect(): void {
    intentionalDisconnect = true
    clearReconnectTimer()
    reconnectAttempt = 0

    if (socket) {
      socket.onopen = null
      socket.onclose = null
      socket.onmessage = null
      socket.onerror = null
      if (socket.readyState === WebSocket.OPEN || socket.readyState === WebSocket.CONNECTING) {
        socket.close()
      }
      socket = null
    }

    connected.value = false
  }

  /**
   * Sends data through the WebSocket connection.
   * Data is JSON-serialized before sending.
   * No-op if not connected.
   */
  function send(data: any): void {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.send(JSON.stringify(data))
    }
  }

  // Auto-connect if configured
  if (autoConnect) {
    connect()
  }

  // Clean up on component unmount to prevent memory leaks
  onUnmounted(() => {
    disconnect()
  })

  return {
    connected,
    connect,
    disconnect,
    send,
  }
}
