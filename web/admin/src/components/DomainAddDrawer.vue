<script setup lang="ts">
import { ref, computed } from 'vue'
import { useDomainsStore } from '@/stores/domains'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import SlideOver from '@koris/ui/SlideOver.vue'
import Button from '@koris/ui/Button.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'

defineProps<{
  open: boolean
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const { t } = useI18n()
const toast = useToast()
const domainsStore = useDomainsStore()

const domainName = ref('')
const ipAddress = ref('')
const submitting = ref(false)
const validationError = ref('')

// ─── Validation ─────────────────────────────────────────────────────────────

/**
 * Validates a domain name per RFC 1123:
 * - Labels separated by dots
 * - Each label: 1-63 chars, lowercase a-z, 0-9, hyphens
 * - No leading/trailing hyphens per label
 * - Total length ≤ 253
 */
function isValidDomainName(name: string): boolean {
  if (!name || name.length > 253) return false
  const labels = name.split('.')
  if (labels.length < 2) return false
  for (const label of labels) {
    if (label.length === 0 || label.length > 63) return false
    if (!/^[a-z0-9]([a-z0-9-]*[a-z0-9])?$/.test(label)) return false
  }
  return true
}

/**
 * Validates an IPv4 or IPv6 address.
 */
function isValidIP(ip: string): boolean {
  if (!ip) return false
  // IPv4: four octets 0-255
  const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/
  const ipv4Match = ip.match(ipv4Regex)
  if (ipv4Match) {
    return ipv4Match.slice(1).every((octet) => {
      const n = parseInt(octet, 10)
      return n >= 0 && n <= 255
    })
  }
  // IPv6: simplified check for valid hex groups separated by colons
  const ipv6Regex = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/
  return ipv6Regex.test(ip) || /^::$/.test(ip) || /^::1$/.test(ip)
}

function validate(): string | null {
  if (!domainName.value.trim()) return 'Domain name is required'
  if (!isValidDomainName(domainName.value.trim().toLowerCase())) {
    return 'Invalid domain name (must be a valid RFC 1123 hostname, e.g. vpn.example.com)'
  }
  if (!ipAddress.value.trim()) return 'IP address is required'
  if (!isValidIP(ipAddress.value.trim())) {
    return 'Invalid IP address (must be a valid IPv4 or IPv6 address)'
  }
  return null
}

// ─── Actions ────────────────────────────────────────────────────────────────

function reset() {
  domainName.value = ''
  ipAddress.value = ''
  validationError.value = ''
}

function handleClose() {
  reset()
  emit('close')
}

async function handleSubmit() {
  const error = validate()
  if (error) {
    validationError.value = error
    toast.warning(error)
    return
  }

  validationError.value = ''
  submitting.value = true

  try {
    const result = await domainsStore.createDomain({
      name: domainName.value.trim().toLowerCase(),
      ip_address: ipAddress.value.trim(),
    })

    if (result) {
      toast.success('Domain created successfully')
      reset()
      emit('close')
    } else {
      toast.error('Failed to create domain')
    }
  } catch (err: any) {
    toast.error(err?.message || 'An error occurred')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <SlideOver :open="open" title="Add Domain" @close="handleClose">
    <form class="entity-form" autocomplete="off" @submit.prevent="handleSubmit">
      <FormField
        name="domain-name"
        label="Domain Name"
        required
        :error="validationError && !domainName ? validationError : ''"
        hint="RFC 1123 hostname (e.g. vpn.example.com)"
      >
        <template #default="{ fieldId }">
          <Input
            :id="fieldId"
            v-model="domainName"
            placeholder="vpn.example.com"
          />
        </template>
      </FormField>

      <FormField
        name="domain-ip"
        label="IP Address"
        required
        :error="validationError && !ipAddress ? validationError : ''"
        hint="IPv4 or IPv6 address"
      >
        <template #default="{ fieldId }">
          <Input
            :id="fieldId"
            v-model="ipAddress"
            placeholder="192.168.1.1"
          />
        </template>
      </FormField>

      <div v-if="validationError" class="validation-error">
        {{ validationError }}
      </div>

      <div class="entity-form__actions">
        <Button type="submit" variant="primary" :loading="submitting" full-width>
          Create Domain
        </Button>
      </div>
    </form>
  </SlideOver>
</template>

<style scoped>
.entity-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3, 0.75rem);
  padding: var(--space-4, 1rem);
}

.entity-form__actions {
  display: flex;
  gap: var(--space-2, 0.5rem);
  padding: var(--space-4, 1rem);
}

.validation-error {
  font-size: var(--text-sm, 0.875rem);
  color: var(--color-error, #ef4444);
  padding: var(--space-2, 0.5rem) var(--space-3, 0.75rem);
  background: rgba(239, 68, 68, 0.08);
  border: 1px solid rgba(239, 68, 68, 0.25);
  border-radius: var(--radius-md, 6px);
}
</style>
