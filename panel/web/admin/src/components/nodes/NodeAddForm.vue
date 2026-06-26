<script setup lang="ts">
import { ref, computed } from 'vue'
import { useNodesStore, type NodeFormData } from '@/stores/nodes'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'
import KTextarea from '@koris/ui/KTextarea.vue'
import KAlert from '@koris/ui/KAlert.vue'

const emit = defineEmits<{
  (e: 'created', nodeId: number): void
  (e: 'close'): void
}>()

const { t } = useI18n()
const nodesStore = useNodesStore()
const toast = useToast()

// ─── Form State ─────────────────────────────────────────────────────────────
const name = ref('')
const address = ref('')
const port = ref(2083)
const apiKey = ref('')
const certPem = ref('')

const saving = ref(false)
const submitted = ref(false)
const feedback = ref<{ type: 'success' | 'error'; message: string } | null>(null)

// ─── Validation ─────────────────────────────────────────────────────────────
const errors = computed(() => {
  if (!submitted.value) return {}
  const e: Record<string, string> = {}
  if (!address.value.trim()) e.address = t('nodes.validation_address')
  const p = Number(port.value)
  if (!Number.isInteger(p) || p < 1 || p > 65535) e.port = t('nodes.validation_port')
  if (!apiKey.value.trim()) e.apiKey = t('nodes.validation_api_key')
  if (!certPem.value.trim()) e.certPem = t('nodes.validation_cert')
  return e
})

const isValid = computed(() => {
  if (!address.value.trim()) return false
  const p = Number(port.value)
  if (!Number.isInteger(p) || p < 1 || p > 65535) return false
  if (!apiKey.value.trim()) return false
  if (!certPem.value.trim()) return false
  return true
})

// ─── Actions ────────────────────────────────────────────────────────────────
async function handleSubmit() {
  submitted.value = true
  if (!isValid.value) return

  saving.value = true
  feedback.value = null

  const payload: NodeFormData = {
    name: name.value.trim() || address.value.trim(),
    address: address.value.trim(),
    port: Number(port.value),
    api_key: apiKey.value.trim(),
    client_cert_pem: '',
    client_key_pem: '',
    ca_cert_pem: certPem.value.trim(),
  }

  const nodeId = await nodesStore.createNode(payload)

  if (nodeId) {
    feedback.value = { type: 'success', message: t('nodes.created_success') }
    toast.success(t('nodes.created_success'))
    emit('created', nodeId)
    resetForm()
  } else {
    feedback.value = { type: 'error', message: t('nodes.created_error') }
  }

  saving.value = false
}

function resetForm() {
  name.value = ''
  address.value = ''
  port.value = 2083
  apiKey.value = ''
  certPem.value = ''
}
</script>

<template>
  <div class="node-add-slide">
    <div class="node-add-slide__header">
      <h3 class="node-add-slide__title">{{ t('nodes.add_node') }}</h3>
      <KButton variant="ghost" size="sm" @click="emit('close')">✕</KButton>
    </div>

    <p class="node-add-slide__hint">
      {{ t('nodes.add_hint') }}
    </p>

    <KAlert v-if="feedback" :variant="feedback.type" closable @close="feedback = null">
      {{ feedback.message }}
    </KAlert>

    <form class="node-add-slide__form" @submit.prevent="handleSubmit">
      <KFormField name="node-name" :label="t('nodes.node_name')" hint="Optional — defaults to address">
        <template #default="{ fieldId }">
          <KInput :id="fieldId" v-model="name" placeholder="e.g. de-1, us-west" />
        </template>
      </KFormField>

      <KFormField name="node-address" :label="t('nodes.address')" :error="errors.address">
        <template #default="{ fieldId }">
          <KInput :id="fieldId" v-model="address" placeholder="IP or hostname (e.g. 185.1.2.3)" />
        </template>
      </KFormField>

      <KFormField name="node-port" :label="t('label.port')" :error="errors.port">
        <template #default="{ fieldId }">
          <KInput :id="fieldId" v-model="port" type="number" placeholder="2083" />
        </template>
      </KFormField>

      <KFormField name="node-api-key" :label="t('nodes.api_key')" :error="errors.apiKey" hint="Shown when knode is installed">
        <template #default="{ fieldId }">
          <KInput :id="fieldId" v-model="apiKey" type="password" placeholder="Paste from knode install output" />
        </template>
      </KFormField>

      <KFormField name="node-cert" :label="t('nodes.certificate')" :error="errors.certPem" hint="CA certificate from knode install output">
        <template #default="{ fieldId }">
          <KTextarea :id="fieldId" v-model="certPem" :rows="5" placeholder="-----BEGIN CERTIFICATE-----&#10;...&#10;-----END CERTIFICATE-----" />
        </template>
      </KFormField>

      <div class="node-add-slide__actions">
        <KButton type="submit" variant="primary" :loading="saving" :disabled="!isValid">
          {{ t('nodes.test_and_save') }}
        </KButton>
        <KButton variant="ghost" @click="emit('close')">
          {{ t('btn.cancel') }}
        </KButton>
      </div>
    </form>
  </div>
</template>

<style scoped>
.node-add-slide {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  max-width: 480px;
  margin-bottom: var(--space-5);
}

.node-add-slide__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.node-add-slide__title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

.node-add-slide__hint {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-muted);
  line-height: 1.5;
}

.node-add-slide__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.node-add-slide__actions {
  display: flex;
  gap: var(--space-2);
  padding-top: var(--space-2);
}
</style>
