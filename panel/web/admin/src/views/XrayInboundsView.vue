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
const { get, post, patch, del } = useApi()
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()

// ─── Types ───────────────────────────────────────────────────────────────────
interface XrayInbound {
  id: number
  customer_id: number
  customer_name?: string
  node_id: number
  uuid: string
  protocol: string
  transport: string
  security: string
  port: number
  server_name: string | null
  public_key: string | null
  short_id: string | null
  private_key: string | null
  path: string | null
  service_name: string | null
  status: string
  rx_bytes: number
  tx_bytes: number
  core_name: string
  created_at: string
  updated_at: string
}

interface CustomerOption {
  id: number
  username: string
}

// ─── State ───────────────────────────────────────────────────────────────────
const inbounds = ref<XrayInbound[]>([])
const customers = ref<CustomerOption[]>([])
const loading = ref(false)
const showCreateForm = ref(false)
const showEditForm = ref(false)
const creating = ref(false)
const saving = ref(false)
const actionInProgress = ref<number | null>(null)
const editingInbound = ref<XrayInbound | null>(null)

// ─── Filters ─────────────────────────────────────────────────────────────────
const filterNode = ref<string | number>('')
const filterCustomer = ref<string | number>('')
const filterProtocol = ref('')
const filterStatus = ref('')

const protocolOptions = [
  { label: 'All', value: '' },
  { label: 'VLESS', value: 'vless' },
  { label: 'VMess', value: 'vmess' },
  { label: 'Trojan', value: 'trojan' },
]

const statusOptions = [
  { label: 'All', value: '' },
  { label: 'Active', value: 'active' },
  { label: 'Disabled', value: 'disabled' },
  { label: 'Pending', value: 'pending' },
]

// ─── Create Form ─────────────────────────────────────────────────────────────
const createForm = ref({
  customer_id: '' as string | number,
  node_id: '' as string | number,
  protocol: 'vless',
  transport: 'tcp',
  security: 'reality',
  port: 443,
  server_name: '',
  public_key: '',
  short_id: '',
  private_key: '',
  path: '',
  service_name: '',
})

// ─── Computed ────────────────────────────────────────────────────────────────
const nodeOptions = computed(() =>
  [{ label: t('xray_inbounds.all_nodes'), value: '' }].concat(
    nodesStore.list.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id as any }))
  )
)

const nodeSelectOptions = computed(() =>
  nodesStore.list.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

const customerSelectOptions = computed(() =>
  customers.value.map(c => ({ label: c.username, value: c.id }))
)

const transportOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'WebSocket', value: 'ws' },
  { label: 'gRPC', value: 'grpc' },
  { label: 'HTTP/2', value: 'h2' },
]

const securityOptions = [
  { label: 'Reality', value: 'reality' },
  { label: 'TLS', value: 'tls' },
  { label: 'None', value: 'none' },
]

const filteredInbounds = computed(() => {
  let list = inbounds.value
  if (filterNode.value) {
    list = list.filter(i => i.node_id === Number(filterNode.value))
  }
  if (filterCustomer.value) {
    list = list.filter(i => i.customer_id === Number(filterCustomer.value))
  }
  if (filterProtocol.value) {
    list = list.filter(i => i.protocol === filterProtocol.value)
  }
  if (filterStatus.value) {
    list = list.filter(i => i.status === filterStatus.value)
  }
  return list
})

const showRealityFields = computed(() =>
  createForm.value.security === 'reality'
)

const showTlsFields = computed(() =>
  createForm.value.security === 'tls'
)

const showPathField = computed(() =>
  ['ws', 'grpc', 'h2'].includes(createForm.value.transport)
)

// ─── API ─────────────────────────────────────────────────────────────────────
async function loadInbounds() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; inbounds: XrayInbound[] }>('/api/xray/inbounds')
    if (res.ok) {
      inbounds.value = res.inbounds || []
    }
  } catch {
    // handled by useApi
  } finally {
    loading.value = false
  }
}

