<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { formatDate } from '@koris/composables/useFormatDate'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import FormField from '@koris/ui/FormField.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import Drawer from '@koris/ui/Drawer.vue'

const { t } = useI18n()
const { get, patch } = useApi()
const toast = useToast()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface Payout {
  id: number
  reseller_username: string
  amount: number
  status: 'pending' | 'approved' | 'rejected'
  payment_details: string
  admin_note: string
  requested_at: string
  processed_at: string | null
  processed_by: string | null
}

interface ResellerInfo {
  username: string
  payout_balance: number
  commission_percent: number
  min_payout_amount: number
}

interface PayoutListResponse {
  ok: boolean
  payouts: Payout[]
  total: number
}

interface ResellerListResponse {
  ok: boolean
  resellers: ResellerInfo[]
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const payouts = ref<Payout[]>([])
const resellers = ref<ResellerInfo[]>([])
const totalCount = ref(0)
const loading = ref(false)
const resellersLoading = ref(false)

// Filters
const filterStatus = ref('')

// Action Drawer
const showActionDrawer = ref(false)
const actionPayout = ref<Payout | null>(null)
const actionType = ref<'approve' | 'reject'>('approve')
const adminNote = ref('')
const actionLoading = ref(false)

// ═══════════════════════════════════════════════════════════════════════════════
// Filter Options
// ═══════════════════════════════════════════════════════════════════════════════

const statusOptions = computed(() => [
  { label: t('payouts.all_statuses'), value: '' },
  { label: t('payouts.status_pending'), value: 'pending' },
  { label: t('payouts.status_approved'), value: 'approved' },
  { label: t('payouts.status_rejected'), value: 'rejected' },
])

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchPayouts() {
  loading.value = true
  try {
    let url = '/api/admin/payouts'
    if (filterStatus.value) url += `?status=${filterStatus.value}`

    const data = await get<PayoutListResponse>(url)
    if (data?.ok) {
      payouts.value = data.payouts || []
      totalCount.value = data.total || 0
    }
  } catch {
    payouts.value = []
    totalCount.value = 0
  } finally {
    loading.value = false
  }
}

async function fetchResellers() {
  resellersLoading.value = true
  try {
    const data = await get<ResellerListResponse>('/api/admin/payouts/resellers')
    if (data?.ok) {
      resellers.value = data.resellers || []
    }
  } catch {
    resellers.value = []
  } finally {
    resellersLoading.value = false
  }
}

function openAction(payout: Payout, type: 'approve' | 'reject') {
  actionPayout.value = payout
  actionType.value = type
  adminNote.value = ''
  showActionDrawer.value = true
}

async function submitAction() {
  if (!actionPayout.value) return

  actionLoading.value = true
  try {
    await patch<{ ok: boolean }>(`/api/admin/payouts/${actionPayout.value.id}`, {
      status: actionType.value === 'approve' ? 'approved' : 'rejected',
      admin_note: adminNote.value || undefined,
    })
    toast.success(
      actionType.value === 'approve'
        ? t('payouts.approve_success')
        : t('payouts.reject_success')
    )
    showActionDrawer.value = false
    actionPayout.value = null
    await fetchPayouts()
    await fetchResellers()
  } catch {
    // error toast handled by useApi
  } finally {
    actionLoading.value = false
  }
}

function applyFilter() {
  fetchPayouts()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function statusVariant(status: string): string {
  switch (status) {
    case 'pending': return 'warning'
    case 'approved': return 'active'
    case 'rejected': return 'danger'
    default: return 'default'
  }
}

const actionDrawerTitle = computed(() =>
  actionType.value === 'approve' ? t('payouts.approve_payout') : t('payouts.reject_payout')
)

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(() => {
  fetchPayouts()
  fetchResellers()
})
</script>

<template>
  <div class="page payouts-view">
    <!-- Reseller Balances -->
    <section class="panel">
      <div class="panel-header">
        <h3 class="panel-title">{{ t('payouts.reseller_balances') }}</h3>
      </div>

      <div v-if="resellersLoading" class="skeleton-wrap">
        <Skeleton variant="card" :count="3" />
      </div>
      <div v-else-if="resellers.length === 0" class="empty-state">
        <p class="text-muted">{{ t('payouts.no_resellers') }}</p>
      </div>
      <div v-else class="reseller-grid">
        <div v-for="r in resellers" :key="r.username" class="reseller-card">
          <div class="reseller-card__header">
            <span class="reseller-card__name">{{ r.username }}</span>
            <span class="reseller-card__commission">{{ r.commission_percent }}%</span>
          </div>
          <div class="reseller-card__body">
            <div class="reseller-stat">
              <span class="reseller-stat__value">{{ r.payout_balance.toFixed(2) }}</span>
              <span class="reseller-stat__label">{{ t('payouts.balance') }}</span>
            </div>
            <div class="reseller-stat">
              <span class="reseller-stat__value">{{ r.min_payout_amount.toFixed(2) }}</span>
              <span class="reseller-stat__label">{{ t('payouts.min_payout') }}</span>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Payout Requests Table -->
    <section class="panel">
      <div class="panel-header">
        <h3 class="panel-title">{{ t('payouts.title') }}</h3>
        <div class="filter-row">
          <Select
            v-model="filterStatus"
            :options="statusOptions"
            :aria-label="t('payouts.filter_status')"
            class="filter-select"
            @update:model-value="applyFilter"
          />
        </div>
      </div>

      <div v-if="loading" class="skeleton-wrap">
        <Skeleton variant="table-row" :count="5" />
      </div>

      <div v-else-if="payouts.length === 0" class="empty-state">
        <p class="text-muted">{{ t('payouts.no_payouts') }}</p>
      </div>

      <div v-else class="table-wrap">
        <table class="data-table" role="table">
          <thead>
            <tr>
              <th>{{ t('payouts.col_reseller') }}</th>
              <th>{{ t('payouts.col_amount') }}</th>
              <th>{{ t('payouts.col_status') }}</th>
              <th>{{ t('payouts.col_requested') }}</th>
              <th>{{ t('payouts.col_processed') }}</th>
              <th>{{ t('payouts.col_actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="payout in payouts" :key="payout.id">
              <td>{{ payout.reseller_username }}</td>
              <td class="amount-cell">{{ payout.amount.toFixed(2) }}</td>
              <td>
                <StatusPill :status="statusVariant(payout.status)" size="sm">
                  {{ t(`payouts.status_${payout.status}`) }}
                </StatusPill>
              </td>
              <td class="text-muted">{{ formatDate(payout.requested_at) }}</td>
              <td class="text-muted">
                {{ payout.processed_at ? formatDate(payout.processed_at) : '—' }}
              </td>
              <td>
                <div v-if="payout.status === 'pending'" class="action-btns">
                  <Button variant="primary" size="sm" @click="openAction(payout, 'approve')">
                    {{ t('payouts.approve') }}
                  </Button>
                  <Button variant="danger" size="sm" @click="openAction(payout, 'reject')">
                    {{ t('payouts.reject') }}
                  </Button>
                </div>
                <span v-else class="text-muted">
                  <template v-if="payout.admin_note">{{ payout.admin_note }}</template>
                  <template v-else>—</template>
                </span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Approve/Reject Drawer -->
    <Drawer :open="showActionDrawer" :title="actionDrawerTitle" side="right" @close="showActionDrawer = false">
      <form v-if="actionPayout" class="action-form" @submit.prevent="submitAction">
        <div class="action-info">
          <div class="action-info__row">
            <span class="action-info__label">{{ t('payouts.col_reseller') }}</span>
            <span>{{ actionPayout.reseller_username }}</span>
          </div>
          <div class="action-info__row">
            <span class="action-info__label">{{ t('payouts.col_amount') }}</span>
            <span class="amount-cell">{{ actionPayout.amount.toFixed(2) }}</span>
          </div>
          <div v-if="actionPayout.payment_details" class="action-info__row">
            <span class="action-info__label">{{ t('payouts.payment_details') }}</span>
            <span class="text-muted">{{ actionPayout.payment_details }}</span>
          </div>
        </div>

        <FormField name="admin-note" :label="t('payouts.admin_note')">
          <template #default="{ fieldId }">
            <textarea
              :id="fieldId"
              v-model="adminNote"
              class="note-textarea"
              rows="3"
              :placeholder="t('payouts.note_placeholder')"
            />
          </template>
        </FormField>

        <Button
          type="submit"
          :variant="actionType === 'approve' ? 'primary' : 'danger'"
          :loading="actionLoading"
          full-width
        >
          {{ actionType === 'approve' ? t('payouts.confirm_approve') : t('payouts.confirm_reject') }}
        </Button>
      </form>
    </Drawer>
  </div>
</template>

<style scoped>
.payouts-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.panel {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.panel-header {
  display: flex;
  align-items: center;
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

/* Reseller Grid */
.reseller-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: var(--space-3);
}
.reseller-card {
  padding: var(--space-3);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.reseller-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.reseller-card__name {
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
  color: var(--color-text);
}
.reseller-card__commission {
  font-size: var(--text-xs);
  color: var(--color-primary);
  font-weight: var(--font-semibold);
  background: rgba(59, 130, 246, 0.1);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
.reseller-card__body {
  display: flex;
  gap: var(--space-4);
}
.reseller-stat {
  display: flex;
  flex-direction: column;
}
.reseller-stat__value {
  font-size: var(--text-base);
  font-weight: var(--font-bold);
  color: var(--color-text);
  font-family: var(--font-mono, monospace);
}
.reseller-stat__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}

/* Filter */
.filter-row {
  display: flex;
  gap: var(--space-2);
}
.filter-select {
  width: 160px;
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
  white-space: nowrap;
}
.amount-cell {
  font-weight: var(--font-semibold);
  color: var(--color-success);
  font-family: var(--font-mono, monospace);
}

/* Actions */
.action-btns {
  display: flex;
  gap: var(--space-1);
}

/* Action Form */
.action-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.action-info {
  padding: var(--space-3);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.action-info__row {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.action-info__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}
.note-textarea {
  width: 100%;
  padding: var(--space-2);
  font-size: var(--text-sm);
  font-family: inherit;
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  resize: vertical;
}
.note-textarea:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
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
  .reseller-grid { grid-template-columns: 1fr; }
  .filter-row { width: 100%; }
  .filter-select { width: 100%; }
}
</style>
