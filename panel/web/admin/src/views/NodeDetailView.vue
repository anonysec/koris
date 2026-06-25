<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { useNodesStore } from '@/stores/nodes'
import { useMetricsStore } from '@/stores/metrics'
import { useEditionStore } from '@/stores/edition'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import { useApi } from '@koris/composables/useApi'
import KButton from '@koris/ui/KButton.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import NodeMetricsPanel from '@/components/nodes/NodeMetricsPanel.vue'
import NodeCoresTab from '@/components/nodes/NodeCoresTab.vue'
import NodeSessionsTab from '@/components/nodes/NodeSessionsTab.vue'
import NodeFirewallTab from '@/components/nodes/NodeFirewallTab.vue'
import NodeTunnelsTab from '@/components/nodes/NodeTunnelsTab.vue'
import NodeCertsTab from '@/components/nodes/NodeCertsTab.vue'
import BandwidthChart from '@/components/metrics/BandwidthChart.vue'

const props = defineProps<{ id: string; tab?: string }>()

const { t } = useI18n()
const router = useRouter()
const route = useRoute()
const store = useNodesStore()
const metrics = useMetricsStore()
const edition = useEditionStore()
const toast = useToast()
const { get } = useApi()

// ─── State ───────────────────────────────────────────────────────────────────
const loading = ref(true)
const node = ref<any>(null)
const activeTab = ref(props.tab || 'overview')

// ─── Tabs definition ─────────────────────────────────────────────────────────
const tabs = computed(() => {
  const list = [
    { key: 'overview', label: t('node_detail.tab_overview') },
    { key: 'cores', label: t('node_detail.tab_cores') },
    { key: 'sessions', label: t('node_detail.tab_sessions') },
    { key: 'firewall', label: t('node_detail.tab_firewall') },
  ]
  if (edition.isFull) {
    list.push({ key: 'tunnels', label: t('node_detail.tab_tunnels') })
  }
  list.push({ key: 'certificates', label: t('node_detail.tab_certificates') })
  return list
})

// ─── Node metrics from realtime store ────────────────────────────────────────
const nodeMetrics = computed(() => {
  return metrics.nodes.get(Number(props.id)) ?? null
})

// ─── Sync tab from route param ───────────────────────────────────────────────
watch(() => props.tab, (newTab) => {
  if (newTab) activeTab.value = newTab
})

watch(activeTab, (newTab) => {
  if (newTab !== props.tab) {
    router.replace({ name: 'node-detail', params: { id: props.id, tab: newTab } })
  }
})

// ─── Helpers ─────────────────────────────────────────────────────────────────
function formatUptime(seconds: number): string {
  if (!seconds) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  if (days > 0) return `${days}d ${hours}h`
  if (hours > 0) return `${hours}h`
  return `${Math.floor(seconds / 60)}m`
}

// ─── Data Loading ────────────────────────────────────────────────────────────
async function loadNodeDetail() {
  loading.value = true
  try {
    const res = await get<any>(`/api/admin/knode/nodes/${props.id}`)
    if (res.ok !== false) {
      node.value = res.node || res
    }
  } catch {
    toast.error(t('node_detail.load_error'))
  } finally {
    loading.value = false
  }
}

onMounted(async () => {
  await loadNodeDetail()
  edition.fetchEdition()
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
      <KSkeleton variant="rect" :width="'100%'" :height="300" />
    </div>

    <template v-else-if="node">
      <!-- Node Summary Card -->
      <div class="node-summary">
        <div class="node-summary__info">
          <h2 class="node-summary__name">{{ node.name }}</h2>
          <KStatusPill :status="node.status" size="sm" />
        </div>
        <div class="node-summary__meta">
          <span class="text-muted">{{ node.address }}</span>
          <span v-if="nodeMetrics" class="text-muted">{{ t('node_detail.uptime') }}: {{ formatUptime(nodeMetrics.uptime) }}</span>
          <span v-if="node.version" class="text-muted">v{{ node.version }}</span>
        </div>
      </div>

      <!-- Tab Navigation -->
      <nav class="tab-nav" role="tablist">
        <button
          v-for="tab in tabs"
          :key="tab.key"
          class="tab-nav__item"
          :class="{ active: activeTab === tab.key }"
          role="tab"
          :aria-selected="activeTab === tab.key"
          @click="activeTab = tab.key"
        >
          {{ tab.label }}
        </button>
      </nav>

      <!-- Tab Content -->
      <div class="tab-content" role="tabpanel">
        <!-- Overview Tab -->
        <div v-if="activeTab === 'overview'" class="overview-tab">
          <NodeMetricsPanel :node-id="Number(id)" />
          <BandwidthChart :node-id="Number(id)" />
        </div>

        <!-- Cores Tab -->
        <NodeCoresTab v-else-if="activeTab === 'cores'" :node-id="Number(id)" />

        <!-- Sessions Tab -->
        <NodeSessionsTab v-else-if="activeTab === 'sessions'" :node-id="Number(id)" />

        <!-- Firewall Tab -->
        <NodeFirewallTab v-else-if="activeTab === 'firewall'" :node-id="Number(id)" />

        <!-- Tunnels Tab (full edition only) -->
        <NodeTunnelsTab v-else-if="activeTab === 'tunnels'" :node-id="Number(id)" />

        <!-- Certificates Tab -->
        <NodeCertsTab v-else-if="activeTab === 'certificates'" :node-id="Number(id)" />
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
  gap: var(--space-5);
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
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
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

/* ─── Tab Navigation ─── */
.tab-nav {
  display: flex;
  gap: var(--space-1);
  border-bottom: 1px solid var(--color-border);
  overflow-x: auto;
}

.tab-nav__item {
  padding: var(--space-3) var(--space-4);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-muted);
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  cursor: pointer;
  white-space: nowrap;
  transition: color var(--duration-normal) ease, border-color var(--duration-normal) ease;
}

.tab-nav__item:hover {
  color: var(--color-text);
}

.tab-nav__item.active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}

/* ─── Tab Content ─── */
.tab-content {
  min-height: 200px;
}

.overview-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

/* ─── Utility ─── */
.text-muted {
  color: var(--color-muted);
}
</style>