async function loadCustomers() {
  try {
    const res = await get<{ ok: boolean; customers: CustomerOption[] }>('/api/customers?fields=id,username')
    if (res.ok) {
      customers.value = res.customers || []
    }
  } catch {
    // handled by useApi
  }
}

async function handleCreate() {
  if (!createForm.value.customer_id || !createForm.value.node_id) return
  creating.value = true
  try {
    const payload: Record<string, any> = {
      customer_id: Number(createForm.value.customer_id),
      node_id: Number(createForm.value.node_id),
      protocol: createForm.value.protocol,
      transport: createForm.value.transport,
      security: createForm.value.security,
      port: createForm.value.port,
    }
    if (showRealityFields.value) {
      payload.server_name = createForm.value.server_name
      payload.public_key = createForm.value.public_key
      payload.short_id = createForm.value.short_id
      payload.private_key = createForm.value.private_key
    }
    if (showPathField.value) {
      payload.path = createForm.value.path
      if (createForm.value.transport === 'grpc') {
        payload.service_name = createForm.value.service_name
      }
    }

    const res = await post<{ ok: boolean }>('/api/xray/inbounds', payload)
    if (res.ok) {
      toast.success(t('xray_inbounds.created_success'))
      showCreateForm.value = false
      resetCreateForm()
      await loadInbounds()
    }
  } catch {
    // handled by useApi
  } finally {
    creating.value = false
  }
}

function openEdit(inbound: XrayInbound) {
  editingInbound.value = inbound
  createForm.value = {
    customer_id: inbound.customer_id,
    node_id: inbound.node_id,
    protocol: inbound.protocol,
    transport: inbound.transport,
    security: inbound.security,
    port: inbound.port,
    server_name: inbound.server_name || '',
    public_key: inbound.public_key || '',
    short_id: inbound.short_id || '',
    private_key: inbound.private_key || '',
    path: inbound.path || '',
    service_name: inbound.service_name || '',
  }
  showEditForm.value = true
}

async function handleEdit() {
  if (!editingInbound.value) return
  saving.value = true
  try {
    const payload: Record<string, any> = {
      protocol: createForm.value.protocol,
      transport: createForm.value.transport,
      security: createForm.value.security,
      port: createForm.value.port,
    }
    if (createForm.value.security === 'reality') {
      payload.server_name = createForm.value.server_name
      payload.public_key = createForm.value.public_key
      payload.short_id = createForm.value.short_id
      payload.private_key = createForm.value.private_key
    }
    if (['ws', 'grpc', 'h2'].includes(createForm.value.transport)) {
      payload.path = createForm.value.path
      if (createForm.value.transport === 'grpc') {
        payload.service_name = createForm.value.service_name
      }
    }

    const res = await patch<{ ok: boolean }>(`/api/xray/inbounds/${editingInbound.value.id}`, payload)
    if (res.ok) {
      toast.success(t('xray_inbounds.updated_success'))
      showEditForm.value = false
      editingInbound.value = null
      resetCreateForm()
      await loadInbounds()
    }
  } catch {
    // handled by useApi
  } finally {
    saving.value = false
  }
}

