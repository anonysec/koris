<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useNodesStore } from '@/stores/nodes'
import KButton from '@koris/ui/KButton.vue'
import KSlideOver from '@koris/ui/KSlideOver.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'

const { t } = useI18n()
const api = useApi({ baseUrl: '/api/admin' })
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()

// ─── State ───────────────────────────────────────────────────────────────────
interface TelegramProxy {
  id: number
  node_id: number
  port: number
  secret: string
  tag: string
  status: string
  share_link: string
  tg_link: string
  connections_count: number
  last_health_check: string | null
  created_at: string
}

const proxies = ref<TelegramProxy[]>([])
const loading = ref(false)
const showCreateForm = ref(false)
const creating = ref(false)
const actionInProgress = ref<number | null>(null)
const rotating = ref(false)

const createForm = ref({
  node_id: '' as string | number,
  port: 8443,
  tag: '',
})

// ─── Computed ────────────────────────────────────────────────────────────────
const nodeOptions = computed(() =>
  nodesStore.list.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

// ─── API ─────────────────────────────────────────────────────────────────────
async function loadProxies() {
  loading.value = true
  try {
    const res = await api.get<{ ok: boolean; proxies: TelegramProxy[] }>('/telegram-proxies')
    if (res.ok) {
      proxies.value = res.proxies || []
    }
  } catch {
    // useApi handles toast
  } finally {
    loading.value = false
  }
}

async function handleCreate() {
  if (!createForm.value.node_id || !createForm.value.port) return
  creating.value = true
  try {
    const res = await api.post<{ ok: boolean; proxy: TelegramProxy }>('/telegram-proxies', {
      node_id: Number(createForm.value.node_id),
      port: createForm.value.port,
      tag: createForm.value.tag,
    })
    if (res.ok) {
      toast.success(t('teleproxy.created_success'))
      showCreateForm.value = false
      createForm.value = { node_id: '', port: 8443, tag: '' }
      await loadProxies()
    }
  } catch {
    // handled by useApi
  } finally {
    creating.value = false
  }
}

async function handleStart(proxy: TelegramProxy) {
  actionInProgress.value = proxy.id
  try {
    await api.post(`/telegram-proxies/${proxy.id}/start`)
    toast.success(t('teleproxy.start_success'))
    await loadProxies()
  } catch {
    // handled
  } finally {
    actionInProgress.value = null
  }
}

async function handleStop(proxy: TelegramProxy) {
  actionInProgress.value = proxy.id
  try {
    await api.post(`/telegram-proxies/${proxy.id}/stop`)
    toast.success(t('teleproxy.stop_success'))
    await loadProxies()
  } catch {
    // handled
  } finally {
    actionInProgress.value = null
  }
}

async function handleDelete(proxy: TelegramProxy) {
  const confirmed = await confirm({
    title: t('teleproxy.confirm_delete_title'),
    message: t('teleproxy.confirm_delete_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.delete'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  actionInProgress.value = proxy.id
  try {
    await api.del('/telegram-proxies', {
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ id: proxy.id }),
    })
    toast.success(t('teleproxy.deleted_success'))
    await loadProxies()
  } catch {
    // handled
  } finally {
    actionInProgress.value = null
  }
}

async function handleRotateAll() {
  const confirmed = await confirm({
    title: t('teleproxy.confirm_rotate_title'),
    message: t('teleproxy.confirm_rotate_msg'),
    variant: 'danger',
    icon: '🔄',
    confirmText: t('teleproxy.rotate_all'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  rotating.value = true
  try {
    await api.post('/telegram-proxies/rotate')
    toast.success(t('teleproxy.rotated_success'))
    await loadProxies()
  } catch {
    // handled
  } finally {
    rotating.value = false
  }
}

function copyShareLink(proxy: TelegramProxy) {
  const link = proxy.tg_link || proxy.share_link
  if (!link) return
  navigator.clipboard.writeText(link)
  toast.success(t('teleproxy.link_copied'))
}

function getNodeName(nodeId: number): string {
  const node = nodesStore.list.find(n => n.id === nodeId)
  return node ? node.name : `#${nodeId}`
}

function formatTime(ts: string | null): string {
  if (!ts) return '—'
  const d = new Date(ts)
  return d.toLocaleString()
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  loadProxies()
  if (nodesStore.list.length === 0) {
    nodesStore.loadNodes()
  }
})
</script>

<template>
  <div class="page teleproxy-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('teleproxy.title') }}</h2>
      <div class="page-header__actions">
        <KButton variant="ghost" :loading="rotating" @click="handleRotateAll">
          🔄 {{ t('teleproxy.rotate_all') }}
        </KButton>
        <KButton variant="primary" icon="+" @click="showCreateForm = true">
          {{ t('teleproxy.add_proxy') }}
        </KButton>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading && proxies.length === 0" class="proxy-loading">
      <KSkeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="60" />
    </div>

    <!-- Empty state -->
    <KEmptyState
      v-else-if="proxies.length === 0"
      icon="📡"
      :title="t('teleproxy.empty_title')"
      :description="t('teleproxy.empty_desc')"
    />

    <!-- Proxy list table -->
    <div v-else class="proxy-table-wrap">
      <table class="proxy-table">
        <thead>
          <tr>
            <th>ID</th>
            <th>{{ t('teleproxy.col_node') }}</th>
            <th>{{ t('teleproxy.col_port') }}</th>
            <th>{{ t('teleproxy.col_status') }}</th>
            <th>{{ t('teleproxy.col_connections') }}</th>
            <th>{{ t('teleproxy.col_health_check') }}</th>
            <th>{{ t('teleproxy.col_tag') }}</th>
            <th>{{ t('teleproxy.col_actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="proxy in proxies" :key="proxy.id">
            <td class="td-id">{{ proxy.id }}</td>
            <td>{{ getNodeName(proxy.node_id) }}</td>
            <td class="td-mono">{{ proxy.port }}</td>
            <td>
              <span class="status-badge" :class="`status-badge--${proxy.status}`">
                <span class="status-dot" :class="`status-dot--${proxy.status}`" />
                {{ t(`teleproxy.status_${proxy.status}`) }}
              </span>
            </td>
            <td class="td-center">{{ proxy.connections_count }}</td>
            <td class="td-muted">{{ formatTime(proxy.last_health_check) }}</td>
            <td class="td-muted">{{ proxy.tag || '—' }}</td>
            <td class="td-actions">
              <KButton
                v-if="proxy.status !== 'active'"
                variant="ghost"
                size="sm"
                :loading="actionInProgress === proxy.id"
                @click="handleStart(proxy)"
              >
                ▶ {{ t('teleproxy.start') }}
              </KButton>
              <KButton
                v-if="proxy.status === 'active'"
                variant="ghost"
                size="sm"
                :loading="actionInProgress === proxy.id"
                @click="handleStop(proxy)"
              >
                ⏹ {{ t('teleproxy.stop') }}
              </KButton>
              <KButton variant="ghost" size="sm" @click="copyShareLink(proxy)">
                📋 {{ t('teleproxy.copy_link') }}
              </KButton>
              <KButton
                variant="danger"
                size="sm"
                :loading="actionInProgress === proxy.id"
                @click="handleDelete(proxy)"
              >
                {{ t('btn.delete') }}
              </KButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Proxy Slide-Over -->
    <KSlideOver :open="showCreateForm" :title="t('teleproxy.create_title')" @close="showCreateForm = false">
      <form class="proxy-form" @submit.prevent="handleCreate">
        <KFormField name="proxy-node" :label="t('teleproxy.field_node')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="createForm.node_id"
              :options="nodeOptions"
              :placeholder="t('teleproxy.select_node')"
            />
          </template>
        </KFormField>

        <KFormField name="proxy-port" :label="t('teleproxy.field_port')" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model.number="createForm.port" type="number" placeholder="8443" />
          </template>
        </KFormField>

        <KFormField name="proxy-tag" :label="t('teleproxy.field_tag')">
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="createForm.tag" :placeholder="t('teleproxy.tag_placeholder')" />
          </template>
        </KFormField>

        <div class="form-actions">
          <KButton variant="ghost" @click="showCreateForm = false">{{ t('btn.cancel') }}</KButton>
          <KButton type="submit" variant="primary" :loading="creating">{{ t('btn.create') }}</KButton>
        </div>
      </form>
    </KSlideOver>
  </div>
</template>

<style scoped>
.teleproxy-view {
  padding: var(--space-6);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.page-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.page-header__actions {
  display: flex;
  gap: var(--space-2);
}

.proxy-loading {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.proxy-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
}

.proxy-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.proxy-table th {
  text-align: left;
  padding: var(--space-3) var(--space-4);
  font-weight: 600;
  color: var(--color-muted);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}

.proxy-table td {
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.proxy-table tbody tr:last-child td {
  border-bottom: none;
}

.proxy-table tbody tr:hover {
  background: var(--color-surface-hover, rgba(255, 255, 255, 0.03));
}

.td-id {
  color: var(--color-muted);
  font-size: var(--text-xs);
}

.td-mono {
  font-family: var(--font-mono, monospace);
}

.td-center {
  text-align: center;
}

.td-muted {
  color: var(--color-muted);
  font-size: var(--text-xs);
}

.td-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}

/* ─── Status Badge ─────────────────────────────────────────────────────────── */
.status-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-xs);
  font-weight: 500;
  padding: 2px 8px;
  border-radius: var(--radius-full, 9999px);
  background: var(--color-surface);
}

.status-dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  flex-shrink: 0;
}

.status-dot--active {
  background-color: #22c55e;
  box-shadow: 0 0 6px rgba(34, 197, 94, 0.5);
}

.status-dot--stopped {
  background-color: #6b7280;
}

.status-dot--error {
  background-color: #ef4444;
  box-shadow: 0 0 6px rgba(239, 68, 68, 0.5);
}

.status-badge--active {
  color: #22c55e;
}

.status-badge--stopped {
  color: #6b7280;
}

.status-badge--error {
  color: #ef4444;
}

/* ─── Form ─────────────────────────────────────────────────────────────────── */
.proxy-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-4);
}

/* ─── Mobile ───────────────────────────────────────────────────────────────── */
@media (max-width: 768px) {
  .proxy-table-wrap {
    font-size: var(--text-xs);
  }

  .td-actions {
    flex-direction: column;
  }
}
</style>
