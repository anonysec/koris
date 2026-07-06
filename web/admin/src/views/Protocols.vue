<script setup lang="ts">
import { ref, reactive, computed, watch, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import PageHeader from '@koris/ui/PageHeader.vue'
import Button from '@koris/ui/Button.vue'
import Select from '@koris/ui/Select.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import SlideOver from '@koris/ui/SlideOver.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'

const { t } = useI18n()
const { get, post } = useApi()
const toast = useToast()

interface Node { id: number; name: string; status?: string }
interface ProtoDef { key: string; name: string; icon: string; blurb: string; defaultPort: number; defaultNetwork: string }
interface FieldDef {
  key: string
  label: string
  type: 'number' | 'text' | 'password' | 'select' | 'toggle' | 'dns'
  default: any
  options?: { label: string; value: string }[]
  tooltip?: string
  showIf?: (f: Record<string, any>) => boolean
}
interface ProtoState { enabled: boolean; port: number; network: string; extra_json?: Record<string, any> }
type VpnConfig = Record<string, ProtoState>

const protocols: ProtoDef[] = [
  { key: 'openvpn',   name: 'OpenVPN',    icon: '\u{1F510}', blurb: 'Battle-tested SSL VPN, works everywhere.', defaultPort: 1194,  defaultNetwork: '10.8.0.0/20' },
  { key: 'wireguard', name: 'WireGuard',  icon: '\u{1F511}', blurb: 'Modern, fast, minimal overhead.',        defaultPort: 51820, defaultNetwork: '10.66.0.0/20' },
  { key: 'l2tp',      name: 'L2TP/IPsec', icon: '\u{1F6E1}\u{FE0F}', blurb: 'Native support on most devices.', defaultPort: 1701,  defaultNetwork: '10.9.0.0/20' },
  { key: 'ikev2',     name: 'IKEv2',      icon: '\u{1F512}', blurb: 'Stable on mobile, fast reconnects.',      defaultPort: 500,   defaultNetwork: '10.10.0.0/20' },
  { key: 'ssh',       name: 'SSH Tunnel', icon: '\u{1F4BB}', blurb: 'Lightweight tunneling over SSH.',         defaultPort: 2222,  defaultNetwork: '' },
  { key: 'mtproto',   name: 'MTProto',    icon: '\u{2708}\u{FE0F}', blurb: 'Telegram proxy (MTProto).',        defaultPort: 443,   defaultNetwork: '' },
]

const dnsPresets = [
  { label: 'Google', value: '8.8.8.8' }, { label: 'Cloudflare', value: '1.1.1.1' },
  { label: 'Quad9', value: '9.9.9.9' }, { label: 'OpenDNS', value: '208.67.222.222' }, { label: 'AdGuard', value: '94.140.14.14' },
]

const protocolFields: Record<string, FieldDef[]> = {
  openvpn: [
    { key: 'port', label: 'Port', type: 'number', default: 1194 },
    { key: 'network', label: 'Client subnet', type: 'text', default: '10.8.0.0/20' },
    { key: 'transport', label: 'Transport', type: 'select', default: 'udp', options: [{ label: 'UDP (faster)', value: 'udp' }, { label: 'TCP (compatible)', value: 'tcp' }] },
    { key: 'auth_mode', label: 'Authentication', type: 'select', default: 'hybrid', options: [{ label: 'Hybrid (cert + password)', value: 'hybrid' }, { label: 'Username / password', value: 'userpass' }, { label: 'Certificate only', value: 'certificate' }] },
    { key: 'cipher', label: 'Cipher', type: 'select', default: 'AES-256-GCM', options: [{ label: 'AES-256-GCM', value: 'AES-256-GCM' }, { label: 'AES-128-GCM', value: 'AES-128-GCM' }, { label: 'CHACHA20-POLY1305', value: 'CHACHA20-POLY1305' }] },
    { key: 'tls_mode', label: 'TLS protection', type: 'select', default: 'tls-crypt', options: [{ label: 'tls-crypt (most secure)', value: 'tls-crypt' }, { label: 'tls-auth (compatible)', value: 'tls-auth' }, { label: 'None', value: 'none' }] },
    { key: 'dns', label: 'DNS', type: 'dns', default: '8.8.8.8' },
    { key: 'mtu', label: 'MTU', type: 'number', default: 1500 },
    { key: 'backup_domain', label: 'Backup domain', type: 'text', default: '', tooltip: 'Alternate domain when the primary IP is blocked (optional).' },
  ],
  wireguard: [
    { key: 'port', label: 'Port', type: 'number', default: 51820 },
    { key: 'network', label: 'Client subnet', type: 'text', default: '10.66.0.0/20' },
    { key: 'dns', label: 'DNS', type: 'dns', default: '1.1.1.1' },
    { key: 'mtu', label: 'MTU', type: 'number', default: 1420 },
    { key: 'gaming_optimize', label: 'Gaming optimization', type: 'toggle', default: false, tooltip: 'Lower MTU + short keepalive for latency-sensitive traffic.' },
    { key: 'backup_domain', label: 'Backup domain', type: 'text', default: '', tooltip: 'Alternate domain when the primary IP is blocked (optional).' },
  ],
  l2tp: [
    { key: 'port', label: 'Port', type: 'number', default: 1701 },
    { key: 'network', label: 'Client subnet', type: 'text', default: '10.9.0.0/20' },
    { key: 'psk', label: 'Pre-shared key', type: 'text', default: '' },
    { key: 'dns', label: 'DNS', type: 'dns', default: '8.8.8.8' },
    { key: 'simple_mode', label: 'Simple mode', type: 'toggle', default: true, tooltip: 'Use sensible defaults; disable for advanced tuning.' },
    { key: 'auth_method', label: 'Auth method', type: 'select', default: 'MS-CHAPv2', showIf: (f) => !f.simple_mode, options: [{ label: 'CHAP', value: 'CHAP' }, { label: 'PAP', value: 'PAP' }, { label: 'MS-CHAPv2', value: 'MS-CHAPv2' }] },
    { key: 'dpd_interval', label: 'DPD interval (s)', type: 'number', default: 30, showIf: (f) => !f.simple_mode },
    { key: 'dpd_timeout', label: 'DPD timeout (s)', type: 'number', default: 120, showIf: (f) => !f.simple_mode },
  ],
  ikev2: [
    { key: 'port', label: 'Port', type: 'number', default: 500 },
    { key: 'network', label: 'Client subnet', type: 'text', default: '10.10.0.0/20' },
    { key: 'psk', label: 'Pre-shared key', type: 'text', default: '' },
    { key: 'domain', label: 'Server domain', type: 'text', default: '', tooltip: 'Required for a valid certificate.' },
    { key: 'cert_source', label: 'Certificate', type: 'select', default: 'letsencrypt', options: [{ label: "Let's Encrypt (auto)", value: 'letsencrypt' }, { label: 'Custom certificate', value: 'custom' }] },
    { key: 'dns', label: 'DNS', type: 'dns', default: '8.8.8.8' },
  ],
  ssh: [
    { key: 'port', label: 'Port', type: 'number', default: 2222 },
    { key: 'max_sessions', label: 'Max sessions', type: 'number', default: 10 },
    { key: 'key_type', label: 'Key type', type: 'select', default: 'ed25519', options: [{ label: 'ed25519 (recommended)', value: 'ed25519' }, { label: 'RSA', value: 'rsa' }] },
  ],
  mtproto: [
    { key: 'port', label: 'Port', type: 'number', default: 443 },
    { key: 'tag', label: 'Promo tag', type: 'text', default: '', tooltip: 'Optional Telegram sponsored-channel tag.' },
    { key: 'secret_mode', label: 'Secret mode', type: 'select', default: 'random', options: [{ label: 'Random per user', value: 'random' }, { label: 'Shared secret', value: 'shared' }] },
    { key: 'fake_tls', label: 'FakeTLS (dd-secret)', type: 'toggle', default: true, tooltip: 'Disguise traffic as TLS to evade DPI.' },
  ],
}

const nodes = ref<Node[]>([])
const loadingNodes = ref(false)
const loadingConfig = ref(false)
const selectedNodeId = ref<number | null>(null)
const config = ref<VpnConfig>({})

const panelOpen = ref(false)
const panelProtocol = ref<string>('')
const panelSaving = ref(false)
const showDnsDropdown = ref(false)
const panelForm = reactive<Record<string, any>>({})

const selectedNode = computed(() => nodes.value.find(n => n.id === selectedNodeId.value) || null)
const nodeOptions = computed(() => nodes.value.map(n => ({ label: n.name, value: n.id })))
const enabledCount = computed(() => protocols.filter(p => config.value[p.key]?.enabled).length)
const panelDef = computed(() => protocols.find(p => p.key === panelProtocol.value) || null)
const panelTitle = computed(() => panelDef.value && selectedNode.value ? `${panelDef.value.name} · ${selectedNode.value.name}` : 'Configure')
const currentFields = computed<FieldDef[]>(() => (protocolFields[panelProtocol.value] || []).filter(f => !f.showIf || f.showIf(panelForm)))

const statCards = computed(() => [
  { label: 'Enabled protocols', value: `${enabledCount.value} / ${protocols.length}`, icon: '\u{1F517}' },
  { label: 'Selected node', value: selectedNode.value?.name || '—', icon: '\u{1F5A5}\u{FE0F}' },
  { label: 'Available protocols', value: protocols.length, icon: '\u{1F9E9}' },
])

async function fetchNodes() {
  loadingNodes.value = true
  try {
    const res = await get<{ nodes: Node[] }>('/api/admin/knode/nodes')
    nodes.value = res.nodes || []
    if (nodes.value.length && selectedNodeId.value == null) selectedNodeId.value = nodes.value[0].id
  } catch { /* handled */ } finally { loadingNodes.value = false }
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
watch(selectedNodeId, (id) => { if (id != null) fetchVpnConfig(id) })
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
function generatePsk(key: string) {
  const chars = 'ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789'
  const arr = new Uint8Array(32); crypto.getRandomValues(arr)
  let r = ''; for (let i = 0; i < 32; i++) r += chars[arr[i] % chars.length]
  panelForm[key] = r
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
  if (panelProtocol.value === 'wireguard' && panelForm.gaming_optimize) { extra.persistent_keepalive = 15; extra.mtu = 1280 }
  try {
    await post(`/api/nodes/vpn-config/${selectedNodeId.value}`, { protocol: panelProtocol.value, enabled: true, port, network, extra_json: extra })
    toast.success('Protocol settings saved')
    config.value[panelProtocol.value] = { enabled: true, port, network, extra_json: extra }
    closePanel()
  } catch { /* handled */ } finally { panelSaving.value = false }
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
    toast.success(`${proto.name} ${newEnabled ? 'enabled' : 'disabled'}`)
  } catch { config.value[protocolKey].enabled = !newEnabled }
}
</script>

<template>
  <div class="page protocols-view">
    <PageHeader title="Protocols" subtitle="Enable and configure VPN & tunneling protocols per node.">
      <template #actions>
        <Select
          v-if="nodes.length"
          v-model="selectedNodeId"
          :options="nodeOptions"
          aria-label="Select node"
        />
      </template>
    </PageHeader>

    <!-- Stat cards (dashboard style) -->
    <section v-if="nodes.length" class="proto-stats" aria-label="Protocol statistics">
      <div v-for="stat in statCards" :key="stat.label" class="stat-card">
        <span class="stat-card__icon">{{ stat.icon }}</span>
        <div class="stat-card__body">
          <span class="stat-card__value">{{ stat.value }}</span>
          <span class="stat-card__label">{{ stat.label }}</span>
        </div>
      </div>
    </section>

    <!-- Loading / empty -->
    <div v-if="loadingNodes" class="proto-grid">
      <Skeleton v-for="i in 6" :key="i" variant="rect" :width="'100%'" :height="150" />
    </div>
    <EmptyState
      v-else-if="!nodes.length"
      icon="🖥️"
      title="No nodes yet"
      description="Add a node under Services to start configuring protocols."
    />

    <!-- Protocol grid -->
    <section v-else class="proto-grid">
      <article
        v-for="proto in protocols"
        :key="proto.key"
        class="proto-card"
        :class="{ 'proto-card--on': config[proto.key]?.enabled }"
      >
        <header class="proto-card__head">
          <span class="proto-card__icon">{{ proto.icon }}</span>
          <div class="proto-card__title-wrap">
            <h3 class="proto-card__name">{{ proto.name }}</h3>
            <StatusPill
              :status="config[proto.key]?.enabled ? 'active' : 'inactive'"
              :label="config[proto.key]?.enabled ? 'Enabled' : 'Disabled'"
              size="sm"
            />
          </div>
          <button
            class="proto-toggle"
            :class="{ 'proto-toggle--on': config[proto.key]?.enabled }"
            :aria-pressed="config[proto.key]?.enabled ? 'true' : 'false'"
            :title="config[proto.key]?.enabled ? 'Disable' : 'Enable'"
            @click="toggleProtocol(proto.key)"
          >
            <span class="proto-toggle__thumb" />
          </button>
        </header>

        <p class="proto-card__blurb">{{ proto.blurb }}</p>

        <footer class="proto-card__foot">
          <span v-if="config[proto.key]?.enabled" class="proto-card__port">
            :{{ config[proto.key]?.port }}
          </span>
          <span v-else class="proto-card__port proto-card__port--muted">not configured</span>
          <Button variant="ghost" size="sm" @click="openProtocolPanel(proto.key)">
            Configure
          </Button>
        </footer>
      </article>
    </section>

    <!-- Config drawer -->
    <SlideOver :open="panelOpen" :title="panelTitle" @close="closePanel">
      <div v-if="panelDef" class="cfg">
        <p class="cfg__desc">{{ panelDef.blurb }}</p>
        <div class="cfg__fields">
          <template v-for="field in currentFields" :key="field.key">
            <!-- toggle -->
            <div v-if="field.type === 'toggle'" class="cfg__toggle-row">
              <div>
                <span class="cfg__toggle-label">{{ field.label }}</span>
                <span v-if="field.tooltip" class="cfg__hint">{{ field.tooltip }}</span>
              </div>
              <button
                class="proto-toggle"
                :class="{ 'proto-toggle--on': panelForm[field.key] }"
                :aria-pressed="panelForm[field.key] ? 'true' : 'false'"
                @click="panelForm[field.key] = !panelForm[field.key]"
              >
                <span class="proto-toggle__thumb" />
              </button>
            </div>

            <!-- select -->
            <FormField v-else-if="field.type === 'select'" :name="field.key" :label="field.label" :hint="field.tooltip">
              <template #default="{ fieldId }">
                <Select :id="fieldId" v-model="panelForm[field.key]" :options="field.options || []" />
              </template>
            </FormField>

            <!-- dns with presets -->
            <FormField v-else-if="field.type === 'dns'" :name="field.key" :label="field.label">
              <template #default="{ fieldId }">
                <div class="cfg__dns">
                  <Input :id="fieldId" v-model="panelForm[field.key]" placeholder="8.8.8.8" />
                  <div class="cfg__dns-presets">
                    <button
                      v-for="p in dnsPresets"
                      :key="p.value"
                      type="button"
                      class="cfg__chip"
                      :class="{ 'cfg__chip--on': panelForm[field.key] === p.value }"
                      @click="selectDns(p.value)"
                    >{{ p.label }}</button>
                  </div>
                </div>
              </template>
            </FormField>

            <!-- psk / text with generator -->
            <FormField
              v-else
              :name="field.key"
              :label="field.label"
              :hint="field.tooltip"
            >
              <template #default="{ fieldId }">
                <div class="cfg__with-btn">
                  <Input
                    :id="fieldId"
                    v-model="panelForm[field.key]"
                    :type="field.type === 'password' ? 'password' : 'text'"
                    :inputmode="field.type === 'number' ? 'numeric' : undefined"
                  />
                  <Button
                    v-if="field.key === 'psk'"
                    variant="ghost"
                    size="sm"
                    type="button"
                    @click="generatePsk(field.key)"
                  >Generate</Button>
                </div>
              </template>
            </FormField>
          </template>
        </div>
      </div>

      <template #footer>
        <Button variant="ghost" @click="closePanel">Cancel</Button>
        <Button variant="primary" :loading="panelSaving" @click="saveProtocolSettings">Save & enable</Button>
      </template>
    </SlideOver>
  </div>
</template>

<style scoped>
.page { padding: var(--space-6); }

/* Stat cards — same look as Dashboard/Users */
.proto-stats { display: grid; grid-template-columns: repeat(3, 1fr); gap: var(--space-4); margin-bottom: var(--space-6); }
@media (max-width: 900px) { .proto-stats { grid-template-columns: 1fr; } }
.stat-card { display: flex; align-items: center; gap: var(--space-3); padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); }
.stat-card__icon { font-size: 1.35rem; display: grid; place-items: center; width: 44px; height: 44px; border-radius: var(--radius-lg); background: color-mix(in srgb, var(--color-primary) 14%, transparent); box-shadow: inset 0 0 0 1px color-mix(in srgb, var(--color-primary) 22%, transparent); }
.stat-card__body { display: flex; flex-direction: column; min-width: 0; }
.stat-card__value { font-size: var(--text-xl); font-weight: var(--font-bold); color: var(--color-text); white-space: nowrap; overflow: hidden; text-overflow: ellipsis; }
.stat-card__label { font-size: var(--text-xs); color: var(--color-muted); text-transform: uppercase; letter-spacing: var(--tracking-wider); }