async function handleDelete(inbound: XrayInbound) {
  const confirmed = await confirm({
    title: t('xray_inbounds.confirm_delete_title'),
    message: t('xray_inbounds.confirm_delete_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.delete'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  actionInProgress.value = inbound.id
  try {
    const res = await del<{ ok: boolean }>(`/api/xray/inbounds/${inbound.id}`)
    if (res.ok) {
      toast.success(t('xray_inbounds.deleted_success'))
      await loadInbounds()
    }
  } catch {
    // handled by useApi
  } finally {
    actionInProgress.value = null
  }
}

function resetCreateForm() {
  createForm.value = {
    customer_id: '',
    node_id: '',
    protocol: 'vless',
    transport: 'tcp',
    security: 'reality',
    port: 443,
    server_name: '',
    public_key: '',
    short_id: '',
    private_key: '',
    path: '',
    service_name: '',
  }
}

function getNodeName(nodeId: number): string {
  const node = nodesStore.list.find(n => n.id === nodeId)
  return node ? node.name : `#${nodeId}`
}

function getCustomerName(inbound: XrayInbound): string {
  if (inbound.customer_name) return inbound.customer_name
  const customer = customers.value.find(c => c.id === inbound.customer_id)
  return customer ? customer.username : `#${inbound.customer_id}`
}

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const units = ['B', 'KB', 'MB', 'GB', 'TB']
  const i = Math.floor(Math.log(bytes) / Math.log(1024))
  return `${(bytes / Math.pow(1024, i)).toFixed(1)} ${units[i]}`
}

function formatTraffic(inbound: XrayInbound): string {
  const total = inbound.rx_bytes + inbound.tx_bytes
  return `↓${formatBytes(inbound.rx_bytes)} ↑${formatBytes(inbound.tx_bytes)} (${formatBytes(total)})`
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  loadInbounds()
  loadCustomers()
  if (nodesStore.list.length === 0) {
    nodesStore.loadNodes()
  }
})
</script>

<template>
  <div class="page xray-inbounds-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('xray_inbounds.title') }}</h2>
      <div class="page-header__actions">
        <KButton variant="primary" icon="+" @click="showCreateForm = true">
          {{ t('xray_inbounds.create_inbound') }}
        </KButton>
      </div>
    </header>

    <!-- Filter Controls -->
    <div class="filter-bar">
      <KSelect
        v-model="filterNode"
        :options="nodeOptions"
        :placeholder="t('xray_inbounds.filter_node')"
        class="filter-select"
      />
      <KSelect
        v-model="filterProtocol"
        :options="protocolOptions"
        :placeholder="t('xray_inbounds.filter_protocol')"
        class="filter-select"
      />
      <KSelect
        v-model="filterStatus"
        :options="statusOptions"
        :placeholder="t('xray_inbounds.filter_status')"
        class="filter-select"
      />
    </div>

    <!-- Loading -->
    <div v-if="loading && inbounds.length === 0" class="table-loading">
      <KSkeleton v-for="i in 4" :key="i" variant="rect" width="100%" :height="60" />
    </div>

    <!-- Empty state -->
    <KEmptyState
      v-else-if="filteredInbounds.length === 0"
      icon="⚡"
      :title="t('xray_inbounds.empty_title')"
      :description="t('xray_inbounds.empty_desc')"
    />

    <!-- Inbounds table -->
    <div v-else class="table-wrap">
      <table class="data-table">
        <thead>
          <tr>
            <th>{{ t('xray_inbounds.col_customer') }}</th>
            <th>{{ t('xray_inbounds.col_node') }}</th>
            <th>{{ t('xray_inbounds.col_protocol') }}</th>
            <th>{{ t('xray_inbounds.col_transport') }}</th>
            <th>{{ t('xray_inbounds.col_security') }}</th>
            <th>{{ t('xray_inbounds.col_port') }}</th>
            <th>{{ t('xray_inbounds.col_traffic') }}</th>
            <th>{{ t('xray_inbounds.col_status') }}</th>
            <th>{{ t('xray_inbounds.col_actions') }}</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="inbound in filteredInbounds" :key="inbound.id">
            <td>{{ getCustomerName(inbound) }}</td>
            <td>{{ getNodeName(inbound.node_id) }}</td>
            <td class="td-mono td-protocol">
              <span class="protocol-badge" :class="`protocol-badge--${inbound.protocol}`">
                {{ inbound.protocol.toUpperCase() }}
              </span>
            </td>
            <td class="td-mono">{{ inbound.transport }}</td>
            <td>
              <span class="security-badge" :class="`security-badge--${inbound.security}`">
                {{ inbound.security }}
              </span>
            </td>
            <td class="td-mono">{{ inbound.port }}</td>
            <td class="td-traffic">{{ formatTraffic(inbound) }}</td>
            <td>
              <KStatusPill
                :status="inbound.status === 'active' ? 'success' : inbound.status === 'disabled' ? 'danger' : 'neutral'"
              >
                {{ t(`xray_inbounds.status_${inbound.status}`) }}
              </KStatusPill>
            </td>
            <td class="td-actions">
              <KButton variant="ghost" size="sm" @click="openEdit(inbound)">
                ✏️ {{ t('btn.edit') }}
              </KButton>
              <KButton
                variant="danger"
                size="sm"
                :loading="actionInProgress === inbound.id"
                @click="handleDelete(inbound)"
              >
                {{ t('btn.delete') }}
              </KButton>
            </td>
          </tr>
        </tbody>
      </table>
    </div>

    <!-- Create Inbound Slide-Over -->
    <KSlideOver :open="showCreateForm" :title="t('xray_inbounds.create_title')" @close="showCreateForm = false">
      <form class="inbound-form" @submit.prevent="handleCreate">
        <KFormField name="inbound-customer" :label="t('xray_inbounds.field_customer')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="createForm.customer_id"
              :options="customerSelectOptions"
              :placeholder="t('xray_inbounds.select_customer')"
            />
          </template>
        </KFormField>

        <KFormField name="inbound-node" :label="t('xray_inbounds.field_node')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="createForm.node_id"
              :options="nodeSelectOptions"
              :placeholder="t('xray_inbounds.select_node')"
            />
          </template>
        </KFormField>

        <KFormField name="inbound-protocol" :label="t('xray_inbounds.field_protocol')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="createForm.protocol"
              :options="protocolOptions.filter(o => o.value !== '')"
            />
          </template>
        </KFormField>

        <KFormField name="inbound-transport" :label="t('xray_inbounds.field_transport')" required>
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="createForm.transport" :options="transportOptions" />
          </template>
        </KFormField>

        <KFormField name="inbound-security" :label="t('xray_inbounds.field_security')" required>
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="createForm.security" :options="securityOptions" />
          </template>
        </KFormField>

        <KFormField name="inbound-port" :label="t('xray_inbounds.field_port')" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model.number="createForm.port" type="number" placeholder="443" />
          </template>
        </KFormField>

        <!-- Reality Fields -->
        <template v-if="showRealityFields">
          <KFormField name="inbound-sni" :label="t('xray_inbounds.field_server_name')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.server_name" placeholder="www.google.com" />
            </template>
          </KFormField>

          <KFormField name="inbound-pubkey" :label="t('xray_inbounds.field_public_key')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.public_key" placeholder="Public key" />
            </template>
          </KFormField>

          <KFormField name="inbound-shortid" :label="t('xray_inbounds.field_short_id')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.short_id" placeholder="Short ID" />
            </template>
          </KFormField>

          <KFormField name="inbound-privkey" :label="t('xray_inbounds.field_private_key')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.private_key" type="password" placeholder="Private key" />
            </template>
          </KFormField>
        </template>

        <!-- Path/Service Name for WS/gRPC/H2 -->
        <template v-if="showPathField">
          <KFormField name="inbound-path" :label="t('xray_inbounds.field_path')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.path" placeholder="/path" />
            </template>
          </KFormField>

          <KFormField v-if="createForm.transport === 'grpc'" name="inbound-svc" :label="t('xray_inbounds.field_service_name')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.service_name" placeholder="grpc-service" />
            </template>
          </KFormField>
        </template>

        <div class="form-actions">
          <KButton variant="ghost" @click="showCreateForm = false">{{ t('btn.cancel') }}</KButton>
          <KButton type="submit" variant="primary" :loading="creating">{{ t('btn.create') }}</KButton>
        </div>
      </form>
    </KSlideOver>

    <!-- Edit Inbound Slide-Over -->
    <KSlideOver :open="showEditForm" :title="t('xray_inbounds.edit_title')" @close="showEditForm = false; editingInbound = null">
      <form class="inbound-form" @submit.prevent="handleEdit">
        <KFormField name="edit-protocol" :label="t('xray_inbounds.field_protocol')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="createForm.protocol"
              :options="protocolOptions.filter(o => o.value !== '')"
            />
          </template>
        </KFormField>

        <KFormField name="edit-transport" :label="t('xray_inbounds.field_transport')" required>
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="createForm.transport" :options="transportOptions" />
          </template>
        </KFormField>

        <KFormField name="edit-security" :label="t('xray_inbounds.field_security')" required>
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="createForm.security" :options="securityOptions" />
          </template>
        </KFormField>

        <KFormField name="edit-port" :label="t('xray_inbounds.field_port')" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model.number="createForm.port" type="number" placeholder="443" />
          </template>
        </KFormField>

        <!-- Reality Fields -->
        <template v-if="showRealityFields">
          <KFormField name="edit-sni" :label="t('xray_inbounds.field_server_name')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.server_name" placeholder="www.google.com" />
            </template>
          </KFormField>

          <KFormField name="edit-pubkey" :label="t('xray_inbounds.field_public_key')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.public_key" placeholder="Public key" />
            </template>
          </KFormField>

          <KFormField name="edit-shortid" :label="t('xray_inbounds.field_short_id')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.short_id" placeholder="Short ID" />
            </template>
          </KFormField>

          <KFormField name="edit-privkey" :label="t('xray_inbounds.field_private_key')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.private_key" type="password" placeholder="Private key" />
            </template>
          </KFormField>
        </template>

        <!-- Path/Service Name for WS/gRPC/H2 -->
        <template v-if="showPathField">
          <KFormField name="edit-path" :label="t('xray_inbounds.field_path')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.path" placeholder="/path" />
            </template>
          </KFormField>

          <KFormField v-if="createForm.transport === 'grpc'" name="edit-svc" :label="t('xray_inbounds.field_service_name')">
            <template #default="{ fieldId }">
              <KInput :id="fieldId" v-model="createForm.service_name" placeholder="grpc-service" />
            </template>
          </KFormField>
        </template>

        <div class="form-actions">
          <KButton variant="ghost" @click="showEditForm = false; editingInbound = null">{{ t('btn.cancel') }}</KButton>
          <KButton type="submit" variant="primary" :loading="saving">{{ t('btn.save') }}</KButton>
        </div>
      </form>
    </KSlideOver>
  </div>
</template>

<style scoped>
.xray-inbounds-view {
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

.filter-bar {
  display: flex;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
}

.filter-select {
  min-width: 160px;
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

.td-traffic {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-xs);
  white-space: nowrap;
}

.td-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}

/* ─── Protocol Badge ──────────────────────────────────────────────────────── */
.protocol-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: var(--radius-full, 9999px);
  font-size: var(--text-xs);
  font-weight: 600;
  text-transform: uppercase;
}

.protocol-badge--vless {
  background: rgba(99, 102, 241, 0.15);
  color: #818cf8;
}

.protocol-badge--vmess {
  background: rgba(34, 197, 94, 0.15);
  color: #4ade80;
}

.protocol-badge--trojan {
  background: rgba(234, 179, 8, 0.15);
  color: #facc15;
}

/* ─── Security Badge ──────────────────────────────────────────────────────── */
.security-badge {
  display: inline-block;
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
  font-weight: 500;
}

.security-badge--reality {
  background: rgba(139, 92, 246, 0.15);
  color: #a78bfa;
}

.security-badge--tls {
  background: rgba(6, 182, 212, 0.15);
  color: #22d3ee;
}

.security-badge--none {
  background: rgba(107, 114, 128, 0.15);
  color: #9ca3af;
}

/* ─── Form ─────────────────────────────────────────────────────────────────── */
.inbound-form {
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
  .filter-bar {
    flex-direction: column;
  }

  .filter-select {
    min-width: 100%;
  }

  .table-wrap {
    font-size: var(--text-xs);
  }

  .td-actions {
    flex-direction: column;
  }
}
</style>
