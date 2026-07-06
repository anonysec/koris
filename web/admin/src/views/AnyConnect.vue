<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useNodesStore } from '@/stores/nodes'
import Button from '@koris/ui/Button.vue'
import SlideOver from '@koris/ui/SlideOver.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import StatusPill from '@koris/ui/StatusPill.vue'

const { t } = useI18n()
const { get, post, del } = useApi()
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()

// ─── Types ───────────────────────────────────────────────────────────────────
interface AnyConnectNode {
  id: number
  node_id: number
  port: number
  cert_path: string | null
  status: string
  created_at: string
  updated_at: string
}

// ─── State ───────────────────────────────────────────────────────────────────
const nodes = ref<AnyConnectNode[]>([])
const loading = ref(false)
const showEnableModal = ref(false)
const enabling = ref(false)
const actionInProgress = ref<number | null>(null)
const uploadingCert = ref<number | null>(null)
const certFileInput = ref<HTMLInputElement | null>(null)
const keyFileInput = ref<HTMLInputElement | null>(null)
const certUploadTarget = ref<AnyConnectNode | null>(null)

const enableForm = ref({
  node_id: '' as string | number,
  port: 443,
})

// ─── Computed ────────────────────────────────────────────────────────────────
const nodeOptions = computed(() =>
  nodesStore.list.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

// ─── API ─────────────────────────────────────────────────────────────────────
async function loadNodes() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; nodes: AnyConnectNode[] }>('/api/anyconnect')
    if (res.ok) {
      nodes.value = res.nodes || []
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
    const res = await post<{ ok: boolean }>('/api/anyconnect', {
      node_id: Number(enableForm.value.node_id),
      port: enableForm.value.port,
    })
    if (res.ok) {
      toast.success(t('anyconnect.enabled_success'))
      showEnableModal.value = false
      enableForm.value = { node_id: '', port: 443 }
      await loadNodes()
    }
  } catch {
    // handled by useApi
  } finally {
    enabling.value = false
  }
}

function openCertUpload(node: AnyConnectNode) {
  certUploadTarget.value = node
  // Trigger file input via hidden element
  certFileInput.value?.click()
}

async function handleCertUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const certFile = target.files?.[0]
  if (!certFile || !certUploadTarget.value) return

  // After cert file is picked, trigger key file input
  keyFileInput.value?.click()
}

async function handleKeyUpload(event: Event) {
  const target = event.target as HTMLInputElement
  const keyFile = target.files?.[0]
  const certFile = certFileInput.value?.files?.[0]
  if (!keyFile || !certFile || !certUploadTarget.value) return

  const nodeItem = certUploadTarget.value
  uploadingCert.value = nodeItem.id

  try {
    const formData = new FormData()
    formData.append('cert', certFile)
    formData.append('key', keyFile)

    const res = await post<{ ok: boolean }>(`/api/anyconnect/${nodeItem.id}/cert`, formData)
    if (res.ok) {
      toast.success(t('anyconnect.cert_uploaded'))
      await loadNodes()
    }
  } catch {
    // handled by useApi
  } finally {
    uploadingCert.value = null
    certUploadTarget.value = null
    // Reset file inputs
    if (certFileInput.value) certFileInput.value.value = ''
    if (keyFileInput.value) keyFileInput.value.value = ''
  }
}

