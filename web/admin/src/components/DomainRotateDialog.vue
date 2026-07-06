<script setup lang="ts">
import { ref, watch, computed } from 'vue'
import { useDomainsStore, type VpnDomain } from '@/stores/domains'
import { useToast } from '@koris/composables/useToast'
import Button from '@koris/ui/Button.vue'

const props = defineProps<{
  open: boolean
  domain: VpnDomain | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'rotated'): void
}>()

const toast = useToast()
const domainsStore = useDomainsStore()

const newIp = ref('')
const submitting = ref(false)
const confirmStep = ref(false)
const validationError = ref('')

// Reset state when dialog opens/closes
watch(() => props.open, (isOpen) => {
  if (!isOpen) {
    newIp.value = ''
    confirmStep.value = false
    validationError.value = ''
    submitting.value = false
  }
})

// ─── Validation ─────────────────────────────────────────────────────────────

function isValidIP(ip: string): boolean {
  if (!ip) return false
  const ipv4Regex = /^(\d{1,3})\.(\d{1,3})\.(\d{1,3})\.(\d{1,3})$/
  const ipv4Match = ip.match(ipv4Regex)
  if (ipv4Match) {
    return ipv4Match.slice(1).every((octet) => {
      const n = parseInt(octet, 10)
      return n >= 0 && n <= 255
    })
  }
  const ipv6Regex = /^([0-9a-fA-F]{0,4}:){2,7}[0-9a-fA-F]{0,4}$/
  return ipv6Regex.test(ip) || /^::$/.test(ip) || /^::1$/.test(ip)
}

function validate(): string | null {
  const trimmed = newIp.value.trim()
  if (!trimmed) return 'New IP address is required'
  if (!isValidIP(trimmed)) return 'Invalid IP address (must be a valid IPv4 or IPv6 address)'
  if (props.domain && trimmed === props.domain.ip_address) {
    return 'New IP must be different from the current IP'
  }
  return null
}

// ─── Actions ────────────────────────────────────────────────────────────────

function handleCancel() {
  emit('close')
}

function handleOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    handleCancel()
  }
}

function handleNext() {
  const error = validate()
  if (error) {
    validationError.value = error
    toast.warning(error)
    return
  }
  validationError.value = ''
  confirmStep.value = true
}

function handleBack() {
  confirmStep.value = false
}

async function handleConfirm() {
  if (!props.domain) return

  submitting.value = true
  try {
    const success = await domainsStore.rotateIP(props.domain.id, {
      new_ip: newIp.value.trim(),
    })

    if (success) {
      toast.success(`IP rotated for ${props.domain.name}`)
      emit('rotated')
      emit('close')
    } else {
      toast.error('Failed to rotate IP')
    }
  } catch (err: any) {
    toast.error(err?.message || 'An error occurred')
  } finally {
    submitting.value = false
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="rotate-dialog">
      <div
        v-if="open && domain"
        class="rotate-dialog__overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="rotate-dialog-title"
        @click="handleOverlayClick"
      >
        <div class="rotate-dialog">
          <h2 id="rotate-dialog-title" class="rotate-dialog__title">
            Rotate IP Address
          </h2>

          <!-- Current domain info -->
          <div class="rotate-dialog__info">
            <div class="rotate-dialog__info-row">
              <span class="rotate-dialog__info-label">Domain</span>
              <code class="rotate-dialog__info-value">{{ domain.name }}</code>
            </div>
            <div class="rotate-dialog__info-row">
              <span class="rotate-dialog__info-label">Current IP</span>
              <code class="rotate-dialog__info-value rotate-dialog__info-value--mono">{{ domain.ip_address }}</code>
            </div>
          </div>

          <!-- Step 1: Input new IP -->
          <template v-if="!confirmStep">
            <div class="rotate-dialog__field">
              <label class="rotate-dialog__label" for="rotate-new-ip">
                New IP Address
              </label>
              <input
                id="rotate-new-ip"
                v-model="newIp"
                type="text"
                class="rotate-dialog__input"
                placeholder="Enter new IP address"
                @keyup.enter="handleNext"
              />
              <span v-if="validationError" class="rotate-dialog__error">
                {{ validationError }}
              </span>
            </div>

            <div class="rotate-dialog__actions">
              <Button variant="ghost" @click="handleCancel">
                Cancel
              </Button>
              <Button variant="primary" @click="handleNext">
                Continue
              </Button>
            </div>
          </template>

          <!-- Step 2: Confirmation -->
          <template v-else>
            <div class="rotate-dialog__confirm">
              <div class="rotate-dialog__warning">
                <span class="rotate-dialog__warning-icon">⚠️</span>
                <p>This will update the DNS A record for this domain. Active connections using the old IP may be disrupted.</p>
              </div>

              <div class="rotate-dialog__change">
                <div class="rotate-dialog__change-row">
                  <span class="rotate-dialog__change-label">From</span>
                  <code class="rotate-dialog__change-value">{{ domain.ip_address }}</code>
                </div>
                <span class="rotate-dialog__change-arrow">→</span>
                <div class="rotate-dialog__change-row">
                  <span class="rotate-dialog__change-label">To</span>
                  <code class="rotate-dialog__change-value rotate-dialog__change-value--new">{{ newIp.trim() }}</code>
                </div>
              </div>
            </div>

            <div class="rotate-dialog__actions">
              <Button variant="ghost" :disabled="submitting" @click="handleBack">
                Back
              </Button>
              <Button variant="primary" :loading="submitting" @click="handleConfirm">
                Confirm Rotation
              </Button>
            </div>
          </template>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.rotate-dialog__overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal, 200);
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  padding: var(--space-4);
}

