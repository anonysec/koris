<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { useTheme } from '@koris/composables/useTheme'
import type { ThemeMode, UITheme } from '@koris/composables/useTheme'

const { isDark, theme, mode, setTheme, setMode, availableThemes } = useTheme()

const isOpen = ref(false)
const dropdownRef = ref<HTMLElement | null>(null)
const triggerRef = ref<HTMLButtonElement | null>(null)
const focusedIndex = ref(-1)

const modeOptions: { value: ThemeMode; label: string }[] = [
  { value: 'system', label: 'System' },
  { value: 'light', label: 'Light' },
  { value: 'dark', label: 'Dark' },
]

const themeList = computed(() =>
  availableThemes.map((t) => ({
    id: t.id,
    name: t.name,
    mode: t.mode,
    primary: t.colors.primary,
    accent: t.colors.accent,
    isActive: theme.value === t.id,
  }))
)

function toggleDropdown() {
  isOpen.value = !isOpen.value
  if (isOpen.value) {
    focusedIndex.value = -1
  }
}

function closeDropdown() {
  isOpen.value = false
  focusedIndex.value = -1
}

function selectTheme(id: UITheme) {
  setTheme(id)
  closeDropdown()
}

function selectMode(m: ThemeMode) {
  setMode(m)
}

function handleClickOutside(event: MouseEvent) {
  if (
    dropdownRef.value &&
    !dropdownRef.value.contains(event.target as Node) &&
    triggerRef.value &&
    !triggerRef.value.contains(event.target as Node)
  ) {
    closeDropdown()
  }
}

function handleKeydown(event: KeyboardEvent) {
  if (!isOpen.value) {
    if (event.key === 'Enter' || event.key === ' ' || event.key === 'ArrowDown') {
      event.preventDefault()
      isOpen.value = true
      focusedIndex.value = 0
    }
    return
  }

  const totalItems = availableThemes.length

  switch (event.key) {
    case 'Escape':
      event.preventDefault()
      closeDropdown()
      triggerRef.value?.focus()
      break
    case 'ArrowDown':
      event.preventDefault()
      focusedIndex.value = (focusedIndex.value + 1) % totalItems
      break
    case 'ArrowUp':
      event.preventDefault()
      focusedIndex.value = (focusedIndex.value - 1 + totalItems) % totalItems
      break
    case 'Enter':
    case ' ':
      event.preventDefault()
      if (focusedIndex.value >= 0 && focusedIndex.value < totalItems) {
        selectTheme(availableThemes[focusedIndex.value].id)
      }
      break
    case 'Tab':
      closeDropdown()
      break
  }
}

onMounted(() => {
  document.addEventListener('mousedown', handleClickOutside)
})

onUnmounted(() => {
  document.removeEventListener('mousedown', handleClickOutside)
})
</script>