async function handleDisable(node: AnyConnectNode) {
  const confirmed = await confirm({
    title: t('anyconnect.confirm_disable_title'),
    message: t('anyconnect.confirm_disable_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.disable'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  actionInProgress.value = node.id
  try {
    const res = await del<{ ok: boolean }>(`/api/anyconnect/${node.id}`)
    if (res.ok) {
      toast.success(t('anyconnect.disabled_success'))
      await loadNodes()
    }
  } catch {
    // handled by useApi
  } finally {
    actionInProgress.value = null
  }
}

function getNodeName(nodeId: number): string {
  const n = nodesStore.list.find(nd => nd.id === nodeId)
  return n ? n.name : `#${nodeId}`
}

function getCertStatus(node: AnyConnectNode): string {
  if (node.cert_path) return t('anyconnect.cert_installed')
  return t('anyconnect.cert_missing')
}

function getCertPillStatus(node: AnyConnectNode): string {
  return node.cert_path ? 'success' : 'warning'
}

function getServicePillStatus(node: AnyConnectNode): string {
  if (node.status === 'active') return 'success'
  if (node.status === 'error') return 'danger'
  return 'neutral'
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  loadNodes()
  if (nodesStore.list.length === 0) {
    nodesStore.loadNodes()
  }
})
</script>

<template>
  <div class="page anyconnect-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('anyconnect.title') }}</h2>
      <div class="page-header__actions">
        <Button variant="primary" icon="+" @click="showEnableModal = true">
          {{ t('anyconnect.enable_on_node') }}
        </Button>
      </div>
    </header>

    <!-- Hidden file inputs for cert upload -->
    <input
      ref="certFileInput"
      type="file"
      accept=".pem,.crt,.cer"
      style="display: none"
      @change="handleCertUpload"
    />
    <input
      ref="keyFileInput"
      type="file"
      accept=".pem,.key"
      style="display: none"
      @change="handleKeyUpload"
    />

    <!-- Loading -->
    <div v-if="loading && nodes.length === 0" class="table-loading">
      <Skeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="60" />
    </div>

    <!-- Empty state -->
    <EmptyState
      v-else-if="nodes.length === 0"
      icon="🔒"
      :title="t('anyconnect.empty_title')"
      :description="t('anyconnect.empty_desc')"
    />

    <!-- AnyConnect nodes table -->
    <div v-else class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>{{ t('anyconnect.col_node') }}</th>
            <th>{{ t('anyconnect.col_port') }}</th>
            <th>{{ t('anyconnect.col_cert_status') }}</th>
            <th>{{ t('anyconnect.col_service_status') }}</th>
            <th>{{ t('anyconnect.col_actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="node in nodes" :key="node.id">
            <td>{{ getNodeName(node.node_id) }}</td>
            <td class="td-mono">{{ node.port }}</td>
            <td>
              <StatusPill :status="getCertPillStatus(node)">
                {{ getCertStatus(node) }}
              </StatusPill>
            </td>
            <td>
              <StatusPill :status="getServicePillStatus(node)">
                {{ t(`anyconnect.status_${node.status}`) }}
              </StatusPill>
            </td>
            <td class="td-actions">
              <Button
                variant="ghost"
                size="sm"
                :loading="uploadingCert === node.id"
                @click="openCertUpload(node)"
              >
                📄 {{ t('anyconnect.upload_cert') }}
              </Button>
              <Button
                variant="danger"
                size="sm"
                :loading="actionInProgress === node.id"
                @click="handleDisable(node)"
              >
                {{ t('btn.disable') }}
              </Button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Enable on Node Slide-Over -->
    <SlideOver :open="showEnableModal" :title="t('anyconnect.enable_title')" @close="showEnableModal = false">
      <form class="enable-form" @submit.prevent="handleEnable">
        <FormField name="ac-node" :label="t('anyconnect.field_node')" required>
          <template #default="{ fieldId }">
            <Select
              :id="fieldId"
              v-model="enableForm.node_id"
              :options="nodeOptions"
              :placeholder="t('anyconnect.select_node')"
            />
          </template>
        </FormField>

        <FormField name="ac-port" :label="t('anyconnect.field_port')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model.number="enableForm.port" type="number" placeholder="443" />
          </template>
        </FormField>

        <div class="form-actions">
          <Button variant="ghost" @click="showEnableModal = false">{{ t('btn.cancel') }}</Button>
          <Button type="submit" variant="primary" :loading="enabling">{{ t('anyconnect.enable') }}</Button>
        </div>
      </form>
    </SlideOver>
  </div>
</template>

<style scoped>
.anyconnect-view {
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

.td-center {
  text-align: center;
}

.td-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
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
