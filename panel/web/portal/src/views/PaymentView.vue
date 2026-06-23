<script setup lang="ts">
import { ref, computed } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useFreshData } from '@koris/composables/useFreshData'
import KButton from '@koris/ui/KButton.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'

interface Gateway {
  id: number
  name: string
  display_name: string
  currency?: string
}

interface GatewaysResponse {
  ok: boolean
  gateways: Gateway[]
}

interface PayResponse {
  ok: boolean
  redirect_url: string
  reference?: string
}

const { get, post, loading } = useApi()
const { t } = useI18n()

const gateways = ref<Gateway[]>([])
const selectedGateway = ref('')
const amount = ref<number>(0)
const notice = ref('')
const errorMsg = ref('')

useFreshData(async () => {
  await fetchGateways()
})

async function fetchGateways() {
  try {
    const res = await get<GatewaysResponse>('/api/portal/gateways')
    gateways.value = (res.gateways || []).filter(g => g.name) // only active gateways
    if (gateways.value.length && !selectedGateway.value) {
      selectedGateway.value = gateways.value[0].name
    }
  } catch {
    // keep empty
  }
}

const currentGateway = computed(() => {
  return gateways.value.find(g => g.name === selectedGateway.value)
})

const currency = computed(() => {
  return currentGateway.value?.currency || 'IRR'
})

const canPay = computed(() => {
  return selectedGateway.value && amount.value > 0
})

async function handlePay() {
  if (!canPay.value) return
  notice.value = ''
  errorMsg.value = ''

  try {
    const res = await post<PayResponse>('/api/portal/pay', {
      gateway: selectedGateway.value,
      amount: amount.value,
    })

    if (res.redirect_url) {
      // Redirect to payment gateway
      window.location.href = res.redirect_url
    } else {
      notice.value = 'Payment initiated. Please wait for redirect...'
    }
  } catch (err: any) {
    errorMsg.value = err?.message || 'Payment failed. Please try again.'
  }
}

function formatMoney(value: number): string {
  if (!value) return `0 ${currency.value}`
  return `${new Intl.NumberFormat('en', { maximumFractionDigits: 0 }).format(value)} ${currency.value}`
}
</script>
<template>
  <div class="payment">
    <h1 class="payment__title">Make Payment</h1>

    <KSkeleton v-if="loading && !gateways.length" type="card" :count="1" />

    <template v-else>
      <div v-if="notice" class="payment__notice" role="status">{{ notice }}</div>
      <div v-if="errorMsg" class="payment__error" role="alert">{{ errorMsg }}</div>

      <section class="payment__section">
        <h2 class="payment__section-title">💳 Payment Details</h2>
        <p class="payment__section-desc">
          Add credit to your wallet through an online payment gateway.
        </p>

        <form class="payment__form" @submit.prevent="handlePay">
          <!-- Gateway Selection -->
          <KFormField label="Payment Gateway" :required="true">
            <KSelect v-model="selectedGateway">
              <option v-for="gw in gateways" :key="gw.id" :value="gw.name">
                {{ gw.display_name }}
              </option>
            </KSelect>
          </KFormField>

          <!-- Amount Input -->
          <KFormField label="Amount" :required="true">
            <div class="payment__amount-row">
              <KInput
                v-model.number="amount"
                type="number"
                :min="1"
                placeholder="Enter amount"
              />
              <span class="payment__currency">{{ currency }}</span>
            </div>
          </KFormField>

          <!-- Summary -->
          <div v-if="amount > 0" class="payment__summary">
            <div class="payment__summary-row">
              <span>Amount</span>
              <span class="payment__summary-value">{{ formatMoney(amount) }}</span>
            </div>
            <div class="payment__summary-row">
              <span>Gateway</span>
              <span>{{ currentGateway?.display_name || selectedGateway }}</span>
            </div>
          </div>

          <!-- Pay Button -->
          <KButton
            type="submit"
            variant="primary"
            :loading="loading"
            :disabled="!canPay"
          >
            💳 Pay Now — {{ formatMoney(amount) }}
          </KButton>
        </form>
      </section>

      <!-- No gateways available -->
      <div v-if="!gateways.length && !loading" class="payment__no-gateways">
        <p>No payment gateways are currently available. Please contact support.</p>
      </div>
    </template>
  </div>
</template>
<style scoped>
.payment {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding-bottom: calc(var(--space-8) + env(safe-area-inset-bottom, 20px));
}
.payment__title {
  font-size: var(--text-xl);
  font-weight: 700;
}
.payment__notice {
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: rgba(34, 197, 94, 0.1);
  color: var(--color-success);
  font-size: var(--text-sm);
  border: 1px solid rgba(34, 197, 94, 0.2);
}
.payment__error {
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: rgba(239, 68, 68, 0.1);
  color: var(--color-danger, #ef4444);
  font-size: var(--text-sm);
  border: 1px solid rgba(239, 68, 68, 0.2);
}
.payment__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.payment__section-title {
  font-size: var(--text-md);
  font-weight: 600;
  margin-bottom: var(--space-2);
}
.payment__section-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-bottom: var(--space-5);
  line-height: 1.5;
}
.payment__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  max-width: 400px;
}
.payment__amount-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.payment__currency {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-muted);
  white-space: nowrap;
  min-width: 40px;
}
.payment__summary {
  padding: var(--space-4);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.payment__summary-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: var(--text-sm);
}
.payment__summary-value {
  font-weight: 600;
}
.payment__no-gateways {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  text-align: center;
  color: var(--color-muted);
  font-size: var(--text-sm);
}
</style>
