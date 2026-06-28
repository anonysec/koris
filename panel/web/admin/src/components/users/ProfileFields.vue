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

    <!-- Expiry Date -->
    <KFormField name="expiry" :label="t('customer.expiry_date')">
      <template #default="{ fieldId }">
        <div class="profile-fields__expiry">
          <!-- Input mode toggle -->
          <div class="profile-fields__expiry-toggle" role="group" :aria-label="t('customer.expiry_input_mode')">
            <button
              type="button"
              :class="['profile-fields__toggle-btn', { 'profile-fields__toggle-btn--active': expiryMode === 'date' }]"
              :aria-pressed="expiryMode === 'date'"
              @click="expiryMode = 'date'"
            >
              {{ t('customer.expiry_mode_date') }}
            </button>
            <button
              type="button"
              :class="['profile-fields__toggle-btn', { 'profile-fields__toggle-btn--active': expiryMode === 'days' }]"
              :aria-pressed="expiryMode === 'days'"
              @click="expiryMode = 'days'"
            >
              {{ t('customer.expiry_mode_days') }}
            </button>
          </div>

          <!-- Date-type input: calendar picker -->
          <div v-if="expiryMode === 'date'" class="profile-fields__expiry-date">
            <input
              :id="fieldId"
              type="date"
              class="profile-fields__date-input"
              :value="expiryDateValue"
              @input="onDateInput"
            />
          </div>

          <!-- Days-type input: shortcut chips -->
          <div v-else class="profile-fields__expiry-chips">
            <KExpiryChips
              :model-value="modelValue.expiry_date"
              @update:model-value="updateField('expiry_date', $event)"
            />
          </div>

          <!-- "Expires in X days" display -->
          <p v-if="expiresInDays !== null" class="profile-fields__expiry-info">
            <span v-if="expiresInDays > 0">
              {{ t('customer.expires_in_days').replace('{days}', String(expiresInDays)) }}
            </span>
            <span v-else-if="expiresInDays === 0" class="profile-fields__expiry-info--warning">
              {{ t('customer.expires_today') }}
            </span>
            <span v-else class="profile-fields__expiry-info--expired">
              {{ t('customer.expired_days_ago').replace('{days}', String(Math.abs(expiresInDays))) }}
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
          :placeholder="t('customer.note_placeholder')"
          :rows="3"
          @update:model-value="updateField('note', $event)"
        />
      </template>
    </KFormField>

    <!-- Proxy settings (allowed protocols) -->
    <KFormField name="proxy-settings" :label="t('customer.proxy_settings')">
      <template #default>
        <div class="profile-fields__protocols">
          <label
            v-for="protocol in availableProtocols"
            :key="protocol.value"
            class="profile-fields__protocol-item"
          >
            <input
              type="checkbox"
              :value="protocol.value"
              :checked="isProtocolEnabled(protocol.value)"
              @change="toggleProtocol(protocol.value)"
            />
            <span class="profile-fields__protocol-label">{{ protocol.label }}</span>
          </label>
          <p v-if="availableProtocols.length === 0" class="profile-fields__no-protocols">
            {{ t('customer.no_protocols_available') }}
          </p>
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
import KExpiryChips from '@koris/ui/KExpiryChips.vue'

const { t } = useI18n()

/**
 * Form data shape for user profile editing.
 * Matches the fields managed by the detail panel.
 */
export interface ProfileFormData {
  username: string
  status: string
  expiry_date: string
  note: string
  allowed_protocols: string[]
}

export interface ProfileFieldsProps {
  modelValue: ProfileFormData
}

const props = defineProps<ProfileFieldsProps>()

const emit = defineEmits<{
  'update:modelValue': [value: ProfileFormData]
}>()

// ─── Expiry Mode Toggle ─────────────────────────────────────────────────────
const expiryMode = ref<'date' | 'days'>('date')

// ─── Status Options ─────────────────────────────────────────────────────────
const statusOptions = [
  { label: t('customer.status_active'), value: 'active' },
  { label: t('customer.status_disabled'), value: 'disabled' },
  { label: t('customer.status_expired'), value: 'expired' },
  { label: t('customer.status_limited'), value: 'limited' },
]

