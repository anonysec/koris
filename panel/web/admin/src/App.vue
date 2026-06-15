<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'

type Screen = 'loading' | 'setup' | 'login' | 'app'
type Section = 'overview' | 'customers' | 'customer-detail' | 'plans' | 'payments' | 'tickets' | 'resellers' | 'nodes' | 'system'

interface SetupStatus { ok: boolean; needs_setup: boolean; setup_key_required: boolean }
interface AuthResponse { ok: boolean; authenticated?: boolean; username?: string; role?: string; credit?: number }
interface ApiError extends Error { status?: number }

interface Customer { id: number; username: string; display_name: string; status: string; plan_id?: number | null; plan: string; credit: number; created_at: string }
interface DeletedCustomer extends Customer { deleted_at: string }
interface RadiusCheck { id: number; username: string; attribute: string; op: string; value: string }
interface WalletTransaction { id: number; username: string; amount: number; type: string; description: string; actor: string; created_at: string }
interface SubscriptionHistory { id: number; username: string; plan: string; status: string; started_at: string; expires_at: string; paid_amount: number; discount_code: string }
interface CustomerDetail extends Customer { notes: string; sub_token: string; radius_checks: RadiusCheck[]; radius_replies: RadiusCheck[]; subscription?: Record<string, unknown>; subscriptions: SubscriptionHistory[]; wallet_transactions: WalletTransaction[] }
interface Plan { id: number; name: string; data_gb: number; speed_mbps: number; duration_days: number; price: number; is_active: boolean; sort_order: number; created_at: string }
interface Payment { id: number; username: string; amount: number; method: string; status: string; intent_type: string; intent_id?: number; intent_label: string; created_at: string; updated_at: string }
interface PaymentMethod { id: number; name: string; type: string; instructions: string; is_active: boolean; sort_order: number; created_at: string }
interface Ticket { id: number; customer_id?: number; username: string; subject: string; status: string; priority: string; created_at: string; updated_at: string; closed_at: string }
interface TicketMessage { id: number; ticket_id: number; sender_type: string; sender_name: string; message: string; created_at: string }
interface TicketDetail extends Ticket { messages: TicketMessage[] }
interface NodeStatus { cpu_percent: number; ram_percent: number; disk_percent: number; rx_bps: number; tx_bps: number; openvpn_status: string; l2tp_status: string; ikev2_status: string; updated_at: string }
interface NodeService { service: string; status: string; updated_at: string }
interface NodeItem { id: number; name: string; public_ip: string; domain: string; status: string; last_seen_at: string; created_at: string; status_metrics: NodeStatus; services: NodeService[]; history?: any[] }
interface NodeTask { id: number; node_id: number; node_name: string; action: string; payload_json: Record<string, unknown>; status: string; error: string; created_at: string; completed_at: string }
interface VPNSettings { id: number; openvpn_port: number; openvpn_protocol: string; openvpn_network: string; l2tp_network: string; ikev2_network: string; ipsec_psk: string; dns_1: string; dns_2: string; updated_at: string; openvpn_service_status: string; ca_file: string; ca_exists: boolean; tls_crypt_file: string; tls_crypt_exists: boolean; remote_host: string; active_node: string }
interface UsageSession { id: number; username: string; start_time: string; update_time: string; stop_time: string; session_seconds: number; input_bytes: number; output_bytes: number; total_bytes: number; framed_ip: string; calling_station_id: string; terminate_cause: string; online: boolean }
interface UsageSummary { online: boolean; active_sessions: number; total_input_bytes: number; total_output_bytes: number; total_usage_bytes: number; max_data_bytes: number; remaining_bytes?: number; last_connected_at: string; last_disconnected_at: string; sessions: UsageSession[] }
interface Stats { ok: boolean; customers: number; active_customers: number; plans: number; nodes: number; open_tickets: number; pending_payments: number; approved_payments: number; total_rx_bps?: number; total_tx_bps?: number }
interface AuditLog { id: number; actor: string; action: string; entity_type: string; entity_id: string; before_json: string; after_json: string; ip: string; created_at: string }
type BlankNumber = number | ''

