<template>
  <div class="profile-fields">
    <!-- Username (read-only) -->
    <KFormField name="username" :label="t('customer.username')">
      <template #default="{ fieldId, describedBy }">
        <KInput
          :id="fieldId"
          :model-value="modelValue.username"
          :aria-describedby="describedBy"
          disabled
        />
      </template>
    </KFormField>

    <!-- Status dropdown -->
    <KFormField name="status" :label="t('customer.status')">
      <template #default="{ fieldId, describedBy }">
        <KSelect
          :id="fieldId"
          :model-value="modelValue.status"
          :options="statusOptions"
          :aria-describedby="describedBy"
          @update:model-value="updateField('status', $event)"
        />
      </template>
    </KFormField>

    <!-- Data Limit -->
    <KFormField name="data-limit" :label="t('customer.data_limit')">
      <template #default="{ fieldId }">
        <div class="profile-fields__data-limit">
          <KInput
            :id="fieldId"
            :model-value="modelValue.data_limit"
            type="number"
            placeholder="0 = unlimited"
            @update:model-value="updateField('data_limit', $event)"
          />
          <KSelect
            :model-value="modelValue.data_limit_unit"
            :options="dataLimitUnitOptions"
            class="profile-fields__unit-select"
            @update:model-value="updateField('data_limit_unit', $event)"
          />
        </div>
      </template>
    </KFormField>

    <!-- Expiry Date — always show date input + chips below -->
    <KFormField name="expiry" :label="t('customer.expiry_date')">
      <template #default="{ fieldId }">
        <div class="profile-fields__expiry">
          <!-- Date input with calendar icon -->
          <div class="profile-fields__date-wrapper">
            <input
              :id="fieldId"
              type="date"
              class="profile-fields__date-input"
              :value="expiryDateValue"
              @input="onDateInput"
            />
          </div>

          <!-- Quick-set chips (always visible, minimal style) -->
          <div class="profile-fields__quick-chips">
            <button
              v-for="chip in expiryChips"
              :key="chip"
              type="button"
              class="profile-fields__chip"
              @click="applyChip(chip)"
            >
              {{ chip }}
            </button>
          </div>

          <!-- "Expires in X days" display -->
          <p v-if="expiresInDays !== null" class="profile-fields__expiry-info">
            <span v-if="expiresInDays > 0">
              Expires in {{ expiresInDays }} day{{ expiresInDays === 1 ? '' : 's' }}
            </span>
            <span v-else-if="expiresInDays === 0" class="profile-fields__expiry-info--warning">
              Expires today
            </span>
            <span v-else class="profile-fields__expiry-info--expired">
              Expired {{ Math.abs(expiresInDays) }} day{{ Math.abs(expiresInDays) === 1 ? '' : 's' }} ago
            </span>
          </p>
        </div>
      </template>
    </KFormField>

    <!-- Note -->
    <KFormField name="note" :label="t('customer.note')">
      <template #default="{ fieldId, describedBy }">
        <KTextarea
          :id="fieldId"
          :model-value="modelValue.note"
          :aria-describedby="describedBy"
          placeholder="Note..."
          :rows="3"
          @update:model-value="updateField('note', $event)"
        />
      </template>
    </KFormField>

    <!-- Proxy settings — dropdown style -->
    <KFormField name="proxy-settings" :label="t('customer.proxy_settings')">
      <template #default>
        <div class="profile-fields__proxy">
          <button
            type="button"
            class="profile-fields__proxy-toggle"
            :aria-expanded="proxyOpen"
            @click="proxyOpen = !proxyOpen"
          >
            <span>{{ proxyButtonLabel }}</span>
            <svg
              class="profile-fields__proxy-chevron"
              :class="{ 'profile-fields__proxy-chevron--open': proxyOpen }"
              width="14" height="14" viewBox="0 0 14 14" fill="none"
            >
              <path d="M4 5.5L7 8.5L10 5.5" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>

          <!-- Dropdown panel -->
          <div v-if="proxyOpen" class="profile-fields__proxy-panel">
            <div
              v-for="protocol in availableProtocols"
              :key="protocol.value"
              class="profile-fields__proxy-item"
            >
              <div class="profile-fields__proxy-item-header">
                <label class="profile-fields__proxy-label">
                  <input
                    type="checkbox"
                    :checked="isProtocolEnabled(protocol.value)"
                    @change="toggleProtocol(protocol.value)"
                  />
                  <span>{{ protocol.label }}</span>
                </label>
              </div>
              <!-- Protocol sub-options (shown when enabled) -->
              <div v-if="isProtocolEnabled(protocol.value) && protocol.options" class="profile-fields__proxy-options">
                <label
                  v-for="opt in protocol.options"
                  :key="opt.value"
                  class="profile-fields__proxy-option"
                >
                  <input
                    type="radio"
                    :name="`proto-opt-${protocol.value}`"
                    :value="opt.value"
                    :checked="getProtocolOption(protocol.value) === opt.value"
                    @change="setProtocolOption(protocol.value, opt.value)"
                  />
                  <span>{{ opt.label }}</span>
                </label>
              </div>
            </div>
          </div>
        </div>
      </template>
    </KFormField>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KTextarea from '@koris/ui/KTextarea.vue'
