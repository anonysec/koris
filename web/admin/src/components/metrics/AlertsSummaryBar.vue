<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import type { Alert } from '@/stores/metrics'

const props = defineProps<{
  alerts: Alert[]
}>()

const emit = defineEmits<{
  'alert-click': [nodeId: number]
}>()

const { t } = useI18n()

const visible = computed(() => props.alerts.length > 0)

function timeSince(dateStr: string): string {
  const diff = Date.now() - new Date(dateStr).getTime()
  const minutes = Math.floor(diff / 60000)
  if (minutes < 1) return t('metrics.just_now')
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  return `${Math.floor(hours / 24)}d`
}

function typeBadgeClass(type: string): string {
  switch (type) {
    case 'cpu': return 'type-badge--cpu'
    case 'ram': return 'type-badge--ram'
    case 'disk': return 'type-badge--disk'
    default: return ''
  }
}
</script>

<template>
  <div v-if="visible" class="alerts-summary-bar" role="alert">
    <div class="alerts-summary-bar__icon">⚠️</div>
    <div class="alerts-summary-bar__list">
      <button
        v-for="(alert, idx) in alerts"
        :key="`${alert.nodeId}-${alert.type}-${idx}`"
        class="alert-chip"
        @click="emit('alert-click', alert.nodeId)"
      >
        <span class="alert-chip__node">{{ alert.nodeName }}</span>
        <span class="type-badge" :class="typeBadgeClass(alert.type)">{{ alert.type.toUpperCase() }}</span>
        <span class="alert-chip__value">{{ Math.round(alert.value) }}%</span>
        <span class="alert-chip__threshold">&gt; {{ alert.threshold }}%</span>
        <span class="alert-chip__time">{{ timeSince(alert.since) }}</span>
      </button>
    </div>
  </div>
</template>

<style scoped>
.alerts-summary-bar {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: rgba(var(--color-warning-rgb, 245, 158, 11), 0.08);
  border: 1px solid var(--color-warning);
  border-radius: var(--radius-lg);
  overflow-x: auto;
}

.alerts-summary-bar__icon {
  font-size: var(--text-lg);
  flex-shrink: 0;
}

.alerts-summary-bar__list {
  display: flex;
  gap: var(--space-2);
  flex-wrap: nowrap;
  overflow-x: auto;
}

.alert-chip {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  font-size: var(--text-xs);
  white-space: nowrap;
  cursor: pointer;
  transition: border-color var(--duration-normal) ease;
}

.alert-chip:hover {
  border-color: var(--color-primary);
}

.alert-chip__node {
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

.type-badge {
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  font-size: 9px;
  font-weight: var(--font-bold);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.type-badge--cpu {
  background: rgba(239, 68, 68, 0.15);
  color: var(--color-danger);
}

.type-badge--ram {
  background: rgba(245, 158, 11, 0.15);
  color: var(--color-warning);
}

.type-badge--disk {
  background: rgba(59, 130, 246, 0.15);
  color: var(--color-primary);
}

.alert-chip__value {
  font-weight: var(--font-semibold);
  color: var(--color-danger);
}

.alert-chip__threshold {
  color: var(--color-muted);
}

.alert-chip__time {
  color: var(--color-muted);
  font-style: italic;
}
</style>
