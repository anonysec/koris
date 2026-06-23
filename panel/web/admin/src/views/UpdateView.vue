<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useNodesStore } from '@/stores/nodes'
import { storeToRefs } from 'pinia'
import KButton from '@koris/ui/KButton.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()
const { list: nodes } = storeToRefs(nodesStore)

// ─── Panel Update State ──────────────────────────────────────────────────────
interface ReleaseInfo {
  version: string
  changelog: string
  url: string
  checksum_sha256: string
}

const checking = ref(false)
const applying = ref(false)
const rollingBack = ref(false)
const releaseInfo = ref<ReleaseInfo | null>(null)
const currentVersion = ref('')
const updateError = ref('')

// ─── Node Agent Update State ─────────────────────────────────────────────────
const selectedNodes = ref<Set<number>>(new Set())
const bulkUpdating = ref(false)
const selectAll = ref(false)

// ─── Computed ────────────────────────────────────────────────────────────────
const hasUpdate = computed(() => {
  if (!releaseInfo.value || !currentVersion.value) return false
  return releaseInfo.value.version !== currentVersion.value
})

const selectedNodesList = computed(() =>
  nodes.value.filter(n => selectedNodes.value.has(n.id))
)

// ─── Panel Update Actions ────────────────────────────────────────────────────
async function checkForUpdates() {
  checking.value = true
  updateError.value = ''
  try {
    const res = await api.get<{
      ok: boolean
      current_version: string
      latest: ReleaseInfo | null
    }>('/api/admin/update/check')
    currentVersion.value = res.current_version
    releaseInfo.value = res.latest || null
    if (!res.latest) {
      toast.success(t('update.up_to_date'))
    }
  } catch (err: any) {
    updateError.value = err.message || 'Failed to check for updates'
  } finally {
    checking.value = false
  }
}

