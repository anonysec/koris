<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { usePaymentsStore } from '@/stores/payments'
import { useToast } from '@koris/composables/useToast'
import KDataTable from '@koris/ui/KDataTable.vue'
import KButton from '@koris/ui/KButton.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'
import KDrawer from '@koris/ui/KDrawer.vue'

const store = usePaymentsStore()
const toast = useToast()
const creatingPayment = ref(false)
const showRecordDrawer = ref(false)
const showMethodDrawer = ref(false)
const savingMethod = ref(false)

const paymentForm = ref({
  username: '',
  amount: '',
  method: '',
  description: '',
})

const methodForm = ref({
  name: '',
  type: '',
  instructions: '',
  is_active: true,
  sort_order: 0,
})

const tableColumns = [
  { key: 'username', label: 'User', sortable: true },
  { key: 'amount', label: 'Amount', sortable: true, align: 'right' as const },
  { key: 'method', label: 'Method', sortable: true },
  { key: 'status', label: 'Status', sortable: true, filterable: true, filterType: 'select' as const, filterOptions: [
    { label: 'Pending', value: 'pending' },
    { label: 'Approved', value: 'approved' },
    { label: 'Rejected', value: 'rejected' },
  ]},
  { key: 'intent_label', label: 'Intent' },
  { key: 'created_at', label: 'Date', sortable: true },
  { key: 'actions', label: 'Actions', align: 'center' as const },
]

const methodTypeOptions = [
  { label: 'Bank Transfer', value: 'bank_transfer' },
  { label: 'Crypto', value: 'crypto' },
  { label: 'Card', value: 'card' },
  { label: 'Other', value: 'other' },
]

async function handleApprove(id: number) {
  await store.approvePayment(id)
}

async function handleReject(id: number) {
  await store.rejectPayment(id)
}

async function submitPayment() {
  creatingPayment.value = true
  const success = await store.createManualPayment({
    username: paymentForm.value.username,
    amount: Number(paymentForm.value.amount),
    method: paymentForm.value.method,
    description: paymentForm.value.description,
  })
  creatingPayment.value = false
  if (success) {
    paymentForm.value = { username: '', amount: '', method: '', description: '' }
    showRecordDrawer.value = false
    toast.success('Payment recorded successfully.')
  } else {
    toast.error('Failed to record payment.')
  }
}

async function submitMethod() {
  savingMethod.value = true
  const success = await store.savePaymentMethod({
    name: methodForm.value.name,
    type: methodForm.value.type,
    instructions: methodForm.value.instructions,
    is_active: methodForm.value.is_active,
    sort_order: Number(methodForm.value.sort_order),
  })
  savingMethod.value = false
  if (success) {
    methodForm.value = { name: '', type: '', instructions: '', is_active: true, sort_order: 0 }
    showMethodDrawer.value = false
    toast.success('Payment method created successfully.')
  } else {
    toast.error('Failed to create payment method.')
  }
}

onMounted(() => {
  store.loadPayments()
})
</script>

