<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useNodesStore } from '@/stores/nodes'
import { useI18n } from '@koris/composables/useI18n'
import { useApi } from '@koris/composables/useApi'
import Button from '@koris/ui/Button.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import type { NodeItem } from '@koris/types'

const { t } = useI18n()
const router = useRouter()
const store = useNodesStore()
const { get } = useApi()

// ─── State ───────────────────────────────────────────────────────────────────
const selectedNodeIds = ref<number[]>([])
const slaMap = ref<Record<number, number | null>>({})
const loadingSla = ref(false)

// ─── Computed ────────────────────────────────────────────────────────────────
const selectedNodes = computed(() =>
  store.list.filter((n) => selectedNodeIds.value.includes(n.id))
)

const canCompare = computed(() =>
  selectedNodeIds.value.length >= 2 && selectedNodeIds.value.length <= 4
)

// ─── Helpers ─────────────────────────────────────────────────────────────────
function formatBps(bps: number): string {
  if (bps < 1000) return `${bps} bps`
  if (bps < 1000000) return `${(bps / 1000).toFixed(1)} Kbps`
  return `${(bps / 1000000).toFixed(1)} Mbps`
}

function getConnectionCount(node: NodeItem): number {
  if (!node.services || !Array.isArray(node.services)) return 0
  return node.services.reduce((sum, s) => sum + ((s as any).connections ?? 0), 0)
}

function getBandwidth(node: NodeItem): string {
  const rx = node.status_metrics?.rx_bps ?? 0
  const tx = node.status_metrics?.tx_bps ?? 0
  return `↓${formatBps(rx)} ↑${formatBps(tx)}`
}

function getBarWidth(value: number): string {
  return `${Math.min(Math.max(value, 0), 100)}%`
}

function getBarColor(value: number): string {
  if (value >= 90) return 'var(--color-danger)'
  if (value >= 70) return 'var(--color-warning)'
  return 'var(--color-success)'
}

// ─── Selection ───────────────────────────────────────────────────────────────
function toggleNode(nodeId: number) {
  const idx = selectedNodeIds.value.indexOf(nodeId)
  if (idx >= 0) {
    selectedNodeIds.value.splice(idx, 1)
  } else if (selectedNodeIds.value.length < 4) {
    selectedNodeIds.value.push(nodeId)
  }
}

function isSelected(nodeId: number): boolean {
  return selectedNodeIds.value.includes(nodeId)
}

// ─── SLA Loading ─────────────────────────────────────────────────────────────
async function loadSlaForNodes() {
  loadingSla.value = true
  for (const id of selectedNodeIds.value) {
    try {
      const res = await get<any>(`/api/admin/nodes/${id}/sla`)
      if (res.ok !== false && res.availability_percent != null) {
        slaMap.value[id] = res.availability_percent
      } else {
        slaMap.value[id] = null
      }
    } catch {
      slaMap.value[id] = null
    }
  }
  loadingSla.value = false
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  if (store.list.length === 0) {
    store.loadNodes()
  }
})
</script>

