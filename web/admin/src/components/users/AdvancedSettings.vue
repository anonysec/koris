<script setup lang="ts">
/**
 * AdvancedSettings — Inline row for speed and connection limit.
 * Minimal style: two small fields side by side with labels.
 */
import { computed } from 'vue'
import KInput from '@koris/ui/KInput.vue'

const props = defineProps<{
  speedLimit: number
  connectionLimit: number
}>()

const emit = defineEmits<{
  (e: 'update:speedLimit', value: number): void
  (e: 'update:connectionLimit', value: number): void
}>()

function onSpeedInput(value: string | number) {
  emit('update:speedLimit', Number(value) || 0)
}

function onConnectionInput(value: string | number) {
  emit('update:connectionLimit', Number(value) || 0)
}
</script>

<template>
  <div class="advanced-settings">
    <div class="advanced-settings__field">
      <label class="advanced-settings__label">Speed Limit (Mbps)</label>
      <KInput
        :model-value="String(props.speedLimit)"
        type="number"
        placeholder="0 = unlimited"
        @update:model-value="onSpeedInput"
      />
    </div>
    <div class="advanced-settings__field">
      <label class="advanced-settings__label">Connection Limit</label>
      <KInput
        :model-value="String(props.connectionLimit)"
        type="number"
        placeholder="0 = unlimited"
        @update:model-value="onConnectionInput"
      />
    </div>
  </div>
</template>

<style scoped>
.advanced-settings {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3, 12px);
}

.advanced-settings__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
}

.advanced-settings__label {
  font-size: 12px;
  color: var(--color-muted, #8b98a5);
  font-weight: 500;
}
</style>
