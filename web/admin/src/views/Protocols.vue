<script setup lang="ts">
import { ref, reactive, computed, onMounted, watch } from 'vue'
import { useApi, getCsrfToken } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import { useConfirm } from '@koris/composables/useConfirm'
import Skeleton from '@koris/ui/Skeleton.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import FormField from '@koris/ui/FormField.vue'
import SlideOver from '@koris/ui/SlideOver.vue'
import EmptyState from '@koris/ui/EmptyState.vue'

const { get, post } = useApi()
const { confirm } = useConfirm()
const toast = useToast()
const { t } = useI18n()

interface Node { id: number; name: string; address: string; status?: string }
interface ProtocolConfig { enabled: boolean; port: number; network: string; extra_json?: Record<string, any> }
type VpnConfig = Record<string, ProtocolConfig>
interface ProtoDef { key: string; name: string; icon: string; defaultPort: number; defaultNetwork: string }
const protocols: ProtoDef[] = [
  { key: 'openvpn', name: 'OpenVPN', icon: '\u{1F510}', defaultPort: 1194, defaultNetwork: '10.8.0.0/20' },
  { key: 'wireguard', name: 'WireGuard', icon: '\u{1F511}', defaultPort: 51820, defaultNetwork: '10.66.0.0/20' },
  { key: 'l2tp', name: 'L2TP/IPsec', icon: '\u{1F6E1}', defaultPort: 1701, defaultNetwork: '10.9.0.0/20' },
  { key: 'ikev2', name: 'IKEv2', icon: '\u{1F512}', defaultPort: 500, defaultNetwork: '10.10.0.0/20' },
  { key: 'ssh', name: 'SSH Tunnel', icon: '\u{1F4BB}', defaultPort: 2222, defaultNetwork: '' },
]

