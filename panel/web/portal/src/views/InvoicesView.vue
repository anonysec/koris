<script setup lang="ts">
import { ref, computed } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useFreshData } from '@koris/composables/useFreshData'
import { formatDate } from '@koris/composables/useFormatDate'
import KButton from '@koris/ui/KButton.vue'
import KDataTable from '@koris/ui/KDataTable.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'

interface Invoice {
  id: number
  invoice_number: string
  amount: number
  tax: number
  total: number
  currency: string
  plan_name: string | null
  payment_method: string | null
  status: string
  refunded_amount: number
  created_at: string
}

interface InvoicesResponse {
  ok: boolean
  invoices: Invoice[]
}

interface InvoiceDetailResponse {
  ok: boolean
  invoice: Invoice
}

const { get, loading } = useApi()
const { t } = useI18n()

const invoices = ref<Invoice[]>([])
const selectedInvoice = ref<Invoice | null>(null)
const showDetail = ref(false)

useFreshData(async () => {
  await fetchInvoices()
})

async function fetchInvoices() {
  try {
    const res = await get<InvoicesResponse>('/api/portal/invoices')
    invoices.value = res.invoices || []
  } catch {
    // keep empty state
  }
}

async function viewInvoice(invoice: Invoice) {
  try {
    const res = await get<InvoiceDetailResponse>(`/api/portal/invoices/${invoice.id}`)
    selectedInvoice.value = res.invoice
    showDetail.value = true
  } catch {
    // fallback to basic data
    selectedInvoice.value = invoice
    showDetail.value = true
  }
}

function closeDetail() {
  showDetail.value = false
  selectedInvoice.value = null
}

function formatMoney(value: number, currency: string): string {
  return `${new Intl.NumberFormat('en', { maximumFractionDigits: 2 }).format(value)} ${currency}`
}

function getStatusVariant(status: string): string {
  switch (status) {
    case 'paid': return 'active'
    case 'refunded': return 'disabled'
    case 'partially_refunded': return 'expired'
    default: return 'expired'
  }
}

const columns = [
  { key: 'invoice_number', label: 'Invoice #', sortable: true },
  { key: 'amount', label: 'Amount', sortable: true },
  { key: 'total', label: 'Total', sortable: true },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'created_at', label: 'Date', sortable: true },
  { key: 'actions', label: '' },
]
</script>
<template>
  <div class="invoices">
    <h1 class="invoices__title">Invoices</h1>

    <KSkeleton v-if="loading && !invoices.length" type="table" :count="5" />

    <template v-else>
      <!-- Invoice Detail Modal -->
      <div v-if="showDetail && selectedInvoice" class="invoices__detail-overlay" @click.self="closeDetail">
        <div class="invoices__detail-card">
          <div class="invoices__detail-header">
            <h2 class="invoices__detail-title">{{ selectedInvoice.invoice_number }}</h2>
            <KButton variant="ghost" size="sm" @click="closeDetail">✕</KButton>
          </div>

          <div class="invoices__detail-body">
            <div class="invoices__detail-row">
              <span class="invoices__detail-label">Status</span>
              <KStatusPill :status="getStatusVariant(selectedInvoice.status)">
                {{ selectedInvoice.status }}
              </KStatusPill>
            </div>
            <div class="invoices__detail-row">
              <span class="invoices__detail-label">Amount</span>
              <span>{{ formatMoney(selectedInvoice.amount, selectedInvoice.currency) }}</span>
            </div>
            <div v-if="selectedInvoice.tax > 0" class="invoices__detail-row">
              <span class="invoices__detail-label">Tax</span>
              <span>{{ formatMoney(selectedInvoice.tax, selectedInvoice.currency) }}</span>
            </div>
            <div class="invoices__detail-row invoices__detail-row--total">
              <span class="invoices__detail-label">Total</span>
              <span class="invoices__detail-total">{{ formatMoney(selectedInvoice.total, selectedInvoice.currency) }}</span>
            </div>
            <div v-if="selectedInvoice.plan_name" class="invoices__detail-row">
              <span class="invoices__detail-label">Plan</span>
              <span>{{ selectedInvoice.plan_name }}</span>
            </div>
            <div v-if="selectedInvoice.payment_method" class="invoices__detail-row">
              <span class="invoices__detail-label">Payment Method</span>
              <span>{{ selectedInvoice.payment_method }}</span>
            </div>
            <div v-if="selectedInvoice.refunded_amount > 0" class="invoices__detail-row">
              <span class="invoices__detail-label">Refunded</span>
              <span class="invoices__detail-refund">{{ formatMoney(selectedInvoice.refunded_amount, selectedInvoice.currency) }}</span>
            </div>
            <div class="invoices__detail-row">
              <span class="invoices__detail-label">Date</span>
              <span>{{ formatDate(selectedInvoice.created_at) }}</span>
            </div>
          </div>
        </div>
      </div>

      <!-- Invoice Table -->
      <section class="invoices__section">
        <KEmptyState
          v-if="!invoices.length"
          title="No invoices yet"
          description="Your invoice history will appear here after your first payment."
          icon="🧾"
        />

        <KDataTable
          v-else
          :columns="columns"
          :data="invoices"
          :loading="loading"
        >
          <template #cell-invoice_number="{ row }">
            <span class="invoices__number">{{ row.invoice_number }}</span>
          </template>
          <template #cell-amount="{ row }">
            {{ formatMoney(row.amount, row.currency) }}
          </template>
          <template #cell-total="{ row }">
            <strong>{{ formatMoney(row.total, row.currency) }}</strong>
          </template>
          <template #cell-status="{ row }">
            <KStatusPill :status="getStatusVariant(row.status)">
              {{ row.status }}
            </KStatusPill>
          </template>
          <template #cell-created_at="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
          <template #cell-actions="{ row }">
            <KButton variant="ghost" size="sm" @click="viewInvoice(row)">
              View
            </KButton>
          </template>
        </KDataTable>
      </section>
    </template>
  </div>
</template>
<style scoped>
.invoices {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding-bottom: calc(var(--space-8) + env(safe-area-inset-bottom, 20px));
}
.invoices__title {
  font-size: var(--text-xl);
  font-weight: 700;
}
.invoices__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.invoices__number {
  font-family: monospace;
  font-size: var(--text-xs);
  font-weight: 600;
}

/* Detail Overlay */
.invoices__detail-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: var(--space-4);
}
.invoices__detail-card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 480px;
  max-height: 90vh;
  overflow-y: auto;
}
.invoices__detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
}
.invoices__detail-title {
  font-size: var(--text-md);
  font-weight: 700;
  font-family: monospace;
}
.invoices__detail-body {
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.invoices__detail-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) 0;
}
.invoices__detail-row--total {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-3);
  margin-top: var(--space-2);
}
.invoices__detail-label {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
.invoices__detail-total {
  font-size: var(--text-md);
  font-weight: 700;
}
.invoices__detail-refund {
  color: var(--color-warning);
  font-weight: 500;
}
</style>
