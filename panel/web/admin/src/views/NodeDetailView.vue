<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useNodesStore } from '@/stores/nodes'
import { useRealtimeStore } from '@/stores/realtime'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import { useApi } from '@koris/composables/useApi'
import KButton from '@koris/ui/KButton.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import KChart from '@koris/ui/KChart.vue'

const props = defineProps<{ id: string }>()

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const store = useNodesStore()
const realtime = useRealtimeStore()
const toast = useToast()
const { get, post } = useApi()

// ─── State ───────────────────────────────────────────────────────────────────
const loading = ref(true)
const node = ref<any>(null)
const slaData = ref<any>(null)
const nodeTasks = ref<any[]>([])
const protocols = ref<any[]>([])
const loadingSla = ref(false)
const loadingTasks = ref(false)
const loadingProtocols = ref(false)

// ─── Active Sessions (from realtime store, filtered by this node) ────────────
const activeSessions = computed(() => {
  if (!node.value) return []
  return realtime.liveSessions.filter(
    (s) => s.node_name === node.value.name
  )
})

// ─── Health Timeline Data ────────────────────────────────────────────────────
const healthTimelineData = computed(() => {
  if (!slaData.value?.downtimes || !Array.isArray(slaData.value.downtimes)) return []
  // Build a simple timeline from downtime events for the last 30 days
  const now = Date.now()
  const thirtyDaysAgo = now - 30 * 24 * 60 * 60 * 1000
  const downtimes = slaData.value.downtimes.filter((d: any) => {
    const start = new Date(d.started_at).getTime()
    return start >= thirtyDaysAgo
  })
  return downtimes
})

const slaPercentage = computed(() => {
  if (!slaData.value?.availability_percent) return null
  return slaData.value.availability_percent
})

const healthChartData = computed(() => {
  if (!slaData.value?.downtimes || !Array.isArray(slaData.value.downtimes)) {
    // Show 30 days all up
    const data = []
    for (let i = 29; i >= 0; i--) {
      const date = new Date(Date.now() - i * 24 * 60 * 60 * 1000)
      data.push({
        label: `${date.getMonth() + 1}/${date.getDate()}`,
        value: 100,
        color: 'var(--color-success)',
      })
    }
    return data
  }

  // Calculate daily uptime % over last 30 days
  const now = new Date()
  const data = []
  for (let i = 29; i >= 0; i--) {
    const dayStart = new Date(now)
    dayStart.setHours(0, 0, 0, 0)
    dayStart.setDate(dayStart.getDate() - i)
    const dayEnd = new Date(dayStart)
    dayEnd.setDate(dayEnd.getDate() + 1)

    let downtimeMs = 0
    for (const d of slaData.value.downtimes) {
      const start = new Date(d.started_at).getTime()
      const end = d.ended_at ? new Date(d.ended_at).getTime() : Date.now()
      const overlapStart = Math.max(start, dayStart.getTime())
      const overlapEnd = Math.min(end, dayEnd.getTime())
      if (overlapStart < overlapEnd) {
        downtimeMs += overlapEnd - overlapStart
      }
    }

    const dayMs = 24 * 60 * 60 * 1000
    const uptimePercent = Math.max(0, Math.round((1 - downtimeMs / dayMs) * 100))
    data.push({
      label: `${dayStart.getMonth() + 1}/${dayStart.getDate()}`,
      value: uptimePercent,
      color: uptimePercent >= 99 ? 'var(--color-success)' : uptimePercent >= 90 ? 'var(--color-warning)' : 'var(--color-danger)',
    })
  }
  return data
})

// ─── Protocol Helpers ────────────────────────────────────────────────────────
const protocolIcons: Record<string, string> = {
  openvpn: '🔐',
  l2tp: '🔒',
  ikev2: '🛡️',
  ssh: '🖥️',
  wireguard: '⚡',
  cisco_ipsec: '🏢',
}
const protocolLabels: Record<string, string> = {
  openvpn: 'OpenVPN',
  l2tp: 'L2TP',
  ikev2: 'IKEv2',
  ssh: 'SSH',
  wireguard: 'WireGuard',
  cisco_ipsec: 'Cisco IPSec',
}

