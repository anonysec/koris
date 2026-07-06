<script setup lang="ts">
import { ref, computed, onUnmounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useWebSocket } from '@koris/composables/useWebSocket'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import FormField from '@koris/ui/FormField.vue'
import Select from '@koris/ui/Select.vue'

const { t } = useI18n()
const api = useApi()
const toast = useToast()

// ─── Types ───────────────────────────────────────────────────────────────────
interface ProvisionStep {
  step: string
  status: 'pending' | 'active' | 'completed' | 'failed'
  message?: string
}

interface NodeGroup {
  id: number
  name: string
}

// ─── State ───────────────────────────────────────────────────────────────────
const form = ref({
  host: '',
  port: 22,
  user: 'root',
  password: '',
  ssh_key: '',
  auth_method: 'password' as 'password' | 'key',
  group_id: '' as number | string,
})

const provisioning = ref(false)
const provisionStarted = ref(false)
const provisionComplete = ref(false)
const provisionError = ref('')
const groups = ref<NodeGroup[]>([])

const PROVISION_STEPS = ['connecting', 'installing', 'configuring', 'verifying', 'completed'] as const
const steps = ref<ProvisionStep[]>([])

// ─── WebSocket ───────────────────────────────────────────────────────────────
const wsProtocol = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
const wsUrl = `${wsProtocol}//${window.location.host}/api/admin/nodes/provision/status`

const { connected: wsConnected, connect: wsConnect, disconnect: wsDisconnect } = useWebSocket({
  url: wsUrl,
  autoConnect: false,
  reconnect: false,
  onMessage(data: any) {
    handleProvisionMessage(data)
  },
  onDisconnect() {
    if (provisionStarted.value && !provisionComplete.value) {
      provisionError.value = t('provision.ws_disconnected')
    }
  },
})

function handleProvisionMessage(data: any) {
  if (data.type === 'step_update') {
    const stepIndex = steps.value.findIndex(s => s.step === data.step)
    if (stepIndex >= 0) {
      steps.value[stepIndex].status = data.status
      steps.value[stepIndex].message = data.message || ''
    }

    // Mark next step as active
    if (data.status === 'completed') {
      const nextIndex = stepIndex + 1
      if (nextIndex < steps.value.length) {
        steps.value[nextIndex].status = 'active'
      }
    }
  } else if (data.type === 'completed') {
    provisionComplete.value = true
    provisioning.value = false
    toast.success(t('provision.success'))
  } else if (data.type === 'failed') {
    provisionComplete.value = true
    provisioning.value = false
    provisionError.value = data.message || t('provision.failed')
    // Mark current active step as failed
    const activeStep = steps.value.find(s => s.status === 'active')
    if (activeStep) {
      activeStep.status = 'failed'
      activeStep.message = data.message || ''
    }
  }
}

// ─── Computed ────────────────────────────────────────────────────────────────
const groupOptions = computed(() =>
  groups.value.map(g => ({ label: g.name, value: g.id }))
)

const canSubmit = computed(() => {
  if (!form.value.host || !form.value.port || !form.value.user) return false
  if (form.value.auth_method === 'password' && !form.value.password) return false
  if (form.value.auth_method === 'key' && !form.value.ssh_key) return false
  return true
})

const overallStatus = computed(() => {
  if (!provisionStarted.value) return 'idle'
  if (provisionError.value) return 'failed'
  if (provisionComplete.value) return 'completed'
  return 'in_progress'
})

// ─── Actions ─────────────────────────────────────────────────────────────────
async function fetchGroups() {
  try {
    const res = await api.get<{ ok: boolean; groups: NodeGroup[] }>('/api/node-groups')
    groups.value = res.groups || []
  } catch {
    // Silent fail — groups is optional
  }
}

async function startProvision() {
  if (!canSubmit.value) return

  provisioning.value = true
  provisionStarted.value = true
  provisionComplete.value = false
  provisionError.value = ''

  // Initialize steps
  steps.value = PROVISION_STEPS.map((step, i) => ({
    step,
    status: i === 0 ? 'active' : 'pending',
  }))

  // Connect WebSocket for progress
  wsConnect()

  try {
    const payload: Record<string, any> = {
      host: form.value.host,
      port: form.value.port,
      user: form.value.user,
      group_id: form.value.group_id ? Number(form.value.group_id) : undefined,
    }

    if (form.value.auth_method === 'password') {
      payload.password = form.value.password
    } else {
      payload.ssh_key = form.value.ssh_key
    }

    await api.post<{ ok: boolean }>('/api/nodes/provision', payload)
  } catch (err: any) {
    provisioning.value = false
    provisionError.value = err.message || t('provision.start_failed')
    wsDisconnect()
  }
}

function resetForm() {
  provisionStarted.value = false
  provisionComplete.value = false
  provisionError.value = ''
  steps.value = []
  form.value = {
    host: '',
    port: 22,
    user: 'root',
    password: '',
    ssh_key: '',
    auth_method: 'password',
    group_id: '',
  }
  wsDisconnect()
}

function getStepIcon(step: ProvisionStep): string {
  switch (step.status) {
    case 'completed': return '✓'
    case 'failed': return '✗'
    case 'active': return '⟳'
    default: return '○'
  }
}

// Fetch groups on mount
fetchGroups()

onUnmounted(() => {
  wsDisconnect()
})
</script>

