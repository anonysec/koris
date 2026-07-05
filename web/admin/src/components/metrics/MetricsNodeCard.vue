<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import MetricsGauge from './MetricsGauge.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'

export interface MetricsNodeCardProps {
  nodeId: number
  name: string
  status: 'online' | 'offline' | 'stale'
  cpu: number
  ram: number
  disk: number
  rxBps: number
  txBps: number
  sessions: number
  uptime: number
  alertThresholds: { cpu: number; ram: number; disk: number }
}

const props = defineProps<MetricsNodeCardProps>()
const emit = defineEmits<{ click: [] }>()
const { t } = useI18n()

const hasWarning = computed(() => {
  return (
    props.cpu > props.alertThresholds.cpu ||
    props.ram > props.alertThresholds.ram ||
    props.disk > props.alertThresholds.disk
  )
})

function formatBps(bps: number): string {
  if (bps < 1000) return `${bps} B/s`
  if (bps < 1000000) return `${(bps / 1000).toFixed(1)} KB/s`
  if (bps < 1000000000) return `${(bps / 1000000).toFixed(1)} MB/s`
  return `${(bps / 1000000000).toFixed(2)} GB/s`
}

function formatUptime(seconds: number): string {
  if (!seconds) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h`
  return `${Math.floor(seconds / 60)}m`
}
</script>

<template>
  <div
    class="metrics-node-card"
    :class="{ 'metrics-node-card--warning': hasWarning, 'metrics-node-card--offline': status === 'offline' }"
    role="button"
    tabindex="0"
    @click="emit('click')"
    @keydown.enter="emit('click')"
  >
    <!-- Warning overlay -->
    <div v-if="hasWarning" class="warning-indicator" />

    <!-- Header -->
    <div class="card-header">
      <h4 class="card-header__name">{{ name }}</h4>
      <KStatusPill :status="status" size="sm" />
    </div>

    <!-- Mini Gauges -->
    <div class="gauges-row">
      <MetricsGauge :value="cpu" label="CPU" :threshold="alertThresholds.cpu" />
      <MetricsGauge :value="ram" label="RAM" :threshold="alertThresholds.ram" />
      <MetricsGauge :value="disk" label="Disk" :threshold="alertThresholds.disk" />
    </div>

    <!-- Stats Row -->
    <div class="stats-row">
      <div class="stat">
        <span class="stat__label">↓ RX</span>
        <span class="stat__value">{{ formatBps(rxBps) }}</span>
      </div>
      <div class="stat">
        <span class="stat__label">↑ TX</span>
        <span class="stat__value">{{ formatBps(txBps) }}</span>
      </div>
      <div class="stat">
        <span class="stat__label">{{ t('metrics.sessions') }}</span>
        <span class="stat__value">{{ sessions }}</span>
      </div>
      <div class="stat">
        <span class="stat__label">{{ t('metrics.uptime') }}</span>
        <span class="stat__value">{{ formatUptime(uptime) }}</span>
      </div>
    </div>
  </div>
</template>

<style scoped>
.metrics-node-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  cursor: pointer;
  transition: border-color var(--duration-normal) ease, box-shadow var(--duration-normal) ease;
  position: relative;
  overflow: hidden;
}

.metrics-node-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 12px rgba(91, 157, 255, 0.1);
}

.metrics-node-card:focus-visible {
  outline: 2px solid var(--color-primary);
  outline-offset: 2px;
}

.metrics-node-card--warning {
  border-color: var(--color-warning);
}

.metrics-node-card--offline {
  opacity: 0.6;
}

.warning-indicator {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  height: 3px;
  background: var(--color-warning);
}

.card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.card-header__name {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.gauges-row {
  display: flex;
  justify-content: space-around;
  gap: var(--space-3);
}

.gauges-row :deep(.metrics-gauge) {
  transform: scale(0.7);
}

.stats-row {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: var(--space-2);
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-3);
}

.stat {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
}

.stat__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.stat__value {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}
</style>