function getProtocolStatus(protocol: string): string {
  if (!node.value) return 'unknown'
  if (node.value.status === 'offline' || node.value.status === 'disabled') return 'offline'
  if (node.value.services && Array.isArray(node.value.services)) {
    const svc = node.value.services.find((s: any) => s.service === protocol || s.name === protocol)
    if (svc?.status) return svc.status
  }
  const metrics = node.value.status_metrics
  if (metrics) {
    if (protocol === 'openvpn' && metrics.openvpn_status) return metrics.openvpn_status
    if (protocol === 'l2tp' && metrics.l2tp_status) return metrics.l2tp_status
    if (protocol === 'ikev2' && metrics.ikev2_status) return metrics.ikev2_status
    if (protocol === 'ssh' && metrics.ssh_status) return metrics.ssh_status
  }
  return 'unknown'
}

// ─── Task Helpers ────────────────────────────────────────────────────────────
function formatTaskTime(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString()
}

// ─── Protocol Actions ────────────────────────────────────────────────────────
const actionLoading = ref<string | null>(null)

async function handleProtocolAction(protocol: string, action: 'start' | 'stop' | 'restart') {
  if (!node.value) return
  actionLoading.value = `${protocol}-${action}`
  try {
    const success = await store.createNodeTask({
      node_id: node.value.id,
      action: `${action}_service`,
      payload_json: { protocol },
    })
    if (success) {
      toast.success(t('node_detail.action_success'))
    } else {
      toast.error(t('node_detail.action_error'))
    }
  } catch {
    toast.error(t('node_detail.action_error'))
  } finally {
    actionLoading.value = null
  }
}

// ─── Data Loading ────────────────────────────────────────────────────────────
async function loadNodeDetail() {
  loading.value = true
  try {
    const res = await get<any>(`/api/nodes/${props.id}`)
    if (res.ok !== false) {
      node.value = res.node || res
    }
  } catch {
    toast.error(t('node_detail.load_error'))
  } finally {
    loading.value = false
  }
}

async function loadSla() {
  loadingSla.value = true
  try {
    const res = await get<any>(`/api/admin/nodes/${props.id}/sla`)
    if (res.ok !== false) {
      slaData.value = res
    }
  } catch {
    // SLA endpoint may not exist yet; silently ignore
  } finally {
    loadingSla.value = false
  }
}

async function loadRecentTasks() {
  loadingTasks.value = true
  try {
    const res = await get<any>(`/api/node/tasks?node_id=${props.id}&limit=10`)
    if (res.ok !== false) {
      nodeTasks.value = (res.tasks || []).slice(0, 10)
    }
  } catch {
    // Fallback: load all tasks and filter
    try {
      await store.loadNodeTasks()
      nodeTasks.value = store.tasks
        .filter((t) => t.node_id === Number(props.id))
        .slice(0, 10)
    } catch {
      // Silently fail
    }
  } finally {
    loadingTasks.value = false
  }
}

async function loadProtocols() {
  loadingProtocols.value = true
  try {
    await store.loadNodeVpnConfigs(Number(props.id))
    protocols.value = store.vpnConfigs[Number(props.id)] || []
  } catch {
    // Silently fail
  } finally {
    loadingProtocols.value = false
  }
}

