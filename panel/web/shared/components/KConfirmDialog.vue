<script setup lang="ts">
import { ref, watch, nextTick, onMounted, onUnmounted } from 'vue'
import { useConfirm } from '@koris/composables/useConfirm'

const { isOpen, options, handleConfirm, handleCancel } = useConfirm()

const cancelButtonRef = ref<HTMLButtonElement | null>(null)
const confirmButtonRef = ref<HTMLButtonElement | null>(null)
const dialogRef = ref<HTMLDivElement | null>(null)

/**
 * Map dialog variant to confirm button CSS class
 */
function getConfirmButtonClass(variant?: string): string {
  switch (variant) {
    case 'danger':
      return 'k-confirm-dialog__btn--danger'
    case 'warning':
      return 'k-confirm-dialog__btn--warning'
    case 'info':
    default:
      return 'k-confirm-dialog__btn--primary'
  }
}

/**
 * Focus the appropriate button when dialog opens.
 * Auto-focus cancel for danger variant to prevent accidental confirmation.
 */
watch(isOpen, async (open) => {
  if (open) {
    await nextTick()
    if (options.value.variant === 'danger') {
      cancelButtonRef.value?.focus()
    } else {
      confirmButtonRef.value?.focus()
    }
  }
})

/**
 * Handle keyboard events for the dialog
 */
function onKeydown(event: KeyboardEvent): void {
  if (!isOpen.value) return

  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    handleCancel()
  } else if (event.key === 'Enter') {
    // Only handle Enter if the active element is not already a button
    // (to avoid double-triggering when a button is focused)
    const target = event.target as HTMLElement
    if (target.tagName !== 'BUTTON') {
      event.preventDefault()
      handleConfirm()
    }
  } else if (event.key === 'Tab') {
    // Trap focus within the dialog
    trapFocus(event)
  }
}

/**
 * Trap focus within the dialog (Tab / Shift+Tab)
 */
function trapFocus(event: KeyboardEvent): void {
  if (!dialogRef.value) return

  const focusableElements = dialogRef.value.querySelectorAll<HTMLElement>(
    'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'
  )

  if (focusableElements.length === 0) return

  const firstEl = focusableElements[0]
  const lastEl = focusableElements[focusableElements.length - 1]

  if (event.shiftKey) {
    if (document.activeElement === firstEl) {
      event.preventDefault()
      lastEl.focus()
    }
  } else {
    if (document.activeElement === lastEl) {
      event.preventDefault()
      firstEl.focus()
    }
  }
}

/**
 * Handle overlay click (click outside the dialog content)
 */
function onOverlayClick(event: MouseEvent): void {
  if (event.target === event.currentTarget) {
    handleCancel()
  }
}

/**
 * Global keydown listener
 */
function onGlobalKeydown(event: KeyboardEvent): void {
  if (!isOpen.value) return
  onKeydown(event)
}

onMounted(() => {
  document.addEventListener('keydown', onGlobalKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', onGlobalKeydown)
})
</script>

<template>
  <Teleport to="body">
    <Transition name="k-confirm-dialog">
      <div
        v-if="isOpen"
        class="k-confirm-dialog__overlay"
        role="alertdialog"
        aria-modal="true"
        :aria-labelledby="'k-confirm-title'"
        :aria-describedby="'k-confirm-message'"
        @click="onOverlayClick"
      >
        <div
          ref="dialogRef"
          class="k-confirm-dialog"
        >
          <!-- Icon -->
          <div
            v-if="options.icon"
            class="k-confirm-dialog__icon"
            :class="`k-confirm-dialog__icon--${options.variant || 'info'}`"
          >
            <span>{{ options.icon }}</span>
          </div>

          <!-- Title -->
          <h2
            id="k-confirm-title"
            class="k-confirm-dialog__title"
          >
            {{ options.title }}
          </h2>

          <!-- Message -->
          <p
            id="k-confirm-message"
            class="k-confirm-dialog__message"
          >
            {{ options.message }}
          </p>

          <!-- Actions -->
          <div class="k-confirm-dialog__actions">
            <button
              ref="cancelButtonRef"
              type="button"
              class="k-confirm-dialog__btn k-confirm-dialog__btn--ghost"
              @click="handleCancel"
            >
              {{ options.cancelText || 'Cancel' }}
            </button>
            <button
              ref="confirmButtonRef"
              type="button"
              class="k-confirm-dialog__btn"
              :class="getConfirmButtonClass(options.variant)"
              @click="handleConfirm"
            >
              {{ options.confirmText || 'Confirm' }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.k-confirm-dialog__overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal, 200);
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  padding: var(--space-4, 16px);
}