interface FieldDef {
  key: string
  label: string
  type: 'number' | 'text' | 'password' | 'select' | 'toggle' | 'dns'
  default?: any
  options?: { label: string; value: string }[]
  showIf?: (form: Record<string, any>) => boolean
  tooltip?: string
}
const protocolFields: Record<string, FieldDef[]> = {
  openvpn: [
    { key: 'port', label: 'services.port', type: 'number', default: 1194 },
    { key: 'transport', label: 'services.transport', type: 'select', default: 'udp', options: [{ label: 'UDP', value: 'udp' }, { label: 'TCP', value: 'tcp' }] },
    { key: 'auth_mode', label: 'services.auth_mode', type: 'select', default: 'hybrid', options: [{ label: 'Hybrid (cert + password per user)', value: 'hybrid' }, { label: 'Username/Password only', value: 'userpass' }, { label: 'Certificate only', value: 'certificate' }] },
    { key: 'cipher', label: 'services.cipher', type: 'select', default: 'AES-256-GCM', options: [{ label: 'AES-256-GCM', value: 'AES-256-GCM' }, { label: 'AES-128-GCM', value: 'AES-128-GCM' }, { label: 'CHACHA20-POLY1305', value: 'CHACHA20-POLY1305' }] },
    { key: 'tls_mode', label: 'services.tls_mode', type: 'select', default: 'tls-crypt', options: [{ label: 'tls-crypt (most secure)', value: 'tls-crypt' }, { label: 'tls-auth (compatible)', value: 'tls-auth' }, { label: 'None (no protection)', value: 'none' }] },
    { key: 'backup_domain', label: 'services.backup_domain', type: 'text', default: '', tooltip: 'Alternate domain when primary IP is blocked (optional)' },
    { key: 'dns', label: 'services.dns_label', type: 'dns', default: '8.8.8.8' },
    { key: 'mtu', label: 'services.mtu', type: 'number', default: 1500 },
  ],
  wireguard: [
    { key: 'port', label: 'services.port', type: 'number', default: 51820 },
    { key: 'backup_domain', label: 'services.backup_domain', type: 'text', default: '', tooltip: 'Alternate domain when primary IP is blocked (optional)' },
    { key: 'dns', label: 'services.dns_label', type: 'dns', default: '1.1.1.1' },
    { key: 'gaming_optimize', label: 'services.gaming_optimize', type: 'toggle', default: false, tooltip: 'services.gaming_desc' },
  ],
  l2tp: [
    { key: 'port', label: 'services.port', type: 'number', default: 1701 },
    { key: 'psk', label: 'services.psk', type: 'text', default: '' },
    { key: 'backup_domain', label: 'services.backup_domain', type: 'text', default: '', tooltip: 'Alternate domain when primary IP is blocked (optional)' },
    { key: 'dns', label: 'services.dns_label', type: 'dns', default: '8.8.8.8' },
    { key: 'simple_mode', label: 'services.simple_mode', type: 'toggle', default: true },
    { key: 'auth_method', label: 'nodes.auth_method', type: 'select', default: 'MS-CHAPv2', showIf: (f) => !f.simple_mode, options: [{ label: 'CHAP', value: 'CHAP' }, { label: 'PAP', value: 'PAP' }, { label: 'MS-CHAPv2', value: 'MS-CHAPv2' }] },
    { key: 'dpd_interval', label: 'nodes.dpd_interval', type: 'number', default: 30, showIf: (f) => !f.simple_mode },
    { key: 'dpd_timeout', label: 'nodes.dpd_timeout', type: 'number', default: 120, showIf: (f) => !f.simple_mode },
  ],
  ikev2: [
    { key: 'port', label: 'services.port', type: 'number', default: 500 },
    { key: 'psk', label: 'services.psk', type: 'text', default: '' },
    { key: 'backup_domain', label: 'services.backup_domain', type: 'text', default: '', tooltip: 'Alternate domain when primary IP is blocked (optional)' },
    { key: 'dns', label: 'services.dns_label', type: 'dns', default: '8.8.8.8' },
    { key: 'domain', label: 'services.domain', type: 'text', default: '' },
    { key: 'cert_source', label: 'services.tls_mode', type: 'select', default: 'letsencrypt', options: [{ label: "Let's Encrypt (auto)", value: 'letsencrypt' }, { label: 'Custom Certificate', value: 'custom' }] },
  ],
  ssh: [
    { key: 'port', label: 'services.port', type: 'number', default: 2222 },
    { key: 'max_sessions', label: 'services.max_connections', type: 'number', default: 10 },
    { key: 'key_type', label: 'services.key_type', type: 'select', default: 'ed25519', options: [{ label: 'ed25519 (recommended)', value: 'ed25519' }, { label: 'RSA', value: 'rsa' }] },
  ],
}
const dnsPresets = [
  { label: 'Google', value: '8.8.8.8' }, { label: 'Cloudflare', value: '1.1.1.1' },
  { label: 'Quad9', value: '9.9.9.9' }, { label: 'OpenDNS', value: '208.67.222.222' }, { label: 'AdGuard', value: '94.140.14.14' },
]

const nodes = ref<Node[]>([])
const selectedNodeId = ref<number | null>(null)
const config = ref<VpnConfig>({})
const loadingNodes = ref(true)
const loadingConfig = ref(false)
const panelOpen = ref(false)
const panelProtocol = ref<string>('')
const panelSaving = ref(false)
const showDnsDropdown = ref(false)
const panelForm = reactive<Record<string, any>>({})

const selectedNode = computed(() => nodes.value.find(n => n.id === selectedNodeId.value) || null)
const nodeOptions = computed(() => nodes.value.map(n => ({ label: n.name, value: n.id })))
const enabledCount = computed(() => protocols.filter(p => config.value[p.key]?.enabled).length)
const panelTitle = computed(() => {
  const proto = protocols.find(p => p.key === panelProtocol.value)
  return proto && selectedNode.value ? `${proto.name} — ${selectedNode.value.name}` : ''
})
const currentFields = computed<FieldDef[]>(() => protocolFields[panelProtocol.value] || [])

async function fetchNodes() {
  loadingNodes.value = true
  try {
    const res = await get<{ nodes: Node[] }>('/api/admin/knode/nodes')
    nodes.value = res.nodes || []
    if (nodes.value.length && selectedNodeId.value == null) selectedNodeId.value = nodes.value[0].id
  } catch { /* handled by useApi */ } finally { loadingNodes.value = false }
}
async function fetchVpnConfig(nodeId: number) {
  loadingConfig.value = true
  try {
    const res = await get<{ configs: any[] }>(`/api/nodes/vpn-config/${nodeId}`)
    const map: VpnConfig = {}
    for (const c of res.configs || []) map[c.protocol] = { enabled: c.enabled, port: c.port, network: c.network || '', extra_json: c.extra_json || {} }
    config.value = map
  } catch { /* node offline */ } finally { loadingConfig.value = false }
}
watch(selectedNodeId, (id) => { if (id != null) { fetchVpnConfig(id); loadTeleProxies() } })
onMounted(fetchNodes)