const screen = ref<Screen>('loading')
const section = ref<Section>('overview')
const setupStatus = ref<SetupStatus>({ ok: true, needs_setup: false, setup_key_required: false })
const user = ref({ username: '', role: '', credit: 0 })
const health = ref<{ ok?: boolean; version?: string; time?: string } | null>(null)
const stats = ref<Stats>({ ok: true, customers: 0, active_customers: 0, plans: 0, nodes: 0, open_tickets: 0, pending_payments: 0, approved_payments: 0 })
const customers = ref<Customer[]>([])
const deletedCustomers = ref<DeletedCustomer[]>([])
const plans = ref<Plan[]>([])
const payments = ref<Payment[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const methodForm = ref({ name: '', type: 'manual', instructions: '', is_active: true, sort_order: 0 })
const editingMethodId = ref<number | null>(null)
const tickets = ref<Ticket[]>([])
const selectedTicket = ref<TicketDetail | null>(null)
const ticketReply = ref('')
const adminTicketForm = ref({ username: '', subject: '', priority: 'normal', message: '' })
const nodes = ref<NodeItem[]>([])
const nodeTasks = ref<NodeTask[]>([])
const vpnSettings = ref<VPNSettings | null>(null)
const selectedCustomer = ref<CustomerDetail | null>(null)
const detailTab = ref<'profile' | 'usage' | 'history'>('profile')
const systemTab = ref<'audit' | 'backups' | 'diagnostics'>('diagnostics')
const infraTab = ref<'nodes' | 'vpn'>('nodes')
const customerView = ref<'active' | 'archived'>('active')
const selectedUsage = ref<UsageSummary | null>(null)
const search = ref('')
const busy = ref(false)
const appLoading = ref(false)
const detailLoading = ref(false)
const error = ref('')
const notice = ref('')
const auditLogs = ref<any[]>([])
const auditLoading = ref(false)
const auditOffset = ref(0)
const auditLimit = ref(100)

const setupForm = ref({ setup_key: '', username: 'owner', password: '' })
const loginForm = ref({ username: '', password: '' })
const createForm = ref<{ username: string; password: string; display_name: string; plan_id: number; data_gb: BlankNumber; speed_mbps: BlankNumber; days: BlankNumber }>({ username: '', password: '', display_name: '', plan_id: 0, data_gb: '', speed_mbps: '', days: '' })
const detailForm = ref({ display_name: '', status: 'active', plan_id: 0, notes: '', data_gb: 0, speed_mbps: 0, days: 0 })
const passwordForm = ref({ password: '' })
const planForm = ref({ name: '', data_gb: 0, speed_mbps: 0, duration_days: 30, price: 0, is_active: true, sort_order: 0 })
const paymentForm = ref({ username: '', amount: 0, method: 'manual', description: '' })
const nodeForm = ref({ name: '', public_ip: '', domain: '' })
const vpnForm = ref({ openvpn_port: 1194, openvpn_protocol: 'udp', openvpn_network: '10.8.0.0/24', l2tp_network: '10.9.0.0/24', ikev2_network: '10.10.0.0/24', ipsec_psk: '', dns_1: '1.1.1.1', dns_2: '8.8.8.8' })
const nodeToken = ref('')
const walletForm = ref({ username: '', amount: 0, description: 'Manual adjustment' })
const walletSetForm = ref({ username: '', balance: 0, description: 'Manual balance set' })
const renewForm = ref({ plan_id: 0 })
const editingPlanId = ref<number | null>(null)
const planModalOpen = ref(false)
const nodeModalOpen = ref(false)
const customerModalOpen = ref(false)
const realtimeConnected = ref(false)
const liveSessions = ref<any[]>([])
let realtimeSocket: WebSocket | null = null
let realtimeRetry: ReturnType<typeof setTimeout> | null = null

const activePlans = computed(() => plans.value.filter((plan) => plan.is_active))
const payAsGoPlan = computed(() => activePlans.value.find((plan) => plan.name.toLowerCase() === 'pay as you go'))
const selectedRenewPlan = computed(() => plans.value.find((plan) => plan.id === Number(renewForm.value.plan_id)))
const panelOrigin = computed(() => window.location.origin)
const nodeInstallCommand = computed(() => `cd koris-next && sudo PANEL_URL=${shQuote(panelOrigin.value)} NODE_TOKEN=${shQuote(nodeToken.value)} NODE_NAME=${shQuote(nodeForm.value.name || 'node1')} bash scripts/install-node.sh`)
const activePercent = computed(() => stats.value.customers ? Math.round((stats.value.active_customers / stats.value.customers) * 100) : 0)
const filteredCustomers = computed(() => {
  const q = search.value.trim().toLowerCase()
  const list = customerView.value === 'active' ? customers.value : deletedCustomers.value
  if (!q) return list
  return list.filter((customer) => `${customer.username} ${customer.display_name} ${customer.status} ${customer.plan}`.toLowerCase().includes(q))
})
const initials = computed(() => (user.value.username || 'K').slice(0, 2).toUpperCase())
const systemScore = computed(() => Math.min(100, (health.value?.ok ? 62 : 24) + (stats.value.customers ? 16 : 0) + (stats.value.plans ? 10 : 0) + (stats.value.nodes ? 12 : 0)))
const statusSummary = computed(() => {
  const summary: Record<string, number> = { active: 0, disabled: 0, expired: 0, limited: 0 }
  for (const customer of customers.value) summary[customer.status] = (summary[customer.status] || 0) + 1
  return summary
})

async function api<T>(url: string, options: RequestInit = {}): Promise<T> {
  const headers = new Headers(options.headers || {})
  if (options.body && !headers.has('Content-Type')) headers.set('Content-Type', 'application/json')
  const response = await fetch(url, { credentials: 'same-origin', ...options, headers })
  const data = await response.json().catch(() => ({ ok: false, error: response.statusText }))
  if (!response.ok || data.ok === false) {
    const err = new Error(data.error || `HTTP ${response.status}`) as ApiError
    err.status = response.status
    throw err
  }
  return data as T
}

function connectRealtime() {
  if (realtimeSocket || screen.value !== 'app') return
  const scheme = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  realtimeSocket = new WebSocket(`${scheme}//${window.location.host}/api/realtime`)
  realtimeSocket.onopen = () => { realtimeConnected.value = true }
  realtimeSocket.onmessage = (event) => {
    try {
      const message = JSON.parse(event.data)
      if (message.type === 'stats' && message.data) stats.value = message.data as Stats
      if (message.type === 'sessions' && message.data) liveSessions.value = message.data
    } catch { /* ignore malformed realtime frame */ }
  }
  realtimeSocket.onclose = () => {
    realtimeSocket = null
    realtimeConnected.value = false
    if (screen.value === 'app') realtimeRetry = setTimeout(connectRealtime, 3000)
  }
  realtimeSocket.onerror = () => realtimeSocket?.close()
}

function disconnectRealtime() {
  if (realtimeRetry) clearTimeout(realtimeRetry)
  realtimeRetry = null
  realtimeConnected.value = false
  if (realtimeSocket) {
    realtimeSocket.onclose = null
    realtimeSocket.close()
  }
  realtimeSocket = null
}

async function boot() {
  error.value = ''
  try {
    setupStatus.value = await api<SetupStatus>('/api/setup/status')
    if (setupStatus.value.needs_setup) { screen.value = 'setup'; return }
    const me = await api<AuthResponse>('/api/auth/me')
    if (me.authenticated) {
      user.value = { username: me.username || 'admin', role: me.role || 'admin', credit: me.credit || 0 }
      screen.value = 'app'
      await loadDashboard()
      return
    }
    screen.value = 'login'
  } catch (err) { error.value = friendlyError(err); screen.value = 'login' }
}

async function submitSetup() {
  busy.value = true; error.value = ''
  try {
    const res = await api<AuthResponse>('/api/setup/owner', { method: 'POST', body: JSON.stringify(setupForm.value) })
    user.value = { username: res.username || setupForm.value.username, role: res.role || 'owner', credit: 0 }
    notice.value = 'Owner account created. Welcome to KorisPanel.'
    screen.value = 'app'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function submitLogin() {
  busy.value = true; error.value = ''
  try {
    const res = await api<AuthResponse>('/api/auth/admin', { method: 'POST', body: JSON.stringify(loginForm.value) })
    user.value = { username: res.username || loginForm.value.username, role: res.role || 'admin', credit: res.credit || 0 }
    screen.value = 'app'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function logout() {
  disconnectRealtime()
  await api<{ ok: boolean }>('/api/auth/logout', { method: 'POST' }).catch(() => null)
  user.value = { username: '', role: '', credit: 0 }
  screen.value = 'login'
}

async function loadDashboard() {
  appLoading.value = true; error.value = ''
  try {
    const [healthRes, statsRes, customersRes, deletedRes, plansRes, paymentsRes, paymentMethodsRes, ticketsRes, nodesRes, nodeTasksRes, vpnRes] = await Promise.all([
      api<{ ok: boolean; version: string; time: string }>('/api/health'),
      api<Stats>('/api/dashboard/stats'),
      api<{ ok: boolean; customers: Customer[] }>(`/api/customers?q=${encodeURIComponent(search.value.trim())}`),
      api<{ ok: boolean; customers: DeletedCustomer[] }>('/api/deleted/customers'),
      api<{ ok: boolean; plans: Plan[] }>('/api/plans'),
      api<{ ok: boolean; payments: Payment[] }>('/api/payments'),
      api<{ ok: boolean; methods: PaymentMethod[] }>('/api/payment-methods'),
      api<{ ok: boolean; tickets: Ticket[] }>('/api/tickets'),
      api<{ ok: boolean; nodes: NodeItem[] }>('/api/nodes'),
      api<{ ok: boolean; tasks: NodeTask[] }>('/api/node/tasks'),
      api<{ ok: boolean; settings: VPNSettings }>('/api/vpn/settings')
    ])
    health.value = healthRes; stats.value = statsRes; customers.value = customersRes.customers || []; deletedCustomers.value = deletedRes.customers || []; plans.value = plansRes.plans || []; payments.value = paymentsRes.payments || []; paymentMethods.value = paymentMethodsRes.methods || []; tickets.value = ticketsRes.tickets || []; nodes.value = nodesRes.nodes || []; nodeTasks.value = nodeTasksRes.tasks || []; vpnSettings.value = vpnRes.settings; vpnForm.value = { openvpn_port: vpnRes.settings.openvpn_port, openvpn_protocol: vpnRes.settings.openvpn_protocol, openvpn_network: vpnRes.settings.openvpn_network, l2tp_network: vpnRes.settings.l2tp_network, ikev2_network: vpnRes.settings.ikev2_network, ipsec_psk: vpnRes.settings.ipsec_psk || '', dns_1: vpnRes.settings.dns_1, dns_2: vpnRes.settings.dns_2 }
    defaultCreatePlanIfNeeded()
    connectRealtime()
    if (user.value.role === 'reseller') {
      await loadResellerPayments()
    }
  } catch (err) {
    const apiErr = err as ApiError
    if (apiErr.status === 401) screen.value = 'login'
    error.value = friendlyError(err)
  } finally { appLoading.value = false }
}

async function loadAuditLogs() {
  auditLoading.value = true; error.value = ''
  try {
    const res = await api<{ ok: boolean; logs: AuditLog[]; limit: number; offset: number }>(`/api/audit-logs?limit=${auditLimit.value}&offset=${auditOffset.value}`)
    auditLogs.value = res.logs || []
  } catch (err) { error.value = friendlyError(err) }
  finally { auditLoading.value = false }
}

const diagnosticsData = ref<any>(null)
const diagnosticsLoading = ref(false)
async function loadDiagnostics() {
  diagnosticsLoading.value = true; error.value = ''
  try {
    const res = await api<any>('/api/diagnostics')
    diagnosticsData.value = res
  } catch (err) { error.value = friendlyError(err) }
  finally { diagnosticsLoading.value = false }
}

const resellersList = ref<any[]>([])
const resellerForm = ref({ username: '', password: '' })
const resellerCreditForm = ref<Record<number, number>>({})
const resellerTxs = ref<any[]>([])

async function loadResellerTxs() {
  try {
    const res = await api<any>('/api/resellers/transactions')
    resellerTxs.value = res.transactions || []
  } catch (err) { error.value = friendlyError(err) }
}

async function loadResellers() {
  error.value = ''
  try {
    const res = await api<any>('/api/resellers')
    resellersList.value = res.resellers || []
    await loadResellerTxs()
  } catch (err) { error.value = friendlyError(err) }
}

async function createReseller() {
  busy.value = true; error.value = ''
  try {
    await api<any>('/api/resellers', {
      method: 'POST',
      body: JSON.stringify(resellerForm.value)
    })
    resellerForm.value = { username: '', password: '' }
    notice.value = 'Reseller created successfully.'
    await loadResellers()
  } catch (err) { error.value = friendlyError(err) }
  finally { busy.value = false }
}

async function adjustResellerCredit(id: number, add: boolean) {
  busy.value = true; error.value = ''
  let amt = resellerCreditForm.value[id] || 0
  if (!add) amt = -amt
  try {
    await api<any>(`/api/resellers/${id}/credit`, {
      method: 'POST',
      body: JSON.stringify({ amount: amt })
    })
    resellerCreditForm.value[id] = 0
    notice.value = 'Reseller credit adjusted successfully.'
    await loadResellers()
  } catch (err) { error.value = friendlyError(err) }
  finally { busy.value = false }
}

async function deleteReseller(id: number) {
  if (!confirm('Are you sure you want to delete this reseller?')) return
  busy.value = true; error.value = ''
  try {
    await api<any>(`/api/resellers/${id}`, { method: 'DELETE' })
    notice.value = 'Reseller deleted.'
    await loadResellers()
  } catch (err) { error.value = friendlyError(err) }
  finally { busy.value = false }
}

async function killSession(id: number) {
  if (!confirm('Are you sure you want to terminate this active VPN connection?')) return
  error.value = ''
  try {
    await api<any>('/api/sessions/kill', {
      method: 'POST',
      body: JSON.stringify({ id })
    })
    notice.value = 'VPN session terminated.'
    liveSessions.value = liveSessions.value.filter(s => s.id !== id)
  } catch (err) { error.value = friendlyError(err) }
}

const rxHistory = ref<number[]>(Array(20).fill(0))
const txHistory = ref<number[]>(Array(20).fill(0))
const resellerTopupAmount = ref(50000)

watch(() => stats.value, (newStats: any) => {
  if (newStats) {
    rxHistory.value.push(newStats.total_rx_bps || 0)
    rxHistory.value.shift()
    txHistory.value.push(newStats.total_tx_bps || 0)
    txHistory.value.shift()
  }
}, { deep: true })

const maxBps = computed(() => {
  const maxVal = Math.max(...rxHistory.value, ...txHistory.value, 1024)
  return maxVal
})

const rxPoints = computed(() => {
  const max = maxBps.value
  return rxHistory.value.map((val, idx) => `${idx * 18},${60 - (val / max) * 50}`).join(' ')
})

const txPoints = computed(() => {
  const max = maxBps.value
  return txHistory.value.map((val, idx) => `${idx * 18},${60 - (val / max) * 50}`).join(' ')
})

async function checkoutResellerCredit() {
  busy.value = true; error.value = ''
  try {
    await api<any>('/api/resellers/checkout', {
      method: 'POST',
      body: JSON.stringify({ amount: resellerTopupAmount.value })
    })
    notice.value = 'Self-checkout completed. Reseller wallet credited.'
    user.value.credit += resellerTopupAmount.value
    resellerTopupAmount.value = 50000
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) }
  finally { busy.value = false }
}

function nodeHistoryPoints(history: any[]) {
  if (!history || !history.length) return '0,40 150,40'
  const maxRx = Math.max(...history.map(h => Number(h.rx_bytes || 0)), 1024)
  const reversed = [...history].reverse()
  return reversed.map((h, idx) => {
    const x = (idx / (reversed.length - 1 || 1)) * 150
    const y = 35 - (Number(h.rx_bytes || 0) / maxRx) * 30
    return `${x},${y}`
  }).join(' ')
}

function copyToClipboard(text: string) {
  if (navigator.clipboard && window.isSecureContext) {
    navigator.clipboard.writeText(text).then(() => {
      notice.value = 'Copied to clipboard!'
    })
  } else {
    const textArea = document.createElement('textarea')
    textArea.value = text
    textArea.style.top = '0'
    textArea.style.left = '0'
    textArea.style.position = 'fixed'
    document.body.appendChild(textArea)
    textArea.focus()
    textArea.select()
    try {
      const successful = document.execCommand('copy')
      if (successful) {
        notice.value = 'Copied to clipboard!'
      } else {
        notice.value = 'Press Ctrl+C to copy'
      }
    } catch (err) {
      notice.value = 'Failed to copy, please copy manually'
    }
    document.body.removeChild(textArea)
  }
}

const resellerPayments = ref<any[]>([])
const resellerManualPayForm = ref({ amount: 100000, description: '' })

async function loadResellerPayments() {
  try {
    const res = await api<any>('/api/resellers/payments')
    resellerPayments.value = res.payments || []
  } catch (err) { error.value = friendlyError(err) }
}

async function submitManualResellerPayment() {
  busy.value = true; error.value = ''
  try {
    await api<any>('/api/resellers/payments', {
      method: 'POST',
      body: JSON.stringify(resellerManualPayForm.value)
    })
    resellerManualPayForm.value = { amount: 100000, description: '' }
    notice.value = 'Manual top-up request submitted for admin review.'
    await loadResellerPayments()
  } catch (err) { error.value = friendlyError(err) }
  finally { busy.value = false }
}

function exportCSV(name: string) {
  window.open(`/api/export/${name}.csv`, '_blank')
}

function defaultCreatePlanIfNeeded() {
  if (!createForm.value.plan_id && payAsGoPlan.value) {
    createForm.value.plan_id = payAsGoPlan.value.id
    applyCreatePlan()
  }
}
function applyCreatePlan() {
  const plan = plans.value.find((item) => item.id === Number(createForm.value.plan_id))
  if (!plan) {
    createForm.value.data_gb = ''
    createForm.value.speed_mbps = ''
    createForm.value.days = ''
    return
  }
  createForm.value.data_gb = plan.data_gb || ''
  createForm.value.speed_mbps = plan.speed_mbps || ''
  createForm.value.days = plan.duration_days || ''
}
function applyDetailPlan() {
  const plan = plans.value.find((item) => item.id === Number(detailForm.value.plan_id))
  if (!plan) return
  detailForm.value.data_gb = plan.data_gb
  detailForm.value.speed_mbps = plan.speed_mbps
  detailForm.value.days = plan.duration_days
}
function resetCreateForm() {
  createForm.value = { username: '', password: '', display_name: '', plan_id: 0, data_gb: '', speed_mbps: '', days: '' }
  defaultCreatePlanIfNeeded()
}
function cleanNumber(value: unknown) { const n = Number(value); return Number.isFinite(n) && n > 0 ? n : 0 }

async function createCustomer() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const payload = { ...createForm.value, data_gb: cleanNumber(createForm.value.data_gb), speed_mbps: cleanNumber(createForm.value.speed_mbps), days: Math.trunc(cleanNumber(createForm.value.days)) }
    await api<{ ok: boolean; id: number }>('/api/customers', { method: 'POST', body: JSON.stringify(payload) })
    notice.value = `Customer ${createForm.value.username} created.`
    customerModalOpen.value = false
    resetCreateForm()
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function openCustomer(customer: Customer) { section.value = 'customer-detail'; await loadCustomer(customer.id) }
async function loadCustomer(id: number) {
  detailLoading.value = true; error.value = ''; selectedCustomer.value = null; selectedUsage.value = null
  try {
    const [res, usageRes] = await Promise.all([
      api<{ ok: boolean; customer: CustomerDetail }>(`/api/customers/${id}`),
      api<{ ok: boolean; usage: UsageSummary }>(`/api/customers/${id}/usage`)
    ])
    selectedCustomer.value = res.customer
    selectedUsage.value = usageRes.usage
    detailForm.value = { display_name: res.customer.display_name || '', status: res.customer.status || 'active', plan_id: Number(res.customer.plan_id || 0), notes: res.customer.notes || '', data_gb: maxDataGB(res.customer.radius_checks || []), speed_mbps: speedMbps(res.customer.radius_replies || []), days: 0 }
    walletForm.value.username = res.customer.username
    walletSetForm.value.username = res.customer.username
    walletSetForm.value.balance = Number(res.customer.credit || 0)
    renewForm.value.plan_id = Number(res.customer.plan_id || payAsGoPlan.value?.id || 0)
    paymentForm.value.username = res.customer.username
  } catch (err) { error.value = friendlyError(err) } finally { detailLoading.value = false }
}

async function saveCustomerDetail() {
  if (!selectedCustomer.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const payload = { ...detailForm.value, data_gb: cleanNumber(detailForm.value.data_gb), speed_mbps: cleanNumber(detailForm.value.speed_mbps), days: Math.trunc(cleanNumber(detailForm.value.days)) }
    await api<{ ok: boolean }>(`/api/customers/${selectedCustomer.value.id}`, { method: 'PATCH', body: JSON.stringify(payload) })
    notice.value = 'Customer details saved.'
    await loadCustomer(selectedCustomer.value.id); await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function setSelectedCustomerStatus(status: 'active' | 'disabled') {
  if (!selectedCustomer.value) return
  busy.value = true; error.value = ''
  try {
    await api<{ ok: boolean }>(`/api/customers/${selectedCustomer.value.id}/${status === 'active' ? 'enable' : 'disable'}`, { method: 'POST' })
    notice.value = status === 'active' ? 'Customer enabled.' : 'Customer disabled.'
    await loadCustomer(selectedCustomer.value.id); await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function resetCustomerPassword() {
  if (!selectedCustomer.value) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/customers/${selectedCustomer.value.id}/reset-password`, { method: 'POST', body: JSON.stringify(passwordForm.value) })
    notice.value = 'VPN password reset.'; passwordForm.value.password = ''; await loadCustomer(selectedCustomer.value.id)
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function renewCustomerPlan() {
  if (!selectedCustomer.value) return
  if (!renewForm.value.plan_id) { error.value = 'plan required'; return }
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const res = await api<{ ok: boolean; wallet_deducted: number }>(`/api/customers/${selectedCustomer.value.id}/renew`, { method: 'POST', body: JSON.stringify(renewForm.value) })
    notice.value = res.wallet_deducted > 0 ? `Plan applied. Wallet deducted ${formatMoney(res.wallet_deducted)}.` : 'Plan applied.'
    await loadCustomer(selectedCustomer.value.id); await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function archiveSelectedCustomer() {
  if (!selectedCustomer.value) return
  if (!confirm(`Archive customer ${selectedCustomer.value.username}? VPN radius rows will be removed until restore.`)) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/customers/${selectedCustomer.value.id}`, { method: 'DELETE' })
    notice.value = 'Customer archived.'
    selectedCustomer.value = null
    customerView.value = 'archived'
    section.value = 'customers'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function restoreDeletedCustomer(customer: DeletedCustomer) {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/customers/${customer.id}/restore`, { method: 'POST' })
    notice.value = `Customer ${customer.username} restored.`
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

function resetPlanForm() { editingPlanId.value = null; planForm.value = { name: '', data_gb: 0, speed_mbps: 0, duration_days: 30, price: 0, is_active: true, sort_order: 0 } }
function openNewPlan() { resetPlanForm(); planModalOpen.value = true }
function editPlan(plan: Plan) { editingPlanId.value = plan.id; planForm.value = { name: plan.name, data_gb: plan.data_gb, speed_mbps: plan.speed_mbps, duration_days: plan.duration_days, price: plan.price, is_active: plan.is_active, sort_order: plan.sort_order }; planModalOpen.value = true }
async function savePlan() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const payload = { ...planForm.value, data_gb: cleanNumber(planForm.value.data_gb), speed_mbps: cleanNumber(planForm.value.speed_mbps), duration_days: Math.trunc(cleanNumber(planForm.value.duration_days)), price: cleanNumber(planForm.value.price) }
    if (editingPlanId.value) { await api<{ ok: boolean }>(`/api/plans/${editingPlanId.value}`, { method: 'PATCH', body: JSON.stringify(payload) }); notice.value = 'Plan updated.' }
    else { await api<{ ok: boolean; id: number }>('/api/plans', { method: 'POST', body: JSON.stringify(payload) }); notice.value = 'Plan created.' }
    resetPlanForm(); planModalOpen.value = false; await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function archivePlan(plan: Plan) {
  if (!confirm(`Deactivate plan ${plan.name}? Existing customers keep their reference.`)) return
  busy.value = true; error.value = ''
  try { await api<{ ok: boolean }>(`/api/plans/${plan.id}`, { method: 'DELETE' }); notice.value = 'Plan deactivated.'; await loadDashboard() }
  catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}


async function saveVPNSettings(apply = false) {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const res = await api<{ ok: boolean; settings: VPNSettings; applied: boolean; apply_error: string }>('/api/vpn/settings', { method: 'PATCH', body: JSON.stringify({ ...vpnForm.value, apply }) })
    vpnSettings.value = res.settings
    if (apply && res.apply_error) notice.value = `Settings saved, but apply failed: ${res.apply_error}`
    else notice.value = apply ? 'VPN settings saved and OpenVPN restarted.' : 'VPN settings saved.'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

function resetNodeForm() { nodeForm.value = { name: '', public_ip: '', domain: '' }; nodeToken.value = '' }
async function createNode() {
  busy.value = true; error.value = ''; notice.value = ''; nodeToken.value = ''
  try {
    const res = await api<{ ok: boolean; id: number; token: string }>('/api/nodes', { method: 'POST', body: JSON.stringify(nodeForm.value) })
    nodeToken.value = res.token
    notice.value = 'Node created. Copy the token now.'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function rotateNodeToken(node: NodeItem) {
  if (!confirm(`Rotate token for ${node.name}? The old node token will stop working.`)) return
  busy.value = true; error.value = ''; notice.value = ''; nodeToken.value = ''
  try {
    const res = await api<{ ok: boolean; token: string }>(`/api/nodes/${node.id}/rotate-token`, { method: 'POST' })
    nodeToken.value = res.token
    nodeModalOpen.value = true
    notice.value = 'Node token rotated. Copy the new token now.'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function setNodeEnabled(node: NodeItem, enabled: boolean) {
  busy.value = true; error.value = ''
  try {
    await api<{ ok: boolean }>(`/api/nodes/${node.id}/${enabled ? 'enable' : 'disable'}`, { method: 'POST' })
    notice.value = enabled ? 'Node enabled.' : 'Node disabled.'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
function serviceLabel(node: NodeItem, key: string) {
  return node.services?.find((service) => service.service === key)?.status || node.status_metrics?.[`${key}_status` as keyof NodeStatus] || 'unknown'
}
function bps(value?: number) {
  const n = Number(value || 0)
  if (n > 1024 * 1024) return `${(n / 1024 / 1024).toFixed(2)} MB/s`
  if (n > 1024) return `${(n / 1024).toFixed(2)} KB/s`
  return `${Math.round(n)} B/s`
}


async function createNodeTask(node: NodeItem, action: string, payload: Record<string, unknown> = {}) {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean; id: number }>('/api/node/tasks', { method: 'POST', body: JSON.stringify({ node_id: node.id, action, payload_json: payload }) })
    notice.value = `Task queued for ${node.name}.`
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}



function resetMethodForm() { editingMethodId.value = null; methodForm.value = { name: '', type: 'manual', instructions: '', is_active: true, sort_order: 0 } }
function editPaymentMethod(method: PaymentMethod) { editingMethodId.value = method.id; methodForm.value = { name: method.name, type: method.type, instructions: method.instructions || '', is_active: method.is_active, sort_order: method.sort_order } }
async function savePaymentMethod() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    if (editingMethodId.value) {
      await api<{ ok: boolean }>(`/api/payment-methods/${editingMethodId.value}`, { method: 'PATCH', body: JSON.stringify(methodForm.value) })
      notice.value = 'Payment method updated.'
    } else {
      await api<{ ok: boolean; id: number }>('/api/payment-methods', { method: 'POST', body: JSON.stringify(methodForm.value) })
      notice.value = 'Payment method created.'
    }
    resetMethodForm(); await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function deactivatePaymentMethod(method: PaymentMethod) {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/payment-methods/${method.id}`, { method: 'DELETE' })
    notice.value = 'Payment method deactivated.'
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function loadTicket(id: number) {
  busy.value = true; error.value = ''
  try {
    const res = await api<{ ok: boolean; ticket: TicketDetail }>(`/api/tickets/${id}`)
    selectedTicket.value = res.ticket
    ticketReply.value = ''
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function replyTicket() {
  if (!selectedTicket.value || !ticketReply.value.trim()) return
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/tickets/${selectedTicket.value.id}/reply`, { method: 'POST', body: JSON.stringify({ message: ticketReply.value }) })
    notice.value = 'Reply sent.'
    await loadTicket(selectedTicket.value.id); await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function setTicketStatus(ticket: Ticket, status: 'open' | 'closed') {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/tickets/${ticket.id}/${status === 'closed' ? 'close' : 'open'}`, { method: 'POST' })
    notice.value = status === 'closed' ? 'Ticket closed.' : 'Ticket reopened.'
    await loadTicket(ticket.id).catch(() => null); await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function createAdminTicket() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    const res = await api<{ ok: boolean; id: number }>('/api/tickets', { method: 'POST', body: JSON.stringify(adminTicketForm.value) })
    notice.value = 'Ticket created.'
    adminTicketForm.value = { username: '', subject: '', priority: 'normal', message: '' }
    await loadDashboard(); await loadTicket(res.id)
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

async function createManualPayment() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean; id: number }>('/api/payments', { method: 'POST', body: JSON.stringify({ ...paymentForm.value, amount: cleanNumber(paymentForm.value.amount) }) })
    notice.value = 'Manual payment recorded and wallet topped up.'
    paymentForm.value = { username: '', amount: 0, method: 'manual', description: '' }
    await loadDashboard()
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function adjustWallet() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/wallets/${encodeURIComponent(walletForm.value.username)}/adjust`, { method: 'POST', body: JSON.stringify({ amount: Number(walletForm.value.amount), description: walletForm.value.description }) })
    notice.value = 'Wallet adjusted.'; walletForm.value.amount = 0; await loadDashboard(); if (selectedCustomer.value) await loadCustomer(selectedCustomer.value.id)
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function setWalletBalance() {
  busy.value = true; error.value = ''; notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/wallets/${encodeURIComponent(walletSetForm.value.username)}/set`, { method: 'POST', body: JSON.stringify({ balance: Number(walletSetForm.value.balance), description: walletSetForm.value.description }) })
    notice.value = 'Wallet balance saved.'; await loadDashboard(); if (selectedCustomer.value) await loadCustomer(selectedCustomer.value.id)
  } catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}
async function approvePayment(payment: Payment, status: 'approve' | 'reject') {
  busy.value = true; error.value = ''
  try { await api<{ ok: boolean }>(`/api/payments/${payment.id}/${status}`, { method: 'POST' }); notice.value = `Payment ${status}d.`; await loadDashboard() }
  catch (err) { error.value = friendlyError(err) } finally { busy.value = false }
}

function friendlyError(err: unknown) { return err instanceof Error ? err.message.replace(/_/g, ' ') : 'Unexpected error' }
function formatDate(value?: string) { return value ? new Intl.DateTimeFormat('en', { month: 'short', day: '2-digit', hour: '2-digit', minute: '2-digit' }).format(new Date(value)) : '—' }
function shQuote(value: string) { return `'${String(value).replace(/'/g, `'\\''`)}'` }
function formatMoney(value?: number) { return `${new Intl.NumberFormat('en', { maximumFractionDigits: 0 }).format(value || 0)} IRT` }
function signedMoney(value?: number) { const n = Number(value || 0); return `${n > 0 ? '+' : ''}${formatMoney(n)}` }
function formatBytes(value?: number) {
  const n = Number(value || 0)
  if (n >= 1024 ** 4) return `${(n / 1024 ** 4).toFixed(2)} TB`
  if (n >= 1024 ** 3) return `${(n / 1024 ** 3).toFixed(2)} GB`
  if (n >= 1024 ** 2) return `${(n / 1024 ** 2).toFixed(2)} MB`
  if (n >= 1024) return `${(n / 1024).toFixed(2)} KB`
  return `${Math.round(n)} B`
}
function formatDuration(seconds?: number) {
  const s = Math.max(0, Math.trunc(Number(seconds || 0)))
  const h = Math.floor(s / 3600)
  const m = Math.floor((s % 3600) / 60)
  const sec = s % 60
  if (h) return `${h}h ${m}m`
  if (m) return `${m}m ${sec}s`
  return `${sec}s`
}
function formatGB(value?: number) { return value && value > 0 ? `${new Intl.NumberFormat('en', { maximumFractionDigits: 2 }).format(value)} GB` : 'Unlimited' }
function formatSpeed(value?: number) { return value && value > 0 ? `${new Intl.NumberFormat('en', { maximumFractionDigits: 2 }).format(value)} Mbps` : 'Unlimited' }
function maxDataGB(checks: RadiusCheck[]) { const raw = Number(checks.find((check) => check.attribute === 'Max-Data')?.value || 0); return raw ? Math.round((raw / 1024 / 1024 / 1024) * 100) / 100 : 0 }
function speedMbps(replies: RadiusCheck[]) { const v = replies.find((reply) => reply.attribute === 'Mikrotik-Rate-Limit')?.value || ''; const m = v.match(/([0-9.]+)M/i); return m ? Number(m[1]) : 0 }
function subscriptionText(customer: CustomerDetail | null) { const sub = customer?.subscription; return sub ? `${sub.status || 'active'} · expires ${formatDate(String(sub.expires_at || ''))}` : 'No subscription yet' }

watch(notice, (message) => {
  if (!message) return
  window.setTimeout(() => {
    if (notice.value === message) notice.value = ''
  }, 4000)
})

watch(section, (newSec) => {
  window.location.hash = '/' + newSec
})

onMounted(() => {
  if (window.location.pathname !== '/dashboard/' && window.location.pathname !== '/dashboard') {
    window.history.replaceState(null, '', '/dashboard/' + window.location.hash)
  }
  const hash = window.location.hash.replace('#/', '').replace('#', '')
  if (hash && ['overview', 'customers', 'plans', 'payments', 'tickets', 'resellers', 'nodes', 'system', 'customer-detail'].includes(hash)) {
    section.value = hash as Section
  }
  boot()
})
</script>

<template>
  <main v-if="screen === 'loading'" class="loading-screen"><div class="orb"></div><p>Loading KorisPanel…</p></main>

  <main v-else-if="screen === 'setup' || screen === 'login'" class="auth-screen">
    <section class="auth-hero">
      <div class="brand-row"><span class="brand-mark">K</span><div><strong>KorisPanel</strong><small>Next generation VPN operations</small></div></div>
      <h1>Premium, compact control for customers, nodes and billing.</h1>
      <p>Go backend, Vue 3 frontend, clean schema, split panel/node architecture.</p>
      <div class="hero-grid"><span>Go API online</span><span>Vue dashboard</span><span>FreeRADIUS ready</span></div>
      <div class="ux-preview" aria-label="KorisPanel product flow preview"><div><small>01</small><b>Secure setup</b><span>Owner bootstrap with signed sessions</span></div><div><small>02</small><b>Customer ops</b><span>Create Radius users in one compact flow</span></div><div><small>03</small><b>Payments</b><span>Manual top-up and wallet adjustment</span></div></div>
    </section>
    <section class="auth-card glass-card"><div class="auth-card-head"><span>{{ screen === 'setup' ? 'First run' : 'Admin access' }}</span><b>{{ screen === 'setup' ? 'Create owner' : 'Sign in' }}</b></div><form v-if="screen === 'setup'" @submit.prevent="submitSetup" class="form-stack"><label v-if="setupStatus.setup_key_required">Setup key<input v-model="setupForm.setup_key" autocomplete="one-time-code" placeholder="Paste setup key" required /></label><label>Owner username<input v-model.trim="setupForm.username" autocomplete="username" placeholder="owner" required /></label><label>Password<input v-model="setupForm.password" type="password" autocomplete="new-password" placeholder="At least 6 characters" required /></label><button :disabled="busy" class="primary-btn">{{ busy ? 'Creating…' : 'Create owner' }}</button></form><form v-else @submit.prevent="submitLogin" class="form-stack"><label>Username<input v-model.trim="loginForm.username" autocomplete="username" placeholder="admin username" required /></label><label>Password<input v-model="loginForm.password" type="password" autocomplete="current-password" placeholder="••••••••" required /></label><button :disabled="busy" class="primary-btn">{{ busy ? 'Signing in…' : 'Enter dashboard' }}</button></form><p v-if="error" class="alert danger">{{ error }}</p><small class="muted">Root path intentionally stays closed; admin UI is served from <b>/dashboard/</b>.</small></section>
  </main>

  <main v-else class="panel-shell">
    <aside class="sidebar"><div class="brand-row compact"><span class="brand-mark">K</span><div><strong>KorisPanel</strong><small>koris-next</small></div></div><nav><button :class="{ active: section === 'overview' }" @click="section = 'overview'"><span>Overview</span><b>{{ activePercent }}%</b></button><button :class="{ active: section === 'customers' || section === 'customer-detail' }" @click="section = 'customers'"><span>Customers</span><b>{{ stats.customers }}</b></button><button :class="{ active: section === 'plans' }" @click="section = 'plans'"><span>Plans</span><b>{{ stats.plans }}</b></button><button v-if="user.role !== 'reseller'" :class="{ active: section === 'payments' }" @click="section = 'payments'"><span>Payments</span><b>{{ stats.pending_payments || payments.length }}</b></button><button :class="{ active: section === 'tickets' }" @click="section = 'tickets'"><span>Tickets</span><b>{{ stats.open_tickets || tickets.length }}</b></button><button v-if="user.role === 'owner' || user.role === 'admin'" :class="{ active: section === 'resellers' }" @click="section = 'resellers'; loadResellers()"><span>Resellers</span><b>System</b></button><button v-if="user.role !== 'reseller'" :class="{ active: section === 'nodes' }" @click="section = 'nodes'"><span>Infrastructure</span><b>{{ nodes.length }}</b></button><button v-if="user.role !== 'reseller'" :class="{ active: section === 'system' }" @click="section = 'system'; loadDiagnostics(); loadAuditLogs()"><span>System logs</span><b>Utility</b></button></nav><div class="release-card"><span>Design system</span><b>Pro compact</b><small>Glass depth · dense tables · clear status language</small></div><div class="sidebar-footer"><small>API status</small><strong :class="health?.ok ? 'online' : 'offline'">{{ health?.ok ? 'Online' : 'Offline' }}</strong></div></aside>

    <section class="workspace">
      <header class="topbar"><div><span class="eyebrow">{{ section }}</span><h1>{{ section === 'overview' ? 'Command dashboard' : section === 'plans' ? 'Plans catalog' : section === 'payments' ? 'Payments & wallets' : section === 'tickets' ? 'Support tickets' : section === 'nodes' ? 'Infrastructure & VPN' : section === 'system' ? 'System Logs & Utilities' : section === 'resellers' ? 'Reseller fleet' : section === 'customer-detail' ? 'Customer detail' : 'Customer operations' }}</h1></div><div class="top-actions"><div class="score-chip" role="status"><span>{{ systemScore }}%</span><small>ready</small></div><div v-if="user.role === 'reseller'" class="notify-chip" style="background: rgba(34, 211, 238, 0.15); color: var(--cyan); border: 1px solid rgba(34, 211, 238, 0.3);">Reseller Credit: {{ formatMoney(user.credit) }}</div><div v-if="stats.pending_payments && user.role !== 'reseller'" class="notify-chip"><b>{{ stats.pending_payments }}</b> pending payment{{ stats.pending_payments === 1 ? '' : 's' }}</div><div :class="['live-chip', { online: realtimeConnected }]"><span></span>{{ realtimeConnected ? 'Live' : 'Offline' }}</div><div class="user-chip"><span>{{ initials }}</span><div><b>{{ user.username }}</b><small>{{ user.role }}</small></div></div><button class="icon-btn" title="Logout" aria-label="Logout" @click="logout">↗</button></div></header>
      <div v-if="notice" class="toast success">{{ notice }}</div><p v-if="error" class="alert danger">{{ error }}</p>

      <section v-if="section === 'overview'" class="dashboard-grid"><article class="metric-card primary"><small>Total customers</small><strong>{{ stats.customers }}</strong><span>{{ stats.active_customers }} active · {{ activePercent }}% healthy</span></article><article class="metric-card"><small>Approved payments</small><strong>{{ formatMoney(stats.approved_payments) }}</strong><span>{{ stats.pending_payments }} pending review</span></article><article class="metric-card"><small>Nodes online</small><strong>{{ stats.nodes }}</strong><span>HTTP node API only</span></article><article class="metric-card"><small>Active plans</small><strong>{{ stats.plans }}</strong><span>{{ activePlans.length }} available packages</span></article>

        <article class="glass-card wide-card" style="grid-column: span 4; height: 180px;">
          <div class="section-head" style="margin-bottom: 4px;">
            <div>
              <span class="eyebrow" style="color: var(--cyan);">Real-Time aggregate bandwidth</span>
              <h2>Multi-Node Bandwidth Speedometer</h2>
            </div>
            <div style="font-size: 13px; font-weight: bold;">
              <span style="color: var(--cyan); margin-right: 12px;">● RX: {{ formatBytes((stats.total_rx_bps || 0) / 8) }}/s</span>
              <span style="color: var(--blue);">● TX: {{ formatBytes((stats.total_tx_bps || 0) / 8) }}/s</span>
            </div>
          </div>
          <div style="width: 100%; height: 90px; margin-top: 10px;">
            <svg viewBox="0 0 360 60" style="width: 100%; height: 100%; overflow: visible;">
              <polyline fill="none" stroke="var(--cyan)" stroke-width="2.5" :points="rxPoints" />
              <polyline fill="none" stroke="var(--blue)" stroke-width="2.5" :points="txPoints" />
            </svg>
          </div>
        </article>

        <article class="glass-card wide-card" style="grid-column: span 4;">
          <div class="section-head">
            <div>
              <span class="eyebrow" style="color: var(--cyan);">Real-Time active throughput</span>
              <h2>3x-ui Live Session Speedometer</h2>
            </div>
            <span class="pill active" style="background: rgba(34, 211, 238, 0.15); color: var(--cyan); border: 1px solid rgba(34, 211, 238, 0.3);">WebSocket Live Streaming</span>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Session ID</th>
                  <th>Customer</th>
                  <th>Framed IP</th>
                  <th>Duration</th>
                  <th>Real-Time Rx / Speed (Download)</th>
                  <th>Real-Time Tx / Speed (Upload)</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="s in liveSessions" :key="s.id">
                  <td>#{{ s.id }}</td>
                  <td><b>{{ s.username }}</b></td>
                  <td><code>{{ s.ip || '—' }}</code></td>
                  <td>{{ formatDuration(s.duration) }}</td>
                  <td>
                    <span>{{ formatBytes(s.rx_bytes) }}</span>
                    <b style="color: var(--cyan); margin-left: 12px; font-family: monospace;">↑ {{ s.rx_speed_kbps.toFixed(1) }} KB/s</b>
                  </td>
                  <td>
                    <span>{{ formatBytes(s.tx_bytes) }}</span>
                    <b style="color: var(--blue); margin-left: 12px; font-family: monospace;">↓ {{ s.tx_speed_kbps.toFixed(1) }} KB/s</b>
                  </td>
                  <td>
                    <button class="mini-btn" style="background: rgba(239, 68, 68, 0.15); color: var(--red); border-color: rgba(239, 68, 68, 0.3);" @click="killSession(s.id)">Disconnect</button>
                  </td>
                </tr>
                <tr v-if="!liveSessions.length">
                  <td colspan="7" class="empty" style="text-align: center; padding: 30px; color: var(--muted);">No active VPN connections right now. Live sessions appear instantly here.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article><article class="glass-card ops-card"><div class="section-head"><div><span class="eyebrow">Operational flow</span><h2>Panel readiness</h2></div><strong>{{ systemScore }}%</strong></div><div class="ops-steps"><div class="done"><i></i><b>Setup</b><small>Owner/session API</small></div><div :class="stats.customers ? 'done' : ''"><i></i><b>Customers</b><small>{{ stats.customers ? 'Radius writes active' : 'Create first user' }}</small></div><div :class="stats.plans ? 'done' : ''"><i></i><b>Plans</b><small>{{ stats.plans ? 'Catalog ready' : 'Create packages' }}</small></div></div></article><article class="glass-card wide-card"><div class="section-head"><div><span class="eyebrow">Recent users</span><h2>Latest customers</h2></div><button class="ghost-btn" @click="section = 'customers'">Manage</button></div><div class="mini-list"><div v-for="customer in customers.slice(0, 6)" :key="customer.id" class="mini-row" @click="openCustomer(customer)"><span class="avatar">{{ customer.username.slice(0, 2).toUpperCase() }}</span><div><b>{{ customer.username }}</b><small>{{ customer.display_name || 'No display name' }}</small></div><em :class="['status-dot', customer.status]">{{ customer.status }}</em></div><p v-if="!customers.length" class="empty">No customers yet. Create the first account.</p></div></article>        <article v-if="user.role === 'reseller'" class="glass-card create-card">
          <div class="section-head">
            <div>
              <span class="eyebrow" style="color: var(--cyan);">Self-service checkout</span>
              <h2>Automatic Top-up</h2>
            </div>
          </div>
          <p class="muted small-text">Top up reseller wallet credits instantly via simulation checkout gateway.</p>
          <form class="form-stack" @submit.prevent="checkoutResellerCredit">
            <label>Top-up Amount (IRT)<input v-model.number="resellerTopupAmount" type="number" min="1000" step="1000" required /></label>
            <button class="primary-btn wide-action" :disabled="busy">Top-up Wallet Credit</button>
          </form>
        </article>

        <article v-if="user.role === 'reseller'" class="glass-card create-card">
          <div class="section-head">
            <div>
              <span class="eyebrow" style="color: var(--cyan);">Bank transfer</span>
              <h2>Submit top-up request</h2>
            </div>
          </div>
          <p class="muted small-text">Submit manual payment transfer receipt to master admin for credit approval.</p>
          <form class="form-stack" @submit.prevent="submitManualResellerPayment">
            <label>Amount (IRT)<input v-model.number="resellerManualPayForm.amount" type="number" min="1000" step="1000" required /></label>
            <label>Receipt details / ID<input v-model.trim="resellerManualPayForm.description" placeholder="Card reference #" required /></label>
            <button class="ghost-btn wide-action" :disabled="busy">Submit Receipt Request</button>
          </form>
        </article>

        <article v-if="user.role === 'reseller'" class="glass-card table-card detail-wide">
          <div class="section-head">
            <div>
              <span class="eyebrow" style="color: var(--cyan);">Receipt history</span>
              <h2>Submitted Top-up Requests</h2>
            </div>
            <button class="ghost-btn" @click="loadResellerPayments">Sync Status</button>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Request ID</th>
                  <th>Amount</th>
                  <th>Status</th>
                  <th>Receipt Details</th>
                  <th>Submitted At</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in resellerPayments" :key="p.id">
                  <td>#{{ p.id }}</td>
                  <td><b>{{ formatMoney(p.amount) }}</b></td>
                  <td><span :class="['pill', p.status === 'approved' ? 'active' : p.status === 'rejected' ? 'disabled' : 'limited']">{{ p.status }}</span></td>
                  <td>{{ p.note }}</td>
                  <td>{{ formatDate(p.created_at) }}</td>
                </tr>
                <tr v-if="!resellerPayments.length">
                  <td colspan="5" class="empty" style="text-align: center; padding: 24px; color: var(--muted);">No manual top-up receipts submitted yet.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>

        <article class="glass-card create-card"><div class="section-head"><div><span class="eyebrow">Quick action</span><h2>New customer</h2></div></div><p class="muted small-text">Create VPN users in a focused popup so the dashboard stays compact.</p><button class="primary-btn wide-action" @click="customerModalOpen = true">+ New customer</button></article></section>

      <section v-else-if="section === 'customers'" class="customers-layout"><article class="glass-card create-panel action-panel"><div class="section-head"><div><span class="eyebrow">Radius user</span><h2>New customer</h2></div></div><p class="muted small-text">Open a focused popup for creating a VPN user with plan defaults, unlimited data/speed options, and Radius policy writes.</p><button class="primary-btn wide-action" @click="customerModalOpen = true">+ New customer</button><div class="modal-hints"><span>0 GB = unlimited data</span><span>0 Mbps = unlimited speed</span><span>Plan values are only defaults</span></div></article><article class="glass-card table-card">
        <!-- Compact Inline View Toggle -->
        <div style="display: flex; gap: 8px; margin-bottom: 16px; background: rgba(0,0,0,0.15); padding: 4px; border-radius: 12px; width: fit-content; border: 1px solid var(--line);">
          <button class="ghost-btn" style="padding: 6px 14px; border: 0; border-radius: 8px; font-weight: 850; cursor: pointer; transition: all 0.2s;" :style="customerView === 'active' ? 'background: linear-gradient(135deg, var(--blue), var(--cyan)); color: #fff;' : 'color: var(--muted); background: none;'" @click="customerView = 'active'">Active ({{ stats.customers }})</button>
          <button class="ghost-btn" style="padding: 6px 14px; border: 0; border-radius: 8px; font-weight: 850; cursor: pointer; transition: all 0.2s;" :style="customerView === 'archived' ? 'background: linear-gradient(135deg, var(--blue), var(--cyan)); color: #fff;' : 'color: var(--muted); background: none;'" @click="customerView = 'archived'">Archived ({{ deletedCustomers.length }})</button>
        </div>

        <div class="section-head table-head"><div><span class="eyebrow">{{ filteredCustomers.length }} records</span><h2>{{ customerView === 'active' ? 'Customers' : 'Archived Deleted Customers' }}</h2></div><div class="search-box"><input v-model="search" @keyup.enter="loadDashboard" placeholder="Search user, name, status…" /><button class="ghost-btn" @click="loadDashboard">Search</button></div></div><div v-if="customerView === 'active'" class="status-summary"><span><b>{{ statusSummary.active || 0 }}</b> active</span><span><b>{{ statusSummary.limited || 0 }}</b> limited</span><span><b>{{ statusSummary.expired || 0 }}</b> expired</span><span><b>{{ statusSummary.disabled || 0 }}</b> disabled</span></div><div class="table-wrap"><table><thead><tr><th>User</th><th>Status</th><th>Plan</th><th>Wallet</th><th>Created</th><th></th></tr></thead><tbody><tr v-for="customer in filteredCustomers" :key="customer.id"><td><div class="identity"><span class="avatar">{{ customer.username.slice(0, 2).toUpperCase() }}</span><div><b>{{ customer.username }}</b><small>{{ customer.display_name || '—' }}</small></div></div></td><td><span :class="['pill', customer.status]">{{ customer.status }}</span></td><td>{{ customer.plan || '—' }}</td><td>{{ formatMoney(customer.credit) }}</td><td>{{ formatDate(customer.created_at) }}</td><td>
          <button v-if="customerView === 'active'" class="mini-btn" @click="openCustomer(customer)">Detail</button>
          <button v-else class="mini-btn" style="background: rgba(34, 211, 238, 0.15); color: var(--cyan); border-color: rgba(34, 211, 238, 0.3);" :disabled="busy" @click="restoreDeletedCustomer(customer as any)">Restore</button>
        </td></tr></tbody></table><p v-if="!filteredCustomers.length" class="empty">No matching customers.</p></div></article></section>

      <section v-else-if="section === 'customer-detail'" class="detail-section"><button class="ghost-btn back-btn" @click="section = 'customers'">← Back to customers</button><p v-if="detailLoading" class="empty">Loading customer detail…</p><div v-else-if="selectedCustomer" class="detail-layout"><article class="glass-card detail-hero" style="grid-column: span 2;"><div><span class="avatar large">{{ selectedCustomer.username.slice(0, 2).toUpperCase() }}</span></div><div><span class="eyebrow">{{ selectedCustomer.status }}</span><h2>{{ selectedCustomer.username }}</h2><p>{{ selectedCustomer.display_name || 'No display name' }} · {{ selectedCustomer.plan || 'No plan' }}</p></div><div class="detail-money"><small>Wallet</small><b>{{ formatMoney(selectedCustomer.credit) }}</b></div></article>

          <!-- Premium Secondary Tabs -->
          <div class="secondary-tabs" style="grid-column: span 2; display: flex; gap: 8px; border-bottom: 1px solid var(--line); padding-bottom: 8px; margin-top: 10px; margin-bottom: 14px;">
            <button class="tab-btn" @click="detailTab = 'profile'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="detailTab === 'profile' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Profile & Wallet</button>
            <button class="tab-btn" @click="detailTab = 'usage'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="detailTab === 'usage' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">VPN Sessions & Usage</button>
            <button class="tab-btn" @click="detailTab = 'history'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="detailTab === 'history' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Ledgers & History</button>
          </div>

          <article v-if="detailTab === 'profile'" class="glass-card detail-card"><div class="section-head"><div><span class="eyebrow">Profile</span><h2>Edit customer</h2></div></div><form class="form-stack" @submit.prevent="saveCustomerDetail"><label>Display name<input v-model.trim="detailForm.display_name" /></label><div class="two-col"><label>Status<select v-model="detailForm.status"><option value="active">active</option><option value="limited">limited</option><option value="expired">expired</option><option value="disabled">disabled</option></select></label><label>Plan<select v-model.number="detailForm.plan_id" @change="applyDetailPlan"><option :value="0">No plan</option><option v-for="plan in plans" :key="plan.id" :value="plan.id">{{ plan.name }}{{ plan.is_active ? '' : ' (inactive)' }}</option></select></label></div><div class="two-col"><label>Max data GB<input v-model.number="detailForm.data_gb" type="number" min="0" step="1" placeholder="0 = unlimited" /></label><label>Speed Mbps<input v-model.number="detailForm.speed_mbps" type="number" min="0" step="1" placeholder="0 = unlimited" /></label></div><label>Add subscription days<input v-model.number="detailForm.days" type="number" min="0" step="1" /></label><label>Notes<textarea v-model.trim="detailForm.notes" placeholder="Internal notes"></textarea></label><button class="primary-btn" :disabled="busy">{{ busy ? 'Saving…' : 'Save detail' }}</button></form></article><article v-if="detailTab === 'profile'" class="glass-card detail-card"><div class="section-head"><div><span class="eyebrow">Renewal</span><h2>Apply plan</h2></div></div><form class="form-stack" @submit.prevent="renewCustomerPlan"><label>Plan<select v-model.number="renewForm.plan_id"><option :value="0">Select plan</option><option v-for="plan in activePlans" :key="plan.id" :value="plan.id">{{ plan.name }} · {{ formatMoney(plan.price) }}</option></select></label><div v-if="selectedRenewPlan" class="renew-summary"><span><b>{{ formatGB(selectedRenewPlan.data_gb) }}</b><small>data</small></span><span><b>{{ formatSpeed(selectedRenewPlan.speed_mbps) }}</b><small>speed</small></span><span><b>{{ selectedRenewPlan.duration_days }}d</b><small>duration</small></span><span><b>{{ formatMoney(selectedRenewPlan.price) }}</b><small>wallet charge</small></span></div><button class="primary-btn" :disabled="busy || !renewForm.plan_id">{{ busy ? 'Applying…' : 'Apply / renew plan' }}</button><small class="muted">Paid plans deduct from wallet immediately. Pay as you go costs 0.</small></form></article><article v-if="detailTab === 'profile'" class="glass-card detail-card"><div class="section-head"><div><span class="eyebrow">Access & wallet</span><h2>Password / funds</h2></div></div><form class="form-stack" @submit.prevent="resetCustomerPassword"><label>New VPN password<input v-model="passwordForm.password" placeholder="New password" /></label><button class="primary-btn" :disabled="busy">Reset password</button></form><form class="form-stack wallet-mini" @submit.prevent="setWalletBalance"><label>Set wallet balance<input v-model.number="walletSetForm.balance" type="number" step="1" placeholder="exact balance" /></label><button class="primary-btn" :disabled="busy">Save balance</button></form><form class="form-stack wallet-mini" @submit.prevent="adjustWallet"><label>Adjust by amount<input v-model.number="walletForm.amount" type="number" step="1" placeholder="positive or negative" /></label><button class="ghost-btn" :disabled="busy">Apply adjustment</button></form><div class="action-row"><button class="ghost-btn" :disabled="busy" @click="setSelectedCustomerStatus('active')">Enable</button><button class="danger-btn" :disabled="busy" @click="setSelectedCustomerStatus('disabled')">Disable</button><button class="danger-btn" :disabled="busy" @click="archiveSelectedCustomer">Archive</button></div><p class="muted small-text">Subscription: {{ subscriptionText(selectedCustomer) }}</p>
<div v-if="selectedCustomer?.sub_token" style="margin-top: 14px; padding-top: 10px; border-top: 1px solid var(--line);">
  <span class="eyebrow" style="color: var(--cyan); font-size: 10px; display: block; margin-bottom: 6px;">Subscriber URL (Unified Config Link)</span>
  <div style="display: flex; gap: 8px; align-items: center;">
    <input readonly :value="panelOrigin + '/portal/sub/' + selectedCustomer.sub_token" style="font-family: monospace; font-size: 12px; background: rgba(0,0,0,0.2); flex: 1; padding: 6px 10px; border-radius: 8px; border: 1px solid var(--line); color: #fff;" />
    <button class="mini-btn ghost-btn" @click="copyToClipboard(panelOrigin + '/portal/sub/' + selectedCustomer.sub_token)" style="white-space: nowrap; height: 32px; border-radius: 8px; padding: 0 12px;">Copy Link</button>
  </div>
</div>
</article><article v-if="detailTab === 'usage'" class="glass-card table-card detail-wide"><div class="section-head"><div><span class="eyebrow">Usage</span><h2>VPN usage & sessions</h2></div><span v-if="selectedUsage" :class="['pill', selectedUsage.online ? 'active' : 'disabled']">{{ selectedUsage.online ? 'online' : 'offline' }}</span></div><div v-if="selectedUsage" class="usage-summary"><span><b>{{ formatBytes(selectedUsage.total_usage_bytes) }}</b><small>total</small></span><span><b>{{ formatBytes(selectedUsage.total_input_bytes) }}</b><small>download</small></span><span><b>{{ formatBytes(selectedUsage.total_output_bytes) }}</b><small>upload</small></span><span><b>{{ selectedUsage.remaining_bytes === undefined ? 'Unlimited' : formatBytes(selectedUsage.remaining_bytes) }}</b><small>remaining</small></span><span><b>{{ selectedUsage.active_sessions }}</b><small>active sessions</small></span></div><div v-if="selectedUsage" class="table-wrap"><table><thead><tr><th>ID</th><th>Status</th><th>IP</th><th>Duration</th><th>Down</th><th>Up</th><th>Started</th><th>Stopped</th></tr></thead><tbody><tr v-for="session in selectedUsage.sessions" :key="session.id"><td>#{{ session.id }}</td><td><span :class="['pill', session.online ? 'active' : 'disabled']">{{ session.online ? 'online' : 'closed' }}</span></td><td>{{ session.framed_ip || '—' }}</td><td>{{ formatDuration(session.session_seconds) }}</td><td>{{ formatBytes(session.input_bytes) }}</td><td>{{ formatBytes(session.output_bytes) }}</td><td>{{ formatDate(session.start_time) }}</td><td>{{ formatDate(session.stop_time) }}</td></tr></tbody></table><p v-if="!selectedUsage.sessions.length" class="empty">No VPN sessions yet.</p></div></article><article v-if="detailTab === 'history'" class="glass-card table-card detail-wide"><div class="section-head"><div><span class="eyebrow">Wallet</span><h2>Transaction history</h2></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>Amount</th><th>Type</th><th>Description</th><th>Actor</th><th>Date</th></tr></thead><tbody><tr v-for="tx in (selectedCustomer.wallet_transactions || [])" :key="tx.id"><td>#{{ tx.id }}</td><td><b :class="tx.amount >= 0 ? 'amount-plus' : 'amount-minus'">{{ signedMoney(tx.amount) }}</b></td><td><span class="pill limited">{{ tx.type }}</span></td><td>{{ tx.description || '—' }}</td><td>{{ tx.actor || '—' }}</td><td>{{ formatDate(tx.created_at) }}</td></tr></tbody></table><p v-if="!selectedCustomer.wallet_transactions || !selectedCustomer.wallet_transactions.length" class="empty">No wallet transactions yet.</p></div></article><article v-if="detailTab === 'history'" class="glass-card table-card detail-wide"><div class="section-head"><div><span class="eyebrow">Subscriptions</span><h2>Subscription history</h2></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>Plan</th><th>Status</th><th>Paid</th><th>Started</th><th>Expires</th></tr></thead><tbody><tr v-for="sub in (selectedCustomer.subscriptions || [])" :key="sub.id"><td>#{{ sub.id }}</td><td>{{ sub.plan || '—' }}</td><td><span :class="['pill', sub.status === 'active' ? 'active' : sub.status === 'expired' ? 'disabled' : 'limited']">{{ sub.status }}</span></td><td>{{ formatMoney(sub.paid_amount) }}</td><td>{{ formatDate(sub.started_at) }}</td><td>{{ formatDate(sub.expires_at) }}</td></tr></tbody></table><p v-if="!selectedCustomer.subscriptions || !selectedCustomer.subscriptions.length" class="empty">No subscription history yet.</p></div></article><article v-if="detailTab === 'history'" class="glass-card table-card detail-wide"><div class="section-head"><div><span class="eyebrow">FreeRADIUS</span><h2>radcheck / radreply</h2></div></div><div class="table-wrap"><table><thead><tr><th>Table</th><th>Attribute</th><th>Op</th><th>Value</th></tr></thead><tbody><tr v-for="check in (selectedCustomer.radius_checks || [])" :key="`c-${check.id}`"><td>radcheck</td><td>{{ check.attribute }}</td><td>{{ check.op }}</td><td><code>{{ check.value }}</code></td></tr><tr v-for="reply in (selectedCustomer.radius_replies || [])" :key="`r-${reply.id}`"><td>radreply</td><td>{{ reply.attribute }}</td><td>{{ reply.op }}</td><td><code>{{ reply.value }}</code></td></tr></tbody></table><p v-if="(!selectedCustomer.radius_checks || !selectedCustomer.radius_checks.length) && (!selectedCustomer.radius_replies || !selectedCustomer.radius_replies.length)" class="empty">No radius policy rows for this customer.</p></div></article></div></section>

      <section v-else-if="section === 'plans'" class="plans-layout"><article class="glass-card plan-editor action-panel"><div class="section-head"><div><span class="eyebrow">Catalog</span><h2>Plan actions</h2></div></div><p class="muted small-text">Create and edit plans in a focused popup. Prices are free-form IRT amounts.</p><button class="primary-btn wide-action" @click="openNewPlan">+ New plan</button><div class="modal-hints"><span>0 GB = unlimited</span><span>0 Mbps = unlimited</span><span>Price is IRT</span></div></article><article class="glass-card table-card"><div class="section-head"><div><span class="eyebrow">{{ plans.length }} plans</span><h2>Plans CRUD</h2></div></div><div class="plan-grid"><div v-for="plan in plans" :key="plan.id" :class="['plan-card', { inactive: !plan.is_active }]"><div class="section-head"><div><span :class="['pill', plan.is_active ? 'active' : 'disabled']">{{ plan.is_active ? 'active' : 'inactive' }}</span><h3>{{ plan.name }}</h3></div></div><div class="plan-numbers"><span><b>{{ formatGB(plan.data_gb) }}</b><small>data</small></span><span><b>{{ formatSpeed(plan.speed_mbps) }}</b><small>speed</small></span><span><b>{{ plan.duration_days }}d</b><small>duration</small></span><span><b>{{ formatMoney(plan.price) }}</b><small>price</small></span></div><div class="action-row"><button class="ghost-btn" @click="editPlan(plan)">Edit</button><button class="danger-btn" :disabled="!plan.is_active" @click="archivePlan(plan)">Deactivate</button></div></div><p v-if="!plans.length" class="empty">No plans yet. Create the first package.</p></div></article></section>

      <section v-else-if="section === 'nodes'" class="nodes-layout">
        <!-- Infrastructure Sub-Tabs -->
        <div class="secondary-tabs" style="grid-column: span 2; display: flex; gap: 8px; border-bottom: 1px solid var(--line); padding-bottom: 8px; margin-bottom: 14px;">
          <button class="tab-btn" @click="infraTab = 'nodes'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="infraTab === 'nodes' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Nodes Status</button>
          <button class="tab-btn" @click="infraTab = 'vpn'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="infraTab === 'vpn' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">OpenVPN Core Settings</button>
        </div>

        <template v-if="infraTab === 'nodes'">
          <article class="glass-card create-panel action-panel"><div class="section-head"><div><span class="eyebrow">Node fleet</span><h2>Management</h2></div></div><p class="muted small-text">Create nodes, copy their token once, and let node agents push status through HTTP. No SSH dependency.</p><button class="primary-btn wide-action" @click="nodeModalOpen = true; resetNodeForm()">+ New node</button><div class="modal-hints"><span>HTTP push only</span><span>Token shown once</span><span>Stale after 2 minutes</span></div></article>
          <article class="glass-card table-card"><div class="section-head"><div><span class="eyebrow">{{ nodes.length }} nodes</span><h2>Node status</h2></div><button class="ghost-btn" @click="loadDashboard">Sync</button></div><div class="node-grid"><div v-for="node in nodes" :key="node.id" class="node-card"><div class="section-head"><div><span :class="['pill', node.status === 'online' ? 'active' : node.status === 'disabled' ? 'disabled' : 'limited']">{{ node.status }}</span><h3>{{ node.name }}</h3><small>{{ node.public_ip }} {{ node.domain ? '· ' + node.domain : '' }}</small></div></div><div class="node-metrics"><span><b>{{ Math.round(node.status_metrics?.cpu_percent || 0) }}%</b><small>CPU</small></span><span><b>{{ Math.round(node.status_metrics?.ram_percent || 0) }}%</b><small>RAM</small></span><span><b>{{ Math.round(node.status_metrics?.disk_percent || 0) }}%</b><small>Disk</small></span><span><b>{{ bps(node.status_metrics?.rx_bps) }}</b><small>RX</small></span><span><b>{{ bps(node.status_metrics?.tx_bps) }}</b><small>TX</small></span></div><div class="status-summary service-summary"><span><b>OpenVPN</b> {{ serviceLabel(node, 'openvpn') }}</span><span><b>L2TP</b> {{ serviceLabel(node, 'l2tp') }}</span><span><b>IKEv2</b> {{ serviceLabel(node, 'ikev2') }}</span></div>
            <div v-if="node.history && node.history.length" style="margin-top: 12px; border-top: 1px solid var(--line); padding-top: 10px; margin-bottom: 8px;">
              <span class="eyebrow" style="color: var(--cyan); font-size: 9px; display: block; margin-bottom: 6px;">Bandwidth Analytics (History)</span>
              <div style="height: 44px; width: 100%;">
                <svg viewBox="0 0 150 40" style="width: 100%; height: 100%; overflow: visible;">
                  <polyline fill="none" stroke="var(--cyan)" stroke-width="1.8" :points="nodeHistoryPoints(node.history)" />
                </svg>
              </div>
            </div>
            <p class="muted small-text">Last seen: {{ formatDate(node.last_seen_at) }}</p><div class="action-row"><button class="ghost-btn" @click="createNodeTask(node, 'agent.status')">Ping agent</button><button class="ghost-btn" @click="createNodeTask(node, 'service.restart', { service: 'openvpn' })">Restart OpenVPN</button><button class="ghost-btn" @click="createNodeTask(node, 'service.restart', { service: 'l2tp' })">Restart L2TP</button><button class="ghost-btn" @click="createNodeTask(node, 'service.restart', { service: 'ikev2' })">Restart IKEv2</button><button class="ghost-btn" @click="rotateNodeToken(node)">Rotate token</button><button v-if="node.status === 'disabled'" class="ghost-btn" @click="setNodeEnabled(node, true)">Enable</button><button v-else class="danger-btn" @click="setNodeEnabled(node, false)">Disable</button></div></div><p v-if="!nodes.length" class="empty">No nodes yet. Create a node and run the node agent with its token.</p></div></article>
          <article class="glass-card table-card detail-wide"><div class="section-head"><div><span class="eyebrow">{{ nodeTasks.length }} tasks</span><h2>Recent node tasks</h2></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>Node</th><th>Action</th><th>Status</th><th>Error</th><th>Created</th></tr></thead><tbody><tr v-for="task in nodeTasks.slice(0, 20)" :key="task.id"><td>#{{ task.id }}</td><td>{{ task.node_name || task.node_id }}</td><td>{{ task.action }}</td><td><span :class="['pill', task.status === 'succeeded' ? 'active' : task.status === 'failed' ? 'disabled' : 'limited']">{{ task.status }}</span></td><td>{{ task.error || '—' }}</td><td>{{ formatDate(task.created_at) }}</td></tr></tbody></table><p v-if="!nodeTasks.length" class="empty">No node tasks yet.</p></div></article>
        </template>

        <template v-else-if="infraTab === 'vpn'">
          <!-- OpenVPN Core Settings Stacked cleanly to prevent side-by-side clutter -->
          <article class="glass-card plan-editor" style="grid-column: span 2;"><div class="section-head"><div><span class="eyebrow">OpenVPN</span><h2>Core settings</h2></div></div><form class="form-stack" @submit.prevent="saveVPNSettings(false)"><div class="two-col"><label>Port<input v-model.number="vpnForm.openvpn_port" type="number" min="1" max="65535" /></label><label>Protocol<select v-model="vpnForm.openvpn_protocol"><option value="udp">udp</option><option value="tcp">tcp</option></select></label></div><label>OpenVPN network<input v-model.trim="vpnForm.openvpn_network" placeholder="10.8.0.0/24" /></label><div class="two-col"><label>DNS 1<input v-model.trim="vpnForm.dns_1" /></label><label>DNS 2<input v-model.trim="vpnForm.dns_2" /></label></div><div class="two-col"><label>L2TP network<input v-model.trim="vpnForm.l2tp_network" /></label><label>IKEv2 network<input v-model.trim="vpnForm.ikev2_network" /></label></div><label>IPSec PSK<input v-model.trim="vpnForm.ipsec_psk" placeholder="optional" /></label><div class="action-row"><button class="primary-btn" :disabled="busy">Save DB settings</button><button class="danger-btn" type="button" :disabled="busy" @click="saveVPNSettings(true)">Save & restart OpenVPN</button></div></form></article>
          <article class="glass-card table-card detail-wide" style="grid-column: span 2; margin-top: 14px;"><div class="section-head"><div><span class="eyebrow">Runtime</span><h2>Status & files</h2></div><span :class="['pill', vpnSettings?.openvpn_service_status === 'active' ? 'active' : 'disabled']">{{ vpnSettings?.openvpn_service_status || 'unknown' }}</span></div><div class="vpn-status-grid"><span><b>{{ vpnSettings?.remote_host || '—' }}</b><small>remote host</small></span><span><b>{{ vpnSettings?.active_node || '—' }}</b><small>active node</small></span><span><b>{{ vpnSettings?.ca_exists ? 'OK' : 'Missing' }}</b><small>CA: {{ vpnSettings?.ca_file || '—' }}</small></span><span><b>{{ vpnSettings?.tls_crypt_exists ? 'OK' : 'Missing' }}</b><small>tls-crypt: {{ vpnSettings?.tls_crypt_file || '—' }}</small></span><span><b>{{ vpnSettings?.openvpn_port }}/{{ vpnSettings?.openvpn_protocol }}</b><small>profile endpoint</small></span><span><b>{{ vpnSettings?.updated_at ? formatDate(vpnSettings.updated_at) : '—' }}</b><small>updated</small></span></div><p class="muted small-text">Save DB settings updates generated profiles. Save & restart also rewrites OpenVPN server.conf and restarts OpenVPN.</p></article>
        </template>
      </section>

      <section v-else-if="section === 'system'" class="customers-layout">
        <!-- System Sub-Tabs -->
        <div class="secondary-tabs" style="grid-column: span 2; display: flex; gap: 8px; border-bottom: 1px solid var(--line); padding-bottom: 8px; margin-bottom: 14px;">
          <button class="tab-btn" @click="systemTab = 'diagnostics'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="systemTab === 'diagnostics' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Diagnostics</button>
          <button class="tab-btn" @click="systemTab = 'audit'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="systemTab === 'audit' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Audit Logs</button>
          <button class="tab-btn" @click="systemTab = 'backups'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 800; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s;" :style="systemTab === 'backups' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">CSV Backups</button>
        </div>

        <template v-if="systemTab === 'diagnostics'">
          <article class="glass-card create-panel action-panel">
            <div class="section-head"><div><span class="eyebrow">System</span><h2>Diagnostics</h2></div></div>
            <p class="muted small-text">System health, service status, and listening ports.</p>
            <div class="status-summary" style="display: flex; flex-direction: column; gap: 8px; margin: 12px 0;">
              <span>Disk: <b>{{ diagnosticsData?.disk || 'N/A' }}</b></span>
              <span>Memory: <b>{{ diagnosticsData?.mem || 'N/A' }}</b></span>
            </div>
            <button class="primary-btn wide-action" :disabled="diagnosticsLoading" @click="loadDiagnostics">
              {{ diagnosticsLoading ? 'Checking…' : 'Run check' }}
            </button>
          </article>
          <article class="glass-card table-card" style="grid-column: span 1;">
            <div class="section-head"><div><span class="eyebrow">Checks</span><h2>Services Status</h2></div></div>
            <div class="table-wrap">
              <table>
                <thead>
                  <tr>
                    <th>Check</th>
                    <th>Status</th>
                    <th>Details</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="c in diagnosticsData?.checks" :key="c.name">
                    <td><b>{{ c.name }}</b></td>
                    <td>
                      <span :class="['pill', c.ok ? 'active' : 'disabled']">
                        {{ c.ok ? 'OK' : 'Issue' }}
                      </span>
                    </td>
                    <td><code>{{ c.detail }}</code></td>
                  </tr>
                </tbody>
              </table>
              <p v-if="diagnosticsLoading" class="empty">Running diagnostic checks…</p>
              <p v-else-if="!diagnosticsData" class="empty">No diagnostics executed yet. Click "Run check" to execute.</p>
            </div>
            <div v-if="diagnosticsData?.ports" style="margin-top: 18px; padding: 12px; border-top: 1px solid var(--line);">
              <h3 style="margin-top: 0;">Listening ports</h3>
              <pre class="code-block" style="white-space: pre-wrap; font-size: 12px; font-family: monospace; color: var(--muted);">{{ diagnosticsData?.ports }}</pre>
            </div>
          </article>
        </template>

        <template v-else-if="systemTab === 'audit'">
          <article class="glass-card table-card detail-wide" style="grid-column: span 2;"><div class="section-head"><div><span class="eyebrow">{{ auditLogs.length }} records</span><h2>Audit logs</h2></div><div class="action-row compact-actions"><button class="ghost-btn" @click="auditOffset = Math.max(0, auditOffset - auditLimit); loadAuditLogs()">Prev</button><button class="ghost-btn" @click="auditOffset += auditLimit; loadAuditLogs()">Next</button></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>Actor</th><th>Action</th><th>Entity</th><th>Entity ID</th><th>IP</th><th>Before</th><th>After</th><th>Created</th></tr></thead><tbody><tr v-for="log in auditLogs" :key="log.id"><td>#{{ log.id }}</td><td>{{ log.actor }}</td><td><span class="pill limited">{{ log.action }}</span></td><td>{{ log.entity_type }}</td><td>{{ log.entity_id }}</td><td>{{ log.ip }}</td><td><pre class="code-block">{{ log.before_json }}</pre></td><td><pre class="code-block">{{ log.after_json }}</pre></td><td>{{ formatDate(log.created_at) }}</td></tr></tbody></table><p v-if="auditLoading" class="empty">Loading audit logs…</p><p v-else-if="!auditLogs.length" class="empty">No audit logs yet.</p></div></article>
        </template>

        <template v-else-if="systemTab === 'backups'">
          <article class="glass-card create-panel action-panel" style="grid-column: span 2;"><div class="section-head"><div><span class="eyebrow">Data</span><h2>Exports & backups</h2></div></div><p class="muted small-text">Download CSV snapshots directly from the database. Daily SQL backups are handled by the panel worker.</p><div class="action-row compact-actions"><button class="primary-btn" @click="exportCSV('customers')">Customers CSV</button><button class="primary-btn" @click="exportCSV('payments')">Payments CSV</button><button class="primary-btn" @click="exportCSV('radacct')">RADIUS CSV</button><button class="primary-btn" @click="exportCSV('wallet-transactions')">Wallet CSV</button></div></article>
        </template>
      </section>

      <section v-else-if="section === 'resellers'" class="customers-layout">
        <article class="glass-card create-panel action-panel">
          <div class="section-head"><div><span class="eyebrow">Ecosystem</span><h2>New Reseller</h2></div></div>
          <p class="muted small-text">Create sub-admin resellers. Resellers can create and manage their own customers using allocated credit.</p>
          <form class="form-stack" @submit.prevent="createReseller">
            <label>Username<input v-model.trim="resellerForm.username" required placeholder="reseller username" /></label>
            <label>Password<input v-model="resellerForm.password" required placeholder="reseller password" type="password" /></label>
            <button class="primary-btn wide-action" :disabled="busy">Create Reseller</button>
          </form>
        </article>
        <article class="glass-card table-card" style="grid-column: span 1;">
          <div class="section-head"><div><span class="eyebrow">{{ resellersList.length }} sub-admins</span><h2>Resellers Fleet</h2></div></div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Username</th>
                  <th>Credit</th>
                  <th>Status</th>
                  <th>Created</th>
                  <th>Credit Adjustment</th>
                  <th>Actions</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="r in resellersList" :key="r.id">
                  <td><b>{{ r.username }}</b></td>
                  <td><b>{{ formatMoney(r.credit) }}</b></td>
                  <td><span :class="['pill', r.is_active ? 'active' : 'disabled']">{{ r.is_active ? 'Active' : 'Inactive' }}</span></td>
                  <td>{{ formatDate(r.created_at) }}</td>
                  <td>
                    <div style="display: flex; gap: 6px; align-items: center;">
                      <input v-model.number="resellerCreditForm[r.id]" type="number" step="1000" style="width: 100px; min-height: 32px;" placeholder="Amount" />
                      <button class="mini-btn" @click="adjustResellerCredit(r.id, true)">Add</button>
                      <button class="mini-btn" style="background: rgba(239, 68, 68, 0.15);" @click="adjustResellerCredit(r.id, false)">Sub</button>
                    </div>
                  </td>
                  <td>
                    <button class="danger-btn mini-btn" @click="deleteReseller(r.id)">Delete</button>
                  </td>
                </tr>
                <tr v-if="!resellersList.length">
                  <td colspan="6" class="empty">No resellers registered. Create your first reseller using the form on the left.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>

        <article class="glass-card table-card detail-wide" style="grid-column: span 2; margin-top: 20px;">
          <div class="section-head">
            <div>
              <span class="eyebrow" style="color: var(--cyan);">Audit Ledger</span>
              <h2>Reseller Credit Transactions Log</h2>
            </div>
            <button class="ghost-btn" @click="loadResellerTxs">Sync Logs</button>
          </div>
          <div class="table-wrap">
            <table>
              <thead>
                <tr>
                  <th>Log ID</th>
                  <th>Reseller</th>
                  <th>Amount Adjustment</th>
                  <th>Type</th>
                  <th>Activity Details</th>
                  <th>Actor</th>
                  <th>Timestamp</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="t in resellerTxs" :key="t.id">
                  <td>#{{ t.id }}</td>
                  <td><b>{{ t.reseller_username }}</b></td>
                  <td><b :class="t.amount >= 0 ? 'amount-plus' : 'amount-minus'">{{ signedMoney(t.amount) }}</b></td>
                  <td><span :class="['pill', t.type === 'allocation' ? 'active' : 'disabled']">{{ t.type }}</span></td>
                  <td>{{ t.description }}</td>
                  <td>{{ t.actor }}</td>
                  <td>{{ formatDate(t.created_at) }}</td>
                </tr>
                <tr v-if="!resellerTxs.length">
                  <td colspan="7" class="empty" style="text-align: center; padding: 24px; color: var(--muted);">No credit transactions recorded in the audit ledger yet.</td>
                </tr>
              </tbody>
            </table>
          </div>
        </article>
      </section>

      <section v-else-if="section === 'payments'" class="payments-layout"><article class="glass-card plan-editor"><div class="section-head"><div><span class="eyebrow">Manual</span><h2>Record payment</h2></div></div><form class="form-stack" @submit.prevent="createManualPayment"><label>Username<input v-model.trim="paymentForm.username" required placeholder="customer username" /></label><label>Amount<input v-model.number="paymentForm.amount" type="number" min="0" step="1" required /></label><label>Method<select v-model="paymentForm.method"><option value="manual">manual</option><option v-for="method in paymentMethods.filter(m => m.is_active)" :key="method.id" :value="method.name">{{ method.name }}</option></select></label><label>Description<textarea v-model.trim="paymentForm.description" placeholder="Receipt or admin note"></textarea></label><button class="primary-btn" :disabled="busy">Record approved payment</button></form><div class="method-editor"><div class="section-head"><div><span class="eyebrow">Methods</span><h2>{{ editingMethodId ? 'Edit method' : 'New method' }}</h2></div><button v-if="editingMethodId" class="ghost-btn" @click="resetMethodForm">Cancel</button></div><form class="form-stack" @submit.prevent="savePaymentMethod"><label>Name<input v-model.trim="methodForm.name" placeholder="Bank transfer" required /></label><label>Type<input v-model.trim="methodForm.type" placeholder="manual / card / crypto" /></label><label>Instructions<textarea v-model.trim="methodForm.instructions" placeholder="Payment instructions shown in portal"></textarea></label><div class="two-col"><label>Sort<input v-model.number="methodForm.sort_order" type="number" step="1" /></label><label class="check-line"><input v-model="methodForm.is_active" type="checkbox" /> Active</label></div><button class="primary-btn" :disabled="busy">{{ editingMethodId ? 'Update method' : 'Create method' }}</button></form><div class="method-list"><div v-for="method in paymentMethods" :key="method.id" class="method-row"><div><b>{{ method.name }}</b><small>{{ method.type }} · {{ method.is_active ? 'active' : 'inactive' }}</small></div><div class="action-row compact-actions"><button class="mini-btn" @click="editPaymentMethod(method)">Edit</button><button class="mini-btn" :disabled="!method.is_active" @click="deactivatePaymentMethod(method)">Disable</button></div></div></div></div></article><article class="glass-card table-card"><div class="section-head"><div><span class="eyebrow">{{ payments.length }} payments</span><h2>Payment ledger</h2></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>User</th><th>Amount</th><th>Method</th><th>Intent</th><th>Status</th><th>Created</th><th></th></tr></thead><tbody><tr v-for="payment in payments" :key="payment.id"><td>#{{ payment.id }}</td><td>{{ payment.username }}</td><td>{{ formatMoney(payment.amount) }}</td><td>{{ payment.method }}</td><td><span class="pill limited">{{ payment.intent_type === 'plan_renewal' ? `renew ${payment.intent_label || payment.intent_id}` : 'wallet topup' }}</span></td><td><span :class="['pill', payment.status === 'approved' ? 'active' : payment.status === 'rejected' ? 'disabled' : 'limited']">{{ payment.status }}</span></td><td>{{ formatDate(payment.created_at) }}</td><td><div class="action-row compact-actions"><button class="mini-btn" :disabled="payment.status === 'approved'" @click="approvePayment(payment, 'approve')">{{ payment.intent_type === 'plan_renewal' ? 'Approve & renew' : 'Approve' }}</button><button class="mini-btn" :disabled="payment.status === 'rejected'" @click="approvePayment(payment, 'reject')">Reject</button></div></td></tr></tbody></table><p v-if="!payments.length" class="empty">No payments yet.</p></div></article></section>

      <section v-else-if="section === 'tickets'" class="tickets-layout"><article class="glass-card create-panel action-panel"><div class="section-head"><div><span class="eyebrow">Support</span><h2>New ticket</h2></div></div><form class="form-stack" @submit.prevent="createAdminTicket"><label>Username<input v-model.trim="adminTicketForm.username" required placeholder="customer username" /></label><label>Subject<input v-model.trim="adminTicketForm.subject" required placeholder="Issue subject" /></label><label>Priority<select v-model="adminTicketForm.priority"><option value="low">low</option><option value="normal">normal</option><option value="high">high</option></select></label><label>Message<textarea v-model.trim="adminTicketForm.message" required placeholder="Initial message"></textarea></label><button class="primary-btn" :disabled="busy">Create ticket</button></form></article><article class="glass-card table-card"><div class="section-head"><div><span class="eyebrow">{{ tickets.length }} tickets</span><h2>Support queue</h2></div></div><div class="table-wrap"><table><thead><tr><th>ID</th><th>User</th><th>Subject</th><th>Priority</th><th>Status</th><th>Updated</th><th></th></tr></thead><tbody><tr v-for="ticket in tickets" :key="ticket.id"><td>#{{ ticket.id }}</td><td>{{ ticket.username }}</td><td>{{ ticket.subject }}</td><td><span class="pill limited">{{ ticket.priority }}</span></td><td><span :class="['pill', ticket.status === 'open' ? 'active' : 'disabled']">{{ ticket.status }}</span></td><td>{{ formatDate(ticket.updated_at) }}</td><td><button class="mini-btn" @click="loadTicket(ticket.id)">Open</button></td></tr></tbody></table><p v-if="!tickets.length" class="empty">No tickets yet.</p></div></article><article v-if="selectedTicket" class="glass-card table-card detail-wide"><div class="section-head"><div><span class="eyebrow">Ticket #{{ selectedTicket.id }}</span><h2>{{ selectedTicket.subject }}</h2><small>{{ selectedTicket.username }} · {{ selectedTicket.priority }}</small></div><div class="action-row"><button v-if="selectedTicket.status === 'open'" class="danger-btn" @click="setTicketStatus(selectedTicket, 'closed')">Close</button><button v-else class="ghost-btn" @click="setTicketStatus(selectedTicket, 'open')">Reopen</button></div></div><div class="ticket-thread"><div v-for="message in selectedTicket.messages" :key="message.id" :class="['ticket-message', message.sender_type]"><b>{{ message.sender_name }} <small>{{ message.sender_type }} · {{ formatDate(message.created_at) }}</small></b><p>{{ message.message }}</p></div></div><form class="form-stack ticket-reply" @submit.prevent="replyTicket"><label>Reply<textarea v-model.trim="ticketReply" placeholder="Write a reply"></textarea></label><button class="primary-btn" :disabled="busy || !ticketReply.trim()">Send reply</button></form></article></section>

      <div v-if="customerModalOpen" class="modal-backdrop" @click.self="customerModalOpen = false">
        <section class="modal-card glass-card" role="dialog" aria-modal="true" aria-label="Create new customer">
          <div class="section-head modal-head"><div><span class="eyebrow">Radius user</span><h2>New customer</h2></div><button class="icon-btn" aria-label="Close" @click="customerModalOpen = false">×</button></div>
          <form class="form-stack" @submit.prevent="createCustomer">
            <div class="two-col"><label>Username<input v-model.trim="createForm.username" required placeholder="customer username" /></label><label>Password<input v-model="createForm.password" required placeholder="VPN password" /></label></div>
            <label>Display name<input v-model.trim="createForm.display_name" placeholder="Optional name" /></label>
            <label>Plan<select v-model.number="createForm.plan_id" @change="applyCreatePlan"><option :value="0">No plan / custom</option><option v-for="plan in activePlans" :key="plan.id" :value="plan.id">{{ plan.name }} · {{ formatGB(plan.data_gb) }} · {{ formatSpeed(plan.speed_mbps) }}</option></select></label>
            <div class="two-col"><label>Data GB<input v-model.number="createForm.data_gb" type="number" min="0" step="1" placeholder="0 = unlimited" /></label><label>Speed Mbps<input v-model.number="createForm.speed_mbps" type="number" min="0" step="1" placeholder="0 = unlimited" /></label></div>
            <label>Duration days<input v-model.number="createForm.days" type="number" min="0" step="1" /></label>
            <div class="modal-hints"><span>Plan values are defaults only</span><span>0 GB removes Max-Data</span><span>0 Mbps removes rate-limit</span></div>
            <div class="action-row"><button class="primary-btn" :disabled="busy">{{ busy ? 'Creating…' : 'Create customer' }}</button><button class="ghost-btn" type="button" @click="customerModalOpen = false">Cancel</button></div>
          </form>
        </section>
      </div>
      <div v-if="nodeModalOpen" class="modal-backdrop" @click.self="nodeModalOpen = false">
        <section class="modal-card glass-card" role="dialog" aria-modal="true" aria-label="Create node">
          <div class="section-head modal-head"><div><span class="eyebrow">Node agent</span><h2>New node</h2></div><button class="icon-btn" aria-label="Close" @click="nodeModalOpen = false">×</button></div>
          <form class="form-stack" @submit.prevent="createNode">
            <div class="two-col"><label>Name<input v-model.trim="nodeForm.name" required placeholder="node-1" /></label><label>Public IP<input v-model.trim="nodeForm.public_ip" required placeholder="203.0.113.10" /></label></div>
            <label>Domain<input v-model.trim="nodeForm.domain" placeholder="optional node domain" /></label>
            <button class="primary-btn" :disabled="busy">{{ busy ? 'Creating…' : 'Create node token' }}</button>
          </form>
          <div v-if="nodeToken" class="token-box"><span class="eyebrow">Copy token now</span><code>{{ nodeToken }}</code><small>This token is shown once. Keep it private.</small><span class="eyebrow">Install command</span><code class="install-command">{{ nodeInstallCommand }}</code><small>Run this on the node server from an extracted koris-next source directory.</small></div>
        </section>
      </div>
      <div v-if="planModalOpen" class="modal-backdrop" @click.self="planModalOpen = false">
        <section class="modal-card glass-card" role="dialog" aria-modal="true" aria-label="Create or edit plan">
          <div class="section-head modal-head"><div><span class="eyebrow">Plan catalog</span><h2>{{ editingPlanId ? 'Edit plan' : 'New plan' }}</h2></div><button class="icon-btn" aria-label="Close" @click="planModalOpen = false">×</button></div>
          <form class="form-stack" @submit.prevent="savePlan">
            <label>Plan name<input v-model.trim="planForm.name" required placeholder="30GB / 30 days" /></label>
            <div class="two-col"><label>Data GB<input v-model.number="planForm.data_gb" type="number" min="0" step="1" placeholder="0 = unlimited" /></label><label>Speed Mbps<input v-model.number="planForm.speed_mbps" type="number" min="0" step="1" placeholder="0 = unlimited" /></label></div>
            <div class="two-col"><label>Days<input v-model.number="planForm.duration_days" type="number" min="0" step="1" /></label><label>Price IRT<input v-model.number="planForm.price" type="number" min="0" step="1" placeholder="Any IRT amount" /></label></div>
            <label>Sort order<input v-model.number="planForm.sort_order" type="number" step="1" /></label>
            <label class="check-line"><input v-model="planForm.is_active" type="checkbox" /> Active plan</label>
            <div class="action-row"><button class="primary-btn" :disabled="busy">{{ busy ? 'Saving…' : editingPlanId ? 'Update plan' : 'Create plan' }}</button><button class="ghost-btn" type="button" @click="planModalOpen = false">Cancel</button></div>
          </form>
        </section>
      </div>
    </section>
  </main>
</template>