// ─── Bytes Formatting ────────────────────────────────────────────────────────
function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatDuration(seconds: number): string {
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  return `${Math.floor(seconds / 3600)}h ${Math.floor((seconds % 3600) / 60)}m`
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(async () => {
  await loadNodeDetail()
  // Load supplementary data in parallel
  loadSla()
  loadRecentTasks()
  loadProtocols()
})
</script>

<template>
  <div class="page node-detail-view">
    <!-- Header -->
    <header class="page-header">
      <KButton variant="ghost" size="sm" @click="router.push({ name: 'nodes' })">
        ← {{ t('node_detail.back') }}
      </KButton>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="node-detail-loading">
      <KSkeleton variant="rect" :width="'100%'" :height="80" />
      <KSkeleton variant="rect" :width="'100%'" :height="200" />
      <KSkeleton variant="rect" :width="'100%'" :height="150" />
    </div>

    <template v-else-if="node">
      <!-- Node Summary Card -->
      <div class="node-summary">
        <div class="node-summary__info">
          <h2 class="node-summary__name">{{ node.name }}</h2>
          <KStatusPill :status="node.status" size="sm" />
        </div>
        <div class="node-summary__meta">
          <span class="text-muted">{{ node.public_ip }}</span>
          <span v-if="node.domain" class="text-muted">{{ node.domain }}</span>
        </div>
        <div v-if="node.status_metrics" class="node-summary__metrics">
          <div class="metric-badge">
            <span class="metric-badge__label">CPU</span>
            <span class="metric-badge__value">{{ node.status_metrics.cpu_percent ?? 0 }}%</span>
          </div>
          <div class="metric-badge">
            <span class="metric-badge__label">RAM</span>
            <span class="metric-badge__value">{{ node.status_metrics.ram_percent ?? 0 }}%</span>
          </div>
          <div class="metric-badge">
            <span class="metric-badge__label">Disk</span>
            <span class="metric-badge__value">{{ node.status_metrics.disk_percent ?? 0 }}%</span>
          </div>
        </div>
      </div>

      <!-- Section Grid -->
      <div class="detail-grid">
        <!-- Health Timeline Chart -->
        <section class="detail-section">
          <h3 class="detail-section__title">{{ t('node_detail.health_timeline') }}</h3>
          <div v-if="loadingSla" class="detail-section__loading">
            <KSkeleton variant="rect" :width="'100%'" :height="160" />
          </div>
          <div v-else class="health-timeline">
            <div v-if="slaPercentage !== null" class="sla-badge">
              <span class="sla-badge__label">{{ t('node_detail.sla_availability') }}</span>
              <span
                class="sla-badge__value"
                :class="{
                  'sla-badge__value--good': slaPercentage >= 99,
                  'sla-badge__value--warn': slaPercentage >= 90 && slaPercentage < 99,
                  'sla-badge__value--bad': slaPercentage < 90,
                }"
              >{{ slaPercentage.toFixed(2) }}%</span>
            </div>
            <KChart
              type="bar"
              :data="healthChartData"
              :height="140"
              :interactive="true"
              :options="{ showGrid: true, showTooltip: true, yAxisFormat: (v: number) => `${v}%` }"
            />
            <p class="text-muted text-sm health-timeline__hint">
              {{ t('node_detail.health_hint') }}
            </p>
          </div>
        </section>

        <!-- Active Sessions -->
        <section class="detail-section">
          <h3 class="detail-section__title">
            {{ t('node_detail.active_sessions') }}
            <span class="detail-section__count">{{ activeSessions.length }}</span>
          </h3>
          <KEmptyState
            v-if="activeSessions.length === 0"
            icon="👤"
            :title="t('node_detail.no_sessions')"
            :description="t('node_detail.no_sessions_desc')"
          />
          <div v-else class="sessions-list">
            <div
              v-for="session in activeSessions"
              :key="session.id"
              class="session-item"
            >
              <div class="session-item__user">
                <span class="session-item__username">{{ session.username }}</span>
                <span class="session-item__ip text-muted">{{ session.framed_ip }}</span>
              </div>
              <div class="session-item__stats">
                <span class="session-item__duration">{{ formatDuration(session.session_seconds) }}</span>
                <span class="session-item__traffic text-muted">
                  ↓{{ formatBytes(session.input_bytes) }} ↑{{ formatBytes(session.output_bytes) }}
                </span>
              </div>
            </div>
          </div>
        </section>

        <!-- Protocol Controls -->
        <section class="detail-section">
          <h3 class="detail-section__title">{{ t('node_detail.protocol_controls') }}</h3>
          <div v-if="loadingProtocols" class="detail-section__loading">
            <KSkeleton variant="rect" :width="'100%'" :height="120" />
          </div>
          <KEmptyState
            v-else-if="protocols.length === 0"
            icon="⚡"
            :title="t('node_detail.no_protocols')"
            :description="t('node_detail.no_protocols_desc')"
          />
          <div v-else class="protocols-list">
            <div
              v-for="proto in protocols"
              :key="proto.protocol"
              class="protocol-row"
            >
              <div class="protocol-row__info">
                <span class="protocol-row__icon">{{ protocolIcons[proto.protocol] || '🔌' }}</span>
                <span class="protocol-row__name">{{ protocolLabels[proto.protocol] || proto.protocol }}</span>
                <KStatusPill :status="getProtocolStatus(proto.protocol)" size="sm" />
              </div>
              <div class="protocol-row__actions">
                <KButton
                  variant="ghost"
                  size="sm"
                  :loading="actionLoading === `${proto.protocol}-start`"
                  :disabled="!proto.enabled"
                  @click="handleProtocolAction(proto.protocol, 'start')"
                >
                  {{ t('node_detail.start') }}
                </KButton>
                <KButton
                  variant="ghost"
                  size="sm"
                  :loading="actionLoading === `${proto.protocol}-stop`"
                  :disabled="!proto.enabled"
                  @click="handleProtocolAction(proto.protocol, 'stop')"
                >
                  {{ t('node_detail.stop') }}
                </KButton>
                <KButton
                  variant="ghost"
                  size="sm"
                  :loading="actionLoading === `${proto.protocol}-restart`"
                  :disabled="!proto.enabled"
                  @click="handleProtocolAction(proto.protocol, 'restart')"
                >
                  {{ t('node_detail.restart') }}
                </KButton>
              </div>
            </div>
          </div>
        </section>

        <!-- Recent Tasks -->
        <section class="detail-section">
          <h3 class="detail-section__title">{{ t('node_detail.recent_tasks') }}</h3>
          <div v-if="loadingTasks" class="detail-section__loading">
            <KSkeleton variant="rect" :width="'100%'" :height="120" />
          </div>
          <KEmptyState
            v-else-if="nodeTasks.length === 0"
            icon="📋"
            :title="t('node_detail.no_tasks')"
            :description="t('node_detail.no_tasks_desc')"
          />
          <div v-else class="tasks-list">
            <div
              v-for="task in nodeTasks"
              :key="task.id"
              class="task-item"
            >
              <div class="task-item__info">
                <span class="task-item__action">{{ task.action }}</span>
                <KStatusPill :status="task.status" size="sm" />
              </div>
              <div class="task-item__meta">
                <span class="task-item__time text-muted">{{ formatTaskTime(task.created_at) }}</span>
                <span v-if="task.error" class="task-item__error text-danger">{{ task.error }}</span>
              </div>
            </div>
          </div>
        </section>
      </div>
    </template>

    <!-- Not Found -->
    <KEmptyState
      v-else
      icon="🖥️"
      :title="t('node_detail.not_found')"
      :description="t('node_detail.not_found_desc')"
    />
  </div>