function openProtocolPanel(protocolKey: string) {
  if (!selectedNodeId.value) return
  panelProtocol.value = protocolKey
  panelOpen.value = true
  showDnsDropdown.value = false
  const proto = protocols.find(p => p.key === protocolKey)!
  const existing = config.value[protocolKey]
  const fields = protocolFields[protocolKey] || []
  Object.keys(panelForm).forEach(k => delete panelForm[k])
  for (const field of fields) {
    if (field.key === 'port') panelForm[field.key] = existing?.port ?? field.default
    else if (field.key === 'network') panelForm[field.key] = existing?.network ?? field.default
    else panelForm[field.key] = existing?.extra_json?.[field.key] ?? field.default
  }
  panelForm.port = existing?.port ?? proto.defaultPort
  panelForm.network = existing?.network ?? proto.defaultNetwork
}
function closePanel() { panelOpen.value = false; showDnsDropdown.value = false }
function selectDns(dns: string) { panelForm.dns = dns; showDnsDropdown.value = false }
function generatePsk() {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  const arr = new Uint8Array(32); crypto.getRandomValues(arr)
  let r = ''; for (let i = 0; i < 32; i++) r += chars[arr[i] % chars.length]
  panelForm.psk = r
}
async function saveProtocolSettings() {
  if (!selectedNodeId.value || !panelProtocol.value) return
  panelSaving.value = true
  const fields = protocolFields[panelProtocol.value] || []
  const extra: Record<string, any> = {}
  let port = 0; let network = ''
  for (const field of fields) {
    const val = panelForm[field.key]
    if (field.key === 'port') port = Number(val) || 0
    else if (field.key === 'network') network = String(val || '')
    else extra[field.key] = val
  }
  const proto = protocols.find(p => p.key === panelProtocol.value)!
  if (panelProtocol.value === 'wireguard' && panelForm.gaming_optimize) { extra.persistent_keepalive = 15; extra.mtu = 1280 }
  try {
    await post(`/api/nodes/vpn-config/${selectedNodeId.value}`, { protocol: panelProtocol.value, enabled: true, port, network, extra_json: extra })
    toast.success(t('services.settings_saved'))
    config.value[panelProtocol.value] = { enabled: true, port, network, extra_json: extra }
    closePanel()
  } catch { /* handled by useApi */ } finally { panelSaving.value = false }
}
async function toggleProtocol(protocolKey: string) {
  if (!selectedNodeId.value) return
  const proto = protocols.find(p => p.key === protocolKey)!
  const current = config.value[protocolKey]
  const newEnabled = !(current?.enabled ?? false)
  if (!config.value[protocolKey]) config.value[protocolKey] = { enabled: newEnabled, port: proto.defaultPort, network: proto.defaultNetwork }
  else config.value[protocolKey].enabled = newEnabled
  try {
    await post(`/api/nodes/vpn-config/${selectedNodeId.value}`, {
      protocol: protocolKey, enabled: newEnabled,
      port: current?.port || proto.defaultPort, network: current?.network || proto.defaultNetwork,
    })
    toast.success(t(newEnabled ? 'services.enabled' : 'services.disabled').replace('{proto}', proto.name))
  } catch { config.value[protocolKey].enabled = !newEnabled }
}


// ─── Telegram Proxy (MTProto) — managed per node, part of Protocols ──────────
const teleProxies = ref<any[]>([])
const loadingTele = ref(false)
const showTeleForm = ref(false)
const creatingTele = ref(false)
const teleForm = ref<{ port: number; tag: string }>({ port: 443, tag: '' })
const nodeTeleProxies = computed(() => teleProxies.value.filter((p: any) => p.node_id === selectedNodeId.value))

