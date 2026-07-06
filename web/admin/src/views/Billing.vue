<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { formatDate } from '@koris/composables/useFormatDate'
import Button from '@koris/ui/Button.vue'
import Select from '@koris/ui/Select.vue'
import Input from '@koris/ui/Input.vue'
import FormField from '@koris/ui/FormField.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import Drawer from '@koris/ui/Drawer.vue'

const { t } = useI18n()
const { get, post, patch } = useApi()
const toast = useToast()

// ═══════════════════════════════════════════════════════════════════════════════
// Revenue Dashboard
// ═══════════════════════════════════════════════════════════════════════════════

type RevenuePeriod = 'daily' | 'weekly' | 'monthly'

interface RevenueData {
  ok: boolean
  total_revenue: number
  mrr: number
  transaction_count: number
  period: RevenuePeriod
  breakdown: { label: string; amount: number; count: number }[]
}

const revenuePeriod = ref<RevenuePeriod>('monthly')
const revenueDateFrom = ref('')
const revenueDateTo = ref('')
const revenueData = ref<RevenueData | null>(null)
const revenueLoading = ref(false)

async function fetchRevenue() {
  revenueLoading.value = true
  try {
    let url = `/api/admin/billing/revenue?period=${revenuePeriod.value}`
    if (revenueDateFrom.value) url += `&from=${revenueDateFrom.value}`
    if (revenueDateTo.value) url += `&to=${revenueDateTo.value}`
    const data = await get<RevenueData>(url)
    if (data?.ok) {
      revenueData.value = data
    }
  } catch {
    // silent — toast handled by useApi
  } finally {
    revenueLoading.value = false
  }
}

watch(revenuePeriod, () => fetchRevenue())

const periodOptions = computed(() => [
  { label: t('billing.period_daily'), value: 'daily' },
  { label: t('billing.period_weekly'), value: 'weekly' },
  { label: t('billing.period_monthly'), value: 'monthly' },
])

// ═══════════════════════════════════════════════════════════════════════════════
// Invoice List
// ═══════════════════════════════════════════════════════════════════════════════

interface Invoice {
  id: number
  customer_name: string
  amount: number
  status: 'paid' | 'draft' | 'cancelled' | 'refunded'
  type: string
  created_at: string
}

interface InvoiceListResponse {
  ok: boolean
  invoices: Invoice[]
  total: number
  page: number
  page_size: number
}

const invoices = ref<Invoice[]>([])
const invoiceTotal = ref(0)
const invoicePage = ref(1)
const invoicePageSize = 15
const invoicesLoading = ref(false)

async function fetchInvoices() {
  invoicesLoading.value = true
  try {
    const data = await get<InvoiceListResponse>(
      `/api/admin/billing/invoices?page=${invoicePage.value}&page_size=${invoicePageSize}`
    )
    if (data?.ok) {
      invoices.value = data.invoices || []
      invoiceTotal.value = data.total || 0
    }
  } catch {
    // endpoint may not exist yet — show empty state
    invoices.value = []
    invoiceTotal.value = 0
  } finally {
    invoicesLoading.value = false
  }
}

const invoiceTotalPages = computed(() => Math.max(1, Math.ceil(invoiceTotal.value / invoicePageSize)))

function prevInvoicePage() {
  if (invoicePage.value > 1) {
    invoicePage.value--
    fetchInvoices()
  }
}
function nextInvoicePage() {
  if (invoicePage.value < invoiceTotalPages.value) {
    invoicePage.value++
    fetchInvoices()
  }
}

function invoiceStatusVariant(status: string): string {
  switch (status) {
    case 'paid': return 'active'
    case 'draft': return 'warning'
    case 'cancelled': return 'danger'
    case 'refunded': return 'info'
    default: return 'default'
  }
}

