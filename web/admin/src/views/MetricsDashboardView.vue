<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { useMetricsStore } from '@/stores/metrics'
import { useI18n } from '@koris/composables/useI18n'
import MetricsNodeCard from '@/components/metrics/MetricsNodeCard.vue'
import AlertsSummaryBar from '@/components/metrics/AlertsSummaryBar.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'

const { t } = useI18n()
const router = useRouter()
const metrics = useMetricsStore()

const rangeOptions = [
  { label: '1h', value: '1h' },
  { label: '6h', value: '6h' },
  { label: '24h', value: '24h' },
]
const selectedRange = ref<'1h' | '6h' | '24h'>('1h')

const nodesList = computed(() => {
  return Array.from(metrics.nodes.values())
})

const alerts = computed(() => metrics.alerts)
const thresholds = computed(() => metrics.thresholds)

function handleNodeClick(nodeId: number) {
  router.push({ name: 'node-detail', params: { id: nodeId } })
}

function handleAlertClick(nodeId: number) {
  router.push({ name: 'node-detail', params: { id: nodeId } })
}

onMounted(() => {
  metrics.connect()
})

onUnmounted(() => {
  metrics.disconnect()
})
</script>

<template>
  <div class="page metrics-dashboard-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('metrics.dashboard_title') }}</h2>
      <div class="page-header__actions">
        <KSelect
          v-model="selectedRange"
          :options="rangeOptions"
          size="sm"
        />
      </div>
    </header>

    <!-- Alerts Bar -->
    <AlertsSummaryBar
      :alerts="alerts"
      @alert-click="handleAlertClick"
    />

    <!-- Connection Status -->
    <div v-if="!metrics.connected" class="connection-banner">
      <span>{{ t('metrics.connecting') }}</span>
    </div>

    <!-- Nodes Grid -->
    <div v-if="nodesList.length === 0 && !metrics.connected" class="loading-grid">
      <KSkeleton v-for="i in 4" :key="i" variant="rect" :width="'100%'" :height="280" />
    </div>

    <KEmptyState
      v-else-if="nodesList.length === 0"
      icon="📊"
      :title="t('metrics.no_nodes')"
      :description="t('metrics.no_nodes_desc')"
    />

    <div v-else class="nodes-grid">
      <MetricsNodeCard
        v-for="node in nodesList"
        :key="node.nodeId"
        :node-id="node.nodeId"
        :name="node.name"
        :status="node.status"
        :cpu="node.cpu"
        :ram="node.ram"
        :disk="node.disk"
        :rx-bps="node.rxBps"
        :tx-bps="node.txBps"
        :sessions="node.sessions"
        :uptime="node.uptime"
        :alert-thresholds="thresholds"
        @click="handleNodeClick(node.nodeId)"
      />
    </div>
  </div>
</template>

<style scoped>
.metrics-dashboard-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-3);
}

.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.page-header__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.connection-banner {
  padding: var(--space-2) var(--space-4);
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-primary);
  text-align: center;
}

.loading-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--space-5);
}

.nodes-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(320px, 1fr));
  gap: var(--space-5);
}
</style>