<template>
  <div class="page node-compare-view">
    <header class="page-header">
      <Button variant="ghost" @click="router.push({ name: 'nodes' })">
        ← {{ t('node_compare.back') }}
      </Button>
      <h2 class="page-title">{{ t('node_compare.title') }}</h2>
    </header>

    <!-- Node Selection -->
    <section class="compare-section">
      <h3 class="section-title">{{ t('node_compare.select_nodes') }}</h3>
      <p class="section-desc text-muted">{{ t('node_compare.select_hint') }}</p>

      <div v-if="store.loading && store.list.length === 0" class="selection-grid">
        <Skeleton v-for="i in 3" :key="i" variant="rect" :width="'100%'" :height="48" />
      </div>

      <EmptyState
        v-else-if="store.list.length === 0"
        icon="🖥️"
        :title="t('nodes.no_nodes')"
        :description="t('nodes.no_nodes_desc')"
      />

      <div v-else class="selection-grid">
        <label
          v-for="node in store.list"
          :key="node.id"
          class="selection-item"
          :class="{ 'selection-item--selected': isSelected(node.id), 'selection-item--disabled': !isSelected(node.id) && selectedNodeIds.length >= 4 }"
        >
          <input
            type="checkbox"
            :checked="isSelected(node.id)"
            :disabled="!isSelected(node.id) && selectedNodeIds.length >= 4"
            class="selection-item__checkbox"
            @change="toggleNode(node.id)"
          />
          <span class="selection-item__name">{{ node.name }}</span>
          <StatusPill :status="node.status" size="sm" />
          <span class="selection-item__ip text-muted">{{ node.public_ip }}</span>
        </label>
      </div>

      <div class="compare-actions">
        <Button
          variant="primary"
          :disabled="!canCompare"
          :loading="loadingSla"
          @click="loadSlaForNodes"
        >
          {{ t('node_compare.compare') }}
        </Button>
        <span v-if="selectedNodeIds.length < 2" class="text-muted text-sm">
          {{ t('node_compare.select_min') }}
        </span>
      </div>
    </section>

    <!-- Comparison Results -->
    <section v-if="selectedNodes.length >= 2" class="compare-section">
      <h3 class="section-title">{{ t('node_compare.results') }}</h3>

      <!-- Side-by-side cards -->
      <div class="compare-grid" :class="`compare-grid--cols-${selectedNodes.length}`">
        <div v-for="node in selectedNodes" :key="node.id" class="compare-card">
          <div class="compare-card__header">
            <h4 class="compare-card__name">{{ node.name }}</h4>
            <StatusPill :status="node.status" size="sm" />
          </div>

          <div class="compare-card__metrics">
            <!-- CPU -->
            <div class="compare-metric">
              <div class="compare-metric__label">CPU</div>
              <div class="compare-metric__bar">
                <div
                  class="compare-metric__fill"
                  :style="{ width: getBarWidth(node.status_metrics?.cpu_percent ?? 0), backgroundColor: getBarColor(node.status_metrics?.cpu_percent ?? 0) }"
                />
              </div>
              <span class="compare-metric__val">{{ node.status_metrics?.cpu_percent ?? 0 }}%</span>
            </div>

            <!-- RAM -->
            <div class="compare-metric">
              <div class="compare-metric__label">RAM</div>
              <div class="compare-metric__bar">
                <div
                  class="compare-metric__fill"
                  :style="{ width: getBarWidth(node.status_metrics?.ram_percent ?? 0), backgroundColor: getBarColor(node.status_metrics?.ram_percent ?? 0) }"
                />
              </div>
              <span class="compare-metric__val">{{ node.status_metrics?.ram_percent ?? 0 }}%</span>
            </div>

            <!-- Disk -->
            <div class="compare-metric">
              <div class="compare-metric__label">{{ t('node_compare.disk') }}</div>
              <div class="compare-metric__bar">
                <div
                  class="compare-metric__fill"
                  :style="{ width: getBarWidth(node.status_metrics?.disk_percent ?? 0), backgroundColor: getBarColor(node.status_metrics?.disk_percent ?? 0) }"
                />
              </div>
              <span class="compare-metric__val">{{ node.status_metrics?.disk_percent ?? 0 }}%</span>
            </div>

            <!-- Connections -->
            <div class="compare-metric">
              <div class="compare-metric__label">{{ t('node_compare.connections') }}</div>
              <span class="compare-metric__badge">{{ getConnectionCount(node) }}</span>
            </div>

            <!-- Bandwidth -->
            <div class="compare-metric">
              <div class="compare-metric__label">{{ t('node_compare.bandwidth') }}</div>
              <span class="compare-metric__badge compare-metric__badge--mono">{{ getBandwidth(node) }}</span>
            </div>

            <!-- SLA -->
            <div class="compare-metric">
              <div class="compare-metric__label">{{ t('node_compare.sla') }}</div>
              <span
                class="compare-metric__badge"
                :class="{ 'compare-metric__badge--success': (slaMap[node.id] ?? 0) >= 99, 'compare-metric__badge--warning': (slaMap[node.id] ?? 0) < 99 && (slaMap[node.id] ?? 0) >= 90, 'compare-metric__badge--danger': (slaMap[node.id] ?? 0) < 90 }"
              >
                {{ slaMap[node.id] != null ? `${slaMap[node.id]}%` : '—' }}
              </span>
            </div>
          </div>
        </div>
      </div>

      <!-- Comparison Table -->
      <div class="compare-table-wrapper">
        <h4 class="compare-table-title">{{ t('node_compare.table_view') }}</h4>
        <div class="compare-table-scroll">
          <table class="compare-table">
            <thead>
              <tr>
                <th>{{ t('node_compare.metric') }}</th>
                <th v-for="node in selectedNodes" :key="node.id">{{ node.name }}</th>
              </tr>
            </thead>
            <tbody>
              <tr>
                <td>{{ t('node_compare.status_label') }}</td>
                <td v-for="node in selectedNodes" :key="node.id">
                  <StatusPill :status="node.status" size="sm" />
                </td>
              </tr>
              <tr>
                <td>CPU %</td>
                <td v-for="node in selectedNodes" :key="node.id">{{ node.status_metrics?.cpu_percent ?? 0 }}%</td>
              </tr>
              <tr>
                <td>RAM %</td>
                <td v-for="node in selectedNodes" :key="node.id">{{ node.status_metrics?.ram_percent ?? 0 }}%</td>
              </tr>
              <tr>
                <td>{{ t('node_compare.disk') }} %</td>
                <td v-for="node in selectedNodes" :key="node.id">{{ node.status_metrics?.disk_percent ?? 0 }}%</td>
              </tr>
              <tr>
                <td>{{ t('node_compare.connections') }}</td>
                <td v-for="node in selectedNodes" :key="node.id">{{ getConnectionCount(node) }}</td>
              </tr>
              <tr>
                <td>{{ t('node_compare.bandwidth') }}</td>
                <td v-for="node in selectedNodes" :key="node.id">{{ getBandwidth(node) }}</td>
              </tr>
              <tr>
                <td>{{ t('node_compare.sla') }}</td>
                <td v-for="node in selectedNodes" :key="node.id">
                  {{ slaMap[node.id] != null ? `${slaMap[node.id]}%` : '—' }}
                </td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.node-compare-view {
  padding: var(--space-4);
}