async function downloadInvoicePdf(invoice: Invoice) {
  try {
    const response = await fetch(`/api/admin/billing/invoices/${invoice.id}/pdf`, {
      credentials: 'same-origin',
    })
    if (!response.ok) {
      toast.error(t('billing.pdf_download_error'))
      return
    }
    const blob = await response.blob()
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = `invoice-${invoice.id}.pdf`
    a.click()
    URL.revokeObjectURL(url)
  } catch {
    toast.error(t('billing.pdf_download_error'))
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Payment Gateways
// ═══════════════════════════════════════════════════════════════════════════════

interface PaymentGateway {
  id: number
  name: string
  type: 'manual' | 'zarinpal' | 'crypto'
  is_active: boolean
  config: Record<string, string>
  created_at: string
}

interface GatewayListResponse {
  ok: boolean
  gateways: PaymentGateway[]
}

const gateways = ref<PaymentGateway[]>([])
const gatewaysLoading = ref(false)
const showGatewayDrawer = ref(false)
const savingGateway = ref(false)

const gatewayForm = ref({
  name: '',
  type: 'manual' as 'manual' | 'zarinpal' | 'crypto',
  merchant_id: '',
  wallet_address: '',
  network: '',
  instructions: '',
})

const gatewayTypeOptions = computed(() => [
  { label: t('billing.gw_type_manual'), value: 'manual' },
  { label: 'Zarinpal', value: 'zarinpal' },
  { label: t('billing.gw_type_crypto'), value: 'crypto' },
])

async function fetchGateways() {
  gatewaysLoading.value = true
  try {
    const data = await get<GatewayListResponse>('/api/admin/billing/gateways')
    if (data?.ok) {
      gateways.value = data.gateways || []
    }
  } catch {
    // endpoint may not exist yet
    gateways.value = []
  } finally {
    gatewaysLoading.value = false
  }
}

async function toggleGateway(gw: PaymentGateway) {
  try {
    await patch<{ ok: boolean }>(`/api/admin/billing/gateways/${gw.id}`, {
      is_active: !gw.is_active,
    })
    gw.is_active = !gw.is_active
    toast.success(t('billing.gw_toggle_success'))
  } catch {
    toast.error(t('billing.gw_toggle_error'))
  }
}

async function submitGateway() {
  savingGateway.value = true
  try {
    const config: Record<string, string> = {}
    if (gatewayForm.value.type === 'zarinpal') {
      config.merchant_id = gatewayForm.value.merchant_id
    } else if (gatewayForm.value.type === 'crypto') {
      config.wallet_address = gatewayForm.value.wallet_address
      config.network = gatewayForm.value.network
    } else {
      config.instructions = gatewayForm.value.instructions
    }

    await post<{ ok: boolean }>('/api/admin/billing/gateways', {
      name: gatewayForm.value.name,
      type: gatewayForm.value.type,
      config,
    })
    toast.success(t('billing.gw_create_success'))
    showGatewayDrawer.value = false
    gatewayForm.value = { name: '', type: 'manual', merchant_id: '', wallet_address: '', network: '', instructions: '' }
    await fetchGateways()
  } catch {
    toast.error(t('billing.gw_create_error'))
  } finally {
    savingGateway.value = false
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(() => {
  fetchRevenue()
  fetchInvoices()
  fetchGateways()
})
</script>

<template>
  <div class="page billing-view">
    <!-- ═══ Revenue Dashboard ═══ -->
    <section class="panel revenue-section">
      <div class="panel-header">
        <div>
          <h3 class="panel-title">{{ t('billing.revenue_dashboard') }}</h3>
          <p class="panel-subtitle">{{ t('billing.revenue_subtitle') }}</p>
        </div>
        <div class="revenue-controls">
          <Select
            v-model="revenuePeriod"
            :options="periodOptions"
            aria-label="Period"
            class="period-select"
          />
          <input
            v-model="revenueDateFrom"
            type="date"
            :placeholder="t('billing.from')"
            class="date-input"
            :aria-label="t('billing.from')"
            @change="fetchRevenue"
          />
          <input
            v-model="revenueDateTo"
            type="date"
            :placeholder="t('billing.to')"
            class="date-input"
            :aria-label="t('billing.to')"
            @change="fetchRevenue"
          />
        </div>
      </div>

      <!-- Stats Cards -->
      <div v-if="revenueLoading && !revenueData" class="revenue-cards">
        <Skeleton variant="card" :count="3" />
      </div>
      <div v-else class="revenue-cards">
        <div class="revenue-card">
          <span class="revenue-card__icon">💰</span>
          <div class="revenue-card__body">
            <span class="revenue-card__value">${{ (revenueData?.total_revenue ?? 0).toFixed(2) }}</span>
            <span class="revenue-card__label">{{ t('billing.total_revenue') }}</span>
          </div>
        </div>
        <div class="revenue-card">
          <span class="revenue-card__icon">📈</span>
          <div class="revenue-card__body">
            <span class="revenue-card__value">${{ (revenueData?.mrr ?? 0).toFixed(2) }}</span>
            <span class="revenue-card__label">{{ t('billing.mrr') }}</span>
          </div>
        </div>
        <div class="revenue-card">
          <span class="revenue-card__icon">🧾</span>
          <div class="revenue-card__body">
            <span class="revenue-card__value">{{ revenueData?.transaction_count ?? 0 }}</span>
            <span class="revenue-card__label">{{ t('billing.transactions') }}</span>
          </div>
        </div>
      </div>

      <!-- Chart Placeholder -->
      <div class="chart-placeholder">
        <div class="chart-placeholder__inner">
          <span class="chart-placeholder__icon">📊</span>
          <p class="chart-placeholder__text">{{ t('billing.chart_coming_soon') }}</p>
        </div>
      </div>
    </section>

    <!-- ═══ Invoice List ═══ -->
    <section class="panel invoices-section">
      <div class="panel-header">
        <h3 class="panel-title">{{ t('billing.invoices') }}</h3>
      </div>

      <div v-if="invoicesLoading" class="invoice-skeleton">
        <Skeleton variant="table-row" :count="5" />
      </div>
      <div v-else-if="invoices.length === 0" class="empty-state">
        <p class="text-muted">{{ t('billing.no_invoices') }}</p>
      </div>
      <div v-else class="invoice-table-wrap">
        <table class="invoice-table" role="table">
          <thead>
            <tr>
              <th>#</th>
              <th>{{ t('billing.col_customer') }}</th>
              <th>{{ t('billing.col_amount') }}</th>
              <th>{{ t('billing.col_status') }}</th>
              <th>{{ t('billing.col_type') }}</th>
              <th>{{ t('billing.col_date') }}</th>
              <th>{{ t('billing.col_actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="inv in invoices" :key="inv.id">
              <td class="text-muted">#{{ inv.id }}</td>
              <td>{{ inv.customer_name }}</td>
              <td class="amount-cell">${{ inv.amount.toFixed(2) }}</td>
              <td>
                <StatusPill :status="invoiceStatusVariant(inv.status)" size="sm">
                  {{ inv.status }}
                </StatusPill>
              </td>
              <td class="text-muted">{{ inv.type }}</td>
              <td class="text-muted">{{ formatDate(inv.created_at) }}</td>
              <td>
                <Button variant="ghost" size="sm" @click="downloadInvoicePdf(inv)">
                  📥 PDF
                </Button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      <!-- Pagination -->
      <div v-if="invoiceTotal > invoicePageSize" class="pagination">
        <Button variant="ghost" size="sm" :disabled="invoicePage <= 1" @click="prevInvoicePage">
          ← {{ t('billing.prev') }}
        </Button>
        <span class="pagination__info">{{ invoicePage }} / {{ invoiceTotalPages }}</span>
        <Button variant="ghost" size="sm" :disabled="invoicePage >= invoiceTotalPages" @click="nextInvoicePage">
          {{ t('billing.next') }} →
        </Button>
      </div>
    </section>

    <!-- ═══ Payment Gateways ═══ -->
    <section class="panel gateways-section">
      <div class="panel-header">
        <h3 class="panel-title">{{ t('billing.payment_gateways') }}</h3>
        <Button variant="primary" size="sm" @click="showGatewayDrawer = true">
          {{ t('billing.add_gateway') }}
        </Button>
      </div>

      <div v-if="gatewaysLoading" class="gateway-skeleton">
        <Skeleton variant="card" :count="2" />
      </div>
      <div v-else-if="gateways.length === 0" class="empty-state">
        <p class="text-muted">{{ t('billing.no_gateways') }}</p>
      </div>
      <div v-else class="gateway-grid">
        <div v-for="gw in gateways" :key="gw.id" class="gateway-card">
          <div class="gateway-card__header">
            <span class="gateway-card__name">{{ gw.name }}</span>
            <StatusPill :status="gw.is_active ? 'active' : 'disabled'" size="sm" />
          </div>
          <div class="gateway-card__meta">
            <span class="gateway-card__type">{{ gw.type }}</span>
          </div>
          <div class="gateway-card__footer">
            <label class="toggle-label">
              <input type="checkbox" :checked="gw.is_active" @change="toggleGateway(gw)" />
              <span>{{ gw.is_active ? t('billing.enabled') : t('billing.disabled') }}</span>
            </label>
          </div>
        </div>
      </div>
    </section>

    <!-- ═══ Add Gateway Drawer ═══ -->
    <Drawer :open="showGatewayDrawer" :title="t('billing.add_gateway')" side="right" @close="showGatewayDrawer = false">
      <form class="gateway-form" @submit.prevent="submitGateway">
        <FormField name="gw-name" :label="t('billing.gw_name')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="gatewayForm.name" placeholder="My Gateway" />
          </template>
        </FormField>
        <FormField name="gw-type" :label="t('billing.gw_type')" required>
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="gatewayForm.type" :options="gatewayTypeOptions" />
          </template>
        </FormField>

        <!-- Zarinpal config -->
        <template v-if="gatewayForm.type === 'zarinpal'">
          <FormField name="gw-merchant" :label="t('billing.gw_merchant_id')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="gatewayForm.merchant_id" placeholder="xxxxxxxx-xxxx-xxxx-xxxx-xxxxxxxxxxxx" />
            </template>
          </FormField>
        </template>

        <!-- Crypto config -->
        <template v-if="gatewayForm.type === 'crypto'">
          <FormField name="gw-wallet" :label="t('billing.gw_wallet_address')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="gatewayForm.wallet_address" placeholder="0x..." />
            </template>
          </FormField>
          <FormField name="gw-network" :label="t('billing.gw_network')">
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="gatewayForm.network" placeholder="TRC20" />
            </template>
          </FormField>
        </template>

        <!-- Manual config -->
        <template v-if="gatewayForm.type === 'manual'">
          <FormField name="gw-instructions" :label="t('billing.gw_instructions')">
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="gatewayForm.instructions" placeholder="Payment instructions..." />
            </template>
          </FormField>
        </template>

        <Button type="submit" variant="primary" :loading="savingGateway" full-width>
          {{ t('billing.create_gateway') }}
        </Button>
      </form>
    </Drawer>
  </div>
</template>

<style scoped>
.billing-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

/* ─── Panel ─── */
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

/* ─── Revenue Controls ─── */
.revenue-controls {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.period-select {
  width: 140px;
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

/* ─── Revenue Cards ─── */
.revenue-cards {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}
.revenue-card {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.revenue-card__icon {
  font-size: 1.5rem;
}
.revenue-card__body {
  display: flex;
  flex-direction: column;
}
.revenue-card__value {
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
  color: var(--color-text);
  font-family: var(--font-mono, monospace);
}
.revenue-card__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
}

/* ─── Chart Placeholder ─── */
.chart-placeholder {
  border: 2px dashed var(--color-border);
  border-radius: var(--radius-md);
  min-height: 200px;
  display: flex;
  align-items: center;
  justify-content: center;
}
.chart-placeholder__inner {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-2);
}
.chart-placeholder__icon {
  font-size: 2rem;
  opacity: 0.5;
}
.chart-placeholder__text {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0;
}

/* ─── Invoice Table ─── */
.invoice-table-wrap {
  overflow-x: auto;
}
.invoice-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.invoice-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  color: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}
.invoice-table td {
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

/* ─── Pagination ─── */
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

/* ─── Gateway Grid ─── */
.gateway-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: var(--space-3);
}
.gateway-card {
  padding: var(--space-3);
  background: var(--color-surface-2, var(--color-surface));
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.gateway-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.gateway-card__name {
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
  color: var(--color-text);
}
.gateway-card__meta {
  font-size: var(--text-xs);
  color: var(--color-muted);
}
.gateway-card__type {
  text-transform: capitalize;
}
.gateway-card__footer {
  padding-top: var(--space-2);
  border-top: 1px solid var(--color-border);
}

/* ─── Toggle ─── */
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

/* ─── Gateway Form ─── */
.gateway-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

/* ─── Empty / Skeleton ─── */
.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 80px;
}
.invoice-skeleton,
.gateway-skeleton {
  padding: var(--space-2) 0;
}

/* ─── Utilities ─── */
.text-muted { color: var(--color-muted); }

/* ─── Responsive ─── */
@media (max-width: 768px) {
  .revenue-controls { flex-direction: column; align-items: stretch; }
  .period-select, .date-input { width: 100%; }
  .revenue-cards { grid-template-columns: 1fr; }
  .gateway-grid { grid-template-columns: 1fr; }
}
</style>
