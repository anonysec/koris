<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'
import KButton from '@koris/ui/KButton.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KSlideOver from '@koris/ui/KSlideOver.vue'

const { get, post } = useApi()
const toast = useToast()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface Node {
  id: number
  name: string
  address: string
  status?: string
}

interface ProtocolConfig {
  enabled: boolean
  port: number
  network: string
  extra_json?: Record<string, any>
}

type VpnConfig = Record<string, ProtocolConfig>

interface NodeWithConfig {
  node: Node
  config: VpnConfig
  loading: boolean
  hovered: boolean
}

// ═══════════════════════════════════════════════════════════════════════════════
// Protocol definitions
// ═══════════════════════════════════════════════════════════════════════════════

const protocols = [
  { key: 'openvpn', name: 'OpenVPN', icon: '🔐', defaultPort: 1194 },
  { key: 'wireguard', name: 'WireGuard', icon: '🛡️', defaultPort: 51820 },
  { key: 'l2tp', name: 'L2TP/IPsec', icon: '🔗', defaultPort: 1701 },
  { key: 'ikev2', name: 'IKEv2', icon: '🔑', defaultPort: 500 },
  { key: 'ssh', name: 'SSH Tunnel', icon: '💻', defaultPort: 2222 },
] as const

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const nodes = ref<NodeWithConfig[]>([])
const loading = ref(true)

// Side panel state
const panelOpen = ref(false)
const panelNodeId = ref<number | null>(null)
const panelProtocol = ref<string>('')
const panelSaving = ref(false)

// Protocol settings form (reactive object for v-model binding)
const panelForm = reactive<Record<string, any>>({})

// ═══════════════════════════════════════════════════════════════════════════════
// Protocol settings schemas
// ═══════════════════════════════════════════════════════════════════════════════

interface FieldDef {
  key: string
  label: string
  type: 'number' | 'text' | 'password' | 'select'
  default?: any
  options?: { label: string; value: string }[]
  showIf?: (form: Record<string, any>) => boolean
}

