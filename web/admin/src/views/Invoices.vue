<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
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
const { get, post } = useApi()
const toast = useToast()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface Invoice {
  id: number
  invoice_number: string
  customer_id: number
  customer_name: string
  amount: number
  tax: number
  total: number
  currency: string
  plan_name: string
  payment_method: string
  status: 'paid' | 'refunded' | 'partially_refunded'
  refunded_amount: number
  created_at: string
}

interface InvoiceListResponse {
  ok: boolean
  invoices: Invoice[]
  total: number
  page: number
  page_size: number
}

interface InvoiceDetailResponse {
  ok: boolean
  invoice: Invoice
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const invoices = ref<Invoice[]>([])
const totalCount = ref(0)
const page = ref(1)
const pageSize = 20
const loading = ref(false)

// Filters
const filterStatus = ref('')
const filterDateFrom = ref('')
const filterDateTo = ref('')
const filterCustomer = ref('')

// Detail view
const selectedInvoice = ref<Invoice | null>(null)
const showDetail = ref(false)

// Refund
const showRefundDrawer = ref(false)
const refundInvoice = ref<Invoice | null>(null)
const refundAmount = ref('')
const refundLoading = ref(false)

// ═══════════════════════════════════════════════════════════════════════════════
// Filter Options
// ═══════════════════════════════════════════════════════════════════════════════

const statusOptions = computed(() => [
  { label: t('invoices.all_statuses'), value: '' },
  { label: t('invoices.status_paid'), value: 'paid' },
  { label: t('invoices.status_refunded'), value: 'refunded' },
  { label: t('invoices.status_partially_refunded'), value: 'partially_refunded' },
])

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchInvoices() {
  loading.value = true
  try {
    let url = `/api/invoices?page=${page.value}&page_size=${pageSize}`
    if (filterStatus.value) url += `&status=${filterStatus.value}`
    if (filterDateFrom.value) url += `&from=${filterDateFrom.value}`
    if (filterDateTo.value) url += `&to=${filterDateTo.value}`
    if (filterCustomer.value) url += `&customer=${encodeURIComponent(filterCustomer.value)}`

    const data = await get<InvoiceListResponse>(url)
    if (data?.ok) {
      invoices.value = data.invoices || []
      totalCount.value = data.total || 0
    }
  } catch {
    invoices.value = []
    totalCount.value = 0
  } finally {
    loading.value = false
  }
}

async function openDetail(invoice: Invoice) {
  selectedInvoice.value = invoice
  showDetail.value = true
}

function downloadInvoice(invoice: Invoice) {
  window.open(`/api/invoices/${invoice.id}/download`, '_blank')
}

function openRefund(invoice: Invoice) {
  refundInvoice.value = invoice
  refundAmount.value = ''
  showRefundDrawer.value = true
}

async function submitRefund() {
  if (!refundInvoice.value) return

  const amount = refundAmount.value ? parseFloat(refundAmount.value) : refundInvoice.value.total

  if (isNaN(amount) || amount <= 0) {
    toast.error(t('invoices.invalid_amount'))
    return
  }

  if (amount > refundInvoice.value.total - refundInvoice.value.refunded_amount) {
    toast.error(t('invoices.refund_exceeds'))
    return
  }

  refundLoading.value = true
  try {
    await post<{ ok: boolean }>(`/api/invoices/${refundInvoice.value.id}/refund`, { amount })
    toast.success(t('invoices.refund_success'))
    showRefundDrawer.value = false
    refundInvoice.value = null
    await fetchInvoices()
  } catch {
    // error toast handled by useApi
  } finally {
    refundLoading.value = false
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Pagination
// ═══════════════════════════════════════════════════════════════════════════════

const totalPages = computed(() => Math.max(1, Math.ceil(totalCount.value / pageSize)))

function prevPage() {
  if (page.value > 1) {
    page.value--
    fetchInvoices()
  }
}

function nextPage() {
  if (page.value < totalPages.value) {
    page.value++
    fetchInvoices()
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function statusVariant(status: string): string {
  switch (status) {
    case 'paid': return 'active'
    case 'refunded': return 'danger'
    case 'partially_refunded': return 'warning'
    default: return 'default'
  }
}

function applyFilters() {
  page.value = 1
  fetchInvoices()
}

function clearFilters() {
  filterStatus.value = ''
  filterDateFrom.value = ''
  filterDateTo.value = ''
  filterCustomer.value = ''
  page.value = 1
  fetchInvoices()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(fetchInvoices)
</script>

<template>
  <div class="page invoices-view">
    <!-- Filter Controls -->
    <section class="panel filter-section">
      <div class="panel-header">
        <h3 class="panel-title">{{ t('invoices.title') }}</h3>
      </div>
      <div class="filter-row">
        <Select
          v-model="filterStatus"
          :options="statusOptions"
          :aria-label="t('invoices.filter_status')"
          class="filter-select"
        />
        <input
          v-model="filterDateFrom"
          type="date"
          class="date-input"
          :aria-label="t('invoices.filter_from')"
        />
        <input
          v-model="filterDateTo"
          type="date"
          class="date-input"
          :aria-label="t('invoices.filter_to')"
        />
        <Input
          v-model="filterCustomer"
          :placeholder="t('invoices.search_customer')"
          class="customer-search"
          @keyup.enter="applyFilters"
        />
        <Button variant="primary" size="sm" @click="applyFilters">
          {{ t('invoices.filter') }}
        </Button>
        <Button variant="ghost" size="sm" @click="clearFilters">
          {{ t('invoices.clear') }}
        </Button>
      </div>
    </section>

    <!-- Invoice Table -->
    <section class="panel">
      <div v-if="loading" class="skeleton-wrap">
        <Skeleton variant="table-row" :count="5" />
      </div>

      <div v-else-if="invoices.length === 0" class="empty-state">
        <p class="text-muted">{{ t('invoices.no_invoices') }}</p>
      </div>

      <div v-else class="table-wrap">
        <table class="data-table" role="table">
          <thead>
            <tr>
              <th>{{ t('invoices.col_number') }}</th>
              <th>{{ t('invoices.col_customer') }}</th>
              <th>{{ t('invoices.col_amount') }}</th>
              <th>{{ t('invoices.col_total') }}</th>
              <th>{{ t('invoices.col_status') }}</th>
              <th>{{ t('invoices.col_date') }}</th>
              <th>{{ t('invoices.col_actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="inv in invoices"
              :key="inv.id"
              class="clickable-row"
              @click="openDetail(inv)"
            >
              <td><code>{{ inv.invoice_number }}</code></td>
              <td>{{ inv.customer_name }}</td>
              <td class="amount-cell">{{ inv.amount.toFixed(2) }} {{ inv.currency }}</td>
              <td class="amount-cell">{{ inv.total.toFixed(2) }} {{ inv.currency }}</td>
              <td>
                <StatusPill :status="statusVariant(inv.status)" size="sm">
                  {{ t(`invoices.status_${inv.status}`) }}
                </StatusPill>
              </td>
              <td class="text-muted">{{ formatDate(inv.created_at) }}</td>
              <td>
                <div class="action-btns" @click.stop>
                  <Button variant="ghost" size="sm" @click="downloadInvoice(inv)">
                    📥 {{ t('invoices.download') }}
                  </Button>
                  <Button
                    v-if="inv.status === 'paid' || inv.status === 'partially_refunded'"
                    variant="ghost"
                    size="sm"
                    @click="openRefund(inv)"
                  >
                    ↩ {{ t('invoices.refund') }}
                  </Button>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="totalCount > pageSize" class="pagination">
        <Button variant="ghost" size="sm" :disabled="page <= 1" @click="prevPage">
          ← {{ t('invoices.prev') }}
        </Button>
        <span class="pagination__info">{{ page }} / {{ totalPages }}</span>
        <Button variant="ghost" size="sm" :disabled="page >= totalPages" @click="nextPage">
          {{ t('invoices.next') }} →
        </Button>
      </div>
    </section>

    <!-- Invoice Detail Drawer -->
    <Drawer :open="showDetail" :title="t('invoices.detail_title')" side="right" @close="showDetail = false">
      <div v-if="selectedInvoice" class="invoice-detail">
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.col_number') }}</span>
          <code>{{ selectedInvoice.invoice_number }}</code>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.col_customer') }}</span>
          <span>{{ selectedInvoice.customer_name }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.detail_plan') }}</span>
          <span>{{ selectedInvoice.plan_name || '—' }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.detail_payment_method') }}</span>
          <span>{{ selectedInvoice.payment_method || '—' }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.col_amount') }}</span>
          <span class="amount-cell">{{ selectedInvoice.amount.toFixed(2) }} {{ selectedInvoice.currency }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.detail_tax') }}</span>
          <span>{{ selectedInvoice.tax.toFixed(2) }} {{ selectedInvoice.currency }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.col_total') }}</span>
          <span class="amount-cell amount-total">{{ selectedInvoice.total.toFixed(2) }} {{ selectedInvoice.currency }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.col_status') }}</span>
          <StatusPill :status="statusVariant(selectedInvoice.status)" size="sm">
            {{ t(`invoices.status_${selectedInvoice.status}`) }}
          </StatusPill>
        </div>
        <div v-if="selectedInvoice.refunded_amount > 0" class="detail-row">
          <span class="detail-label">{{ t('invoices.detail_refunded') }}</span>
          <span class="text-danger">{{ selectedInvoice.refunded_amount.toFixed(2) }} {{ selectedInvoice.currency }}</span>
        </div>
        <div class="detail-row">
          <span class="detail-label">{{ t('invoices.col_date') }}</span>
          <span>{{ formatDate(selectedInvoice.created_at) }}</span>
        </div>

        <div class="detail-actions">
          <Button variant="primary" size="sm" @click="downloadInvoice(selectedInvoice)">
            📥 {{ t('invoices.download') }}
          </Button>
          <Button
            v-if="selectedInvoice.status === 'paid' || selectedInvoice.status === 'partially_refunded'"
            variant="danger"
            size="sm"
            @click="openRefund(selectedInvoice); showDetail = false"
          >
            ↩ {{ t('invoices.refund') }}
          </Button>
        </div>
      </div>
    </Drawer>

    <!-- Refund Drawer -->
    <Drawer :open="showRefundDrawer" :title="t('invoices.refund_title')" side="right" @close="showRefundDrawer = false">
      <form v-if="refundInvoice" class="refund-form" @submit.prevent="submitRefund">
        <div class="refund-info">
          <p class="refund-invoice-num">{{ refundInvoice.invoice_number }}</p>
          <p class="text-muted">
            {{ t('invoices.refund_max') }}: {{ (refundInvoice.total - refundInvoice.refunded_amount).toFixed(2) }} {{ refundInvoice.currency }}
          </p>
        </div>

        <FormField name="refund-amount" :label="t('invoices.refund_amount')">
          <template #default="{ fieldId }">
            <Input
              :id="fieldId"
              v-model="refundAmount"
              type="number"
              step="0.01"
              min="0.01"
              :max="(refundInvoice.total - refundInvoice.refunded_amount).toString()"
              :placeholder="t('invoices.refund_full_placeholder')"
            />
          </template>
        </FormField>

        <p class="text-muted text-xs">
          {{ t('invoices.refund_hint') }}
        </p>

        <Button type="submit" variant="danger" :loading="refundLoading" full-width>
          {{ t('invoices.confirm_refund') }}
        </Button>
      </form>
    </Drawer>
  </div>
</template>

<style scoped>
.invoices-view {
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
  align-items: flex-start;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}
.panel-title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

/* Filters */
.filter-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.filter-select {
  width: 180px;
}
.date-input {
  width: 140px;
  height: 32px;
  padding: 0 var(--space-2);
  font-size: var(--text-sm);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
  background: transparent;
  color: var(--color-text);
  font-family: inherit;
}
.date-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.15);
}
.customer-search {
  width: 200px;
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
.data-table code {
  padding: 2px 6px;
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  font-size: var(--text-xs);
}
.clickable-row {
  cursor: pointer;
  transition: background 0.15s;
}
.clickable-row:hover {
  background: var(--color-surface-2, rgba(0, 0, 0, 0.02));
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

/* Pagination */
.pagination {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}
.pagination__info {
  font-size: var(--text-sm);
  color: var(--color-muted);
}

/* Detail Drawer */
.invoice-detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.detail-row {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: var(--space-2) 0;
  border-bottom: 1px solid var(--color-border);
}
.detail-row:last-of-type {
  border-bottom: none;
}
.detail-label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}
.amount-total {
  font-size: var(--text-base);
  font-weight: var(--font-bold);
}
.detail-actions {
  display: flex;
  gap: var(--space-2);
  margin-top: var(--space-4);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Refund Drawer */
.refund-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.refund-info {
  padding: var(--space-3);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.refund-invoice-num {
  margin: 0 0 var(--space-1);
  font-weight: var(--font-semibold);
  font-family: var(--font-mono, monospace);
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
.text-danger { color: var(--color-danger); }
.text-xs { font-size: var(--text-xs); }

@media (max-width: 768px) {
  .filter-row { flex-direction: column; align-items: stretch; }
  .filter-select, .date-input, .customer-search { width: 100%; }
}
</style>
