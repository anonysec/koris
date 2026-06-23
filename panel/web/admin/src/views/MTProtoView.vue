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
import KStatusPill from '@koris/ui/KStatusPill.vue'

const { t } = useI18n()
const { get, post, del } = useApi()
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()

// ─── Types ───────────────────────────────────────────────────────────────────
interface MTProtoProxy {
  id: number
  node_id: number
  port: number
  secret: string
  status: string
  connections: number
  rx_bytes: number
  tx_bytes: number
  created_at: string
  updated_at: string
}

// ─── State ───────────────────────────────────────────────────────────────────
const proxies = ref<MTProtoProxy[]>([])
const loading = ref(false)
const showEnableModal = ref(false)
const enabling = ref(false)
const actionInProgress = ref<number | null>(null)
const shareLinkProxy = ref<MTProtoProxy | null>(null)
const shareLink = ref('')

const enableForm = ref({
  node_id: '' as string | number,
  port: 443,
})

// ─── Computed ────────────────────────────────────────────────────────────────
const nodeOptions = computed(() =>
  nodesStore.list.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

// ─── API ─────────────────────────────────────────────────────────────────────
async function loadProxies() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; proxies: MTProtoProxy[] }>('/api/mtproto')
    if (res.ok) {
      proxies.value = res.proxies || []
    }
  } catch {
    // handled by useApi
  } finally {
    loading.value = false
  }
}

async function handleEnable() {
  if (!enableForm.value.node_id || !enableForm.value.port) return
  enabling.value = true
  try {
    const res = await post<{ ok: boolean }>('/api/mtproto', {
      node_id: Number(enableForm.value.node_id),
      port: enableForm.value.port,
    })
    if (res.ok) {
      toast.success(t('mtproto.enabled_success'))
      showEnableModal.value = false
      enableForm.value = { node_id: '', port: 443 }
      await loadProxies()
    }
  } catch {
    // handled by useApi
  } finally {
    enabling.value = false
  }
}