async function loadTeleProxies() {
  if (!selectedNodeId.value) return
  loadingTele.value = true
  try {
    const res = await get<{ ok: boolean; proxies: any[] }>('/api/admin/telegram-proxies')
    teleProxies.value = res.proxies || []
  } catch { /* surfaced by useApi */ } finally { loadingTele.value = false }
}
async function createTeleProxy() {
  if (!selectedNodeId.value || !teleForm.value.port) { toast.error(t('teleproxy.field_port') + ' required'); return }
  creatingTele.value = true
  try {
    const res = await post<{ ok: boolean }>('/api/admin/telegram-proxies', { node_id: selectedNodeId.value, port: teleForm.value.port, tag: teleForm.value.tag })
    if (res.ok) { toast.success(t('teleproxy.created_success')); showTeleForm.value = false; teleForm.value = { port: 443, tag: '' }; await loadTeleProxies() }
  } catch { /* surfaced */ } finally { creatingTele.value = false }
}
async function startTeleProxy(p: any) {
  try { const res = await post<{ ok: boolean }>(`/api/admin/telegram-proxies/${p.id}/start`, {}); if (res.ok) { toast.success(t('teleproxy.start_success')); await loadTeleProxies() } } catch {}
}
async function stopTeleProxy(p: any) {
  try { const res = await post<{ ok: boolean }>(`/api/admin/telegram-proxies/${p.id}/stop`, {}); if (res.ok) { toast.success(t('teleproxy.stop_success')); await loadTeleProxies() } } catch {}
}
async function deleteTeleProxy(p: any) {
  const ok = await confirm({ title: t('teleproxy.confirm_delete_title'), message: t('teleproxy.confirm_delete_msg'), variant: 'danger' })
  if (!ok) return
  const token = getCsrfToken()
  try {
    const res = await fetch('/api/admin/telegram-proxies', { method: 'DELETE', headers: { 'Content-Type': 'application/json', ...(token ? { 'X-CSRF-Token': token } : {}) }, credentials: 'same-origin', body: JSON.stringify({ id: p.id }) })
    const data = await res.json()
    if (data.ok) { toast.success(t('teleproxy.deleted_success')); await loadTeleProxies() } else toast.error(data.error || 'Delete failed')
  } catch { toast.error('Delete failed') }
}
async function copyTeleLink(p: any) {
  const link = p.share_link || p.tg_link
  if (!link) return
  try { await navigator.clipboard.writeText(link); toast.success(t('teleproxy.link_copied')) } catch { toast.error('Copy failed') }
}
</script>