</template>

<style scoped>
.node-detail-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.node-detail-loading {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

/* ─── Node Summary ─── */
.node-summary {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.node-summary__info {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.node-summary__name {
  font-size: var(--text-2xl);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.node-summary__meta {
  display: flex;
  gap: var(--space-4);
  font-size: var(--text-sm);
}

.node-summary__metrics {
  display: flex;
  gap: var(--space-4);
  margin-top: var(--space-2);
}

.metric-badge {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-2) var(--space-4);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
  min-width: 60px;
}

.metric-badge__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wide);
}

.metric-badge__value {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

/* ─── Detail Grid ─── */
.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-6);
}

@media (max-width: 900px) {
  .detail-grid {
    grid-template-columns: 1fr;
  }
}

/* ─── Detail Section ─── */
.detail-section {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.detail-section__title {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.detail-section__count {
  font-size: var(--text-xs);
  background: var(--color-primary);
  color: #fff;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-weight: var(--font-medium);
}

.detail-section__loading {
  width: 100%;
}

/* ─── Health Timeline ─── */
.health-timeline {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.health-timeline__hint {
  margin: 0;
}

.sla-badge {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.sla-badge__label {
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.sla-badge__value {
  font-size: var(--text-md);
  font-weight: var(--font-bold);
}

.sla-badge__value--good {
  color: var(--color-success);
}

.sla-badge__value--warn {
  color: var(--color-warning);
}

.sla-badge__value--bad {
  color: var(--color-danger);
}

/* ─── Sessions List ─── */
.sessions-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 300px;
  overflow-y: auto;
}

.session-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
}

.session-item__user {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.session-item__username {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}

.session-item__ip {
  font-size: var(--text-xs);
}

.session-item__stats {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.session-item__duration {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-accent);
}

.session-item__traffic {
  font-size: var(--text-xs);
}

/* ─── Protocol List ─── */
.protocols-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.protocol-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-3);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
  flex-wrap: wrap;
  gap: var(--space-2);
}

.protocol-row__info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.protocol-row__icon {
  font-size: var(--text-lg);
}

.protocol-row__name {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}

.protocol-row__actions {
  display: flex;
  gap: var(--space-1);
}

/* ─── Tasks List ─── */
.tasks-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-height: 300px;
  overflow-y: auto;
}

.task-item {
  display: flex;
  justify-content: space-between;
  align-items: flex-start;
  padding: var(--space-3);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
  flex-wrap: wrap;
  gap: var(--space-2);
}

.task-item__info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.task-item__action {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
  font-family: var(--font-mono);
}

.task-item__meta {
  display: flex;
  flex-direction: column;
  align-items: flex-end;
  gap: 2px;
}

.task-item__time {
  font-size: var(--text-xs);
}

.task-item__error {
  font-size: var(--text-xs);
  color: var(--color-danger);
}

/* ─── Utility ─── */
.text-muted {
  color: var(--color-muted);
}

.text-sm {
  font-size: var(--text-sm);
}

.text-danger {
  color: var(--color-danger);
}
</style>
