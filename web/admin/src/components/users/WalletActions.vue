<script setup lang="ts">
/**
 * WalletActions — Top Up / Deduct modal for wallet management.
 *
 * Opens a KModal with amount and description inputs.
 * - Top Up mode: submits positive amount
 * - Deduct mode: shows confirmation warning if amount exceeds balance, submits negative amount
 * - Submit button disabled for invalid amounts (zero, non-numeric, out of range)
 * - On success: emits 'success', closes modal
 * - On failure: shows error in modal, retains inputs for retry
 *
 * Requirements: 11.3, 11.4, 11.5, 11.6, 11.7, 11.8, 11.9, 11.10
 */
import { ref, computed, watch } from 'vue'
import KModal from '@koris/ui/KModal.vue'
import KButton from '@koris/ui/KButton.vue'
import KInput from '@koris/ui/KInput.vue'
import KFormField from '@koris/ui/KFormField.vue'
import { useWalletActions } from '@/composables/useWalletActions'
import { useConfirm } from '@koris/composables/useConfirm'
import { formatCurrency } from '@/utils/formatCurrency'

export interface WalletActionsProps {
  open: boolean
  mode: 'top-up' | 'deduct'
  username: string
  currentBalance: number
}

const props = defineProps<WalletActionsProps>()

const emit = defineEmits<{
  close: []
  success: []
}>()

// --- Form state ---
const amount = ref<string>('')
const description = ref<string>('')
const apiError = ref<string | null>(null)
const showOverBalanceWarning = ref(false)

// --- Composables ---
const usernameRef = computed(() => props.username)
const { loading, topUp, deduct } = useWalletActions(usernameRef)
const { confirm } = useConfirm()

// --- Computed ---
const modalTitle = computed(() =>
  props.mode === 'top-up' ? 'Top Up Wallet' : 'Deduct from Wallet'
)

const parsedAmount = computed(() => {
  const val = parseFloat(amount.value)
  return val
})

const isAmountValid = computed(() => {
  const val = parsedAmount.value
  if (isNaN(val)) return false
  if (val < 1) return false
  if (val > 999_999_999) return false
  return true
})

const amountError = computed(() => {
  if (amount.value === '') return undefined
  if (isNaN(parsedAmount.value)) return 'Amount must be a valid number'
  if (parsedAmount.value <= 0) return 'Amount must be greater than zero'
  if (parsedAmount.value < 1) return 'Minimum amount is 1'
  if (parsedAmount.value > 999_999_999) return 'Maximum amount is 999,999,999'
  return undefined
})

const descriptionError = computed(() => {
  if (description.value.length > 200) return 'Description must be 200 characters or less'
  return undefined
})

const isSubmitDisabled = computed(() => {
  if (!isAmountValid.value) return true
  if (description.value.length > 200) return true
  return false
})

const exceedsBalance = computed(() =>
  props.mode === 'deduct' && parsedAmount.value > props.currentBalance
)

// --- Methods ---
async function handleSubmit() {
  if (isSubmitDisabled.value || loading.value) return

  apiError.value = null

  // If deducting more than balance, show confirmation
  if (exceedsBalance.value && !showOverBalanceWarning.value) {
    const confirmed = await confirm({
      title: 'Balance will go negative',
      message: `Deducting ${formatCurrency(parsedAmount.value)} will result in a negative balance of ${formatCurrency(props.currentBalance - parsedAmount.value)}. Do you want to proceed?`,
      variant: 'warning',
      confirmText: 'Proceed',
      cancelText: 'Cancel',
    })

    if (!confirmed) return
  }

  let success: boolean

  if (props.mode === 'top-up') {
    success = await topUp(parsedAmount.value, description.value)
  } else {
    success = await deduct(parsedAmount.value, description.value)
  }

  if (success) {
    emit('success')
    handleClose()
  } else {
    // On failure, retain inputs for retry and show error
    apiError.value = 'Operation failed. Please try again.'
  }
}

function handleClose() {
  if (loading.value) return
  emit('close')
}

// --- Reset form when modal opens ---
watch(() => props.open, (isOpen) => {
  if (isOpen) {
    amount.value = ''
    description.value = ''
    apiError.value = null
    showOverBalanceWarning.value = false
  }
})
</script>