import { computeExpiryDate, type ExpiryOffset } from '@/utils/computeExpiryDate'

const { t } = useI18n()

/**
 * Form data shape for user profile editing.
 */
export interface ProfileFormData {
  username: string
  status: string
  data_limit: string
  data_limit_unit: string
  expiry_date: string
  note: string
  allowed_protocols: string[]
  protocol_options: Record<string, string>
}

export interface ProfileFieldsProps {
  modelValue: ProfileFormData
}

const props = defineProps<ProfileFieldsProps>()

const emit = defineEmits<{
  'update:modelValue': [value: ProfileFormData]
}>()

// ─── Proxy dropdown state ───────────────────────────────────────────────────
const proxyOpen = ref(false)

// ─── Status Options ─────────────────────────────────────────────────────────
const statusOptions = [
  { label: t('customer.status_active'), value: 'active' },
  { label: t('customer.status_disabled'), value: 'disabled' },
  { label: t('customer.status_expired'), value: 'expired' },
  { label: t('customer.status_limited'), value: 'limited' },
]

// ─── Data Limit Unit Options ────────────────────────────────────────────────
const dataLimitUnitOptions = [
  { label: 'MB', value: 'MB' },
  { label: 'GB', value: 'GB' },
  { label: 'TB', value: 'TB' },
]

// ─── Expiry Chips ───────────────────────────────────────────────────────────
const expiryChips: ExpiryOffset[] = ['+7d', '+1m', '+2m', '+3m', '+6m', '+1y']

// ─── Available Protocols with sub-options ───────────────────────────────────
const availableProtocols = [
  {
    label: 'OpenVPN',
    value: 'openvpn',
    options: [
      { label: 'Password + Certificate', value: 'auth' },
      { label: 'Passwordless (cert only)', value: 'noauth' },
    ],
  },
  { label: 'WireGuard', value: 'wireguard', options: null },
  { label: 'IKEv2', value: 'ikev2', options: null },
  {
    label: 'L2TP',
    value: 'l2tp',
    options: [
      { label: 'L2TP/IPsec', value: 'ipsec' },
      { label: 'L2TP (plain)', value: 'plain' },
    ],
  },
  { label: 'SSH Tunnel', value: 'ssh', options: null },
  { label: 'MTProto', value: 'mtproto', options: null },
]

// ─── Computed ───────────────────────────────────────────────────────────────

const proxyButtonLabel = computed(() => {
  const count = props.modelValue.allowed_protocols.length
  if (count === 0) return 'No protocols selected'
  if (count === availableProtocols.length) return 'All protocols'
  return `${count} protocol${count > 1 ? 's' : ''} selected`
})

const expiryDateValue = computed(() => {
  if (!props.modelValue.expiry_date) return ''
  try {
    const date = new Date(props.modelValue.expiry_date)
    if (isNaN(date.getTime())) return ''
    return date.toISOString().split('T')[0]
  } catch {
    return ''
  }
})

const expiresInDays = computed<number | null>(() => {
  if (!props.modelValue.expiry_date) return null
  try {
    const expiry = new Date(props.modelValue.expiry_date)
    if (isNaN(expiry.getTime())) return null
    const now = new Date()
    const expiryDay = new Date(expiry.getFullYear(), expiry.getMonth(), expiry.getDate())
    const today = new Date(now.getFullYear(), now.getMonth(), now.getDate())
    const diffMs = expiryDay.getTime() - today.getTime()
    return Math.round(diffMs / (1000 * 60 * 60 * 24))
  } catch {
    return null
  }
})

// ─── Methods ────────────────────────────────────────────────────────────────

function updateField(field: keyof ProfileFormData, value: any) {
  emit('update:modelValue', { ...props.modelValue, [field]: value })
}

function onDateInput(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.value) {
    const date = new Date(target.value + 'T00:00:00')
    updateField('expiry_date', date.toISOString())
  } else {
    updateField('expiry_date', '')
  }
}

function applyChip(chip: ExpiryOffset) {
  const result = computeExpiryDate(new Date(), chip)
  updateField('expiry_date', result.toISOString())
}

