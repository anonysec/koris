<template>
  <Transition name="k-toast">
    <div
      v-if="visible"
      :class="['k-toast', `k-toast--${type}`]"
      role="alert"
      aria-live="assertive"
      aria-atomic="true"
    >
      <span class="k-toast__icon" aria-hidden="true">{{ iconChar }}</span>

      <span class="k-toast__message">{{ message }}</span>

      <button
        class="k-toast__close"
        aria-label="Dismiss notification"
        @click="emit('close')"
      >
        &times;
      </button>
    </div>
  </Transition>
</template>

<script setup lang="ts">
import { computed, watch, onUnmounted } from 'vue'

interface KToastProps {
  message: string
  type?: 'success' | 'error' | 'info' | 'warning'
  duration?: number
  visible: boolean
}

const props = withDefaults(defineProps<KToastProps>(), {
  type: 'info',
  duration: 4000,
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const iconChar = computed(() => {
  switch (props.type) {
    case 'success':
      return '✓'
    case 'error':
      return '✕'
    case 'info':
      return 'ℹ'
    case 'warning':
      return '⚠'
    default:
      return 'ℹ'
  }
})

let timer: ReturnType<typeof setTimeout> | null = null

function startTimer() {
  clearTimer()
  if (props.duration > 0 && props.visible) {
    timer = setTimeout(() => {
      emit('close')
    }, props.duration)
  }
}

function clearTimer() {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
}

watch(
  () => props.visible,
  (val) => {
    if (val) {
      startTimer()
    } else {
      clearTimer()
    }
  },
  { immediate: true }
)

onUnmounted(() => {
  clearTimer()
})
</script>

<style scoped>
.k-toast {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  box-shadow: var(--shadow-md);
  font-family: var(--font-family);
  font-size: var(--text-base);
  color: var(--color-text);
  min-width: 280px;
  max-width: 420px;
}

/* ─── Type Variants ─── */

.k-toast--success {
  border-left: 3px solid var(--color-success);
}

.k-toast--success .k-toast__icon {
  color: var(--color-success);
}

.k-toast--error {
  border-left: 3px solid var(--color-danger);
}

.k-toast--error .k-toast__icon {
  color: var(--color-danger);
}

.k-toast--info {
  border-left: 3px solid var(--color-primary);
}

.k-toast--info .k-toast__icon {
  color: var(--color-primary);
}

.k-toast--warning {
  border-left: 3px solid var(--color-warning);
}

.k-toast--warning .k-toast__icon {
  color: var(--color-warning);
}

/* ─── Elements ─── */

.k-toast__icon {
  font-size: 16px;
  font-weight: var(--font-bold);
  flex-shrink: 0;
  width: 20px;
  text-align: center;
}

.k-toast__message {
  flex: 1;
  line-height: var(--leading-snug);
}

.k-toast__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-muted);
  font-size: 18px;
  cursor: pointer;
  flex-shrink: 0;
  transition: background var(--duration-fast) var(--ease-default);
}

.k-toast__close:hover {
  background: var(--color-surface-2);
  color: var(--color-text);
}

.k-toast__close:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 1px;
}

/* ─── Transition ─── */

.k-toast-enter-active,
.k-toast-leave-active {
  transition:
    opacity var(--duration-slow) var(--ease-out),
    transform var(--duration-slow) var(--ease-out);
}

.k-toast-enter-from {
  opacity: 0;
  transform: translateX(20px);
}

.k-toast-leave-to {
  opacity: 0;
  transform: translateX(20px);
}

@media (prefers-reduced-motion: reduce) {
  .k-toast-enter-active,
  .k-toast-leave-active {
    transition: opacity var(--duration-fast) var(--ease-default);
  }
  .k-toast-enter-from,
  .k-toast-leave-to {
    transform: none;
  }
}
</style>