const protocolFields: Record<string, FieldDef[]> = {
  openvpn: [
    { key: 'port', label: 'Port', type: 'number', default: 1194 },
    { key: 'transport', label: 'Transport', type: 'select', default: 'udp', options: [
      { label: 'UDP', value: 'udp' }, { label: 'TCP', value: 'tcp' },
    ]},
    { key: 'cipher', label: 'Cipher', type: 'select', default: 'AES-256-GCM', options: [
      { label: 'AES-256-GCM', value: 'AES-256-GCM' },
      { label: 'AES-128-GCM', value: 'AES-128-GCM' },
      { label: 'CHACHA20-POLY1305', value: 'CHACHA20-POLY1305' },
    ]},
    { key: 'tls_mode', label: 'TLS Mode', type: 'select', default: 'tls-crypt', options: [
      { label: 'tls-crypt', value: 'tls-crypt' },
      { label: 'tls-auth', value: 'tls-auth' },
      { label: 'None', value: 'none' },
    ]},
    { key: 'topology', label: 'Topology', type: 'select', default: 'subnet', options: [
      { label: 'subnet', value: 'subnet' },
      { label: 'net30', value: 'net30' },
      { label: 'p2p', value: 'p2p' },
    ]},
    { key: 'dns1', label: 'DNS 1', type: 'text', default: '8.8.8.8' },
    { key: 'dns2', label: 'DNS 2', type: 'text', default: '8.8.4.4' },
    { key: 'network', label: 'Network (CIDR)', type: 'text', default: '10.8.0.0/20' },
    { key: 'mtu', label: 'MTU', type: 'number', default: 1500 },
    { key: 'fragment', label: 'Fragment', type: 'number', default: '' },
    { key: 'push_routes', label: 'Push Routes', type: 'text', default: '' },
  ],
  wireguard: [
    { key: 'port', label: 'Port', type: 'number', default: 51820 },
    { key: 'network', label: 'Network (CIDR)', type: 'text', default: '10.66.0.0/20' },
    { key: 'dns1', label: 'DNS 1', type: 'text', default: '1.1.1.1' },
    { key: 'dns2', label: 'DNS 2', type: 'text', default: '8.8.8.8' },
    { key: 'mtu', label: 'MTU', type: 'number', default: 1420 },
  ],
  l2tp: [
    { key: 'port', label: 'Port', type: 'number', default: 1701 },
    { key: 'ipsec_mode', label: 'IPsec Mode', type: 'select', default: 'ipsec', options: [
      { label: 'IPsec', value: 'ipsec' }, { label: 'Plain', value: 'plain' },
    ]},
    { key: 'psk', label: 'PSK', type: 'password', default: '' },
    { key: 'auth_method', label: 'Auth Method', type: 'select', default: 'MS-CHAPv2', options: [
      { label: 'CHAP', value: 'CHAP' },
      { label: 'PAP', value: 'PAP' },
      { label: 'MS-CHAPv2', value: 'MS-CHAPv2' },
    ]},
    { key: 'dns1', label: 'DNS 1', type: 'text', default: '8.8.8.8' },
    { key: 'dns2', label: 'DNS 2', type: 'text', default: '8.8.4.4' },
    { key: 'network', label: 'Network (CIDR)', type: 'text', default: '10.9.0.0/20' },
  ],
  ikev2: [
    { key: 'port', label: 'Port', type: 'number', default: 500 },
    { key: 'auth_type', label: 'Auth Type', type: 'select', default: 'PSK', options: [
      { label: 'PSK', value: 'PSK' }, { label: 'Certificate', value: 'Certificate' },
    ]},
    { key: 'psk', label: 'PSK', type: 'password', default: '', showIf: (f) => f.auth_type === 'PSK' },
    { key: 'dns1', label: 'DNS 1', type: 'text', default: '8.8.8.8' },
    { key: 'dns2', label: 'DNS 2', type: 'text', default: '8.8.4.4' },
    { key: 'network', label: 'Network (CIDR)', type: 'text', default: '10.10.0.0/20' },
    { key: 'domain', label: 'Domain', type: 'text', default: '' },
  ],
  ssh: [
    { key: 'port', label: 'Port', type: 'number', default: 2222 },
    { key: 'listen_address', label: 'Listen Address', type: 'text', default: '0.0.0.0' },
    { key: 'max_sessions', label: 'Max Sessions', type: 'number', default: 10 },
    { key: 'key_type', label: 'Key Type', type: 'select', default: 'ed25519', options: [
      { label: 'ed25519', value: 'ed25519' }, { label: 'RSA', value: 'rsa' },
    ]},
    { key: 'idle_timeout', label: 'Idle Timeout (s)', type: 'number', default: 0 },
  ],
}

// ═══════════════════════════════════════════════════════════════════════════════
// Computed
// ═══════════════════════════════════════════════════════════════════════════════

const panelTitle = computed(() => {
  const proto = protocols.find(p => p.key === panelProtocol.value)
  const node = nodes.value.find(n => n.node.id === panelNodeId.value)
  return proto && node ? `${proto.name} — ${node.node.name}` : ''
})

const currentFields = computed<FieldDef[]>(() => {
  return protocolFields[panelProtocol.value] || []
})

// ═══════════════════════════════════════════════════════════════════════════════
// Data fetching
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(async () => {
  await fetchNodes()
})

async function fetchNodes() {
  loading.value = true
  try {
    const res = await get('/api/admin/knode/nodes')
    const nodeList: Node[] = res.nodes || []
    nodes.value = nodeList.map(node => ({
      node,
      config: {},
      loading: true,
      hovered: false,
    }))
    await Promise.allSettled(
      nodes.value.map((entry, idx) => fetchVpnConfig(entry.node.id, idx))
    )
  } catch {
    // error toast handled by useApi
  } finally {
    loading.value = false
  }
}

