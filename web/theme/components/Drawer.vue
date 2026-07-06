<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'
import type { KDrawerProps } from '@koris/types/components'

const props = withDefaults(defineProps<KDrawerProps>(), {
  side: 'right',
  width: '480px',
  closable: true,
  overlay: true
})

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'after-enter'): void
  (e: 'after-leave'): void
}>()

const drawerRef = ref<HTMLElement | null>(null)
const previouslyFocusedElement = ref<HTMLElement | null>(null)

// --- Focus Trap ---

function getFocusableElements(): HTMLElement[] {
  if (!drawerRef.value) return []
  const selectors = [
    'a[href]',
    'button:not([disabled])',
    'textarea:not([disabled])',
    'input:not([disabled])',
    'select:not([disabled])',
    '[tabindex]:not([tabindex="-1"])'
  ].join(', ')
  return Array.from(drawerRef.value.querySelectorAll<HTMLElement>(selectors))
}

function focusFirstElement() {
  const focusable = getFocusableElements()
  if (focusable.length > 0) {
    focusable[0].focus()
  } else if (drawerRef.value) {
    drawerRef.value.focus()
  }
}

function handleTabKey(event: KeyboardEvent) {
  if (!props.open) return

  const focusable = getFocusableElements()
  if (focusable.length === 0) return

  const firstEl = focusable[0]
  const lastEl = focusable[focusable.length - 1]

  if (event.shiftKey) {
    // Shift+Tab: if on first element, wrap to last
    if (document.activeElement === firstEl) {
      event.preventDefault()
      lastEl.focus()
    }
  } else {
    // Tab: if on last element, wrap to first
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

// --- Overlay Click ---

function handleOverlayClick() {
  if (props.closable) {
    emit('close')
  }
}

// --- Transition Hooks ---

function onAfterEnter() {
  emit('after-enter')
}

function onAfterLeave() {
  emit('after-leave')
  // Restore focus to the element that was focused before drawer opened
  if (previouslyFocusedElement.value) {
    previouslyFocusedElement.value.focus()
    previouslyFocusedElement.value = null
  }
}

// --- Watch open prop ---

watch(() => props.open, async (isOpen) => {
  if (isOpen) {
    // Save currently focused element
    previouslyFocusedElement.value = document.activeElement as HTMLElement | null
    lockBodyScroll()
    await nextTick()
    focusFirstElement()
  } else {
    unlockBodyScroll()
  }
})

// --- Event Listeners ---

onMounted(() => {
  document.addEventListener('keydown', handleKeydown)
})

onUnmounted(() => {
  document.removeEventListener('keydown', handleKeydown)
  // Ensure scroll lock is cleaned up if component unmounts while open
  if (props.open) {
    unlockBodyScroll()
  }
})
</script>

<template>
  <Teleport to="body">
    <!-- Overlay -->
    <Transition name="k-drawer-overlay">
      <div
        v-if="open && overlay"
        class="k-drawer-overlay"
        @click="handleOverlayClick"
        aria-hidden="true"
      />
    </Transition>

    <!-- Drawer Panel -->
    <Transition
      :name="`k-drawer-slide-${side}`"
      @after-enter="onAfterEnter"
      @after-leave="onAfterLeave"
    >
      <div
        v-if="open"
        ref="drawerRef"
        class="k-drawer"
        :class="[`k-drawer--${side}`]"
        :style="{ width }"
        role="dialog"
        aria-modal="true"
        :aria-label="title"
        tabindex="-1"
      >
        <!-- Header -->
        <header v-if="title || closable" class="k-drawer__header">
          <h2 v-if="title" class="k-drawer__title">{{ title }}</h2>
          <button
            v-if="closable"
            type="button"
            class="k-drawer__close-btn"
            aria-label="Close drawer"
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
        <div class="k-drawer__content">
          <slot />
        </div>

        <!-- Footer (optional) -->
        <footer v-if="$slots.footer" class="k-drawer__footer">
          <slot name="footer" />
        </footer>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
/* --- Overlay --- */
.k-drawer-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  z-index: var(--z-modal, 200);
}

/* Overlay transitions */
.k-drawer-overlay-enter-active,
.k-drawer-overlay-leave-active {
  transition: opacity var(--duration-slower, 0.3s) var(--ease-default, ease);
}

.k-drawer-overlay-enter-from,
.k-drawer-overlay-leave-to {
  opacity: 0;
}

/* --- Drawer Panel --- */
.k-drawer {
  position: fixed;
  top: 0;
  bottom: 0;
  z-index: calc(var(--z-modal, 200) + 1);
  display: flex;
  flex-direction: column;
  max-width: 100vw;
  background: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
  outline: none;
}

.k-drawer--right {
  right: 0;
  border-right: none;
  border-top-left-radius: var(--radius-lg, 10px);
  border-bottom-left-radius: var(--radius-lg, 10px);
}

.k-drawer--left {
  left: 0;
  border-left: none;
  border-top-right-radius: var(--radius-lg, 10px);
  border-bottom-right-radius: var(--radius-lg, 10px);
}

/* --- Header --- */
.k-drawer__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-5, 20px) var(--space-6, 24px);
  border-bottom: 1px solid var(--color-border, #28333f);
  flex-shrink: 0;
}

.k-drawer__title {
  margin: 0;
  font-size: var(--text-lg, 16px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text, #e6edf3);
  line-height: var(--leading-tight, 1.1);
}

.k-drawer__close-btn {
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

.k-drawer__close-btn:hover {
  background: var(--color-surface-2, #1e2630);
  color: var(--color-text, #e6edf3);
}

.k-drawer__close-btn:focus-visible {
  outline: 2px solid var(--color-primary, #2563eb);
  outline-offset: 2px;
}

/* --- Content --- */
.k-drawer__content {
  flex: 1;
  overflow-y: auto;
  padding: var(--space-6, 24px);
}

/* --- Footer --- */
.k-drawer__footer {
  padding: var(--space-4, 16px) var(--space-6, 24px);
  border-top: 1px solid var(--color-border, #28333f);
  flex-shrink: 0;
}

/* --- Slide-from-right transitions --- */
.k-drawer-slide-right-enter-active,
.k-drawer-slide-right-leave-active {
  transition: transform var(--duration-slower, 0.3s) var(--ease-out, ease-out);
}

.k-drawer-slide-right-enter-from,
.k-drawer-slide-right-leave-to {
  transform: translateX(100%);
}

/* --- Slide-from-left transitions --- */
.k-drawer-slide-left-enter-active,
.k-drawer-slide-left-leave-active {
  transition: transform var(--duration-slower, 0.3s) var(--ease-out, ease-out);
}

.k-drawer-slide-left-enter-from,
.k-drawer-slide-left-leave-to {
  transform: translateX(-100%);
}
</style>
