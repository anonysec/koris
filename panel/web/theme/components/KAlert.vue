<template>
  <div
    v-if="!dismissed"
    :class="['k-alert', `k-alert--${variant}`]"
    role="alert"
  >
    <span class="k-alert__icon" aria-hidden="true">{{ iconChar }}</span>

    <div class="k-alert__body">
      <p v-if="title" class="k-alert__title">{{ title }}</p>
      <div class="k-alert__content">
        <slot />
      </div>
    </div>

    <button
      v-if="closable"
      class="k-alert__close"
      aria-label="Dismiss alert"
      @click="dismiss"
    >
      &times;
    </button>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'

interface KAlertProps {
  variant?: 'info' | 'success' | 'warning' | 'error'
  title?: string
  closable?: boolean
}

const props = withDefaults(defineProps<KAlertProps>(), {
  variant: 'info',
  closable: false,
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const dismissed = ref(false)

const iconChar = computed(() => {
  switch (props.variant) {
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

function dismiss() {
  dismissed.value = true
  emit('close')
}
</script>

<style scoped>
.k-alert {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  font-family: var(--font-family);
  font-size: var(--text-base);
  color: var(--color-text);
}

/* ─── Variant Left Borders ─── */

.k-alert--info {
  border-left: 3px solid var(--color-primary);
}

.k-alert--success {
  border-left: 3px solid var(--color-success);
}

.k-alert--warning {
  border-left: 3px solid var(--color-warning);
}

.k-alert--error {
  border-left: 3px solid var(--color-danger);
}

/* ─── Icon Colors ─── */

.k-alert--info .k-alert__icon {
  color: var(--color-primary);
}

.k-alert--success .k-alert__icon {
  color: var(--color-success);
}

.k-alert--warning .k-alert__icon {
  color: var(--color-warning);
}

.k-alert--error .k-alert__icon {
  color: var(--color-danger);
}

/* ─── Elements ─── */

.k-alert__icon {
  font-size: 16px;
  font-weight: var(--font-bold);
  flex-shrink: 0;
  width: 20px;
  text-align: center;
  margin-top: 1px;
}

.k-alert__body {
  flex: 1;
  min-width: 0;
}

.k-alert__title {
  font-weight: var(--font-semibold);
  font-size: var(--text-base);
  margin: 0 0 var(--space-1) 0;
  line-height: var(--leading-snug);
}

.k-alert__content {
  line-height: var(--leading-normal);
  color: var(--color-muted);
}

.k-alert__content:empty {
  display: none;
}

.k-alert__close {
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

.k-alert__close:hover {
  background: var(--color-surface-2);
  color: var(--color-text);
}

.k-alert__close:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 1px;
}
</style>