async function fetchVpnConfig(nodeId: number, idx: number) {
  try {
    const res = await get(`/api/nodes/vpn-config/${nodeId}`)
    const configMap: VpnConfig = {}
    const configs = res.configs || []
    for (const c of configs) {
      configMap[c.protocol] = {
        enabled: c.enabled,
        port: c.port,
        network: c.network || '',
        extra_json: c.extra_json || {},
      }
    }
    nodes.value[idx].config = configMap
  } catch {
    // Node might be offline
  } finally {
    nodes.value[idx].loading = false
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Toggle protocol
// ═══════════════════════════════════════════════════════════════════════════════

async function toggleProtocol(nodeId: number, idx: number, protocolKey: string, event: Event) {
  event.stopPropagation()
  const entry = nodes.value[idx]
  const current = entry.config[protocolKey]
  const newEnabled = !(current?.enabled ?? false)

  if (!entry.config[protocolKey]) {
    const proto = protocols.find(p => p.key === protocolKey)
    entry.config[protocolKey] = {
      enabled: newEnabled,
      port: proto?.defaultPort || 0,
      network: '',
    }
  } else {
    entry.config[protocolKey].enabled = newEnabled
  }

  try {
    await post(`/api/nodes/vpn-config/${nodeId}`, {
      protocol: protocolKey,
      enabled: newEnabled,
    })
    toast.success(`${protocolKey} ${newEnabled ? 'enabled' : 'disabled'}`)
  } catch {
    entry.config[protocolKey].enabled = !newEnabled
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Side panel
// ═══════════════════════════════════════════════════════════════════════════════

function openProtocolPanel(nodeId: number, protocolKey: string) {
  panelNodeId.value = nodeId
  panelProtocol.value = protocolKey
  panelOpen.value = true

  // Populate form from existing config
  const entry = nodes.value.find(n => n.node.id === nodeId)
  const config = entry?.config[protocolKey]
  const fields = protocolFields[protocolKey] || []

  // Reset form
  Object.keys(panelForm).forEach(k => delete panelForm[k])

  for (const field of fields) {
    if (field.key === 'port') {
      panelForm[field.key] = config?.port ?? field.default
    } else if (field.key === 'network') {
      panelForm[field.key] = config?.network ?? field.default
    } else {
      panelForm[field.key] = config?.extra_json?.[field.key] ?? field.default
    }
  }
}

function closePanel() {
  panelOpen.value = false
}

async function saveProtocolSettings() {
  if (!panelNodeId.value || !panelProtocol.value) return
  panelSaving.value = true

  const fields = protocolFields[panelProtocol.value] || []
  const extra_json: Record<string, any> = {}
  let port = 0
  let network = ''

  for (const field of fields) {
    const val = panelForm[field.key]
    if (field.key === 'port') {
      port = Number(val) || 0
    } else if (field.key === 'network') {
      network = String(val || '')
    } else {
      extra_json[field.key] = val
    }
  }

  try {
    await post(`/api/nodes/vpn-config/${panelNodeId.value}`, {
      protocol: panelProtocol.value,
      port,
      enabled: true,
      network,
      extra_json,
    })
    toast.success('Settings saved')

    // Update local state
    const idx = nodes.value.findIndex(n => n.node.id === panelNodeId.value)
    if (idx >= 0) {
      nodes.value[idx].config[panelProtocol.value] = {
        enabled: true,
        port,
        network,
        extra_json,
      }
    }
    closePanel()
  } catch {
    // error handled by useApi
  } finally {
    panelSaving.value = false
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function getProtoConfig(config: VpnConfig, key: string): ProtocolConfig | null {
  return config[key] || null
}

function coreCountSummary(config: VpnConfig): string {
  const enabled = protocols.filter(p => config[p.key]?.enabled).length
  return `${enabled}/${protocols.length} protocols`
}

function isNodeOnline(node: Node): boolean {
  return node.status !== 'offline'
}
</script>

<template>
  <div class="page services-view">
    <header class="page-header">
      <h1>Services</h1>
      <p class="subtitle">Manage VPN protocols across all nodes</p>
    </header>

    <!-- Loading state -->
    <div v-if="loading" class="loading-grid">
      <KSkeleton v-for="i in 3" :key="i" height="72px" />
    </div>

    <!-- Empty state -->
    <div v-else-if="nodes.length === 0" class="empty-state">
      <p>No nodes found. Add nodes to manage VPN protocols.</p>
    </div>

    <!-- Node list -->
    <div v-else class="nodes-list">
      <div
        v-for="(entry, idx) in nodes"
        :key="entry.node.id"
        class="node-row"
        :class="{ expanded: entry.hovered }"
        @mouseenter="entry.hovered = true"
        @mouseleave="entry.hovered = false"
      >
        <!-- Node summary row -->
        <div class="node-summary">
          <div class="node-info">
            <span class="node-name">{{ entry.node.name }}</span>
            <span class="node-ip">{{ entry.node.address }}</span>
          </div>
          <div class="node-meta">
            <KStatusPill
              :status="isNodeOnline(entry.node) ? 'active' : 'inactive'"
              :label="isNodeOnline(entry.node) ? 'Online' : 'Offline'"
            />
            <span class="core-count">{{ coreCountSummary(entry.config) }}</span>
          </div>
        </div>

        <!-- Expandable protocol cards area -->
        <Transition name="expand">
          <div v-if="entry.hovered" class="protocols-area">
            <div v-if="entry.loading" class="protocol-cards">
              <KSkeleton v-for="i in 5" :key="i" height="56px" />
            </div>
            <div v-else class="protocol-cards">
              <div
                v-for="proto in protocols"
                :key="proto.key"
                class="protocol-card"
                :class="{ enabled: getProtoConfig(entry.config, proto.key)?.enabled }"
                @click="openProtocolPanel(entry.node.id, proto.key)"
              >
                <span class="proto-icon">{{ proto.icon }}</span>
                <div class="proto-details">
                  <span class="proto-name">{{ proto.name }}</span>
                  <span class="proto-port">
                    :{{ getProtoConfig(entry.config, proto.key)?.port || proto.defaultPort }}
                  </span>
                </div>
                <KStatusPill
                  :status="getProtoConfig(entry.config, proto.key)?.enabled ? 'active' : 'inactive'"
                  :label="getProtoConfig(entry.config, proto.key)?.enabled ? 'Enabled' : 'Disabled'"
                />
                <button
                  class="toggle-btn"
                  :class="{ active: getProtoConfig(entry.config, proto.key)?.enabled }"
                  :title="getProtoConfig(entry.config, proto.key)?.enabled ? 'Disable' : 'Enable'"
                  @click="toggleProtocol(entry.node.id, idx, proto.key, $event)"
                >
                  <span class="toggle-track">
                    <span class="toggle-thumb" />
                  </span>
                </button>
              </div>
            </div>
          </div>
        </Transition>
      </div>
    </div>

    <!-- Protocol settings side panel -->
    <KSlideOver
      :open="panelOpen"
      :title="panelTitle"
      width="400px"
      @close="closePanel"
    >
      <div class="panel-body">
        <template v-for="field in currentFields" :key="field.key">
          <KFormField
            v-if="!field.showIf || field.showIf(panelForm)"
            :label="field.label"
            :name="field.key"
          >
            <KSelect
              v-if="field.type === 'select'"
              v-model="panelForm[field.key]"
              :options="field.options || []"
            />
            <KInput
              v-else
              v-model="panelForm[field.key]"
              :type="field.type"
              :placeholder="String(field.default || '')"
            />
          </KFormField>
        </template>
      </div>

      <template #footer>
        <div class="panel-footer">
          <KButton variant="ghost" @click="closePanel">Cancel</KButton>
          <KButton :loading="panelSaving" @click="saveProtocolSettings">Save</KButton>
        </div>
      </template>
    </KSlideOver>
  </div>
</template>

<style scoped>
.services-view {
  padding: var(--space-6, 24px);
  max-width: 1200px;
}

.page-header {
  margin-bottom: var(--space-6, 24px);
}

.page-header h1 {
  font-size: var(--text-2xl, 24px);
  font-weight: var(--font-bold, 700);
  margin: 0;
}

.page-header .subtitle {
  color: var(--color-muted, #8b98a5);
  font-size: var(--text-sm, 13px);
  margin: var(--space-1, 4px) 0 0;
}

.loading-grid {
  display: grid;
  gap: var(--space-3, 12px);
}

.empty-state {
  text-align: center;
  padding: var(--space-10, 60px) var(--space-4, 16px);
  color: var(--color-muted, #8b98a5);
}

/* ═══════════════════════════════════════════════════════════════════
   Node list
   ═══════════════════════════════════════════════════════════════════ */

.nodes-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3, 12px);
}

.node-row {
  background: var(--color-surface, #161b22);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-lg, 12px);
  overflow: hidden;
  transition: box-shadow 0.2s ease;
}

.node-row:hover {
  box-shadow: 0 2px 12px rgba(0, 0, 0, 0.2);
}

.node-summary {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4, 16px) var(--space-5, 20px);
  cursor: default;
}

.node-info {
  display: flex;
  align-items: baseline;
  gap: var(--space-3, 12px);
}

.node-name {
  font-size: var(--text-base, 14px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text, #e6edf3);
}

.node-ip {
  font-size: var(--text-sm, 13px);
  color: var(--color-muted, #8b98a5);
  font-family: var(--font-mono, monospace);
}

.node-meta {
  display: flex;
  align-items: center;
  gap: var(--space-4, 16px);
}

.core-count {
  font-size: var(--text-sm, 13px);
  color: var(--color-muted, #8b98a5);
}

/* ═══════════════════════════════════════════════════════════════════
   Expandable protocols area
   ═══════════════════════════════════════════════════════════════════ */

.protocols-area {
  padding: 0 var(--space-5, 20px) var(--space-4, 16px);
  background: var(--color-surface-2, #1e2630);
  border-top: 1px solid var(--color-border, #28333f);
}

.protocol-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-3, 12px);
  padding-top: var(--space-4, 16px);
}

.protocol-card {
  display: flex;
  align-items: center;
  gap: var(--space-3, 12px);
  padding: var(--space-3, 12px) var(--space-4, 16px);
  background: var(--color-surface, #161b22);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 8px);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.protocol-card:hover {
  border-color: var(--color-primary, #2563eb);
  background: rgba(37, 99, 235, 0.04);
}

.protocol-card.enabled {
  border-color: var(--color-success, #22c55e);
}

.proto-icon {
  font-size: 20px;
  flex-shrink: 0;
}

.proto-details {
  flex: 1;
  min-width: 0;
}

.proto-name {
  font-size: var(--text-sm, 13px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text, #e6edf3);
  display: block;
}

.proto-port {
  font-size: var(--text-xs, 11px);
  color: var(--color-muted, #8b98a5);
  font-family: var(--font-mono, monospace);
}

/* ═══════════════════════════════════════════════════════════════════
   Toggle button
   ═══════════════════════════════════════════════════════════════════ */

.toggle-btn {
  flex-shrink: 0;
  background: none;
  border: none;
  cursor: pointer;
  padding: 0;
}

.toggle-track {
  display: block;
  width: 36px;
  height: 20px;
  border-radius: 10px;
  background: var(--color-border, #28333f);
  position: relative;
  transition: background 0.15s;
}

.toggle-btn.active .toggle-track {
  background: var(--color-success, #22c55e);
}

.toggle-thumb {
  display: block;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: #fff;
  position: absolute;
  top: 2px;
  left: 2px;
  transition: transform 0.15s;
}

.toggle-btn.active .toggle-thumb {
  transform: translateX(16px);
}

/* ═══════════════════════════════════════════════════════════════════
   Expand transition
   ═══════════════════════════════════════════════════════════════════ */

.expand-enter-active,
.expand-leave-active {
  transition: all 300ms ease;
  overflow: hidden;
  max-height: 300px;
  opacity: 1;
}

.expand-enter-from,
.expand-leave-to {
  max-height: 0;
  opacity: 0;
  padding-top: 0;
  padding-bottom: 0;
}

/* ═══════════════════════════════════════════════════════════════════
   Side panel body
   ═══════════════════════════════════════════════════════════════════ */

.panel-body {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
  padding: var(--space-4, 16px);
}

.panel-footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3, 12px);
}
</style>