async function handleRotateSecret(proxy: MTProtoProxy) {
  const confirmed = await confirm({
    title: t('mtproto.confirm_rotate_title'),
    message: t('mtproto.confirm_rotate_msg'),
    variant: 'default',
    icon: '🔄',
    confirmText: t('mtproto.rotate_secret'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  actionInProgress.value = proxy.id
  try {
    const res = await post<{ ok: boolean }>(`/api/mtproto/${proxy.id}/rotate`)
    if (res.ok) {
      toast.success(t('mtproto.rotated_success'))
      await loadProxies()
    }
  } catch {
    // handled by useApi
  } finally {
    actionInProgress.value = null
  }
}

async function handleGetShareLink(proxy: MTProtoProxy) {
  actionInProgress.value = proxy.id
  try {
    const res = await get<{ ok: boolean; link: string }>(`/api/mtproto/${proxy.id}/link`)
    if (res.ok) {
      shareLink.value = res.link
      shareLinkProxy.value = proxy
    }
  } catch {
    // handled by useApi
  } finally {
    actionInProgress.value = null
  }
}

async function handleDisable(proxy: MTProtoProxy) {
  const confirmed = await confirm({
    title: t('mtproto.confirm_disable_title'),
    message: t('mtproto.confirm_disable_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.disable'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  actionInProgress.value = proxy.id
  try {
    const res = await del<{ ok: boolean }>(`/api/mtproto/${proxy.id}`)
    if (res.ok) {
      toast.success(t('mtproto.disabled_success'))
      await loadProxies()
    }
  } catch {
    // handled by useApi
  } finally {
    actionInProgress.value = null
  }
}

function copyShareLink() {
  if (!shareLink.value) return
  navigator.clipboard.writeText(shareLink.value)
  toast.success(t('mtproto.link_copied'))
}

function closeShareLink() {
  shareLinkProxy.value = null
  shareLink.value = ''
}

function getNodeName(nodeId: number): string {
  const node = nodesStore.list.find(n => n.id === nodeId)
  return node ? node.name : `#${nodeId}`
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatBandwidth(proxy: MTProtoProxy): string {
  const total = proxy.rx_bytes + proxy.tx_bytes
  return formatBytes(total)
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
  <div class="page mtproto-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('mtproto.title') }}</h2>
      <div class="page-header__actions">
        <KButton variant="primary" icon="+" @click="showEnableModal = true">
          {{ t('mtproto.enable_on_node') }}
        </KButton>
      </div>
    </header>

    <!-- Loading -->
    <div v-if="loading && proxies.length === 0" class="table-loading">
      <KSkeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="60" />
    </div>

    <!-- Empty state -->
    <KEmptyState
      v-else-if="proxies.length === 0"
      icon="📡"
      :title="t('mtproto.empty_title')"
      :description="t('mtproto.empty_desc')"
    />

    <!-- Proxy list table -->
    <div v-else class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>{{ t('mtproto.col_node') }}</th>
            <th>{{ t('mtproto.col_port') }}</th>
            <th>{{ t('mtproto.col_secret') }}</th>
            <th>{{ t('mtproto.col_status') }}</th>
            <th>{{ t('mtproto.col_connections') }}</th>
            <th>{{ t('mtproto.col_bandwidth') }}</th>
            <th>{{ t('mtproto.col_actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="proxy in proxies" :key="proxy.id">
            <td>{{ getNodeName(proxy.node_id) }}</td>
            <td class="td-mono">{{ proxy.port }}</td>
            <td class="td-mono td-secret">
              <span class="secret-text">{{ proxy.secret.substring(0, 8) }}…</span>
            </td>
            <td>
              <KStatusPill :status="proxy.status === 'active' ? 'success' : proxy.status === 'error' ? 'danger' : 'neutral'">
                {{ t(`mtproto.status_${proxy.status}`) }}
              </KStatusPill>
            </td>
            <td class="td-center">{{ proxy.connections }}</td>
            <td class="td-center">{{ formatBandwidth(proxy) }}</td>
            <td class="td-actions">
              <KButton
                variant="ghost"
                size="sm"
                :loading="actionInProgress === proxy.id"
                @click="handleRotateSecret(proxy)"
              >
                🔄 {{ t('mtproto.rotate_secret') }}
              </KButton>
              <KButton
                variant="ghost"
                size="sm"
                :loading="actionInProgress === proxy.id"
                @click="handleGetShareLink(proxy)"
              >
                🔗 {{ t('mtproto.get_link') }}
              </KButton>
              <KButton
                variant="danger"
                size="sm"
                :loading="actionInProgress === proxy.id"
                @click="handleDisable(proxy)"
              >
                {{ t('btn.disable') }}
              </KButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Share Link Display -->
    <div v-if="shareLinkProxy" class="share-link-overlay" @click.self="closeShareLink">
      <div class="share-link-card">
        <h3 class="share-link-title">{{ t('mtproto.share_link_title') }}</h3>
        <p class="share-link-node">{{ getNodeName(shareLinkProxy.node_id) }} :{{ shareLinkProxy.port }}</p>
        <div class="share-link-input-row">
          <input
            type="text"
            class="share-link-input"
            :value="shareLink"
            readonly
          />
          <KButton variant="primary" size="sm" @click="copyShareLink">
            📋 {{ t('btn.copy') }}
          </KButton>
        </div>
        <div class="share-link-actions">
          <KButton variant="ghost" @click="closeShareLink">{{ t('btn.close') }}</KButton>
        </div>
      </div>
    </div>

    <!-- Enable on Node Slide-Over -->
    <KSlideOver :open="showEnableModal" :title="t('mtproto.enable_title')" @close="showEnableModal = false">
      <form class="enable-form" @submit.prevent="handleEnable">
        <KFormField name="mtproto-node" :label="t('mtproto.field_node')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="enableForm.node_id"
              :options="nodeOptions"
              :placeholder="t('mtproto.select_node')"
            />
          </template>
        </KFormField>

        <KFormField name="mtproto-port" :label="t('mtproto.field_port')" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model.number="enableForm.port" type="number" placeholder="443" />
          </template>
        </KFormField>

        <div class="form-actions">
          <KButton variant="ghost" @click="showEnableModal = false">{{ t('btn.cancel') }}</KButton>
          <KButton type="submit" variant="primary" :loading="enabling">{{ t('mtproto.enable') }}</KButton>
        </div>
      </form>
    </KSlideOver>
  </div>
</template>

<style scoped>
.mtproto-view {
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

.table-loading {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
}

.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.data-table th {
  text-align: left;
  padding: var(--space-3) var(--space-4);
  font-weight: 600;
  color: var(--color-muted);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}

.data-table td {
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  vertical-align: middle;
}

.data-table tbody tr:last-child td {
  border-bottom: none;
}

.data-table tbody tr:hover {
  background: var(--color-surface-hover, rgba(255, 255, 255, 0.03));
}

.td-mono {
  font-family: var(--font-mono, monospace);
}

.td-secret {
  max-width: 120px;
}

.secret-text {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.td-center {
  text-align: center;
}

.td-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}

/* ─── Share Link Overlay ───────────────────────────────────────────────────── */
.share-link-overlay {
  position: fixed;
  inset: 0;
  z-index: 1000;
  display: flex;
  align-items: center;
  justify-content: center;
  background: rgba(0, 0, 0, 0.5);
}

.share-link-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
  max-width: 500px;
  width: 90%;
}

.share-link-title {
  margin: 0 0 var(--space-2);
  font-size: var(--text-lg);
  font-weight: 600;
  color: var(--color-text);
}

.share-link-node {
  margin: 0 0 var(--space-4);
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.share-link-input-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

.share-link-input {
  flex: 1;
  font-family: var(--font-mono, monospace);
  font-size: var(--text-xs);
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg);
  color: var(--color-text);
  outline: none;
}

.share-link-actions {
  display: flex;
  justify-content: flex-end;
  margin-top: var(--space-4);
}

/* ─── Form ─────────────────────────────────────────────────────────────────── */
.enable-form {
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
  .table-wrap {
    font-size: var(--text-xs);
  }

  .td-actions {
    flex-direction: column;
  }
}
</style>