.rotate-dialog {
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-xl, 14px);
  padding: var(--space-6, 24px);
  max-width: 480px;
  width: 100%;
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.rotate-dialog__title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.rotate-dialog__info {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--color-surface-2, #1e2630);
  border-radius: var(--radius-md, 6px);
}

.rotate-dialog__info-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.rotate-dialog__info-label {
  font-size: var(--text-sm);
  color: var(--color-muted, #888);
}

.rotate-dialog__info-value {
  font-size: var(--text-sm);
  color: var(--color-text);
}

.rotate-dialog__info-value--mono {
  font-family: var(--font-mono, monospace);
}

.rotate-dialog__field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.rotate-dialog__label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}

.rotate-dialog__input {
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface-2, #1e2630);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 6px);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: var(--font-mono, monospace);
  outline: none;
  transition: border-color 0.15s;
}

.rotate-dialog__input:focus {
  border-color: var(--color-primary, #6366f1);
}

.rotate-dialog__error {
  font-size: var(--text-xs);
  color: var(--color-error, #ef4444);
}

.rotate-dialog__confirm {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.rotate-dialog__warning {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.25);
  border-radius: var(--radius-md, 6px);
}

.rotate-dialog__warning-icon {
  font-size: 1.2rem;
  flex-shrink: 0;
}

.rotate-dialog__warning p {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-warning, #f59e0b);
  line-height: var(--leading-normal, 1.5);
}

.rotate-dialog__change {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface-2, #1e2630);
  border-radius: var(--radius-md, 6px);
}

.rotate-dialog__change-row {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  flex: 1;
}

.rotate-dialog__change-label {
  font-size: var(--text-xs);
  color: var(--color-muted, #888);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.rotate-dialog__change-value {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-sm);
  color: var(--color-text);
}

.rotate-dialog__change-value--new {
  color: var(--color-success, #22c55e);
}

.rotate-dialog__change-arrow {
  font-size: var(--text-lg);
  color: var(--color-muted, #888);
  flex-shrink: 0;
}

.rotate-dialog__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-2);
}

/* Transition */
.rotate-dialog-enter-active,
.rotate-dialog-leave-active {
  transition: opacity 0.2s ease-out;
}

.rotate-dialog-enter-active .rotate-dialog,
.rotate-dialog-leave-active .rotate-dialog {
  transition: transform 0.2s ease-out, opacity 0.2s ease-out;
}

.rotate-dialog-enter-from,
.rotate-dialog-leave-to {
  opacity: 0;
}

.rotate-dialog-enter-from .rotate-dialog,
.rotate-dialog-leave-to .rotate-dialog {
  transform: scale(0.95);
  opacity: 0;
}
</style>