<template>
  <KModal
    :open="open"
    :title="modalTitle"
    width="440px"
    @close="handleClose"
  >
    <div class="wallet-actions__form">
      <!-- Error banner -->
      <div v-if="apiError" class="wallet-actions__error" role="alert">
        {{ apiError }}
      </div>

      <!-- Amount field -->
      <KFormField
        label="Amount"
        name="wallet-amount"
        :error="amountError"
      >
        <template #default="{ fieldId, describedBy }">
          <KInput
            :id="fieldId"
            v-model="amount"
            type="number"
            placeholder="Enter amount"
            :aria-describedby="describedBy"
            :disabled="loading"
          />
        </template>
      </KFormField>

      <!-- Description field -->
      <KFormField
        label="Description"
        name="wallet-description"
        :error="descriptionError"
        hint="Optional note for this transaction (max 200 characters)"
      >
        <template #default="{ fieldId, describedBy }">
          <textarea
            :id="fieldId"
            v-model="description"
            class="wallet-actions__textarea"
            placeholder="Enter a description (optional)"
            :maxlength="200"
            :aria-describedby="describedBy"
            :disabled="loading"
            rows="3"
          />
        </template>
      </KFormField>

      <!-- Over-balance warning (inline, for deduct mode) -->
      <div
        v-if="mode === 'deduct' && exceedsBalance && isAmountValid"
        class="wallet-actions__warning"
        role="alert"
      >
        <svg
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M8 1.5L1 14h14L8 1.5z"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linejoin="round"
          />
          <path
            d="M8 6v3M8 11.5v.5"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
          />
        </svg>
        <span>
          This amount exceeds the current balance ({{ formatCurrency(currentBalance) }}).
          The wallet will go negative.
        </span>
      </div>
    </div>

    <template #footer>
      <div class="wallet-actions__footer">
        <KButton
          variant="ghost"
          :disabled="loading"
          @click="handleClose"
        >
          Cancel
        </KButton>
        <KButton
          :variant="mode === 'deduct' ? 'danger' : 'primary'"
          :loading="loading"
          :disabled="isSubmitDisabled"
          @click="handleSubmit"
        >
          {{ mode === 'top-up' ? 'Top Up' : 'Deduct' }}
        </KButton>
      </div>
    </template>
  </KModal>
</template>

<style scoped>
.wallet-actions__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.wallet-actions__error {
  padding: var(--space-3, 12px);
  border-radius: var(--radius-md, 8px);
  background-color: rgba(239, 68, 68, 0.1);
  border: 1px solid rgba(239, 68, 68, 0.3);
  color: var(--color-danger, #ef4444);
  font-size: var(--text-sm, 13px);
  line-height: 1.4;
}

.wallet-actions__textarea {
  display: block;
  width: 100%;
  min-height: 72px;
  padding: var(--space-2, 8px) var(--space-3, 12px);
  background: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 8px);
  color: var(--color-text, #e6edf3);
  font-family: var(--font-family);
  font-size: var(--text-base, 14px);
  line-height: var(--leading-normal, 1.5);
  resize: vertical;
  outline: none;
  transition:
    border-color var(--duration-normal, 0.15s) var(--ease-default, ease),
    box-shadow var(--duration-normal, 0.15s) var(--ease-default, ease);
}

.wallet-actions__textarea::placeholder {
  color: var(--color-muted, #8b98a5);
}

.wallet-actions__textarea:focus-visible {
  border-color: var(--color-primary, #2563eb);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.25);
}

.wallet-actions__textarea:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.wallet-actions__warning {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2, 8px);
  padding: var(--space-3, 12px);
  border-radius: var(--radius-md, 8px);
  background-color: rgba(245, 158, 11, 0.1);
  border: 1px solid rgba(245, 158, 11, 0.3);
  color: var(--color-warning, #f59e0b);
  font-size: var(--text-sm, 13px);
  line-height: 1.4;
}

.wallet-actions__warning svg {
  flex-shrink: 0;
  margin-top: 1px;
}

.wallet-actions__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3, 12px);
}
</style>
