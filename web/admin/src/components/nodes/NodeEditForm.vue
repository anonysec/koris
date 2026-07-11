<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useNodesStore, type KnodeNode, type NodeFormData } from '@/stores/nodes'
import { useToast } from '@koris/composables/useToast'
import Button from '@koris/ui/Button.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'
import Textarea from '@koris/ui/Textarea.vue'
import Alert from '@koris/ui/Alert.vue'

const props = defineProps<{
  node: KnodeNode
}>()

const emit = defineEmits<{
  (e: 'updated'): void
}>()

const nodesStore = useNodesStore()
const toast = useToast()

const MASKED = '••••••••'

// ─── Form State ─────────────────────────────────────────────────────────────
const name = ref(props.node.name)
const address = ref(props.node.address)
const domain = ref(props.node.domain ?? '')
const port = ref(props.node.port)
const apiKey = ref(MASKED)
const clientCertPem = ref(MASKED)
const clientKeyPem = ref(MASKED)
const caCertPem = ref(MASKED)

const saving = ref(false)
const feedback = ref<{ type: 'success' | 'error'; message: string } | null>(null)

// Reset form when node prop changes
watch(() => props.node, (n) => {
  name.value = n.name
  address.value = n.address
  domain.value = n.domain ?? ''
  port.value = n.port
  apiKey.value = MASKED
  clientCertPem.value = MASKED
  clientKeyPem.value = MASKED
  caCertPem.value = MASKED
  feedback.value = null
})

// ─── Validation ─────────────────────────────────────────────────────────────
const errors = computed(() => {
  const e: Record<string, string> = {}
  if (!address.value.trim()) e.address = 'Address is required'
  const p = Number(port.value)
  if (!Number.isInteger(p) || p < 1 || p > 65535) e.port = 'Port must be 1–65535'
  // Cert fields: only validate if user clears the masked value (non-masked + empty = error)
  if (clientCertPem.value !== MASKED && !clientCertPem.value.trim()) e.clientCertPem = 'Client certificate is required'
  if (clientKeyPem.value !== MASKED && !clientKeyPem.value.trim()) e.clientKeyPem = 'Client key is required'
  if (caCertPem.value !== MASKED && !caCertPem.value.trim()) e.caCertPem = 'CA certificate is required'
  return e
})

const isValid = computed(() => Object.keys(errors.value).length === 0)

// ─── Detect changes ─────────────────────────────────────────────────────────
const hasChanges = computed(() => {
  return (
    name.value !== props.node.name ||
    address.value !== props.node.address ||
    domain.value.trim() !== (props.node.domain ?? '') ||
    Number(port.value) !== props.node.port ||
    apiKey.value !== MASKED ||
    clientCertPem.value !== MASKED ||
    clientKeyPem.value !== MASKED ||
    caCertPem.value !== MASKED
  )
})

// ─── Actions ────────────────────────────────────────────────────────────────
async function handleSubmit() {
  if (!isValid.value || !hasChanges.value) return

  saving.value = true
  feedback.value = null

  const domainChanged = domain.value.trim() !== (props.node.domain ?? '')
  const coreChanged =
    name.value !== props.node.name ||
    address.value !== props.node.address ||
    Number(port.value) !== props.node.port ||
    apiKey.value !== MASKED ||
    clientCertPem.value !== MASKED ||
    clientKeyPem.value !== MASKED ||
    caCertPem.value !== MASKED

  // Only send changed fields
  const payload: Partial<NodeFormData> = {}
  if (name.value !== props.node.name) payload.name = name.value.trim()
  if (address.value !== props.node.address) payload.address = address.value.trim()
  if (Number(port.value) !== props.node.port) payload.port = Number(port.value)
  if (apiKey.value !== MASKED) payload.api_key = apiKey.value
  if (clientCertPem.value !== MASKED) payload.client_cert_pem = clientCertPem.value
  if (clientKeyPem.value !== MASKED) payload.client_key_pem = clientKeyPem.value
  if (caCertPem.value !== MASKED) payload.ca_cert_pem = caCertPem.value

  const results: boolean[] = []
  if (coreChanged) results.push(await nodesStore.updateNode(props.node.id, payload))
  if (domainChanged) results.push(await nodesStore.setNodeDomain(props.node.id, domain.value))

  const ok = results.length > 0 && results.every(Boolean)
  if (ok) {
    feedback.value = { type: 'success', message: 'Node updated successfully' }
    toast.success('Node updated')
    emit('updated')
  } else {
    feedback.value = { type: 'error', message: 'Failed to update node. Check the values and try again.' }
  }

  saving.value = false
}

