<template>
  <div class="k-tabs">
    <div class="k-tabs__list" role="tablist" :aria-label="ariaLabel">
      <button
        v-for="(tab, index) in tabs"
        :key="tab.key"
        :id="`tab-${tab.key}`"
        role="tab"
        :aria-selected="modelValue === tab.key"
        :aria-controls="`tabpanel-${tab.key}`"
        :tabindex="modelValue === tab.key ? 0 : -1"
        :class="[
          'k-tabs__tab',
          { 'k-tabs__tab--active': modelValue === tab.key },
        ]"
        @click="selectTab(tab.key)"
        @keydown="handleKeydown($event, index)"
      >
        <span class="k-tabs__tab-label">{{ tab.label }}</span>
        <span v-if="tab.badge != null" class="k-tabs__tab-badge">
          {{ tab.badge }}
        </span>
      </button>
    </div>
    <div
      v-for="tab in tabs"
      :key="`panel-${tab.key}`"
      :id="`tabpanel-${tab.key}`"
      role="tabpanel"
      :aria-labelledby="`tab-${tab.key}`"
      :hidden="modelValue !== tab.key"
    >
      <slot :name="tab.key" v-if="modelValue === tab.key" />
    </div>
  </div>
</template>

<script setup lang="ts">
export interface TabItem {
  key: string
  label: string
  badge?: string | number
}

export interface KTabsProps {
  modelValue: string
  tabs: TabItem[]
  ariaLabel?: string
}

const props = withDefaults(defineProps<KTabsProps>(), {
  ariaLabel: 'Tabs',
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
}>()

function selectTab(key: string) {
  emit('update:modelValue', key)
}

function handleKeydown(event: KeyboardEvent, currentIndex: number) {
  let newIndex = currentIndex

  switch (event.key) {
    case 'ArrowRight':
    case 'ArrowDown':
      event.preventDefault()
      newIndex = (currentIndex + 1) % props.tabs.length
      break
    case 'ArrowLeft':
    case 'ArrowUp':
      event.preventDefault()
      newIndex = (currentIndex - 1 + props.tabs.length) % props.tabs.length
      break
    case 'Home':
      event.preventDefault()
      newIndex = 0
      break
    case 'End':
      event.preventDefault()
      newIndex = props.tabs.length - 1
      break
    default:
      return
  }

  const newTab = props.tabs[newIndex]
  emit('update:modelValue', newTab.key)

  // Focus the new tab button
  const tabEl = document.getElementById(`tab-${newTab.key}`)
  tabEl?.focus()
}
</script>

<style scoped>
.k-tabs {
  width: 100%;
}

.k-tabs__list {
  display: flex;
  gap: var(--space-1);
  border-bottom: 1px solid var(--color-border);
  padding: 0;
  margin: 0;
}

.k-tabs__tab {
  position: relative;
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-4);
  border: none;
  background: transparent;
  color: var(--color-muted);
  font-family: var(--font-family);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition:
    color var(--duration-normal) var(--ease-default),
    background var(--duration-normal) var(--ease-default);
  white-space: nowrap;
  outline: none;
  border-radius: var(--radius-sm) var(--radius-sm) 0 0;
}

.k-tabs__tab::after {
  content: '';
  position: absolute;
  bottom: -1px;
  left: 0;
  right: 0;
  height: 2px;
  background: transparent;
  transition: background var(--duration-normal) var(--ease-default);
}

.k-tabs__tab:hover:not(.k-tabs__tab--active) {
  color: var(--color-text);
  background: var(--color-surface-2);
}

.k-tabs__tab:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: -2px;
}

.k-tabs__tab--active {
  color: var(--color-primary);
}

.k-tabs__tab--active::after {
  background: var(--gradient-brand);
}

.k-tabs__tab-label {
  display: inline-flex;
  align-items: center;
}

.k-tabs__tab-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  border-radius: var(--radius-full);
  background: var(--color-surface-2);
  color: var(--color-muted);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  line-height: 1;
}

.k-tabs__tab--active .k-tabs__tab-badge {
  background: rgba(37, 99, 235, 0.12);
  color: var(--color-primary);
}
</style>