// ─── Available Protocols ────────────────────────────────────────────────────
const availableProtocols = [
  { label: 'OpenVPN', value: 'openvpn' },
  { label: 'WireGuard', value: 'wireguard' },
  { label: 'IKEv2', value: 'ikev2' },
  { label: 'L2TP', value: 'l2tp' },
  { label: 'SSH', value: 'ssh' },
  { label: 'MTProto', value: 'mtproto' },
]

// ─── Computed Values ────────────────────────────────────────────────────────

/**
 * Formats the expiry ISO string to YYYY-MM-DD for the native date input.
 */
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

/**
 * Calculates "Expires in X days" from the expiry_date field.
 * Returns null if no expiry date is set.
 */
const expiresInDays = computed<number | null>(() => {
  if (!props.modelValue.expiry_date) return null
  try {
    const expiry = new Date(props.modelValue.expiry_date)
    if (isNaN(expiry.getTime())) return null
    const now = new Date()
    // Compare date-only (strip time)
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
    // Convert YYYY-MM-DD to ISO string
    const date = new Date(target.value + 'T00:00:00')
    updateField('expiry_date', date.toISOString())
  } else {
    updateField('expiry_date', '')
  }
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
</script>

<style scoped>
.profile-fields {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

/* ─── Expiry Section ──────────────────────────────────────────────────────── */
.profile-fields__expiry {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.profile-fields__expiry-toggle {
  display: inline-flex;
  border-radius: var(--radius-md, 6px);
  border: 1px solid var(--color-border);
  overflow: hidden;
}

.profile-fields__toggle-btn {
  padding: var(--space-1, 4px) var(--space-3, 12px);
  border: none;
  background: var(--color-surface-2, #f5f5f5);
  color: var(--color-text);
  font-size: var(--text-sm, 13px);
  font-family: var(--font-family);
  cursor: pointer;
  transition: background var(--duration-fast, 100ms) var(--ease-out, ease-out),
    color var(--duration-fast, 100ms) var(--ease-out, ease-out);
}

.profile-fields__toggle-btn:not(:last-child) {
  border-right: 1px solid var(--color-border);
}

.profile-fields__toggle-btn:hover:not(.profile-fields__toggle-btn--active) {
  background: var(--color-surface-3, #ebebeb);
}

.profile-fields__toggle-btn--active {
  background: var(--color-primary, #2563eb);
  color: #fff;
}

.profile-fields__expiry-date,
.profile-fields__expiry-chips {
  margin-top: var(--space-1, 4px);
}

.profile-fields__date-input {
  display: block;
  width: 100%;
  height: 36px;
  padding: 0 var(--space-3, 12px);
  background: var(--color-surface, #fff);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md, 6px);
  color: var(--color-text);
  font-family: var(--font-family);
  font-size: var(--text-base, 14px);
  line-height: var(--leading-normal);
  outline: none;
  transition: border-color var(--duration-normal, 150ms) var(--ease-default, ease),
    box-shadow var(--duration-normal, 150ms) var(--ease-default, ease);
}

.profile-fields__date-input:focus-visible {
  border-color: var(--color-primary, #2563eb);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.25);
}

.profile-fields__expiry-info {
  margin: 0;
  font-size: var(--text-sm, 13px);
  color: var(--color-muted, #6b7280);
}

.profile-fields__expiry-info--warning {
  color: var(--color-warning, #d97706);
  font-weight: var(--font-medium, 500);
}

.profile-fields__expiry-info--expired {
  color: var(--color-danger, #dc2626);
  font-weight: var(--font-medium, 500);
}

/* ─── Proxy / Protocol Settings ───────────────────────────────────────────── */
.profile-fields__protocols {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3, 12px);
}

.profile-fields__protocol-item {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2, 8px);
  cursor: pointer;
  font-size: var(--text-sm, 13px);
}

.profile-fields__protocol-item input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--color-primary, #2563eb);
  cursor: pointer;
}

.profile-fields__protocol-label {
  color: var(--color-text);
  user-select: none;
}

.profile-fields__no-protocols {
  margin: 0;
  font-size: var(--text-sm, 13px);
  color: var(--color-muted, #6b7280);
}
</style>