<template>
  <div class="page payments-view">
    <header class="page-header">
      <KButton variant="primary" @click="showRecordDrawer = true">
        Record Payment
      </KButton>
    </header>

    <!-- Payments Table -->
    <section class="payments-table-section">
      <KDataTable
        :columns="tableColumns"
        :data="store.paginatedList"
        :loading="store.loading"
        :page-size="store.pageSize"
        row-key="id"
      >
        <template #cell-amount="{ value }">
          <span class="amount-cell">${{ typeof value === 'number' ? value.toFixed(2) : value }}</span>
        </template>
        <template #cell-status="{ value }">
          <KStatusPill :status="value" size="sm" />
        </template>
        <template #cell-created_at="{ value }">
          {{ value?.slice(0, 10) }}
        </template>
        <template #cell-actions="{ row }">
          <div v-if="row.status === 'pending'" class="action-btns">
            <KButton variant="primary" size="sm" @click.stop="handleApprove(row.id)">Approve</KButton>
            <KButton variant="danger" size="sm" @click.stop="handleReject(row.id)">Reject</KButton>
          </div>
          <span v-else class="text-muted">-</span>
        </template>
      </KDataTable>
    </section>

    <!-- Payment Methods Section (always visible) -->
    <section class="panel">
      <div class="panel-header">
        <h4 class="panel-title">Payment Methods</h4>
        <KButton variant="ghost" size="sm" @click="showMethodDrawer = true">Add Method</KButton>
      </div>
      <div class="methods-list">
        <div v-for="method in store.paymentMethods" :key="method.id" class="method-item">
          <div class="method-item__info">
            <span class="method-item__name">{{ method.name }}</span>
            <span class="method-item__type text-muted">{{ method.type }}</span>
          </div>
          <KStatusPill :status="method.is_active ? 'active' : 'disabled'" size="sm" />
        </div>
        <p v-if="store.paymentMethods.length === 0" class="text-muted text-sm">No payment methods configured.</p>
      </div>
    </section>

    <!-- Record Payment Drawer -->
    <KDrawer :open="showRecordDrawer" title="Record Payment" side="right" @close="showRecordDrawer = false">
      <form class="payment-form" @submit.prevent="submitPayment">
        <KFormField name="pay-username" label="Username" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="paymentForm.username" placeholder="customer_username" />
          </template>
        </KFormField>
        <KFormField name="pay-amount" label="Amount ($)" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="paymentForm.amount" type="number" placeholder="10.00" />
          </template>
        </KFormField>
        <KFormField name="pay-method" label="Method" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="paymentForm.method"
              :options="store.activePaymentMethods.map(m => ({ label: m.name, value: m.name }))"
              placeholder="Select method"
            />
          </template>
        </KFormField>
        <KFormField name="pay-desc" label="Description">
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="paymentForm.description" placeholder="Optional note" />
          </template>
        </KFormField>
        <KButton type="submit" variant="primary" :loading="creatingPayment" full-width>
          Record Payment
        </KButton>
      </form>
    </KDrawer>

    <!-- Add Payment Method Drawer -->
    <KDrawer :open="showMethodDrawer" title="Add Payment Method" side="right" @close="showMethodDrawer = false">
      <form class="payment-form" @submit.prevent="submitMethod">
        <KFormField name="method-name" label="Name" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="methodForm.name" placeholder="e.g. Bank Transfer" />
          </template>
        </KFormField>
        <KFormField name="method-type" label="Type" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="methodForm.type"
              :options="methodTypeOptions"
              placeholder="Select type"
            />
          </template>
        </KFormField>
        <KFormField name="method-instructions" label="Instructions">
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="methodForm.instructions" placeholder="Payment instructions for customers" />
          </template>
        </KFormField>
        <KFormField name="method-sort" label="Sort Order">
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="methodForm.sort_order" type="number" placeholder="0" />
          </template>
        </KFormField>
        <div class="toggle-field">
          <label class="toggle-label">
            <input type="checkbox" v-model="methodForm.is_active" />
            <span>Active</span>
          </label>
        </div>
        <KButton type="submit" variant="primary" :loading="savingMethod" full-width>
          Create Method
        </KButton>
      </form>
    </KDrawer>
  </div>
</template>

<style scoped>
.payments-view { display: flex; flex-direction: column; gap: var(--space-5); }
.page-header { display: flex; align-items: center; justify-content: flex-end; }

.panel { padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); }
.panel-header { display: flex; align-items: center; justify-content: space-between; margin-bottom: var(--space-3); }
.panel-title { margin: 0; font-size: var(--text-sm); font-weight: var(--font-semibold); }

.payment-form { display: flex; flex-direction: column; gap: var(--space-3); }

.methods-list { display: flex; flex-direction: column; gap: var(--space-2); }
.method-item { display: flex; justify-content: space-between; align-items: center; padding: var(--space-2) 0; border-bottom: 1px solid var(--color-border); }
.method-item:last-child { border-bottom: none; }
.method-item__info { display: flex; flex-direction: column; }
.method-item__name { font-size: var(--text-sm); font-weight: var(--font-medium); }
.method-item__type { font-size: var(--text-xs); }

.payments-table-section { min-width: 0; }

.amount-cell { font-weight: var(--font-semibold); color: var(--color-success); }
.action-btns { display: flex; gap: var(--space-1); }

.toggle-field { padding: var(--space-2) 0; }
.toggle-label { display: flex; align-items: center; gap: var(--space-2); font-size: var(--text-sm); color: var(--color-text); cursor: pointer; }
.toggle-label input[type="checkbox"] { width: 1rem; height: 1rem; accent-color: var(--color-primary); }

.text-muted { color: var(--color-muted); }
.text-sm { font-size: var(--text-sm); }
</style>