<template>
  <div class="theme-toggle" @keydown="handleKeydown">
    <button
      ref="triggerRef"
      class="theme-toggle__trigger"
      :aria-label="isDark ? 'Open theme menu (dark mode active)' : 'Open theme menu (light mode active)'"
      :aria-expanded="isOpen"
      aria-haspopup="true"
      @click="toggleDropdown"
    >
      <span class="theme-toggle__icon" aria-hidden="true">
        <!-- Sun icon (shown when dark — click to explore themes) -->
        <svg v-if="isDark" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <circle cx="12" cy="12" r="5" />
          <line x1="12" y1="1" x2="12" y2="3" />
          <line x1="12" y1="21" x2="12" y2="23" />
          <line x1="4.22" y1="4.22" x2="5.64" y2="5.64" />
          <line x1="18.36" y1="18.36" x2="19.78" y2="19.78" />
          <line x1="1" y1="12" x2="3" y2="12" />
          <line x1="21" y1="12" x2="23" y2="12" />
          <line x1="4.22" y1="19.78" x2="5.64" y2="18.36" />
          <line x1="18.36" y1="5.64" x2="19.78" y2="4.22" />
        </svg>
        <!-- Moon icon (shown when light) -->
        <svg v-else width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2" stroke-linecap="round" stroke-linejoin="round">
          <path d="M21 12.79A9 9 0 1 1 11.21 3 7 7 0 0 0 21 12.79z" />
        </svg>
      </span>
    </button>

    <Transition name="theme-dropdown">
      <div
        v-if="isOpen"
        ref="dropdownRef"
        class="theme-toggle__dropdown"
        role="menu"
        aria-label="Theme selection"
      >
        <!-- Theme presets -->
        <div class="theme-toggle__section">
          <span class="theme-toggle__section-label">Themes</span>
          <button
            v-for="(t, index) in themeList"
            :key="t.id"
            class="theme-toggle__option"
            :class="{
              'theme-toggle__option--active': t.isActive,
              'theme-toggle__option--focused': focusedIndex === index,
            }"
            role="menuitem"
            :aria-current="t.isActive ? 'true' : undefined"
            @click="selectTheme(t.id)"
          >
            <span class="theme-toggle__option-colors" aria-hidden="true">
              <span class="theme-toggle__dot" :style="{ background: t.primary }" />
              <span class="theme-toggle__dot" :style="{ background: t.accent }" />
            </span>
            <span class="theme-toggle__option-name">{{ t.name }}</span>
            <svg
              v-if="t.isActive"
              class="theme-toggle__check"
              width="14"
              height="14"
              viewBox="0 0 24 24"
              fill="none"
              stroke="currentColor"
              stroke-width="3"
              stroke-linecap="round"
              stroke-linejoin="round"
              aria-hidden="true"
            >
              <polyline points="20 6 9 17 4 12" />
            </svg>
          </button>
        </div>

        <!-- Mode toggle -->
        <div class="theme-toggle__divider" />
        <div class="theme-toggle__section">
          <span class="theme-toggle__section-label">Mode</span>
          <div class="theme-toggle__modes" role="radiogroup" aria-label="Color mode">
            <button
              v-for="opt in modeOptions"
              :key="opt.value"
              class="theme-toggle__mode-btn"
              :class="{ 'theme-toggle__mode-btn--active': mode === opt.value }"
              role="radio"
              :aria-checked="mode === opt.value"
              @click="selectMode(opt.value)"
            >
              {{ opt.label }}
            </button>
          </div>
        </div>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.theme-toggle {
  position: relative;
  display: inline-flex;
}

.theme-toggle__trigger {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--color-text);
  cursor: pointer;
  transition: all 0.15s ease;
}

.theme-toggle__trigger:hover {
  background: var(--color-surface-2);
  border-color: var(--color-muted);
}

.theme-toggle__trigger:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.theme-toggle__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  line-height: 1;
}

/* ─── Dropdown ─── */

.theme-toggle__dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  min-width: 200px;
  padding: var(--space-2);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg, 0 10px 25px rgba(0, 0, 0, 0.15));
  z-index: 1000;
}

/* ─── Transition ─── */

.theme-dropdown-enter-active,
.theme-dropdown-leave-active {
  transition: opacity 0.15s ease, transform 0.15s ease;
}

.theme-dropdown-enter-from,
.theme-dropdown-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}

/* ─── Sections ─── */

.theme-toggle__section {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.theme-toggle__section-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: 0.04em;
  padding: var(--space-1) var(--space-2);
}

.theme-toggle__divider {
  height: 1px;
  background: var(--color-border);
  margin: var(--space-2) 0;
}

/* ─── Theme Options ─── */

.theme-toggle__option {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-2) var(--space-2);
  border: none;
  border-radius: calc(var(--radius-md) - 2px);
  background: transparent;
  color: var(--color-text);
  font-size: 13px;
  text-align: left;
  cursor: pointer;
  transition: background 0.1s ease;
}

.theme-toggle__option:hover,
.theme-toggle__option--focused {
  background: var(--color-surface-2);
}

.theme-toggle__option--active {
  color: var(--color-primary);
  font-weight: 500;
}

.theme-toggle__option-colors {
  display: flex;
  gap: 3px;
}

.theme-toggle__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  border: 1px solid var(--color-border);
}

.theme-toggle__option-name {
  flex: 1;
}

.theme-toggle__check {
  color: var(--color-primary);
  flex-shrink: 0;
}

/* ─── Mode Toggle ─── */

.theme-toggle__modes {
  display: flex;
  gap: 2px;
  padding: 2px;
  background: var(--color-surface-2);
  border-radius: calc(var(--radius-md) - 2px);
}

.theme-toggle__mode-btn {
  flex: 1;
  padding: var(--space-1) var(--space-2);
  border: none;
  border-radius: calc(var(--radius-md) - 4px);
  background: transparent;
  color: var(--color-muted);
  font-size: 12px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
}

.theme-toggle__mode-btn:hover {
  color: var(--color-text);
}

.theme-toggle__mode-btn--active {
  background: var(--color-surface);
  color: var(--color-text);
  box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
}

.theme-toggle__mode-btn:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: -2px;
}
</style>
