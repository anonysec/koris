<template>
  <div class="k-expiry-chips" role="group" aria-label="Expiry shortcut chips">
    <button
      v-for="chip in chipsList"
      :key="chip"
      type="button"
      :class="['k-expiry-chips__chip', { 'k-expiry-chips__chip--active': isActive(chip) }]"
      :aria-pressed="isActive(chip)"
      @click="selectChip(chip)"
    >
      {{ chip }}
    </button>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export type ExpiryOffset = '+1d' | '+7d' | '+1m' | '+2m' | '+3m' | '+6m' | '+1y'

export interface KExpiryChipsProps {
  modelValue?: string
  chips?: string[]
}

const props = withDefaults(defineProps<KExpiryChipsProps>(), {
  modelValue: undefined,
  chips: () => ['+1d', '+7d', '+1m', '+2m', '+3m', '+6m', '+1y'],
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const chipsList = computed(() => props.chips)

/**
 * Computes a target date from a base date plus an offset string.
 * Days are added directly; months/years via calendar month addition (clamped to month end).
 */
function computeExpiryDate(baseDate: Date, offset: ExpiryOffset): Date {
  const result = new Date(baseDate.getTime())

  switch (offset) {
    case '+1d':
      result.setDate(result.getDate() + 1)
      break
    case '+7d':
      result.setDate(result.getDate() + 7)
      break
    case '+1m':
      addMonths(result, 1)
      break
    case '+2m':
      addMonths(result, 2)
      break
    case '+3m':
      addMonths(result, 3)
      break
    case '+6m':
      addMonths(result, 6)
      break
    case '+1y':
      addMonths(result, 12)
      break
  }

  return result
}

function addMonths(date: Date, months: number): void {
  const originalDay = date.getDate()
  date.setMonth(date.getMonth() + months)

  if (date.getDate() !== originalDay) {
    date.setDate(0)
  }
}

function selectChip(chip: string): void {
  const now = new Date()
  const target = computeExpiryDate(now, chip as ExpiryOffset)
  emit('update:modelValue', target.toISOString())
}

function isActive(chip: string): boolean {
  if (!props.modelValue) return false

  const now = new Date()
  const expected = computeExpiryDate(now, chip as ExpiryOffset)
  const current = new Date(props.modelValue)

  // Compare dates within a 1-minute tolerance (since "today" shifts)
  return Math.abs(expected.getTime() - current.getTime()) < 60_000
}
</script>

<style scoped>
.k-expiry-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.k-expiry-chips__chip {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1) var(--space-3);
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
  background: var(--color-surface-2);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: var(--font-family);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition: all var(--duration-fast) var(--ease-out);
  white-space: nowrap;
  line-height: 1;
  user-select: none;
}

.k-expiry-chips__chip:hover {
  border-color: var(--color-primary);
  color: var(--color-primary);
  background: rgba(37, 99, 235, 0.08);
}

.k-expiry-chips__chip:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.k-expiry-chips__chip--active {
  border-color: var(--color-primary);
  background: var(--color-primary);
  color: #fff;
}

.k-expiry-chips__chip--active:hover {
  background: var(--color-primary);
  color: #fff;
  opacity: 0.9;
}
</style>