<template>
  <div class="page protocols-view">
    <header class="page-header">
      <div>
        <h1>Protocols</h1>
        <p class="subtitle">Manage VPN &amp; tunneling protocols across your nodes.</p>
      </div>
      <Select v-if="nodeOptions.length" :options="nodeOptions" v-model="selectedNodeId" placeholder="Select node" class="node-select" />
    </header>

    <div v-if="loadingNodes" class="loading-grid">
      <Skeleton v-for="i in 5" :key="i" height="104px" />
    </div>

    <EmptyState v-else-if="!nodes.length" title="No nodes yet" description="Add a node to start configuring protocols." />

    <template v-else>
      <div v-if="loadingConfig" class="loading-grid">
        <Skeleton v-for="i in 5" :key="i" height="104px" />
      </div>

      <div v-else class="protocol-grid">
        <div v-for="proto in protocols" :key="proto.key" class="protocol-card" :class="{ enabled: config[proto.key]?.enabled }">
          <div class="card-top">
            <span class="proto-icon">{{ proto.icon }}</span>
            <div class="proto-meta">
              <span class="proto-name">{{ proto.name }}</span>
              <StatusPill :status="config[proto.key]?.enabled ? 'active' : 'inactive'" :label="config[proto.key]?.enabled ? 'Enabled' : 'Disabled'" />
            </div>
            <button class="toggle-btn" :class="{ active: config[proto.key]?.enabled }" :title="config[proto.key]?.enabled ? 'Disable' : 'Enable'" @click="toggleProtocol(proto.key)">
              <span class="toggle-track"><span class="toggle-thumb" /></span>
            </button>
          </div>
          <div class="proto-detail">
            <div class="detail-row"><span>Port</span><code>{{ config[proto.key]?.port || proto.defaultPort }}</code></div>
            <div class="detail-row" v-if="proto.defaultNetwork || config[proto.key]?.network">
              <span>Network</span><code>{{ config[proto.key]?.network || proto.defaultNetwork || '—' }}</code>
            </div>
          </div>
          <Button variant="ghost" size="sm" class="edit-btn" @click="openProtocolPanel(proto.key)">Configure</Button>
        </div>
      </div>
    </template>



    <!-- Telegram Proxy (MTProto) — part of Protocols, managed per node -->
    <section class="tele-section">
      <div class="tele-head">
        <div>
          <h3>Telegram Proxy (MTProto)</h3>
          <p class="muted">Per-node MTProto proxy. Each customer gets their own secret/token with limits (managed per user).</p>
        </div>
        <Button variant="primary" size="sm" @click="showTeleForm = !showTeleForm">{{ t('teleproxy.add_proxy') }}</Button>
      </div>

      <div v-if="showTeleForm" class="tele-form card">
        <Input v-model.number="teleForm.port" type="number" :placeholder="t('teleproxy.field_port')" />
        <Input v-model="teleForm.tag" :placeholder="t('teleproxy.tag_placeholder')" />
        <div class="form-actions-row">
          <Button variant="primary" size="sm" :loading="creatingTele" @click="createTeleProxy">{{ t('teleproxy.add_proxy') }}</Button>
          <Button variant="ghost" size="sm" @click="showTeleForm = false">Cancel</Button>
        </div>
      </div>

      <div v-if="loadingTele" class="skeleton-wrap">
        <Skeleton v-for="i in 2" :key="i" variant="rect" width="100%" :height="56" />
      </div>
      <div v-else-if="!nodeTeleProxies.length" class="muted tele-empty">No Telegram proxies on this node yet.</div>
      <div v-else class="tele-list">
        <div v-for="p in nodeTeleProxies" :key="p.id" class="tele-row card">
          <div class="tele-row-main">
            <StatusPill :status="p.status" size="sm" />
            <span class="tele-port">:{{ p.port }}</span>
            <span v-if="p.tag" class="tele-tag">{{ p.tag }}</span>
          </div>
          <div class="row-actions">
            <Button v-if="p.status !== 'active'" variant="ghost" size="sm" @click="startTeleProxy(p)">{{ t('teleproxy.start') }}</Button>
            <Button v-else variant="ghost" size="sm" @click="stopTeleProxy(p)">{{ t('teleproxy.stop') }}</Button>
            <Button variant="ghost" size="sm" @click="copyTeleLink(p)">{{ t('teleproxy.copy_link') }}</Button>
            <Button variant="danger" size="sm" @click="deleteTeleProxy(p)">X</Button>
          </div>
        </div>
      </div>
    </section>

    <SlideOver :open="panelOpen" :title="panelTitle" width="440px" @close="closePanel">
      <div class="panel-body">
        <template v-for="field in currentFields" :key="field.key">
          <div v-if="!field.showIf || field.showIf(panelForm)" class="panel-field">
            <FormField :label="t(field.label)" :name="field.key">
              <Select v-if="field.type === 'select'" v-model="panelForm[field.key]" :options="field.options || []" />
              <div v-else-if="field.type === 'toggle'" class="toggle-field">
                <button class="toggle-btn" :class="{ active: panelForm[field.key] }" @click="panelForm[field.key] = !panelForm[field.key]">
                  <span class="toggle-track"><span class="toggle-thumb" /></span>
                </button>
                <span v-if="field.tooltip" class="field-hint">{{ t(field.tooltip) }}</span>
              </div>
              <div v-else-if="field.type === 'dns'" class="dns-field">
                <div class="dns-input-wrap">
                  <Input v-model="panelForm[field.key]" type="text" placeholder="8.8.8.8" />
                  <button type="button" class="dns-dropdown-btn" @click.stop="showDnsDropdown = !showDnsDropdown">&#9662;</button>
                </div>
                <Transition name="fade">
                  <div v-if="showDnsDropdown" class="dns-dropdown">
                    <button v-for="dns in dnsPresets" :key="dns.value" class="dns-option" @click="selectDns(dns.value)">
                      <span class="dns-label">{{ dns.label }}</span><span class="dns-value">{{ dns.value }}</span>
                    </button>
                  </div>
                </Transition>
              </div>
              <div v-else-if="field.type === 'password' || field.key === 'psk'" class="password-field">
                <Input v-model="panelForm[field.key]" type="text" autocomplete="off" :placeholder="t(field.label)" />
                <Button size="sm" variant="ghost" @click="generatePsk">{{ t('services.auto_generate') }}</Button>
              </div>
              <Input v-else v-model="panelForm[field.key]" :type="field.type" :placeholder="String(field.default || '')" />
            </FormField>
          </div>
        </template>
      </div>
      <template #footer>
        <div class="panel-footer">
          <Button variant="ghost" @click="closePanel">Cancel</Button>
          <Button :loading="panelSaving" @click="saveProtocolSettings">{{ t('nodes.save_config') }}</Button>
        </div>
      </template>
    </SlideOver>
  </div>
