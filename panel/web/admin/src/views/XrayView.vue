<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useNodesStore } from '@/stores/nodes'
import KButton from '@koris/ui/KButton.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KInput from '@koris/ui/KInput.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import XrayConfigEditor from '@/components/XrayConfigEditor.vue'
import XrayProtocolSelector from '@/components/XrayProtocolSelector.vue'
import XrayTemplates from '@/components/XrayTemplates.vue'
import type { ProtocolConfig } from '@/components/XrayProtocolSelector.vue'
import type { XrayTemplate } from '@/components/XrayTemplates.vue'

const { t } = useI18n()
const api = useApi({ baseUrl: '/api/admin' })
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()

// ─── State ───────────────────────────────────────────────────────────────────
interface XrayConfig {
  node_id: number
  enabled: boolean
  inbounds: Array<{
    protocol: string
    port: number
    transport: string
    tag?: string
    settings?: any
  }>
  routing: {
    domain_strategy?: string
    rules?: Array<{
      type?: string
      domain?: string[]
      ip?: string[]
      outbound_tag?: string
    }>
  }
  tls: {
    cert_path?: string
    key_path?: string
    server_name?: string
    alpn?: string[]
  }
  reality_config?: {
    server_names?: string[]
    private_key?: string
    public_key?: string
    short_ids?: string[]
  }
  last_synced_at?: string
}

interface FallbackEntry {
  dest: string
  path?: string
  xver?: number
}

const selectedNodeId = ref<number | string>('')
const loading = ref(false)
const deploying = ref(false)
const syncing = ref(false)
const config = ref<XrayConfig | null>(null)
const rawConfigJson = ref('')
const showRawEditor = ref(false)
const fallbacks = ref<FallbackEntry[]>([])

// Protocol selector state
const protocolConfigs = ref<ProtocolConfig[]>([])

// Reality config
const realityServerNames = ref('')
const realityPublicKey = ref('')
const realityPrivateKey = ref('')

