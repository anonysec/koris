<script setup lang="ts">
/**
 * AdvancedSettings — Collapsible section for speed and connection limit fields.
 *
 * Collapsed by default, displays a one-line summary (speed Mbps / connection limit or "unlimited").
 * Clicking the header toggles expansion with a 200ms CSS grid height transition.
 * Respects prefers-reduced-motion (instant transition when enabled).
 *
 * Requirements: 10.1, 10.2, 10.3, 10.4, 10.5
 */
import { ref, computed, useId } from 'vue'
import { formatAdvancedSummary } from '@/utils/formatAdvancedSummary'
import KInput from '@koris/ui/KInput.vue'
import KFormField from '@koris/ui/KFormField.vue'

const props = defineProps<{
  speedLimit: number
  connectionLimit: number
}>()

const emit = defineEmits<{
  (e: 'update:speedLimit', value: number): void
  (e: 'update:connectionLimit', value: number): void
}>()

const expanded = ref(false)
const panelId = useId()

const summary = computed(() => formatAdvancedSummary(props.speedLimit, props.connectionLimit))

function toggle() {
  expanded.value = !expanded.value
}

function onSpeedInput(value: string | number) {
  emit('update:speedLimit', Number(value) || 0)
}

function onConnectionInput(value: string | number) {
  emit('update:connectionLimit', Number(value) || 0)
}
</script>

<template>
  <div class="advanced-settings">
    <button
      type="button"
      class="advanced-settings__header"
      :aria-expanded="expanded"
      :aria-controls="panelId"
      @click="toggle"
    >
      <span class="advanced-settings__title">Advanced Settings</span>
      <span v-if="!expanded" class="advanced-settings__summary">{{ summary }}</span>
    </button>

    <div
      :id="panelId"
      class="advanced-settings__panel"
      :class="{ 'advanced-settings__panel--open': expanded }"
      role="region"
      :aria-hidden="!expanded"
    >
      <div class="advanced-settings__panel-inner">
        <div class="advanced-settings__fields">
          <KFormField name="speed-limit" label="Speed Limit (Mbps)">
            <template #default="{ fieldId }">
              <KInput
                :id="fieldId"
                :model-value="String(props.speedLimit)"
                type="number"
                placeholder="0 = unlimited"
                @update:model-value="onSpeedInput"
              />
            </template>
          </KFormField>

          <KFormField name="connection-limit" label="Connection Limit">
            <template #default="{ fieldId }">
              <KInput
                :id="fieldId"
                :model-value="String(props.connectionLimit)"
                type="number"
                placeholder="0 = unlimited"
                @update:model-value="onConnectionInput"
              />
            </template>
          </KFormField>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.advanced-settings {
  border-top: 1px solid var(--color-border, #2d3748);
  padding-top: var(--space-3, 0.75rem);
}

.advanced-settings__header {
  display: flex;
  align-items: center;
  gap: var(--space-3, 0.75rem);
  width: 100%;
  padding: var(--space-2, 0.5rem) 0;
  border: none;
  background: transparent;
  color: var(--color-text, #e6edf3);
  cursor: pointer;
  text-align: left;
  font-size: 0.875rem;
  font-weight: 500;
  border-radius: var(--radius-sm, 6px);
}

.advanced-settings__header:hover {
  color: var(--color-primary, #2563eb);
}

.advanced-settings__header:focus-visible {
  outline: 2px solid var(--color-primary, #2563eb);
  outline-offset: 2px;
}

.advanced-settings__title {
  flex-shrink: 0;
}

.advanced-settings__summary {
  color: var(--color-muted, #8b98a5);
  font-weight: 400;
  font-size: 0.8125rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

/* Expandable panel — uses CSS grid for smooth height animation */
.advanced-settings__panel {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 200ms ease-out;
}

.advanced-settings__panel--open {
  grid-template-rows: 1fr;
}

.advanced-settings__panel-inner {
  overflow: hidden;
}

.advanced-settings__fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-3, 0.75rem);
  padding-top: var(--space-3, 0.75rem);
}

/* Respect reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  .advanced-settings__panel {
    transition-duration: 0ms;
  }
}
</style>