</template>

<style scoped>
.protocols-view { padding: var(--space-6, 24px); max-width: 1200px; margin: 0 auto; }
.page-header { display: flex; align-items: center; justify-content: space-between; gap: 16px; margin-bottom: 24px; flex-wrap: wrap; }
.page-header h1 { font-size: var(--text-2xl, 24px); font-weight: var(--font-bold, 700); margin: 0; }
.subtitle { color: var(--color-muted, #8b98a5); margin: 6px 0 0; font-size: var(--text-sm, 13px); }
.node-select { min-width: 220px; max-width: 320px; flex: 1; }
.loading-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 16px; }
.protocol-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(260px, 1fr)); gap: 16px; }
.protocol-card { background: var(--color-surface); border: 1px solid var(--color-border); border-left: 3px solid var(--color-border); border-radius: var(--radius-lg, 12px); padding: 16px; display: flex; flex-direction: column; gap: 12px; transition: border-color .15s, box-shadow .15s, transform .15s; }
.protocol-card.enabled { border-left-color: var(--color-success, #22c55e); }
.protocol-card:hover { box-shadow: var(--shadow-md, 0 12px 28px rgba(0,0,0,.32)); transform: translateY(-2px); }
.card-top { display: flex; align-items: center; gap: 12px; }
.proto-icon { width: 40px; height: 40px; border-radius: 10px; display: grid; place-items: center; font-size: 20px; background: var(--color-surface-2, #1e2630); flex-shrink: 0; }
.proto-meta { display: flex; flex-direction: column; gap: 3px; min-width: 0; flex: 1; }
.proto-name { font-weight: 600; color: var(--color-text); }
.proto-detail { display: flex; flex-direction: column; gap: 6px; font-size: var(--text-sm, 13px); }
.detail-row { display: flex; justify-content: space-between; gap: 8px; color: var(--color-muted, #8b98a5); }
.detail-row code { color: var(--color-text); font-family: var(--font-mono, monospace); }
.edit-btn { align-self: flex-start; }
.toggle-btn { width: 44px; height: 24px; border-radius: 999px; border: 1px solid var(--color-border, #28333f); background: var(--color-surface-2, #1e2630); position: relative; cursor: pointer; flex-shrink: 0; transition: background .15s, border-color .15s; }
.toggle-btn.active { background: var(--gradient-brand, linear-gradient(135deg, var(--color-primary), var(--color-brand-2))); border-color: transparent; }
.toggle-track { position: absolute; inset: 0; }
.toggle-thumb { position: absolute; top: 2px; left: 2px; width: 18px; height: 18px; border-radius: 50%; background: #fff; transition: transform .15s; }
.toggle-btn.active .toggle-thumb { transform: translateX(20px); }
.panel-field { margin-bottom: 14px; }
.toggle-field { display: flex; align-items: center; gap: 10px; }
.field-hint { font-size: var(--text-xs, 11px); color: var(--color-muted, #8b98a5); }
.dns-input-wrap { display: flex; gap: 6px; }
.dns-dropdown-btn { width: 38px; border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-surface-2); color: var(--color-muted); cursor: pointer; }
.dns-dropdown { margin-top: 6px; border: 1px solid var(--color-border); border-radius: var(--radius-md); overflow: hidden; background: var(--color-surface); }
.dns-option { display: flex; justify-content: space-between; width: 100%; padding: 8px 12px; background: none; border: none; cursor: pointer; color: var(--color-text); font-size: var(--text-sm); }
.dns-option:hover { background: var(--color-surface-2); }
.dns-value { color: var(--color-muted); font-family: var(--font-mono, monospace); }
.password-field { display: flex; gap: 6px; }
@media (max-width: 640px) {
  .protocol-grid { grid-template-columns: 1fr; }
  .node-select { max-width: none; width: 100%; }
  .page-header { flex-direction: column; align-items: stretch; }
}
</style>

