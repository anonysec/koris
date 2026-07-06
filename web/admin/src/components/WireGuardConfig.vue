<script setup lang="ts">
import { ref, watch, onMounted } from 'vue'
import { useWireGuard, type WireGuardNodeConfig } from '@/composables/useWireGuard'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'

const props = defineProps<{
  nodeId: number
  currentConfig?: {
    port?: number
    network?: string
    enabled?: boolean
    mtu?: number
    extra_json?: Record<string, unknown>
  } | null
}>()

const emit = defineEmits<{
  saved: []
}>()

const { t } = useI18n()
const toast = useToast()
const { saveNodeWireGuardConfig } = useWireGuard()
const saving = ref(false)

const form = ref<WireGuardNodeConfig>({
  port: 51820,
  network: '10.66.66.0/24',
  dns_1: '1.1.1.1',
  dns_2: '8.8.8.8',
  mtu: 1420,
  gaming_optimize: false,
  enabled: false,
})

function loadFromConfig() {
  if (props.currentConfig) {
    form.value.port = props.currentConfig.port ?? 51820
    form.value.network = props.currentConfig.network ?? '10.66.66.0/24'
    form.value.enabled = props.currentConfig.enabled ?? false
    form.value.mtu = props.currentConfig.mtu ?? 1420
    const extra = props.currentConfig.extra_json || {}
    form.value.dns_1 = (extra.dns_1 as string) || '1.1.1.1'
    form.value.dns_2 = (extra.dns_2 as string) || '8.8.8.8'
    form.value.gaming_optimize = (extra.gaming_optimize as boolean) || false
  }
}

watch(() => props.currentConfig, loadFromConfig, { immediate: true })

async function handleSave() {
  saving.value = true
  const success = await saveNodeWireGuardConfig(props.nodeId, form.value)
  saving.value = false
  if (success) {
    toast.success(t('wireguard.config_saved'))
    emit('saved')
  } else {
    toast.error(t('wireguard.config_save_error'))
  }
}

onMounted(loadFromConfig)
</script>

<template>
  <div class="wireguard-config">
    <div class="wg-config-header">
      <span class="wg-config-icon">🔐</span>
      <span class="wg-config-title">WireGuard</span>
      <label class="toggle-switch">
        <input type="checkbox" v-model="form.enabled" />
        <span class="toggle-switch__slider" />
      </label>
    </div>

    <form class="wg-config-form" @submit.prevent="handleSave">
      <div class="form-grid">
        <FormField name="wg-port" :label="t('wireguard.listen_port')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.port" type="number" placeholder="51820" />
          </template>
        </FormField>

        <FormField name="wg-network" :label="t('wireguard.network_cidr')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.network" placeholder="10.66.66.0/24" />
          </template>
        </FormField>

        <FormField name="wg-dns1" :label="t('wireguard.primary_dns')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.dns_1" placeholder="1.1.1.1" />
          </template>
        </FormField>

        <FormField name="wg-dns2" :label="t('wireguard.secondary_dns')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.dns_2" placeholder="8.8.8.8" />
          </template>
        </FormField>

        <FormField name="wg-mtu" :label="t('wireguard.mtu')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="form.mtu" type="number" placeholder="1420" />
          </template>
        </FormField>
      </div>

      <!-- Gaming Optimize Toggle -->
      <div class="wg-toggle-row">
        <label class="toggle-label">
          <span class="toggle-switch">
            <input type="checkbox" v-model="form.gaming_optimize" />
            <span class="toggle-switch__slider" />
          </span>
          <span>{{ t('wireguard.gaming_optimize') }}</span>
        </label>
        <span class="toggle-hint">{{ t('wireguard.gaming_optimize_hint') }}</span>
      </div>

      <div class="form-actions">
        <Button type="submit" variant="primary" :loading="saving">
          {{ t('wireguard.save_config') }}
        </Button>
      </div>
    </form>
  </div>
</template>

<style scoped>
.wireguard-config {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  background: var(--color-surface);
}

.wg-config-header {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-bottom: var(--space-4);
}

.wg-config-icon {
  font-size: 1.2rem;
}

.wg-config-title {
  font-weight: var(--font-semibold);
  font-size: var(--text-base);
  flex: 1;
}

.wg-config-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--space-3);
}

.wg-toggle-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.toggle-label {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}

.toggle-hint {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
}

/* Shared toggle switch styling */
.toggle-switch {
  position: relative;
  display: inline-block;
  width: 36px;
  height: 20px;
}

.toggle-switch input {
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-switch__slider {
  position: absolute;
  cursor: pointer;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: var(--color-border);
  border-radius: 10px;
  transition: background var(--duration-fast);
}

.toggle-switch__slider::before {
  content: '';
  position: absolute;
  height: 14px;
  width: 14px;
  left: 3px;
  bottom: 3px;
  background: white;
  border-radius: 50%;
  transition: transform var(--duration-fast);
}

.toggle-switch input:checked + .toggle-switch__slider {
  background: var(--color-primary);
}

.toggle-switch input:checked + .toggle-switch__slider::before {
  transform: translateX(16px);
}
</style>
