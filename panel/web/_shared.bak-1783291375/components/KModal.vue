<script setup lang="ts">
/**
 * KModal — A centered modal dialog with blurred darkened overlay.
 *
 * - Props: open, title, width (default '520px'), closable (default true)
 * - Emits: 'close'
 * - Accessible: role="dialog", aria-modal, focus trap, Escape to close
 * - Click-outside (overlay click) closes modal when closable
 * - Fade-in 200ms, fade-out 150ms
 * - backdrop-filter: blur(4px) on overlay
 * - Prevents body scroll while open
 * - Renders to <body> via Teleport
 */
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'

const props = withDefaults(defineProps<{
  open: boolean
  title?: string
  width?: string
  closable?: boolean
}>(), {
  title: '',
  width: '520px',
  closable: true,
})

const emit = defineEmits<{
  (e: 'close'): void
}>()

const modalRef = ref<HTMLElement | null>(null)
const previouslyFocusedElement = ref<HTMLElement | null>(null)

// --- Focus Trap ---

function getFocusableElements(): HTMLElement[] {
  if (!modalRef.value) return []
  const selectors = [
    'a[href]',
    'button:not([disabled])',
    'textarea:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    '[tabindex]:not([tabindex="-1"])',
  ].join(', ')
  return Array.from(modalRef.value.querySelectorAll<HTMLElement>(selectors))
}

function focusFirstElement() {
  const focusable = getFocusableElements()
  if (focusable.length > 0) {
    focusable[0].focus()
  } else if (modalRef.value) {
    modalRef.value.focus()
  }
}

function handleTabKey(event: KeyboardEvent) {
  if (!props.open) return

  const focusable = getFocusableElements()
  if (focusable.length === 0) return

  const firstEl = focusable[0]
  const lastEl = focusable[focusable.length - 1]

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

// --- Keyboard Handling ---

function handleKeydown(event: KeyboardEvent) {
  if (!props.open) return

  if (event.key === 'Escape' && props.closable) {
    event.preventDefault()
    event.stopPropagation()
    emit('close')
    return
  }

  if (event.key === 'Tab') {
    handleTabKey(event)
  }
}

// --- Body Scroll Lock ---

function lockBodyScroll() {
  document.body.style.overflow = 'hidden'
}

function unlockBodyScroll() {
  document.body.style.overflow = ''
}

// --- Overlay Click (click outside) ---

function handleOverlayClick(event: MouseEvent) {
  if (props.closable && event.target === event.currentTarget) {
    emit('close')
  }
}

// --- Transition Hooks ---

function onAfterLeave() {
  if (previouslyFocusedElement.value) {
    previouslyFocusedElement.value.focus()
    previouslyFocusedElement.value = null
  }
}

// --- Watch open prop ---

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    previouslyFocusedElement.value = document.activeElement as HTMLElement | null
    lockBodyScroll()
    await nextTick()
    focusFirstElement()
  } else {
    unlockBodyScroll()
  }
}, { immediate: true })

// --- Event Listeners ---

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  if (props.open) {
    unlockBodyScroll()
  }
})
</script>

<template>
  <Teleport to="body">
    <Transition name="k-modal" @after-leave="onAfterLeave">
      <div
        v-if="open"
        class="k-modal__overlay"
        role="dialog"
        aria-modal="true"
        :aria-labelledby="title ? 'k-modal-title' : undefined"
        :aria-label="!title ? 'Dialog' : undefined"
        @click="handleOverlayClick"
      >
        <div
          ref="modalRef"
          class="k-modal"
          :style="{ width, maxWidth: width }"
          tabindex="-1"
        >
          <!-- Header -->
          <header v-if="title || closable" class="k-modal__header">
            <h2 v-if="title" id="k-modal-title" class="k-modal__title">
              {{ title }}
            </h2>
            <button
              v-if="closable"
              type="button"
              class="k-modal__close-btn"
              aria-label="Close modal"
              @click="emit('close')"
            >
              <svg
                width="20"
                height="20"
                viewBox="0 0 20 20"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                aria-hidden="true"
              >
                <path
                  d="M15 5L5 15M5 5l10 10"
                  stroke="currentColor"
                  stroke-width="1.5"
                  stroke-linecap="round"
                  stroke-linejoin="round"
                />
              </svg>
            </button>
          </header>

          <!-- Content -->
          <div class="k-modal__content">
            <slot />
          </div>

          <!-- Footer (optional) -->
          <footer v-if="$slots.footer" class="k-modal__footer">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* --- Overlay --- */
.k-modal__overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal, 200);
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(4px);
  padding: var(--space-4, 16px);
}

/* --- Modal Panel --- */
.k-modal {
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-xl, 14px);
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
  display: flex;
  flex-direction: column;
  max-height: calc(100vh - 64px);
  width: 100%;
  outline: none;
}

/* --- Header --- */
.k-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5, 20px) var(--space-6, 24px);
  border-bottom: 1px solid var(--color-border, #28333f);
  flex-shrink: 0;
}

.k-modal__title {
  margin: 0;
  font-size: var(--text-lg, 16px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text, #e6edf3);
  line-height: var(--leading-tight, 1.1);
}

.k-modal__close-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm, 6px);
  background: transparent;
  color: var(--color-muted, #8b98a5);
  cursor: pointer;
  transition: background var(--duration-fast, 0.12s) var(--ease-default, ease),
              color var(--duration-fast, 0.12s) var(--ease-default, ease);
}

.k-modal__close-btn:hover {
  background: var(--color-surface-2, #1e2630);
  color: var(--color-text, #e6edf3);
}

.k-modal__close-btn:focus-visible {
  outline: 2px solid var(--color-primary, #2563eb);
  outline-offset: 2px;
}

/* --- Content --- */
.k-modal__content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6, 24px);
}

/* --- Footer --- */
.k-modal__footer {
  padding: var(--space-4, 16px) var(--space-6, 24px);
  border-top: 1px solid var(--color-border, #28333f);
  flex-shrink: 0;
}

/* --- Transition: Fade-in 200ms, Fade-out 150ms --- */
.k-modal-enter-active {
  transition: opacity var(--transition-modal-in, 200ms) var(--ease-out, ease-out);
}

.k-modal-enter-active .k-modal {
  transition: transform var(--transition-modal-in, 200ms) var(--ease-out, ease-out),
              opacity var(--transition-modal-in, 200ms) var(--ease-out, ease-out);
}

.k-modal-leave-active {
  transition: opacity var(--transition-modal-out, 150ms) var(--ease-out, ease-out);
}

.k-modal-leave-active .k-modal {
  transition: transform var(--transition-modal-out, 150ms) var(--ease-out, ease-out),
              opacity var(--transition-modal-out, 150ms) var(--ease-out, ease-out);
}

.k-modal-enter-from,
.k-modal-leave-to {
  opacity: 0;
}

.k-modal-enter-from .k-modal,
.k-modal-leave-to .k-modal {
  transform: scale(0.95);
  opacity: 0;
}

/* --- Reduced motion support --- */
@media (prefers-reduced-motion: reduce) {
  .k-modal-enter-active,
  .k-modal-leave-active,
  .k-modal-enter-active .k-modal,
  .k-modal-leave-active .k-modal {
    transition-duration: 0ms;
  }
}
</style>
