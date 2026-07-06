<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { formatDate } from '@koris/composables/useFormatDate'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import FormField from '@koris/ui/FormField.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import Drawer from '@koris/ui/Drawer.vue'

const { t } = useI18n()
const { get, post, patch, del } = useApi()
const toast = useToast()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface PaymentGateway {
  id: number
  name: string
  display_name: string
  config_json: string
  is_active: boolean
  created_at: string
}

interface GatewayListResponse {
  ok: boolean
  gateways: PaymentGateway[]
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const gateways = ref<PaymentGateway[]>([])
const loading = ref(false)
const showDrawer = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)
const confirmDeleteId = ref<number | null>(null)

const form = ref({
  name: '',
  display_name: '',
  config_json: '{}',
  is_active: true,
})

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchGateways() {
  loading.value = true
  try {
    const data = await get<GatewayListResponse>('/api/gateways')
    if (data?.ok) {
      gateways.value = data.gateways || []
    }
  } catch {
    gateways.value = []
  } finally {
    loading.value = false
  }
}

async function submitGateway() {
  // Validate JSON
  try {
    JSON.parse(form.value.config_json)
  } catch {
    toast.error(t('gateways.invalid_json'))
    return
  }

  saving.value = true
  try {
    if (editingId.value) {
      await patch<{ ok: boolean }>(`/api/gateways/${editingId.value}`, {
        name: form.value.name,
        display_name: form.value.display_name,
        config_json: form.value.config_json,
        is_active: form.value.is_active,
      })
      toast.success(t('gateways.update_success'))
    } else {
      await post<{ ok: boolean }>('/api/gateways', {
        name: form.value.name,
        display_name: form.value.display_name,
        config_json: form.value.config_json,
        is_active: form.value.is_active,
      })
      toast.success(t('gateways.create_success'))
    }
    closeDrawer()
    await fetchGateways()
  } catch {
    // error toast handled by useApi
  } finally {
    saving.value = false
  }
}

async function toggleActive(gw: PaymentGateway) {
  try {
    await patch<{ ok: boolean }>(`/api/gateways/${gw.id}`, {
      is_active: !gw.is_active,
    })
    gw.is_active = !gw.is_active
    toast.success(t('gateways.toggle_success'))
  } catch {
    // error toast handled by useApi
  }
}