// ─── Computed ────────────────────────────────────────────────────────────────
const nodeOptions = computed(() =>
  nodesStore.list.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

const hasConfig = computed(() => config.value !== null)

const lastSyncLabel = computed(() => {
  if (!config.value?.last_synced_at) return t('xray.never_synced')
  return new Date(config.value.last_synced_at).toLocaleString()
})

// ─── Watchers ────────────────────────────────────────────────────────────────
watch(selectedNodeId, (nodeId) => {
  if (nodeId) {
    loadXrayConfig(Number(nodeId))
  } else {
    config.value = null
    protocolConfigs.value = []
  }
})

// ─── API ─────────────────────────────────────────────────────────────────────
async function loadXrayConfig(nodeId: number) {
  loading.value = true
  try {
    const res = await api.get<{ ok: boolean; config: XrayConfig }>(`/xray/nodes/${nodeId}`)
    if (res.ok && res.config) {
      config.value = res.config
      rawConfigJson.value = JSON.stringify(res.config, null, 2)
      syncProtocolsFromConfig(res.config)
      syncRealityFromConfig(res.config)
      syncFallbacksFromConfig(res.config)
    }
  } catch {
    config.value = null
    rawConfigJson.value = ''
  } finally {
    loading.value = false
  }
}

function syncProtocolsFromConfig(cfg: XrayConfig) {
  const protocols: ProtocolConfig[] = [
    { protocol: 'vless', enabled: false, port: 443, transport: 'tcp', path: '', realityEnabled: false },
    { protocol: 'vmess', enabled: false, port: 10086, transport: 'ws', path: '/vmess' },
    { protocol: 'trojan', enabled: false, port: 8443, transport: 'tcp', path: '' },
    { protocol: 'shadowsocks', enabled: false, port: 1080, transport: 'tcp', path: '' },
  ]

  for (const inbound of cfg.inbounds || []) {
    const proto = protocols.find(p => p.protocol === inbound.protocol)
    if (proto) {
      proto.enabled = true
      proto.port = inbound.port
      proto.transport = inbound.transport
      if (inbound.protocol === 'vless' && cfg.reality_config) {
        proto.realityEnabled = true
      }
    }
  }

  protocolConfigs.value = protocols
}

function syncRealityFromConfig(cfg: XrayConfig) {
  if (cfg.reality_config) {
    realityServerNames.value = (cfg.reality_config.server_names || []).join(', ')
    realityPublicKey.value = cfg.reality_config.public_key || ''
    realityPrivateKey.value = cfg.reality_config.private_key || ''
  } else {
    realityServerNames.value = ''
    realityPublicKey.value = ''
    realityPrivateKey.value = ''
  }
}

function syncFallbacksFromConfig(_cfg: XrayConfig) {
  // Fallbacks are stored in inbound settings; parse them out
  fallbacks.value = []
}

async function handleSaveConfig() {
  if (!selectedNodeId.value) return

  const nodeId = Number(selectedNodeId.value)
  const payload = buildConfigPayload(nodeId)

  loading.value = true
  try {
    const res = await api.post<{ ok: boolean }>(`/xray/nodes/${nodeId}`, payload)
    if (res.ok) {
      toast.success(t('xray.config_saved'))
      await loadXrayConfig(nodeId)
    }
  } catch {
    // Handled by useApi
  } finally {
    loading.value = false
  }
}

function buildConfigPayload(nodeId: number) {
  const inbounds = protocolConfigs.value
    .filter(p => p.enabled)
    .map(p => ({
      protocol: p.protocol,
      port: p.port,
      transport: p.transport,
      tag: `${p.protocol}-in`,
    }))

  const realityConfig = protocolConfigs.value.find(p => p.protocol === 'vless')?.realityEnabled
    ? {
        server_names: realityServerNames.value.split(',').map(s => s.trim()).filter(Boolean),
        public_key: realityPublicKey.value,
        private_key: realityPrivateKey.value,
      }
    : undefined

  return {
    node_id: nodeId,
    enabled: true,
    inbounds,
    routing: config.value?.routing || { domain_strategy: 'AsIs', rules: [] },
    tls: config.value?.tls || {},
    reality_config: realityConfig,
  }
}

async function handleDeploy() {
  if (!selectedNodeId.value) return

  const confirmed = await confirm({
    title: t('xray.deploy_title'),
    message: t('xray.deploy_msg'),
    variant: 'default',
    icon: '🚀',
    confirmText: t('xray.deploy'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  deploying.value = true
  try {
    const res = await api.post<{ ok: boolean }>('/node/tasks', {
      node_id: Number(selectedNodeId.value),
      action: 'xray_deploy',
      payload_json: {},
    })
    if (res.ok) {
      toast.success(t('xray.deploy_success'))
    }
  } catch {
    // Handled by useApi
  } finally {
    deploying.value = false
  }
}

async function handleSyncUsers() {
  if (!selectedNodeId.value) return

  syncing.value = true
  try {
    const res = await api.post<{ ok: boolean }>('/node/tasks', {
      node_id: Number(selectedNodeId.value),
      action: 'xray_sync_users',
      payload_json: {},
    })
    if (res.ok) {
      toast.success(t('xray.sync_users_success'))
    }
  } catch {
    // Handled by useApi
  } finally {
    syncing.value = false
  }
}

function addFallback() {
  fallbacks.value.push({ dest: '', path: '', xver: 0 })
}

function removeFallback(index: number) {
  fallbacks.value.splice(index, 1)
}

async function copyToClipboard(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    toast.success(t('xray.copied'))
  } catch {
    toast.error(t('xray.copy_failed'))
  }
}

function handleApplyTemplate(tmpl: XrayTemplate) {
  try {
    const parsed = JSON.parse(tmpl.config_json)
    rawConfigJson.value = JSON.stringify(parsed, null, 2)
    toast.success(t('xray.template_applied'))
    // Reload config from the template content
    if (parsed.inbounds) {
      config.value = {
        node_id: Number(selectedNodeId.value),
        enabled: true,
        inbounds: parsed.inbounds || [],
        routing: parsed.routing || {},
        tls: parsed.tls || {},
        reality_config: parsed.reality_config,
      }
      syncProtocolsFromConfig(config.value)
      syncRealityFromConfig(config.value)
    }
  } catch {
    toast.error(t('xray.invalid_json'))
  }
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  if (nodesStore.list.length === 0) {
    nodesStore.loadNodes()
  }
})
</script>

<template>
  <div class="page xray-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('xray.title') }}</h2>
      <div class="page-header__actions">
        <KButton
          variant="ghost"
          :loading="syncing"
          :disabled="!selectedNodeId"
          @click="handleSyncUsers"
        >
          🔄 {{ t('xray.sync_users') }}
        </KButton>
        <KButton
          variant="primary"
          :loading="deploying"
          :disabled="!selectedNodeId"
          @click="handleDeploy"
        >
          🚀 {{ t('xray.deploy_to_node') }}
        </KButton>
      </div>
    </header>

    <!-- Node Selector -->
    <div class="node-selector">
      <KFormField name="xray-node" :label="t('xray.select_node')">
        <template #default="{ fieldId }">
          <KSelect
            :id="fieldId"
            v-model="selectedNodeId"
            :options="nodeOptions"
            :placeholder="t('xray.choose_node')"
          />
        </template>
      </KFormField>
      <div v-if="config" class="sync-info">
        <span class="sync-label">{{ t('xray.last_sync') }}:</span>
        <span class="sync-value">{{ lastSyncLabel }}</span>
      </div>
    </div>

    <!-- Loading State -->
    <div v-if="loading" class="xray-loading">
      <KSkeleton variant="rect" width="100%" :height="200" />
      <KSkeleton variant="rect" width="100%" :height="150" />
    </div>

    <!-- Empty State -->
    <KEmptyState
      v-else-if="!selectedNodeId"
      icon="⚡"
      :title="t('xray.empty_title')"
      :description="t('xray.empty_desc')"
    />

    <!-- Config Sections -->
    <div v-else-if="!loading" class="xray-sections">
      <!-- Protocol Configuration -->
      <section class="xray-section">
        <XrayProtocolSelector v-model="protocolConfigs" />
      </section>

      <!-- Reality Config -->
      <section
        v-if="protocolConfigs.find(p => p.protocol === 'vless' && p.realityEnabled)"
        class="xray-section"
      >
        <h3 class="section-title">{{ t('xray.reality_config') }}</h3>
        <div class="reality-grid">
          <KFormField name="reality-sni" :label="t('xray.server_names')">
            <template #default="{ fieldId }">
              <KInput
                :id="fieldId"
                v-model="realityServerNames"
                placeholder="www.google.com, www.microsoft.com"
              />
            </template>
          </KFormField>

          <div class="key-display">
            <KFormField name="reality-pubkey" :label="t('xray.public_key')">
              <template #default="{ fieldId }">
                <div class="key-row">
                  <KInput :id="fieldId" v-model="realityPublicKey" readonly />
                  <KButton variant="ghost" size="sm" @click="copyToClipboard(realityPublicKey)">
                    📋
                  </KButton>
                </div>
              </template>
            </KFormField>
          </div>

          <div class="key-display">
            <KFormField name="reality-privkey" :label="t('xray.private_key')">
              <template #default="{ fieldId }">
                <div class="key-row">
                  <KInput :id="fieldId" v-model="realityPrivateKey" readonly type="password" />
                  <KButton variant="ghost" size="sm" @click="copyToClipboard(realityPrivateKey)">
                    📋
                  </KButton>
                </div>
              </template>
            </KFormField>
          </div>
        </div>
      </section>

      <!-- Fallback Chain -->
      <section class="xray-section">
        <div class="section-header">
          <h3 class="section-title">{{ t('xray.fallback_chain') }}</h3>
          <KButton variant="ghost" size="sm" icon="+" @click="addFallback">
            {{ t('xray.add_fallback') }}
          </KButton>
        </div>

        <div v-if="fallbacks.length === 0" class="fallback-empty">
          {{ t('xray.no_fallbacks') }}
        </div>

        <div v-else class="fallback-list">
          <div v-for="(fb, index) in fallbacks" :key="index" class="fallback-entry">
            <KInput
              v-model="fb.dest"
              :placeholder="t('xray.fallback_dest')"
              class="fallback-input"
            />
            <KInput
              v-model="fb.path"
              :placeholder="t('xray.fallback_path')"
              class="fallback-input"
            />
            <KButton variant="danger" size="sm" @click="removeFallback(index)">
              ✕
            </KButton>
          </div>
        </div>
      </section>

      <!-- Raw JSON Editor -->
      <section class="xray-section">
        <div class="section-header">
          <h3 class="section-title">{{ t('xray.raw_config') }}</h3>
          <KButton variant="ghost" size="sm" @click="showRawEditor = !showRawEditor">
            {{ showRawEditor ? t('xray.hide_editor') : t('xray.show_editor') }}
          </KButton>
        </div>
        <XrayConfigEditor
          v-if="showRawEditor"
          v-model="rawConfigJson"
        />
      </section>

      <!-- Templates -->
      <section class="xray-section">
        <XrayTemplates
          :node-id="Number(selectedNodeId)"
          @apply-template="handleApplyTemplate"
        />
      </section>

      <!-- Save Button -->
      <div class="xray-footer">
        <KButton variant="primary" :loading="loading" @click="handleSaveConfig">
          {{ t('btn.save') }}
        </KButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.xray-view {
  padding: var(--space-6);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
  gap: var(--space-3);
}

.page-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.page-header__actions {
  display: flex;
  gap: var(--space-2);
}

.node-selector {
  display: flex;
  align-items: flex-end;
  gap: var(--space-4);
  margin-bottom: var(--space-6);
  flex-wrap: wrap;
}

.sync-info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-xs);
  padding-bottom: var(--space-2);
}

.sync-label {
  color: var(--color-muted);
}

.sync-value {
  color: var(--color-text);
}

.xray-loading {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.xray-sections {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.xray-section {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  background: var(--color-surface);
}

.section-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.section-title {
  margin: 0 0 var(--space-4);
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--color-text);
}

.section-header .section-title {
  margin-bottom: 0;
}

.reality-grid {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.key-row {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

.key-row :deep(input) {
  flex: 1;
}

.fallback-empty {
  font-size: var(--text-sm);
  color: var(--color-muted);
  text-align: center;
  padding: var(--space-4);
}

.fallback-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.fallback-entry {
  display: flex;
  gap: var(--space-2);
  align-items: center;
}

.fallback-input {
  flex: 1;
}

.xray-footer {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-4);
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: flex-start;
  }

  .node-selector {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