<template>
  <div class="page provision-view">
    <h3 class="section-title">{{ t('provision.title') }}</h3>

    <!-- SSH Credential Form -->
    <section v-if="!provisionStarted" class="provision-form-section">
      <form class="provision-form" @submit.prevent="startProvision">
        <div class="form-grid">
          <FormField name="prov-host" :label="t('provision.host')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.host" placeholder="192.168.1.100" />
            </template>
          </FormField>
          <FormField name="prov-port" :label="t('provision.port')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model.number="form.port" type="number" placeholder="22" />
            </template>
          </FormField>
          <FormField name="prov-user" :label="t('provision.user')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.user" placeholder="root" />
            </template>
          </FormField>
          <FormField name="prov-auth" :label="t('provision.auth_method')">
            <template #default="{ fieldId }">
              <Select
                :id="fieldId"
                v-model="form.auth_method"
                :options="[
                  { label: t('provision.password'), value: 'password' },
                  { label: t('provision.ssh_key'), value: 'key' },
                ]"
              />
            </template>
          </FormField>
        </div>

        <FormField v-if="form.auth_method === 'password'" name="prov-password" :label="t('provision.password')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.password" type="password" :placeholder="t('provision.password_placeholder')" />
          </template>
        </FormField>

        <FormField v-if="form.auth_method === 'key'" name="prov-key" :label="t('provision.ssh_key')" required>
          <template #default="{ fieldId }">
            <textarea
              :id="fieldId"
              v-model="form.ssh_key"
              class="ssh-key-input"
              rows="5"
              :placeholder="t('provision.key_placeholder')"
            />
          </template>
        </FormField>

        <FormField name="prov-group" :label="t('provision.target_group')">
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="form.group_id" :options="groupOptions" :placeholder="t('provision.no_group')" />
          </template>
        </FormField>

        <div class="form-actions">
          <Button type="submit" variant="primary" :disabled="!canSubmit" :loading="provisioning">
            {{ t('provision.start') }}
          </Button>
        </div>
      </form>
    </section>

    <!-- Provisioning Progress -->
    <section v-else class="provision-progress-section">
      <div class="progress-card">
        <div class="progress-card__header">
          <h4 class="progress-card__title">
            {{ t('provision.provisioning') }} {{ form.host }}
          </h4>
          <span v-if="wsConnected" class="ws-indicator ws-indicator--connected" :title="t('provision.ws_connected')">●</span>
          <span v-else class="ws-indicator ws-indicator--disconnected" :title="t('provision.ws_disconnected')">●</span>
        </div>

        <!-- Steps Progress -->
        <div class="steps-list">
          <div
            v-for="step in steps"
            :key="step.step"
            class="step-item"
            :class="`step-item--${step.status}`"
          >
            <span class="step-item__icon">{{ getStepIcon(step) }}</span>
            <div class="step-item__content">
              <span class="step-item__label">{{ t(`provision.step_${step.step}`) }}</span>
              <span v-if="step.message" class="step-item__message">{{ step.message }}</span>
            </div>
          </div>
        </div>

        <!-- Error Display -->
        <div v-if="provisionError" class="provision-error">
          <span class="provision-error__icon">⚠</span>
          <span class="provision-error__text">{{ provisionError }}</span>
        </div>

        <!-- Completion Actions -->
        <div v-if="provisionComplete" class="progress-card__actions">
          <Button variant="primary" @click="resetForm">
            {{ t('provision.provision_another') }}
          </Button>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.provision-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.section-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

/* Form */
.provision-form-section {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  padding: var(--space-5);
}

.provision-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
  gap: var(--space-4);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-3);
}

.ssh-key-input {
  width: 100%;
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-bg, #0b1120);
  color: var(--color-text);
  font-family: var(--font-mono, monospace);
  font-size: var(--text-sm);
  resize: vertical;
}

.ssh-key-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.2);
}

/* Progress */
.provision-progress-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.progress-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.progress-card__header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.progress-card__title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  flex: 1;
}

.ws-indicator {
  font-size: var(--text-sm);
}

.ws-indicator--connected {
  color: var(--color-success, #10b981);
}

.ws-indicator--disconnected {
  color: var(--color-danger, #ef4444);
}

/* Steps */
.steps-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.step-item {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3);
  padding: var(--space-3);
  border-radius: var(--radius-md);
  transition: background 0.15s;
}

.step-item--active {
  background: rgba(59, 130, 246, 0.06);
}

.step-item--completed {
  background: rgba(16, 185, 129, 0.06);
}

.step-item--failed {
  background: rgba(239, 68, 68, 0.06);
}

.step-item__icon {
  width: 20px;
  height: 20px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-sm);
  flex-shrink: 0;
}

.step-item--completed .step-item__icon {
  color: var(--color-success, #10b981);
}

.step-item--failed .step-item__icon {
  color: var(--color-danger, #ef4444);
}

.step-item--active .step-item__icon {
  color: var(--color-primary);
  animation: spin 1s linear infinite;
}

.step-item--pending .step-item__icon {
  color: var(--color-muted);
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.step-item__content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.step-item__label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}

.step-item__message {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

/* Error */
.provision-error {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: var(--radius-md);
  background: rgba(239, 68, 68, 0.06);
}

.provision-error__icon {
  color: var(--color-danger, #ef4444);
  flex-shrink: 0;
}

.provision-error__text {
  font-size: var(--text-sm);
  color: var(--color-text);
  word-break: break-word;
}

.progress-card__actions {
  display: flex;
  gap: var(--space-2);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

@media (max-width: 768px) {
  .form-grid {
    grid-template-columns: 1fr;
  }
}
</style>
