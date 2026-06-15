import { ref } from 'vue'

export interface Toast {
  id: number
  message: string
  type: 'success' | 'error' | 'info' | 'warning'
  duration: number
}

const toasts = ref<Toast[]>([])
let nextId = 0

function showToast(
  message: string,
  type: 'success' | 'error' | 'info' | 'warning' = 'info',
  duration = 4000
): void {
  const id = nextId++
  toasts.value.push({ id, message, type, duration })
}

function removeToast(id: number): void {
  const index = toasts.value.findIndex((t) => t.id === id)
  if (index !== -1) {
    toasts.value.splice(index, 1)
  }
}

function success(message: string, duration = 4000): void {
  showToast(message, 'success', duration)
}

function error(message: string, duration = 4000): void {
  showToast(message, 'error', duration)
}

function info(message: string, duration = 4000): void {
  showToast(message, 'info', duration)
}

function warning(message: string, duration = 4000): void {
  showToast(message, 'warning', duration)
}

export function useToast() {
  return {
    toasts,
    showToast,
    removeToast,
    success,
    error,
    info,
    warning,
  }
}
