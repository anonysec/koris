<template>
  <button
    :type="type || 'button'"
    :class="[
      'k-btn',
      `k-btn--${variant}`,
      `k-btn--${size}`,
      {
        'k-btn--loading': loading,
        'k-btn--disabled': disabled,
        'k-btn--full-width': fullWidth,
        'k-btn--icon-only': icon && !$slots.default,
      },
    ]"
    :disabled="disabled || loading"
    :aria-disabled="disabled || loading"
    :aria-busy="loading"
    @click="handleClick"
  >
    <span v-if="loading" class="k-btn__spinner" aria-hidden="true">
      <svg class="k-btn__spinner-icon" viewBox="0 0 24 24" fill="none">
        <circle
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="3"
          stroke-linecap="round"
          stroke-dasharray="50 20"
        />
      </svg>
    </span>

    <span class="k-btn__content" :class="{ 'k-btn__content--hidden': loading }">
      <span
        v-if="icon && iconPosition === 'left'"
        class="k-btn__icon k-btn__icon--left"
        aria-hidden="true"
      >
        {{ icon }}
      </span>

      <span v-if="$slots.default" class="k-btn__label">
        <slot />
      </span>

      <span
        v-if="icon && iconPosition === 'right'"
        class="k-btn__icon k-btn__icon--right"
        aria-hidden="true"
      >
        {{ icon }}
      </span>
    </span>
  </button>
</template>

<script setup lang="ts">
import type { KButtonProps } from '@koris/types/components'

const props = withDefaults(defineProps<KButtonProps>(), {
  variant: 'primary',
  size: 'md',
  loading: false,
  disabled: false,
  iconPosition: 'left',
  fullWidth: false,
})

const emit = defineEmits<{
  (e: 'click', event: MouseEvent): void
}>()

function handleClick(event: MouseEvent) {
  if (props.loading || props.disabled) {
    event.preventDefault()
    event.stopPropagation()
    return
  }
  emit('click', event)
}
</script>

<style scoped>
.k-btn {
  position: relative;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  border: none;
  border-radius: var(--radius-md);
  font-family: var(--font-family);
  font-weight: var(--font-medium);
  line-height: 1;
  cursor: pointer;
  transition:
    background var(--duration-normal) var(--ease-default),
    box-shadow var(--duration-normal) var(--ease-default),
    border-color var(--duration-normal) var(--ease-default),
    opacity var(--duration-normal) var(--ease-default);
  white-space: nowrap;
  user-select: none;
  outline: none;
}

.k-btn:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

/* ─── Sizes ─── */

.k-btn--sm {
  height: 30px;
  font-size: 12px;
  padding: 6px 10px;
}

.k-btn--md {
  height: 36px;
  font-size: 13px;
  padding: 8px 14px;
}

.k-btn--lg {
  height: 42px;
  font-size: 14px;
  padding: 10px 18px;
}

/* ─── Variants ─── */

.k-btn--primary {
  background: var(--gradient-brand);
  color: #fff;
  box-shadow: var(--shadow-brand);
}

.k-btn--primary:hover:not(:disabled) {
  box-shadow: 0 6px 20px rgba(37, 99, 235, 0.35);
}

.k-btn--primary:active:not(:disabled) {
  box-shadow: 0 2px 8px rgba(37, 99, 235, 0.2);
}

.k-btn--ghost {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.k-btn--ghost:hover:not(:disabled) {
  background: var(--color-surface-2);
  border-color: var(--color-muted);
}

.k-btn--ghost:active:not(:disabled) {
  background: var(--color-surface);
}

.k-btn--danger {
  background: var(--color-danger);
  color: #fff;
  border: none;
}

.k-btn--danger:hover:not(:disabled) {
  background: #dc2626;
  box-shadow: 0 4px 14px rgba(239, 68, 68, 0.3);
}

.k-btn--danger:active:not(:disabled) {
  background: #b91c1c;
}

.k-btn--text {
  background: transparent;
  color: var(--color-primary);
  border: none;
  padding-left: var(--space-2);
  padding-right: var(--space-2);
}

.k-btn--text:hover:not(:disabled) {
  background: rgba(37, 99, 235, 0.08);
}

.k-btn--text:active:not(:disabled) {
  background: rgba(37, 99, 235, 0.12);
}

/* ─── States ─── */

.k-btn--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.k-btn--loading {
  cursor: wait;
}

.k-btn--full-width {
  width: 100%;
}

.k-btn--icon-only.k-btn--sm {
  width: 30px;
  padding: 0;
}

.k-btn--icon-only.k-btn--md {
  width: 36px;
  padding: 0;
}

.k-btn--icon-only.k-btn--lg {
  width: 42px;
  padding: 0;
}

/* ─── Content ─── */

.k-btn__content {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  visibility: visible;
}

.k-btn__content--hidden {
  visibility: hidden;
}

.k-btn__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 1.1em;
}

.k-btn__label {
  display: inline-flex;
  align-items: center;
}

/* ─── Spinner ─── */

.k-btn__spinner {
  position: absolute;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
}

.k-btn__spinner-icon {
  width: 1.2em;
  height: 1.2em;
  animation: k-btn-spin 0.75s linear infinite;
}

@keyframes k-btn-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}
</style>