async function deleteGateway(id: number) {
  try {
    await del<{ ok: boolean }>(`/api/gateways/${id}`)
    toast.success(t('gateways.delete_success'))
    gateways.value = gateways.value.filter(g => g.id !== id)
  } catch {
    // error toast handled by useApi
  } finally {
    confirmDeleteId.value = null
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function openAdd() {
  editingId.value = null
  form.value = { name: '', display_name: '', config_json: '{}', is_active: true }
  showDrawer.value = true
}

function openEdit(gw: PaymentGateway) {
  editingId.value = gw.id
  form.value = {
    name: gw.name,
    display_name: gw.display_name,
    config_json: formatConfigJson(gw.config_json),
    is_active: gw.is_active,
  }
  showDrawer.value = true
}

function closeDrawer() {
  showDrawer.value = false
  editingId.value = null
  form.value = { name: '', display_name: '', config_json: '{}', is_active: true }
}

function formatConfigJson(raw: string): string {
  try {
    return JSON.stringify(JSON.parse(raw), null, 2)
  } catch {
    return raw
  }
}

const drawerTitle = computed(() =>
  editingId.value ? t('gateways.edit_gateway') : t('gateways.add_gateway')
)

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(fetchGateways)
</script>

<template>
  <div class="page gateways-view">
    <section class="panel">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ t('gateways.title') }}</h3>
          <p class="panel-subtitle">{{ t('gateways.subtitle') }}</p>
        </div>
        <Button variant="primary" size="sm" @click="openAdd">
          {{ t('gateways.add_gateway') }}
        </Button>
      </div>

      <!-- Loading -->
      <div v-if="loading" class="skeleton-wrap">
        <Skeleton variant="table-row" :count="3" />
      </div>

      <!-- Empty -->
      <div v-else-if="gateways.length === 0" class="empty-state">
        <p class="text-muted">{{ t('gateways.no_gateways') }}</p>
      </div>

      <!-- Gateway Table -->
      <div v-else class="table-wrap">
        <table class="data-table" role="table">
          <thead>
            <tr>
              <th>{{ t('gateways.col_name') }}</th>
              <th>{{ t('gateways.col_display_name') }}</th>
              <th>{{ t('gateways.col_status') }}</th>
              <th>{{ t('gateways.col_config') }}</th>
              <th>{{ t('gateways.col_created') }}</th>
              <th>{{ t('gateways.col_actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="gw in gateways" :key="gw.id">
              <td class="cell-name">
                <code>{{ gw.name }}</code>
              </td>
              <td>{{ gw.display_name }}</td>
              <td>
                <StatusPill :status="gw.is_active ? 'active' : 'disabled'" size="sm">
                  {{ gw.is_active ? t('gateways.active') : t('gateways.inactive') }}
                </StatusPill>
              </td>
              <td class="cell-config">
                <pre class="config-preview">{{ formatConfigJson(gw.config_json) }}</pre>
              </td>
              <td class="text-muted">{{ formatDate(gw.created_at) }}</td>
              <td>
                <div class="action-btns">
                  <Button variant="ghost" size="sm" @click="openEdit(gw)">
                    {{ t('gateways.edit') }}
                  </Button>
                  <Button variant="ghost" size="sm" @click="toggleActive(gw)">
                    {{ gw.is_active ? t('gateways.disable') : t('gateways.enable') }}
                  </Button>
                  <Button
                    v-if="confirmDeleteId !== gw.id"
                    variant="danger"
                    size="sm"
                    @click="confirmDeleteId = gw.id"
                  >
                    {{ t('gateways.delete') }}
                  </Button>
                  <template v-else>
                    <Button variant="danger" size="sm" @click="deleteGateway(gw.id)">
                      {{ t('gateways.confirm') }}
                    </Button>
                    <Button variant="ghost" size="sm" @click="confirmDeleteId = null">
                      {{ t('btn.cancel') }}
                    </Button>
                  </template>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Add/Edit Drawer -->
    <Drawer :open="showDrawer" :title="drawerTitle" side="right" @close="closeDrawer">
      <form class="gateway-form" @submit.prevent="submitGateway">
        <FormField name="gw-name" :label="t('gateways.form_name')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.name" placeholder="zarinpal" :disabled="!!editingId" />
          </template>
        </FormField>

        <FormField name="gw-display-name" :label="t('gateways.form_display_name')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.display_name" placeholder="Zarinpal" />
          </template>
        </FormField>

        <FormField name="gw-config" :label="t('gateways.form_config_json')">
          <template #default="{ fieldId }">
            <textarea
              :id="fieldId"
              v-model="form.config_json"
              class="config-editor"
              rows="8"
              spellcheck="false"
              :placeholder="'{\n  &quot;merchant_id&quot;: &quot;xxx&quot;\n}'"
            />
          </template>
        </FormField>

        <div class="toggle-field">
          <label class="toggle-label">
            <input type="checkbox" v-model="form.is_active" />
            <span>{{ t('gateways.form_active') }}</span>
          </label>
        </div>

        <Button type="submit" variant="primary" :loading="saving" full-width>
          {{ editingId ? t('gateways.save_changes') : t('gateways.create_gateway') }}
        </Button>
      </form>
    </Drawer>
  </div>
</template>

<style scoped>
.gateways-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.panel {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.panel-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: var(--space-4);
  flex-wrap: wrap;
  gap: var(--space-3);
}
.panel-title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}
.panel-subtitle {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin: 4px 0 0;
}

/* Table */
.table-wrap {
  overflow-x: auto;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.data-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  color: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}
.data-table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  vertical-align: top;
}
.cell-name code {
  padding: 2px 6px;
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}
.cell-config {
  max-width: 280px;
}
.config-preview {
  margin: 0;
  padding: var(--space-2);
  font-size: var(--text-xs);
  font-family: var(--font-mono, monospace);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  max-height: 80px;
  overflow: auto;
  white-space: pre-wrap;
  word-break: break-all;
}

/* Actions */
.action-btns {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
}

/* Form */
.gateway-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.config-editor {
  width: 100%;
  padding: var(--space-2);
  font-size: var(--text-sm);
  font-family: var(--font-mono, monospace);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  resize: vertical;
  min-height: 120px;
  line-height: 1.5;
}
.config-editor:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}

/* Toggle */
.toggle-field {
  padding: var(--space-2) 0;
}
.toggle-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--color-text);
  cursor: pointer;
}
.toggle-label input[type="checkbox"] {
  width: 1rem;
  height: 1rem;
  accent-color: var(--color-primary);
}

/* Empty / Skeleton */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80px;
}
.skeleton-wrap {
  padding: var(--space-2) 0;
}

.text-muted { color: var(--color-muted); }

@media (max-width: 768px) {
  .data-table th:nth-child(4),
  .data-table td:nth-child(4) {
    display: none;
  }
}
</style>
