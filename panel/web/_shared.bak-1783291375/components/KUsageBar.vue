<template>
  <div
    :class="['k-usage-bar', `k-usage-bar--${size}`]"
    role="progressbar"
    :aria-valuenow="limit > 0 ? Math.min(100, Math.round((used / limit) * 100)) : undefined"
    :aria-valuemin="0"
    :aria-valuemax="100"
    :aria-label="ariaLabel"
  >
    <div v-if="limit > 0" class="k-usage-bar__track">
      <div
        :class="['k-usage-bar__fill', `k-usage-bar__fill--${colorClass}`]"
        :style="{ width: `${fillPercent}%` }"
      />
    </div>
    <span v-if="showLabel" class="k-usage-bar__label">
      {{ labelText }}
    </span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface KUsageBarProps {
  used: number         // Bytes used in current period
  limit: number        // Bytes limit (0 = unlimited)
  showLabel?: boolean  // Show "X GB / Y GB" text (default: true)
  size?: 'sm' | 'md'  // Height variant
}

const props = withDefaults(defineProps<KUsageBarProps>(), {
  showLabel: true,
  size: 'md',
})

/** Percentage of limit used, capped at 100% visually */
const fillPercent = computed(() => {
  if (props.limit <= 0) return 0
  return Math.min(100, (props.used / props.limit) * 100)
})

/** Ratio of used/limit for color thresholds */
const ratio = computed(() => {
  if (props.limit <= 0) return 0
  return props.used / props.limit
})

/** Color class based on usage ratio */
const colorClass = computed<'normal' | 'warning' | 'error'>(() => {
  if (ratio.value > 1.0) return 'error'
  if (ratio.value > 0.8) return 'warning'
  return 'normal'
})

/** Label text: "X GB / Y GB" or "X GB / Unlimited" */
const labelText = computed(() => {
  if (props.limit === 0) {
    return `${formatBytes(props.used)} / Unlimited`
  }
  return `${formatBytes(props.used)} / ${formatBytes(props.limit)}`
})

/** Aria label for accessibility */
const ariaLabel = computed(() => {
  if (props.limit === 0) {
    return `Usage: ${formatBytes(props.used)}, Unlimited`
  }
  return `Usage: ${formatBytes(props.used)} of ${formatBytes(props.limit)}`
})

/**
 * Converts a byte value to a human-readable string.
 * Produces output matching "X.X UNIT" where UNIT is B, KB, MB, GB, or TB.
 */
function formatBytes(bytes: number): string {
  if (bytes < 0) bytes = 0

  const units = ['B', 'KB', 'MB', 'GB', 'TB'] as const
  const base = 1024

  if (bytes === 0) {
    return '0.0 B'
  }

  let unitIndex = 0
  let value = bytes

  while (value >= base && unitIndex < units.length - 1) {
    value /= base
    unitIndex++
  }

  return `${value.toFixed(1)} ${units[unitIndex]}`
}
</script>

<style scoped>
.k-usage-bar {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  width: 100%;
}

/* ─── Track ─── */

.k-usage-bar__track {
  width: 100%;
  background: var(--color-surface-2);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.k-usage-bar--sm .k-usage-bar__track {
  height: 4px;
}

.k-usage-bar--md .k-usage-bar__track {
  height: 6px;
}

/* ─── Fill ─── */

.k-usage-bar__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width var(--duration-normal) var(--ease-out);
}

.k-usage-bar__fill--normal {
  background: var(--color-primary);
}

.k-usage-bar__fill--warning {
  background: var(--color-warning);
}

.k-usage-bar__fill--error {
  background: var(--color-danger);
}

/* ─── Label ─── */

.k-usage-bar__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  font-family: var(--font-family);
  line-height: 1;
}
</style>
