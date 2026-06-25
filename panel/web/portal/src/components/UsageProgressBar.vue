<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  used: number    // bytes
  total: number   // bytes
}>()

const percentage = computed(() => {
  if (props.total <= 0) return 0
  return Math.min(100, Math.round((props.used / props.total) * 100))
})

const isWarning = computed(() => percentage.value >= 80)

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}
</script>

<template>
  <div class="usage-progress">
    <div class="usage-progress__header">
      <span class="usage-progress__label">{{ formatBytes(used) }} / {{ formatBytes(total) }}</span>
      <span class="usage-progress__percent" :class="{ 'usage-progress__percent--warning': isWarning }">
        {{ percentage }}%
      </span>
    </div>
    <div class="usage-progress__track">
      <div
        class="usage-progress__fill"
        :class="{ 'usage-progress__fill--warning': isWarning }"
        :style="{ width: `${percentage}%` }"
        role="progressbar"
        :aria-valuenow="percentage"
        aria-valuemin="0"
        aria-valuemax="100"
      />
    </div>
  </div>
</template>

<style scoped>
.usage-progress {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.usage-progress__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.usage-progress__label {
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.usage-progress__percent {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

.usage-progress__percent--warning {
  color: var(--color-warning);
}

.usage-progress__track {
  width: 100%;
  height: 8px;
  background: var(--color-border);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.usage-progress__fill {
  height: 100%;
  background: var(--color-primary);
  border-radius: var(--radius-full);
  transition: width 0.3s ease;
}

.usage-progress__fill--warning {
  background: var(--color-warning);
}
</style>
