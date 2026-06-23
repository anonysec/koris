<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KFormField from '@koris/ui/KFormField.vue'

export interface ProtocolConfig {
  protocol: string
  enabled: boolean
  port: number
  transport: string
  path: string
  realityEnabled?: boolean
}

const props = defineProps<{
  modelValue: ProtocolConfig[]
}>()

const emit = defineEmits<{
  'update:modelValue': [value: ProtocolConfig[]]
}>()

const { t } = useI18n()

const protocols = ref<ProtocolConfig[]>(props.modelValue.length > 0 ? [...props.modelValue] : [
  { protocol: 'vless', enabled: false, port: 443, transport: 'tcp', path: '', realityEnabled: false },
  { protocol: 'vmess', enabled: false, port: 10086, transport: 'ws', path: '/vmess' },
  { protocol: 'trojan', enabled: false, port: 8443, transport: 'tcp', path: '' },
  { protocol: 'shadowsocks', enabled: false, port: 1080, transport: 'tcp', path: '' },
])

watch(() => props.modelValue, (val) => {
  if (val.length > 0) {
    protocols.value = [...val]
  }
}, { deep: true })

const transportOptions = [
  { label: 'TCP', value: 'tcp' },
  { label: 'WebSocket', value: 'ws' },
  { label: 'gRPC', value: 'grpc' },
  { label: 'HTTP/2', value: 'h2' },
]

const protocolLabels: Record<string, string> = {
  vless: 'VLESS',
  vmess: 'VMess',
  trojan: 'Trojan',
  shadowsocks: 'Shadowsocks',
}

function toggleProtocol(index: number) {
  protocols.value[index].enabled = !protocols.value[index].enabled
  emitUpdate()
}

function toggleReality(index: number) {
  protocols.value[index].realityEnabled = !protocols.value[index].realityEnabled
  emitUpdate()
}

function updatePort(index: number, value: number) {
  protocols.value[index].port = value
  emitUpdate()
}

function updateTransport(index: number, value: string) {
  protocols.value[index].transport = value
  emitUpdate()
}

function updatePath(index: number, value: string) {
  protocols.value[index].path = value
  emitUpdate()
}

function emitUpdate() {
  emit('update:modelValue', [...protocols.value])
}

const enabledCount = computed(() => protocols.value.filter(p => p.enabled).length)
</script>

<template>
  <div class="protocol-selector">
    <div class="protocol-selector__header">
      <h4>{{ t('xray.protocols') }}</h4>
      <span class="enabled-count">{{ enabledCount }} {{ t('xray.enabled') }}</span>
    </div>

    <div class="protocol-list">
      <div
        v-for="(proto, index) in protocols"
        :key="proto.protocol"
        class="protocol-card"
        :class="{ 'protocol-card--active': proto.enabled }"
      >
        <div class="protocol-card__header">
          <label class="protocol-toggle">
            <input
              type="checkbox"
              :checked="proto.enabled"
              @change="toggleProtocol(index)"
            />
            <span class="protocol-name">{{ protocolLabels[proto.protocol] || proto.protocol }}</span>
          </label>
          <span v-if="proto.enabled" class="protocol-badge">{{ t('label.enabled') }}</span>
        </div>

        <div v-if="proto.enabled" class="protocol-card__settings">
          <div class="settings-grid">
            <KFormField :name="`proto-port-${index}`" :label="t('label.port')">
              <template #default="{ fieldId }">
                <KInput
                  :id="fieldId"
                  :model-value="proto.port"
                  type="number"
                  placeholder="443"
                  @update:model-value="updatePort(index, Number($event))"
                />
              </template>
            </KFormField>

            <KFormField :name="`proto-transport-${index}`" :label="t('xray.transport')">
              <template #default="{ fieldId }">
                <KSelect
                  :id="fieldId"
                  :model-value="proto.transport"
                  :options="transportOptions"
                  @update:model-value="updateTransport(index, $event as string)"
                />
              </template>
            </KFormField>

            <KFormField
              v-if="proto.transport === 'ws' || proto.transport === 'grpc' || proto.transport === 'h2'"
              :name="`proto-path-${index}`"
              :label="t('xray.path')"
            >
              <template #default="{ fieldId }">
                <KInput
                  :id="fieldId"
                  :model-value="proto.path"
                  :placeholder="proto.transport === 'grpc' ? 'service-name' : '/path'"
                  @update:model-value="updatePath(index, $event as string)"
                />
              </template>
            </KFormField>
          </div>

          <!-- Reality toggle for VLESS -->
          <div v-if="proto.protocol === 'vless'" class="reality-toggle">
            <label class="protocol-toggle">
              <input
                type="checkbox"
                :checked="proto.realityEnabled"
                @change="toggleReality(index)"
              />
              <span class="reality-label">{{ t('xray.reality_enabled') }}</span>
            </label>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.protocol-selector__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.protocol-selector__header h4 {
  margin: 0;
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--color-text);
}

.enabled-count {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.protocol-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.protocol-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  background: var(--color-surface);
  transition: border-color 0.15s, box-shadow 0.15s;
}

.protocol-card--active {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 1px rgba(91, 157, 255, 0.2);
}

.protocol-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.protocol-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  cursor: pointer;
}

.protocol-toggle input[type="checkbox"] {
  width: 16px;
  height: 16px;
  accent-color: var(--color-primary);
  cursor: pointer;
}

.protocol-name {
  font-weight: 600;
  font-size: var(--text-sm);
  color: var(--color-text);
}

.protocol-badge {
  font-size: var(--text-xs);
  color: #22c55e;
  font-weight: 500;
}

.protocol-card__settings {
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

.settings-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: var(--space-3);
}

.reality-toggle {
  margin-top: var(--space-3);
  padding-top: var(--space-3);
  border-top: 1px dashed var(--color-border);
}

.reality-label {
  font-size: var(--text-sm);
  color: var(--color-text);
}
</style>
