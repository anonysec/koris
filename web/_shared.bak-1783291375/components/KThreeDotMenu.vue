<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

export interface MenuItem {
  key: string
  label: string
  icon?: string
  danger?: boolean
  disabled?: boolean
}

export interface KThreeDotMenuProps {
  items: MenuItem[]
  placement?: 'bottom-end' | 'bottom-start' | 'top-end' | 'top-start'
}

const props = withDefaults(defineProps<KThreeDotMenuProps>(), {
  placement: 'bottom-end',
})

const emit = defineEmits<{
  select: [key: string]
}>()

const isOpen = ref(false)
const triggerRef = ref<HTMLButtonElement | null>(null)
const menuRef = ref<HTMLDivElement | null>(null)

function toggle() {
  isOpen.value = !isOpen.value
}

function close() {
  isOpen.value = false
}

function selectItem(item: MenuItem) {
  if (item.disabled) return
  emit('select', item.key)
  close()
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    event.preventDefault()
    event.stopPropagation()
    close()
    triggerRef.value?.focus()
  }
}

function onClickOutside(event: MouseEvent) {
  if (!isOpen.value) return
  const target = event.target as Node
  if (
    triggerRef.value?.contains(target) ||
    menuRef.value?.contains(target)
  ) {
    return
  }
  close()
}

onMounted(() => {
  document.addEventListener('mousedown', onClickOutside)
  document.addEventListener('keydown', onKeydown)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', onClickOutside)
  document.removeEventListener('keydown', onKeydown)
})
</script>

<template>
  <div class="k-three-dot-menu">
    <button
      ref="triggerRef"
      type="button"
      class="k-three-dot-menu__trigger"
      aria-haspopup="menu"
      :aria-expanded="isOpen"
      aria-label="More actions"
      @click.stop="toggle"
    >
      <svg
        class="k-three-dot-menu__icon"
        width="16"
        height="16"
        viewBox="0 0 16 16"
        fill="currentColor"
        aria-hidden="true"
      >
        <circle cx="8" cy="3" r="1.5" />
        <circle cx="8" cy="8" r="1.5" />
        <circle cx="8" cy="13" r="1.5" />
      </svg>
    </button>

    <Transition name="k-three-dot-menu-dropdown">
      <div
        v-if="isOpen"
        ref="menuRef"
        class="k-three-dot-menu__dropdown"
        :class="`k-three-dot-menu__dropdown--${placement}`"
        role="menu"
      >
        <button
          v-for="item in items"
          :key="item.key"
          type="button"
          class="k-three-dot-menu__item"
          :class="{
            'k-three-dot-menu__item--danger': item.danger,
            'k-three-dot-menu__item--disabled': item.disabled,
          }"
          role="menuitem"
          :disabled="item.disabled"
          :aria-disabled="item.disabled"
          @click.stop="selectItem(item)"
        >
          <span v-if="item.icon" class="k-three-dot-menu__item-icon" aria-hidden="true">
            {{ item.icon }}
          </span>
          <span class="k-three-dot-menu__item-label">{{ item.label }}</span>
        </button>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.k-three-dot-menu {
  position: relative;
  display: inline-flex;
}

.k-three-dot-menu__trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  border-radius: var(--radius-md, 8px);
  border: none;
  background: transparent;
  color: var(--color-muted, #8b98a5);
  cursor: pointer;
  transition: background-color var(--transition-hover, 100ms ease-out),
              color var(--transition-hover, 100ms ease-out);
  outline: none;
}

.k-three-dot-menu__trigger:hover {
  background-color: var(--color-surface-2, #1e2630);
  color: var(--color-text, #e6edf3);
}

.k-three-dot-menu__trigger:focus-visible {
  outline: 2px solid var(--color-accent, #22d3ee);
  outline-offset: 2px;
}

.k-three-dot-menu__icon {
  flex-shrink: 0;
}

/* ─── Dropdown ─── */

.k-three-dot-menu__dropdown {
  position: absolute;
  z-index: var(--z-dropdown, 100);
  min-width: 180px;
  padding: var(--space-1, 4px);
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-lg, 12px);
  box-shadow: var(--shadow-lg, 0 16px 48px rgba(0, 0, 0, 0.4));
}

.k-three-dot-menu__dropdown--bottom-end {
  top: calc(100% + 4px);
  right: 0;
}

.k-three-dot-menu__dropdown--bottom-start {
  top: calc(100% + 4px);
  left: 0;
}

.k-three-dot-menu__dropdown--top-end {
  bottom: calc(100% + 4px);
  right: 0;
}

.k-three-dot-menu__dropdown--top-start {
  bottom: calc(100% + 4px);
  left: 0;
}

/* ─── Menu Items ─── */

.k-three-dot-menu__item {
  display: flex;
  align-items: center;
  gap: var(--space-2, 8px);
  width: 100%;
  padding: var(--space-2, 8px) var(--space-3, 12px);
  border: none;
  border-radius: var(--radius-md, 8px);
  background: transparent;
  color: var(--color-text, #e6edf3);
  font-size: var(--text-sm, 13px);
  font-family: var(--font-family);
  line-height: var(--leading-normal, 1.5);
  cursor: pointer;
  transition: background-color var(--transition-hover, 100ms ease-out);
  text-align: left;
  outline: none;
}

.k-three-dot-menu__item:hover:not(:disabled) {
  background-color: var(--color-surface-2, #1e2630);
}

.k-three-dot-menu__item:focus-visible {
  outline: 2px solid var(--color-accent, #22d3ee);
  outline-offset: -2px;
}

.k-three-dot-menu__item--danger {
  color: var(--color-danger, #ef4444);
}

.k-three-dot-menu__item--danger:hover:not(:disabled) {
  background-color: rgba(239, 68, 68, 0.1);
}

.k-three-dot-menu__item--disabled {
  color: var(--color-muted, #8b98a5);
  opacity: 0.5;
  cursor: not-allowed;
}

.k-three-dot-menu__item-icon {
  flex-shrink: 0;
  width: 16px;
  text-align: center;
}

.k-three-dot-menu__item-label {
  flex: 1;
}

/* ─── Dropdown Transition ─── */

.k-three-dot-menu-dropdown-enter-active {
  transition: opacity 150ms ease-out, transform 150ms ease-out;
}

.k-three-dot-menu-dropdown-leave-active {
  transition: opacity 100ms ease-in, transform 100ms ease-in;
}

.k-three-dot-menu-dropdown-enter-from,
.k-three-dot-menu-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px) scale(0.95);
}
</style>