async function applyUpdate() {
  if (!releaseInfo.value) return
  const confirmed = await confirm({
    title: t('update.confirm_apply_title'),
    message: t('update.confirm_apply_msg').replace('{version}', releaseInfo.value.version),
    variant: 'info',
    icon: '⬆',
    confirmText: t('update.apply'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  applying.value = true
  try {
    await api.post<{ ok: boolean }>('/api/admin/update/apply', {
      version: releaseInfo.value.version,
    })
    toast.success(t('update.apply_started'))
  } catch {
    toast.error(t('update.apply_failed'))
  } finally {
    applying.value = false
  }
}

async function rollback() {
  const confirmed = await confirm({
    title: t('update.confirm_rollback_title'),
    message: t('update.confirm_rollback_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('update.rollback'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  rollingBack.value = true
  try {
    await api.post<{ ok: boolean }>('/api/admin/update/rollback', {})
    toast.success(t('update.rollback_started'))
  } catch {
    toast.error(t('update.rollback_failed'))
  } finally {
    rollingBack.value = false
  }
}

// ─── Node Agent Bulk Update ──────────────────────────────────────────────────
function toggleSelectAll() {
  if (selectAll.value) {
    nodes.value.forEach(n => selectedNodes.value.add(n.id))
  } else {
    selectedNodes.value.clear()
  }
}

function toggleNodeSelection(nodeId: number) {
  if (selectedNodes.value.has(nodeId)) {
    selectedNodes.value.delete(nodeId)
  } else {
    selectedNodes.value.add(nodeId)
  }
  selectAll.value = selectedNodes.value.size === nodes.value.length
}

async function bulkUpdateNodes() {
  if (selectedNodes.value.size === 0) return
  const confirmed = await confirm({
    title: t('update.confirm_bulk_title'),
    message: t('update.confirm_bulk_msg').replace('{count}', String(selectedNodes.value.size)),
    variant: 'info',
    icon: '⬆',
    confirmText: t('update.update_selected'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  bulkUpdating.value = true
  try {
    await api.post<{ ok: boolean }>('/api/admin/nodes/update/bulk', {
      node_ids: Array.from(selectedNodes.value),
    })
    toast.success(t('update.bulk_started'))
    selectedNodes.value.clear()
    selectAll.value = false
  } catch {
    toast.error(t('update.bulk_failed'))
  } finally {
    bulkUpdating.value = false
  }
}

onMounted(() => {
  nodesStore.loadNodes()
})
</script>

<template>
  <div class="page update-view">
    <!-- Panel Update Section -->
    <section class="update-section">
      <h3 class="section-title">{{ t('update.panel_update') }}</h3>
      <div class="update-card">
        <div class="update-card__info">
          <div v-if="currentVersion" class="update-card__version">
            <span class="text-muted">{{ t('update.current_version') }}:</span>
            <strong>{{ currentVersion }}</strong>
          </div>
          <div v-if="releaseInfo && hasUpdate" class="update-card__available">
            <span class="text-muted">{{ t('update.available_version') }}:</span>
            <strong class="text-primary">{{ releaseInfo.version }}</strong>
          </div>
        </div>

        <div class="update-card__actions">
          <KButton variant="primary" :loading="checking" @click="checkForUpdates">
            {{ t('update.check_for_updates') }}
          </KButton>
          <KButton
            v-if="hasUpdate"
            variant="primary"
            :loading="applying"
            @click="applyUpdate"
          >
            {{ t('update.apply') }}
          </KButton>
          <KButton
            variant="danger"
            :loading="rollingBack"
            @click="rollback"
          >
            {{ t('update.rollback') }}
          </KButton>
        </div>

        <p v-if="updateError" class="update-error">{{ updateError }}</p>
      </div>

      <!-- Changelog -->
      <div v-if="releaseInfo && releaseInfo.changelog" class="changelog-section">
        <h4 class="changelog-title">{{ t('update.changelog') }}</h4>
        <div class="changelog-content">{{ releaseInfo.changelog }}</div>
      </div>
    </section>

    <!-- Node Agent Update Section -->
    <section class="update-section">
      <h3 class="section-title">{{ t('update.node_agent_update') }}</h3>

      <div v-if="nodesStore.loading && nodes.length === 0">
        <KSkeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="48" />
      </div>

      <KEmptyState
        v-else-if="nodes.length === 0"
        icon="🖥️"
        :title="t('update.no_nodes')"
        :description="t('update.no_nodes_desc')"
      />

      <div v-else class="node-update-table-wrap">
        <div class="node-update-actions">
          <label class="select-all-label">
            <input
              type="checkbox"
              :checked="selectAll"
              @change="selectAll = ($event.target as HTMLInputElement).checked; toggleSelectAll()"
            />
            {{ t('update.select_all') }}
          </label>
          <KButton
            variant="primary"
            size="sm"
            :loading="bulkUpdating"
            :disabled="selectedNodes.size === 0"
            @click="bulkUpdateNodes"
          >
            {{ t('update.update_selected') }} ({{ selectedNodes.size }})
          </KButton>
        </div>

        <table class="node-update-table">
          <thead>
            <tr>
              <th class="col-check"></th>
              <th>{{ t('update.node_name') }}</th>
              <th>{{ t('update.ip') }}</th>
              <th>{{ t('update.agent_version') }}</th>
              <th>{{ t('update.status') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="node in nodes" :key="node.id">
              <td class="col-check">
                <input
                  type="checkbox"
                  :checked="selectedNodes.has(node.id)"
                  @change="toggleNodeSelection(node.id)"
                />
              </td>
              <td>{{ node.name }}</td>
              <td class="text-muted">{{ node.public_ip }}</td>
              <td>
                <code>{{ node.agent_version || '—' }}</code>
              </td>
              <td>
                <KStatusPill :status="node.status" size="sm" />
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>
  </div>
</template>

<style scoped>
.update-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.update-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.section-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

.update-card {
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.update-card__info {
  display: flex;
  gap: var(--space-5);
  flex-wrap: wrap;
}

.update-card__version,
.update-card__available {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
}

.update-card__actions {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.update-error {
  color: var(--color-danger, #ef4444);
  font-size: var(--text-sm);
  margin: 0;
}

.text-primary {
  color: var(--color-primary);
}

.changelog-section {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.changelog-title {
  margin: 0;
  padding: var(--space-3) var(--space-4);
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.changelog-content {
  padding: var(--space-4);
  font-size: var(--text-sm);
  line-height: var(--leading-relaxed);
  white-space: pre-wrap;
  color: var(--color-text);
}

/* Node Update Table */
.node-update-table-wrap {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.node-update-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
}

.select-all-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  cursor: pointer;
}

.node-update-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.node-update-table th {
  text-align: left;
  padding: var(--space-3) var(--space-3);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  font-weight: var(--font-semibold);
  color: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.node-update-table td {
  padding: var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
}

.node-update-table tr:last-child td {
  border-bottom: none;
}

.col-check {
  width: 40px;
}

@media (max-width: 768px) {
  .update-card__info {
    flex-direction: column;
    gap: var(--space-2);
  }

  .update-card__actions {
    flex-direction: column;
  }

  .node-update-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