.page-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.page-title {
  font-size: var(--text-xl);
  font-weight: 600;
  margin: 0;
}

.compare-section {
  margin-bottom: var(--space-6);
}

.section-title {
  font-size: var(--text-lg);
  font-weight: 600;
  margin: 0 0 var(--space-1) 0;
}

.section-desc {
  margin: 0 0 var(--space-3) 0;
  font-size: var(--text-sm);
}

/* ─── Selection Grid ────────────────────────────────────────────────────── */
.selection-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-2);
  margin-bottom: var(--space-3);
}

.selection-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.selection-item:hover {
  border-color: var(--color-primary);
}

.selection-item--selected {
  border-color: var(--color-primary);
  background: var(--color-primary-subtle, rgba(var(--color-primary-rgb, 99, 102, 241), 0.06));
}

.selection-item--disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.selection-item__checkbox {
  flex-shrink: 0;
  width: 16px;
  height: 16px;
  accent-color: var(--color-primary);
}

.selection-item__name {
  font-weight: 500;
  flex: 1;
}

.selection-item__ip {
  font-size: var(--text-sm);
  font-family: var(--font-mono, monospace);
}

.compare-actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

/* ─── Compare Grid (Side-by-side cards) ─────────────────────────────────── */
.compare-grid {
  display: grid;
  gap: var(--space-3);
  margin-bottom: var(--space-5);
}

.compare-grid--cols-2 { grid-template-columns: repeat(2, 1fr); }
.compare-grid--cols-3 { grid-template-columns: repeat(3, 1fr); }
.compare-grid--cols-4 { grid-template-columns: repeat(4, 1fr); }

@media (max-width: 768px) {
  .compare-grid--cols-2,
  .compare-grid--cols-3,
  .compare-grid--cols-4 {
    grid-template-columns: 1fr;
  }
}

.compare-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  background: var(--color-surface);
}

.compare-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
  padding-bottom: var(--space-2);
  border-bottom: 1px solid var(--color-border);
}

.compare-card__name {
  margin: 0;
  font-size: var(--text-base);
  font-weight: 600;
}

.compare-card__metrics {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

/* ─── Metric rows ───────────────────────────────────────────────────────── */
.compare-metric {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.compare-metric__label {
  width: 90px;
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  flex-shrink: 0;
}

.compare-metric__bar {
  flex: 1;
  height: 8px;
  background: var(--color-border);
  border-radius: 4px;
  overflow: hidden;
}

.compare-metric__fill {
  height: 100%;
  border-radius: 4px;
  transition: width 0.3s ease;
}

.compare-metric__val {
  width: 42px;
  text-align: right;
  font-size: var(--text-sm);
  font-weight: 500;
  flex-shrink: 0;
}

.compare-metric__badge {
  font-size: var(--text-sm);
  font-weight: 500;
  padding: 2px 8px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-elevated, var(--color-border));
}

.compare-metric__badge--mono {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-xs);
}

.compare-metric__badge--success {
  color: var(--color-success);
  background: rgba(var(--color-success-rgb, 34, 197, 94), 0.1);
}

.compare-metric__badge--warning {
  color: var(--color-warning);
  background: rgba(var(--color-warning-rgb, 234, 179, 8), 0.1);
}

.compare-metric__badge--danger {
  color: var(--color-danger);
  background: rgba(var(--color-danger-rgb, 239, 68, 68), 0.1);
}

/* ─── Comparison Table ──────────────────────────────────────────────────── */
.compare-table-wrapper {
  margin-top: var(--space-4);
}

.compare-table-title {
  font-size: var(--text-base);
  font-weight: 600;
  margin: 0 0 var(--space-3) 0;
}

.compare-table-scroll {
  overflow-x: auto;
}

.compare-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.compare-table th,
.compare-table td {
  padding: var(--space-2) var(--space-3);
  text-align: left;
  border-bottom: 1px solid var(--color-border);
}

.compare-table th {
  font-weight: 600;
  background: var(--color-surface-elevated, var(--color-surface));
  white-space: nowrap;
}

.compare-table td:first-child {
  font-weight: 500;
  color: var(--color-text-muted);
  white-space: nowrap;
}
</style>