.k-confirm-dialog {
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-xl, 14px);
  padding: var(--space-6, 24px);
  max-width: 420px;
  width: 100%;
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
  text-align: center;
}

.k-confirm-dialog__icon {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 48px;
  height: 48px;
  margin: 0 auto var(--space-4, 16px);
  border-radius: var(--radius-full, 9999px);
  font-size: 24px;
}

.k-confirm-dialog__icon--danger {
  background-color: rgba(239, 68, 68, 0.1);
  color: var(--color-danger, #ef4444);
}

.k-confirm-dialog__icon--warning {
  background-color: rgba(245, 158, 11, 0.1);
  color: var(--color-warning, #f59e0b);
}

.k-confirm-dialog__icon--info {
  background-color: rgba(37, 99, 235, 0.1);
  color: var(--color-primary, #2563eb);
}

.k-confirm-dialog__title {
  font-size: var(--text-lg, 16px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text, #e6edf3);
  margin: 0 0 var(--space-2, 8px);
  line-height: var(--leading-snug, 1.3);
}

.k-confirm-dialog__message {
  font-size: var(--text-base, 13.5px);
  color: var(--color-muted, #8b98a5);
  margin: 0 0 var(--space-6, 24px);
  line-height: var(--leading-normal, 1.5);
}

.k-confirm-dialog__actions {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3, 12px);
}

.k-confirm-dialog__btn {
  padding: var(--space-2, 8px) var(--space-5, 20px);
  border-radius: var(--radius-md, 8px);
  font-size: var(--text-base, 13.5px);
  font-weight: var(--font-medium, 500);
  cursor: pointer;
  border: none;
  transition: all var(--duration-normal, 0.15s) var(--ease-default, ease);
  outline: none;
  line-height: var(--leading-normal, 1.5);
}

.k-confirm-dialog__btn:focus-visible {
  outline: 2px solid var(--color-accent, #22d3ee);
  outline-offset: 2px;
}

.k-confirm-dialog__btn--ghost {
  background-color: transparent;
  color: var(--color-muted, #8b98a5);
  border: 1px solid var(--color-border, #28333f);
}

.k-confirm-dialog__btn--ghost:hover {
  background-color: var(--color-surface-2, #1e2630);
  color: var(--color-text, #e6edf3);
}

.k-confirm-dialog__btn--primary {
  background: var(--gradient-brand, linear-gradient(135deg, #2563eb, #7c5cff));
  color: #ffffff;
}

.k-confirm-dialog__btn--primary:hover {
  opacity: 0.9;
  box-shadow: var(--shadow-brand, 0 4px 14px rgba(37, 99, 235, 0.25));
}

.k-confirm-dialog__btn--danger {
  background-color: var(--color-danger, #ef4444);
  color: #ffffff;
}

.k-confirm-dialog__btn--danger:hover {
  opacity: 0.9;
  box-shadow: 0 4px 14px rgba(239, 68, 68, 0.3);
}

.k-confirm-dialog__btn--warning {
  background-color: var(--color-warning, #f59e0b);
  color: #000000;
}

.k-confirm-dialog__btn--warning:hover {
  opacity: 0.9;
  box-shadow: 0 4px 14px rgba(245, 158, 11, 0.3);
}

/* Transition animations */
.k-confirm-dialog-enter-active,
.k-confirm-dialog-leave-active {
  transition: opacity var(--duration-slow, 0.2s) var(--ease-out, ease-out);
}

.k-confirm-dialog-enter-active .k-confirm-dialog,
.k-confirm-dialog-leave-active .k-confirm-dialog {
  transition: transform var(--duration-slow, 0.2s) var(--ease-out, ease-out),
              opacity var(--duration-slow, 0.2s) var(--ease-out, ease-out);
}

.k-confirm-dialog-enter-from,
.k-confirm-dialog-leave-to {
  opacity: 0;
}

.k-confirm-dialog-enter-from .k-confirm-dialog,
.k-confirm-dialog-leave-to .k-confirm-dialog {
  transform: scale(0.95);
  opacity: 0;
}
</style>