function isProtocolEnabled(protocol: string): boolean {
  return props.modelValue.allowed_protocols.includes(protocol)
}

function toggleProtocol(protocol: string) {
  const current = [...props.modelValue.allowed_protocols]
  const index = current.indexOf(protocol)
  if (index >= 0) {
    current.splice(index, 1)
  } else {
    current.push(protocol)
  }
  updateField('allowed_protocols', current)
}

function getProtocolOption(protocol: string): string {
  return props.modelValue.protocol_options?.[protocol] ?? ''
}

function setProtocolOption(protocol: string, value: string) {
  const opts = { ...(props.modelValue.protocol_options || {}), [protocol]: value }
  updateField('protocol_options', opts)
}
</script>

<style scoped>
.profile-fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

/* ─── Data Limit ──────────────────────────────────────────────────────────── */
.profile-fields__data-limit {
  display: grid;
  grid-template-columns: 1fr 80px;
  gap: var(--space-2, 8px);
}

.profile-fields__unit-select {
  width: 80px;
}

/* ─── Expiry Section ──────────────────────────────────────────────────────── */
.profile-fields__expiry {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.profile-fields__date-wrapper {
  position: relative;
}

.profile-fields__date-input {
  display: block;
  width: 100%;
  height: 36px;
  padding: 0 var(--space-3, 12px);
  background: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 6px);
  color: var(--color-text, #e6edf3);
  font-family: var(--font-family);
  font-size: var(--text-base, 14px);
  line-height: var(--leading-normal);
  outline: none;
  cursor: pointer;
  transition: border-color 150ms ease, box-shadow 150ms ease;
}

.profile-fields__date-input:focus-visible {
  border-color: var(--color-primary, #2563eb);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.25);
}

/* Quick-set chips — minimal, no bg/border */
.profile-fields__quick-chips {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2, 8px);
}

.profile-fields__chip {
  padding: 2px 8px;
  border: none;
  background: transparent;
  color: var(--color-primary, #2563eb);
  font-size: var(--text-sm, 13px);
  font-family: var(--font-family);
  font-weight: 500;
  cursor: pointer;
  border-radius: var(--radius-sm, 4px);
  transition: background 100ms ease;
}

.profile-fields__chip:hover {
  background: rgba(37, 99, 235, 0.1);
}

.profile-fields__expiry-info {
  margin: 0;
  font-size: var(--text-sm, 13px);
  color: var(--color-muted, #6b7280);
}

.profile-fields__expiry-info--warning {
  color: var(--color-warning, #d97706);
  font-weight: 500;
}

.profile-fields__expiry-info--expired {
  color: var(--color-danger, #dc2626);
  font-weight: 500;
}

/* ─── Proxy Settings Dropdown ─────────────────────────────────────────────── */
.profile-fields__proxy {
  position: relative;
}

.profile-fields__proxy-toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  height: 36px;
  padding: 0 var(--space-3, 12px);
  background: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 6px);
  color: var(--color-text, #e6edf3);
  font-size: var(--text-sm, 13px);
  font-family: var(--font-family);
  cursor: pointer;
  transition: border-color 150ms ease;
}

.profile-fields__proxy-toggle:hover {
  border-color: var(--color-primary, #2563eb);
}

.profile-fields__proxy-chevron {
  transition: transform 150ms ease;
}

.profile-fields__proxy-chevron--open {
  transform: rotate(180deg);
}

.profile-fields__proxy-panel {
  margin-top: var(--space-2, 8px);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 8px);
  background: var(--color-surface, #0b1120);
  padding: var(--space-2, 8px);
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
}

.profile-fields__proxy-item {
  border-radius: var(--radius-sm, 4px);
  padding: var(--space-2, 8px) var(--space-3, 12px);
}

.profile-fields__proxy-item:hover {
  background: var(--color-surface-2, #1e2630);
}

.profile-fields__proxy-item-header {
  display: flex;
  align-items: center;
}

.profile-fields__proxy-label {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2, 8px);
  cursor: pointer;
  font-size: var(--text-sm, 13px);
  color: var(--color-text, #e6edf3);
}

.profile-fields__proxy-label input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--color-primary, #2563eb);
  cursor: pointer;
}

.profile-fields__proxy-options {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
  padding-left: 24px;
  margin-top: var(--space-1, 4px);
}

.profile-fields__proxy-option {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2, 8px);
  cursor: pointer;
  font-size: 12px;
  color: var(--color-muted, #8b98a5);
}

.profile-fields__proxy-option input[type="radio"] {
  width: 14px;
  height: 14px;
  accent-color: var(--color-primary, #2563eb);
  cursor: pointer;
}
</style>