function clearField(field: 'apiKey' | 'clientCertPem' | 'clientKeyPem' | 'caCertPem') {
  if (field === 'apiKey') apiKey.value = ''
  else if (field === 'clientCertPem') clientCertPem.value = ''
  else if (field === 'clientKeyPem') clientKeyPem.value = ''
  else if (field === 'caCertPem') caCertPem.value = ''
}
</script>

<template>
  <form class="node-edit-form" @submit.prevent="handleSubmit">
    <h3 class="node-edit-form__title">Edit Node</h3>

    <Alert v-if="feedback" :variant="feedback.type" closable @close="feedback = null">
      {{ feedback.message }}
    </Alert>

    <FormField name="node-name" label="Name">
      <template #default="{ fieldId }">
        <Input :id="fieldId" v-model="name" placeholder="Node name" />
      </template>
    </FormField>

    <FormField name="node-address" label="Address" :error="errors.address">
      <template #default="{ fieldId }">
        <Input :id="fieldId" v-model="address" placeholder="IP or hostname" />
      </template>
    </FormField>

    <FormField name="node-domain" label="Public Domain (client endpoint)" hint="What VPN clients connect to, e.g. vpn.example.com or 91.107.164.180">
      <template #default="{ fieldId }">
        <Input :id="fieldId" v-model="domain" placeholder="Domain or public IP" />
      </template>
    </FormField>

    <FormField name="node-port" label="Port" :error="errors.port">
      <template #default="{ fieldId }">
        <Input :id="fieldId" v-model="port" type="number" />
      </template>
    </FormField>

    <FormField name="node-api-key" label="API Key" hint="Clear to enter a new key">
      <template #default="{ fieldId }">
        <div class="node-edit-form__masked-field">
          <Input
            :id="fieldId"
            v-model="apiKey"
            :type="apiKey === MASKED ? 'text' : 'password'"
            :disabled="apiKey === MASKED"
          />
          <Button
            v-if="apiKey === MASKED"
            variant="ghost"
            size="sm"
            @click="clearField('apiKey')"
          >
            Change
          </Button>
        </div>
      </template>
    </FormField>

    <FormField name="node-client-cert" label="Client Certificate (PEM)" :error="errors.clientCertPem">
      <template #default="{ fieldId }">
        <div class="node-edit-form__masked-field">
          <Textarea
            v-if="clientCertPem !== MASKED"
            :id="fieldId"
            v-model="clientCertPem"
            :rows="4"
            placeholder="-----BEGIN CERTIFICATE-----"
          />
          <div v-else class="node-edit-form__masked-value">
            <span class="node-edit-form__masked-text">{{ MASKED }}</span>
            <Button variant="ghost" size="sm" @click="clearField('clientCertPem')">
              Change
            </Button>
          </div>
        </div>
      </template>
    </FormField>

    <FormField name="node-client-key" label="Client Key (PEM)" :error="errors.clientKeyPem">
      <template #default="{ fieldId }">
        <div class="node-edit-form__masked-field">
          <Textarea
            v-if="clientKeyPem !== MASKED"
            :id="fieldId"
            v-model="clientKeyPem"
            :rows="4"
            placeholder="-----BEGIN PRIVATE KEY-----"
          />
          <div v-else class="node-edit-form__masked-value">
            <span class="node-edit-form__masked-text">{{ MASKED }}</span>
            <Button variant="ghost" size="sm" @click="clearField('clientKeyPem')">
              Change
            </Button>
          </div>
        </div>
      </template>
    </FormField>

    <FormField name="node-ca-cert" label="CA Certificate (PEM)" :error="errors.caCertPem">
      <template #default="{ fieldId }">
        <div class="node-edit-form__masked-field">
          <Textarea
            v-if="caCertPem !== MASKED"
            :id="fieldId"
            v-model="caCertPem"
            :rows="4"
            placeholder="-----BEGIN CERTIFICATE-----"
          />
          <div v-else class="node-edit-form__masked-value">
            <span class="node-edit-form__masked-text">{{ MASKED }}</span>
            <Button variant="ghost" size="sm" @click="clearField('caCertPem')">
              Change
            </Button>
          </div>
        </div>
      </template>
    </FormField>

    <Button
      type="submit"
      variant="primary"
      :loading="saving"
      :disabled="!isValid || !hasChanges"
    >
      Test &amp; Save
    </Button>
  </form>
</template>

<style scoped>
.node-edit-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  max-width: 520px;
}

.node-edit-form__title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

.node-edit-form__masked-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.node-edit-form__masked-value {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.node-edit-form__masked-text {
  flex: 1;
  color: var(--color-muted);
  font-family: monospace;
  letter-spacing: 2px;
}
</style>