/* Protocol grid */
.proto-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(280px, 1fr)); gap: var(--space-4); }
.proto-card {
  display: flex; flex-direction: column; gap: var(--space-3);
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  transition: border-color var(--duration-normal), box-shadow var(--duration-normal), transform var(--duration-normal);
}
.proto-card:hover { border-color: color-mix(in srgb, var(--color-primary) 28%, var(--color-border)); box-shadow: var(--shadow-md); }
.proto-card--on { border-color: color-mix(in srgb, var(--color-primary) 40%, var(--color-border)); }
.proto-card__head { display: flex; align-items: center; gap: var(--space-3); }
.proto-card__icon { font-size: 1.4rem; display: grid; place-items: center; width: 46px; height: 46px; border-radius: var(--radius-lg); background: color-mix(in srgb, var(--color-primary) 12%, transparent); }
.proto-card__title-wrap { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.proto-card__name { margin: 0; font-size: var(--text-lg); font-weight: var(--font-semibold); color: var(--color-text); }
.proto-card__blurb { margin: 0; font-size: var(--text-sm); color: var(--color-muted); line-height: var(--leading-snug); flex: 1; }
.proto-card__foot { display: flex; align-items: center; justify-content: space-between; padding-top: var(--space-2); border-top: 1px solid color-mix(in srgb, var(--color-border) 60%, transparent); }
.proto-card__port { font-family: var(--font-mono); font-size: var(--text-sm); color: var(--color-text); }
.proto-card__port--muted { color: var(--color-muted); font-family: var(--font-family); }

/* Toggle switch */
.proto-toggle { margin-left: auto; position: relative; width: 42px; height: 24px; border-radius: var(--radius-full); border: none; background: var(--color-surface-2); cursor: pointer; transition: background var(--duration-fast); flex-shrink: 0; }
.proto-toggle--on { background: var(--gradient-brand); }
.proto-toggle__thumb { position: absolute; top: 3px; left: 3px; width: 18px; height: 18px; border-radius: 50%; background: #fff; transition: transform var(--duration-fast); box-shadow: var(--shadow-sm); }
.proto-toggle--on .proto-toggle__thumb { transform: translateX(18px); }

/* Config drawer */
.cfg__desc { margin: 0 0 var(--space-4); color: var(--color-muted); font-size: var(--text-sm); }
.cfg__fields { display: flex; flex-direction: column; gap: var(--space-4); }
.cfg__toggle-row { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); padding: var(--space-3); border: 1px solid var(--color-border); border-radius: var(--radius-md); background: var(--color-surface); }
.cfg__toggle-label { display: block; font-weight: var(--font-medium); color: var(--color-text); font-size: var(--text-sm); }
.cfg__hint { display: block; margin-top: 2px; font-size: var(--text-xs); color: var(--color-muted); }
.cfg__dns { display: flex; flex-direction: column; gap: var(--space-2); }
.cfg__dns-presets { display: flex; flex-wrap: wrap; gap: var(--space-1); }
.cfg__chip { padding: 3px 10px; border-radius: var(--radius-full); border: 1px solid var(--color-border); background: var(--color-surface); color: var(--color-muted); font-size: var(--text-xs); cursor: pointer; transition: all var(--duration-fast); }
.cfg__chip:hover { border-color: var(--color-primary); color: var(--color-text); }
.cfg__chip--on { background: color-mix(in srgb, var(--color-primary) 15%, transparent); border-color: var(--color-primary); color: var(--color-primary); }
.cfg__with-btn { display: flex; gap: var(--space-2); align-items: center; }
.cfg__with-btn > :first-child { flex: 1; }
</style>
