<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'

type Screen = 'loading' | 'login' | 'portal'

interface PortalCustomer {
  id?: number
  username: string
  display_name?: string
  status?: string
  plan?: string
  credit?: number
  created_at?: string
  max_data_bytes?: string
  sub_token?: string
  subscription?: {
    plan?: string
    status?: string
    expires_at?: string
  }
}

interface Payment { id: number; username: string; amount: number; method: string; status: string; intent_type: string; intent_id?: number; intent_label: string; created_at: string; updated_at: string }
interface PaymentMethod { id: number; name: string; type: string; instructions: string; is_active: boolean; sort_order: number; created_at: string }
interface Ticket { id: number; username: string; subject: string; status: string; priority: string; created_at: string; updated_at: string; closed_at: string }
interface TicketMessage { id: number; ticket_id: number; sender_type: string; sender_name: string; message: string; created_at: string }
interface TicketDetail extends Ticket { messages: TicketMessage[] }
interface Plan { id: number; name: string; data_gb: number; speed_mbps: number; duration_days: number; price: number; is_active: boolean; sort_order: number; created_at: string }
interface VpnProfile { type: string; name: string; filename: string; available: boolean; remote: string; port: number; protocol: string; node: string; download: string }
interface UsageSession { id: number; start_time: string; stop_time: string; session_seconds: number; input_bytes: number; output_bytes: number; total_bytes: number; framed_ip: string; online: boolean }
interface UsageSummary { online: boolean; active_sessions: number; total_input_bytes: number; total_output_bytes: number; total_usage_bytes: number; max_data_bytes: number; remaining_bytes?: number; last_connected_at: string; last_disconnected_at: string; sessions: UsageSession[] }
interface ApiError extends Error { status?: number }

const screen = ref<Screen>('loading')
const loginForm = ref({ username: '', password: '' })
const customer = ref<PortalCustomer | null>(null)
const payments = ref<Payment[]>([])
const paymentMethods = ref<PaymentMethod[]>([])
const tickets = ref<Ticket[]>([])
const selectedTicket = ref<TicketDetail | null>(null)
const ticketForm = ref({ subject: '', priority: 'normal', message: '' })
const ticketReply = ref('')
const plans = ref<Plan[]>([])
const profiles = ref<VpnProfile[]>([])
const portalTab = ref<'overview' | 'billing' | 'support'>('overview')
const usage = ref<UsageSummary | null>(null)
const paymentForm = ref({ amount: 0, method: 'manual', receipt: '' })
const renewForm = ref({ plan_id: 0 })
const busy = ref(false)
const error = ref('')
const notice = ref('')

const titleName = computed(() => customer.value?.display_name || customer.value?.username || 'Customer')
const planName = computed(() => customer.value?.subscription?.plan || customer.value?.plan || 'Starter')
const status = computed(() => customer.value?.subscription?.status || customer.value?.status || 'active')
const dataLimit = computed(() => {
  const raw = Number(customer.value?.max_data_bytes || 0)
  if (!raw) return 'Unlimited'
  return `${Math.round((raw / 1024 / 1024 / 1024) * 10) / 10} GB`
})
const accountScore = computed(() => {
  let score = status.value === 'active' ? 70 : 38
  if (customer.value?.subscription?.expires_at) score += 15
  if ((customer.value?.credit || 0) > 0) score += 10
  if (customer.value?.max_data_bytes) score += 5
  return Math.min(100, score)
})
const selectedPlan = computed(() => plans.value.find((plan) => plan.id === Number(renewForm.value.plan_id)))
const openvpnProfile = computed(() => profiles.value.find((profile) => profile.type === 'openvpn'))
const l2tpProfile = computed(() => profiles.value.find((profile) => profile.type === 'l2tp'))
const ikev2Profile = computed(() => profiles.value.find((profile) => profile.type === 'ikev2'))
const windowOrigin = computed(() => window.location.origin)

function copyToClipboard() {
  const input = document.getElementById('sub-url-input') as HTMLInputElement
  if (input) {
    const text = input.value
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
}

const walletCredit = computed(() => Number(customer.value?.credit || 0))
const requiredTopup = computed(() => Math.max(0, Number(selectedPlan.value?.price || 0) - walletCredit.value))
const selectedPaymentMethod = computed(() => paymentMethods.value.find((method) => method.name === paymentForm.value.method))
const usagePercent = computed(() => {
  if (!usage.value?.max_data_bytes) return 0
  return Math.min(100, Math.round((usage.value.total_usage_bytes / usage.value.max_data_bytes) * 100))
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

async function boot() {
  error.value = ''
  try {
    const [res, paymentRes, methodRes, ticketRes, plansRes, profilesRes, usageRes] = await Promise.all([
      api<{ ok: boolean; customer: PortalCustomer }>('/api/portal/me'),
      api<{ ok: boolean; payments: Payment[] }>('/api/portal/payments'),
      api<{ ok: boolean; methods: PaymentMethod[] }>('/api/portal/payment-methods'),
      api<{ ok: boolean; tickets: Ticket[] }>('/api/portal/tickets'),
      api<{ ok: boolean; plans: Plan[] }>('/api/portal/plans'),
      api<{ ok: boolean; profiles: VpnProfile[] }>('/api/portal/profiles'),
      api<{ ok: boolean; usage: UsageSummary }>('/api/portal/usage')
    ])
    customer.value = res.customer
    payments.value = paymentRes.payments || []
    paymentMethods.value = methodRes.methods || []
    if (paymentMethods.value.length && (!paymentForm.value.method || paymentForm.value.method === 'manual')) paymentForm.value.method = paymentMethods.value[0].name
    tickets.value = ticketRes.tickets || []
    plans.value = plansRes.plans || []
    profiles.value = profilesRes.profiles || []
    usage.value = usageRes.usage
    if (!renewForm.value.plan_id && plans.value.length) renewForm.value.plan_id = plans.value[0].id
    screen.value = 'portal'
  } catch (err) {
    const apiErr = err as ApiError
    if (apiErr.status && apiErr.status !== 401) error.value = friendlyError(err)
    screen.value = 'login'
  }
}

async function login() {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await api<{ ok: boolean; username: string }>('/api/auth/customer', { method: 'POST', body: JSON.stringify(loginForm.value) })
    await boot()
    notice.value = 'Signed in successfully.'
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}

async function logout() {
  await api<{ ok: boolean }>('/api/auth/customer/logout', { method: 'POST' }).catch(() => null)
  customer.value = null
  payments.value = []
  paymentMethods.value = []
  tickets.value = []
  selectedTicket.value = null
  plans.value = []
  profiles.value = []
  usage.value = null
  screen.value = 'login'
}

async function submitRenewal() {
  if (!renewForm.value.plan_id) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const res = await api<{ ok: boolean; renewed: boolean; payment_required: boolean; required_amount?: number; payment_id?: number }>('/api/portal/renew', { method: 'POST', body: JSON.stringify(renewForm.value) })
    if (res.renewed) notice.value = 'Plan activated. Wallet was charged.'
    else if (res.payment_required) notice.value = `Payment request #${res.payment_id} created for ${formatMoney(res.required_amount)}.`
    await boot()
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}

async function submitPaymentRequest() {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await api<{ ok: boolean; id: number }>('/api/portal/payments', { method: 'POST', body: JSON.stringify(paymentForm.value) })
    notice.value = 'Payment request submitted. Admin will review it.'
    paymentForm.value = { amount: 0, method: 'manual', receipt: '' }
    await boot()
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}


async function createTicket() {
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    const res = await api<{ ok: boolean; id: number }>('/api/portal/tickets', { method: 'POST', body: JSON.stringify(ticketForm.value) })
    notice.value = 'Ticket created.'
    ticketForm.value = { subject: '', priority: 'normal', message: '' }
    await boot()
    await openTicket(res.id)
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}

async function openTicket(id: number) {
  busy.value = true
  error.value = ''
  try {
    const res = await api<{ ok: boolean; ticket: TicketDetail }>(`/api/portal/tickets/${id}`)
    selectedTicket.value = res.ticket
    ticketReply.value = ''
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}

async function replyTicket() {
  if (!selectedTicket.value || !ticketReply.value.trim()) return
  busy.value = true
  error.value = ''
  notice.value = ''
  try {
    await api<{ ok: boolean }>(`/api/portal/tickets/${selectedTicket.value.id}/reply`, { method: 'POST', body: JSON.stringify({ message: ticketReply.value }) })
    notice.value = 'Reply sent.'
    await openTicket(selectedTicket.value.id)
    await boot()
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}

async function closeTicket() {
  if (!selectedTicket.value) return
  busy.value = true
  error.value = ''
  try {
    await api<{ ok: boolean }>(`/api/portal/tickets/${selectedTicket.value.id}/close`, { method: 'POST' })
    notice.value = 'Ticket closed.'
    await openTicket(selectedTicket.value.id)
    await boot()
  } catch (err) {
    error.value = friendlyError(err)
  } finally {
    busy.value = false
  }
}

function friendlyError(err: unknown) {
  if (err instanceof Error) return err.message.replace(/_/g, ' ')
  return 'Unexpected error'
}

function formatMoney(value?: number) {
  return `${new Intl.NumberFormat('en', { maximumFractionDigits: 0 }).format(value || 0)} IRT`
}
function formatGB(value?: number) { return value && value > 0 ? `${new Intl.NumberFormat('en', { maximumFractionDigits: 2 }).format(value)} GB` : 'Unlimited' }
function formatSpeed(value?: number) { return value && value > 0 ? `${new Intl.NumberFormat('en', { maximumFractionDigits: 2 }).format(value)} Mbps` : 'Unlimited' }
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
  if (h) return `${h}h ${m}m`
  if (m) return `${m}m`
  return `${s}s`
}

function formatDate(value?: string) {
  if (!value) return 'Not set'
  return new Intl.DateTimeFormat('en', { year: 'numeric', month: 'short', day: '2-digit' }).format(new Date(value))
}

onMounted(boot)
</script>

<template>
  <main v-if="screen === 'loading'" class="portal loading"><span class="loader"></span><p>Opening portal…</p></main>

  <main v-else-if="screen === 'login'" class="portal auth">
    <section class="hero-card">
      <span class="brand">K</span>
      <h1>Customer portal</h1>
      <p>Access subscription status, wallet, VPN profile and support from a clean compact dashboard.</p>
      <div class="chips"><span>OpenVPN</span><span>L2TP</span><span>IKEv2</span></div>
      <div class="trust-stack">
        <div><b>Secure login</b><small>Uses the same Radius credentials</small></div>
        <div><b>Live account</b><small>Plan, wallet and status in one view</small></div>
      </div>
    </section>
    <section class="login-card">
      <span class="eyebrow">Portal login</span>
      <h2>Welcome back</h2>
      <form @submit.prevent="login" class="form-stack">
        <label>Username<input v-model.trim="loginForm.username" autocomplete="username" required placeholder="VPN username" /></label>
        <label>Password<input v-model="loginForm.password" type="password" autocomplete="current-password" required placeholder="VPN password" /></label>
        <button :disabled="busy">{{ busy ? 'Checking…' : 'Enter portal' }}</button>
      </form>
      <p v-if="error" class="alert danger">{{ error }}</p>
      <p class="hint">Use the same username/password created in the admin dashboard.</p>
    </section>
  </main>

  <main v-else class="portal shell">
    <header class="topbar">
      <div class="brand-row"><span class="brand small">K</span><div><b>KorisPanel</b><small>Customer portal</small></div></div>
      <div class="top-actions"><div class="score-pill"><b>{{ accountScore }}%</b><small>account ready</small></div><button class="ghost" @click="logout">Logout</button></div>
    </header>

    <section class="welcome">
      <div>
        <span class="eyebrow">{{ status }}</span>
        <h1>Hello, {{ titleName }}</h1>
        <p>Your VPN account is connected to KorisPanel Next.</p>
      </div>
      <a v-if="openvpnProfile?.available" class="primary" :href="openvpnProfile.download" download>Download profile</a>
      <button v-else class="primary" disabled>No node available</button>
    </section>

    <p v-if="notice" class="alert success">{{ notice }}</p>

    <!-- Tabbed Portal Navigation (Prevents Long Page Scrolls) -->
    <div class="secondary-tabs" style="display: flex; gap: 8px; border-bottom: 1px solid var(--line); padding-bottom: 8px; margin-bottom: 18px; width: 100%;">
      <button class="tab-btn" @click="portalTab = 'overview'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 850; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s; font-size: 14px;" :style="portalTab === 'overview' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Dashboard Overview</button>
      <button class="tab-btn" @click="portalTab = 'billing'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 850; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s; font-size: 14px;" :style="portalTab === 'billing' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Subscriptions & Billing</button>
      <button class="tab-btn" @click="portalTab = 'support'" style="background: none; border: 0; color: var(--muted); padding: 8px 16px; font-weight: 850; cursor: pointer; border-bottom: 2px solid transparent; transition: all 0.2s; font-size: 14px;" :style="portalTab === 'support' ? 'color: var(--cyan); border-bottom-color: var(--cyan);' : ''">Support Tickets</button>
    </div>

    <section v-if="portalTab === 'overview'" class="grid">
      <article class="metric main">
        <small>Current plan</small>
        <strong>{{ planName }}</strong>
        <span>Expires: {{ formatDate(customer?.subscription?.expires_at) }}</span>
      </article>
      <article class="metric">
        <small>Data limit</small>
        <strong>{{ dataLimit }}</strong>
        <span>{{ usage?.online ? `${usage.active_sessions} active session` : 'Offline now' }}</span>
      </article>
      <article class="metric">
        <small>Wallet</small>
        <strong>{{ formatMoney(customer?.credit) }}</strong>
        <span>Manual payments ready</span>
      </article>
    </section>

    <section v-if="portalTab === 'overview' && usage" class="usage-card card">
      <div class="section-head"><div><span class="eyebrow">Usage</span><h2>VPN sessions</h2></div><span :class="['pay-status', usage.online ? 'approved' : 'pending']">{{ usage.online ? 'online' : 'offline' }}</span></div>
      <div class="portal-plan-summary usage-summary">
        <span><b>{{ formatBytes(usage.total_usage_bytes) }}</b><small>total used</small></span>
        <span><b>{{ formatBytes(usage.total_input_bytes) }}</b><small>download</small></span>
        <span><b>{{ formatBytes(usage.total_output_bytes) }}</b><small>upload</small></span>
        <span><b>{{ usage.remaining_bytes === undefined ? 'Unlimited' : formatBytes(usage.remaining_bytes) }}</b><small>remaining</small></span>
      </div>
      <div v-if="usage.max_data_bytes" class="usage-bar"><i :style="{ width: `${usagePercent}%` }"></i></div>
      <div class="payment-list usage-list">
        <div v-for="session in usage.sessions.slice(0, 4)" :key="session.id" class="payment-row">
          <div><b>{{ session.online ? 'Online' : 'Closed' }} · {{ formatBytes(session.total_bytes) }}</b><small>{{ session.framed_ip || '—' }} · {{ formatDuration(session.session_seconds) }} · {{ formatDate(session.start_time) }}</small></div>
          <span :class="['pay-status', session.online ? 'approved' : 'pending']">{{ session.online ? 'online' : 'closed' }}</span>
        </div>
        <p v-if="!usage.sessions.length" class="hint">No VPN sessions yet.</p>
      </div>
    </section>

    <section v-if="portalTab === 'overview'" class="journey card">
      <div class="section-head"><div><span class="eyebrow">Account journey</span><h2>What is ready</h2></div></div>
      <div class="journey-steps">
        <div class="done"><i></i><b>Credentials</b><small>Radius login confirmed</small></div>
        <div :class="customer?.subscription ? 'done' : ''"><i></i><b>Subscription</b><small>{{ customer?.subscription ? 'Plan attached' : 'Awaiting plan' }}</small></div>
        <div :class="customer?.max_data_bytes ? 'done' : ''"><i></i><b>Data policy</b><small>{{ dataLimit }}</small></div>
      </div>
    </section>

    <section class="cards">
      <article v-if="portalTab === 'overview'" class="card">
        <div class="section-head"><div><span class="eyebrow">VPN access</span><h2>Connection profiles</h2></div></div>
        <div v-if="customer?.sub_token" class="profile-row" style="flex-direction: column; align-items: stretch; gap: 8px; border-bottom: 1px solid var(--line); padding-bottom: 14px;">
          <b style="color: var(--cyan); font-size: 11px; text-transform: uppercase; letter-spacing: 0.1em; display: block;">Unified Subscription Link</b>
          <div style="display: flex; gap: 8px; align-items: center;">
            <input readonly :value="windowOrigin + '/portal/sub/' + customer.sub_token" style="font-family: monospace; font-size: 12px; background: rgba(0,0,0,0.2); flex: 1; padding: 6px 10px; border-radius: 8px; border: 1px solid var(--line); color: #fff;" id="sub-url-input" />
            <button class="ghost" @click="copyToClipboard" style="white-space: nowrap; font-size: 12px; min-height: 32px; border-radius: 8px; padding: 0 12px;">Copy Link</button>
          </div>
        </div>
        <div class="profile-row"><div><b>OpenVPN</b><span v-if="openvpnProfile">{{ openvpnProfile.remote }}:{{ openvpnProfile.port }} · {{ openvpnProfile.protocol }}</span><span v-else>Profile endpoint unavailable</span></div><a v-if="openvpnProfile?.available" class="ghost profile-download" :href="openvpnProfile.download" download>Download</a><span v-else>Unavailable</span></div>
        <div class="profile-row"><div><b>L2TP/IPSec (Apple)</b><span v-if="l2tpProfile">{{ l2tpProfile.remote }}</span><span v-else>L2TP profile unavailable</span></div><a v-if="l2tpProfile?.available" class="ghost profile-download" :href="l2tpProfile.download" download>Download</a><span v-else>Unavailable</span></div>
        <div class="profile-row"><div><b>IKEv2 (Apple)</b><span v-if="ikev2Profile">{{ ikev2Profile.remote }}</span><span v-else>IKEv2 profile unavailable</span></div><a v-if="ikev2Profile?.available" class="ghost profile-download" :href="ikev2Profile.download" download>Download</a><span v-else>Unavailable</span></div>

        <!-- L2TP / IKEv2 Manual Connection Credentials -->
        <div class="profile-row" style="flex-direction: column; align-items: stretch; gap: 8px; border-top: 1px solid var(--line); margin-top: 14px; padding-top: 14px;">
          <b style="color: var(--cyan); font-size: 11px; text-transform: uppercase; letter-spacing: 0.1em; display: block;">L2TP / IKEv2 Manual Setup Credentials</b>
          <div style="font-size: 13px; display: grid; gap: 6px; line-height: 1.5; color: var(--muted);">
            <div>● Server Address: <code style="color: #fff; font-size: 12px; background: rgba(0,0,0,0.2); padding: 2px 6px; border-radius: 4px; font-family: monospace;">{{ l2tpProfile?.remote || 'luna.koris.space' }}</code></div>
            <div>● IPSec PSK (Shared Secret): <code style="color: #fff; font-size: 12px; background: rgba(0,0,0,0.2); padding: 2px 6px; border-radius: 4px; font-family: monospace;">testing123</code></div>
            <div>● Username: <code style="color: #fff; font-size: 12px; background: rgba(0,0,0,0.2); padding: 2px 6px; border-radius: 4px; font-family: monospace;">{{ customer?.username }}</code></div>
            <div>● Password: <code style="color: #fff; font-size: 12px; background: rgba(0,0,0,0.2); padding: 2px 6px; border-radius: 4px; font-family: monospace;">(Your login password)</code></div>
          </div>
        </div>
      </article>
      <article v-if="portalTab === 'billing'" class="card plan-renew-card">
        <div class="section-head"><div><span class="eyebrow">Plans</span><h2>Choose / renew</h2></div></div>
        <form class="form-stack" @submit.prevent="submitRenewal">
          <label>Plan<select v-model.number="renewForm.plan_id"><option v-for="plan in plans" :key="plan.id" :value="plan.id">{{ plan.name }} · {{ formatMoney(plan.price) }}</option></select></label>
          <div v-if="selectedPlan" class="portal-plan-summary">
            <span><b>{{ formatGB(selectedPlan.data_gb) }}</b><small>data</small></span>
            <span><b>{{ formatSpeed(selectedPlan.speed_mbps) }}</b><small>speed</small></span>
            <span><b>{{ selectedPlan.duration_days }}d</b><small>duration</small></span>
            <span><b>{{ formatMoney(selectedPlan.price) }}</b><small>price</small></span>
          </div>
          <p v-if="selectedPlan" class="hint">Wallet: {{ formatMoney(walletCredit) }} · {{ requiredTopup > 0 ? `Needs ${formatMoney(requiredTopup)} top-up` : 'Enough wallet balance' }}</p>
          <button :disabled="busy || !renewForm.plan_id">{{ busy ? 'Processing…' : requiredTopup > 0 ? 'Request payment for this plan' : 'Renew now' }}</button>
        </form>
      </article>
      <article v-if="portalTab === 'billing'" class="card">
        <div class="section-head"><div><span class="eyebrow">Payment</span><h2>Request top-up</h2></div></div>
        <form class="form-stack" @submit.prevent="submitPaymentRequest">
          <label>Amount<input v-model.number="paymentForm.amount" type="number" min="1" step="1" required placeholder="Amount" /></label>
          <label>Method<select v-model="paymentForm.method"><option value="manual">manual</option><option v-for="method in paymentMethods" :key="method.id" :value="method.name">{{ method.name }}</option></select></label>
          <p v-if="selectedPaymentMethod?.instructions" class="method-instructions">{{ selectedPaymentMethod.instructions }}</p>
          <label>Receipt / note<textarea v-model.trim="paymentForm.receipt" placeholder="Transaction id, card digits, or note for admin"></textarea></label>
          <button :disabled="busy">{{ busy ? 'Sending…' : 'Submit payment request' }}</button>
        </form>
      </article>
      <article v-if="portalTab === 'billing'" class="card">
        <div class="section-head"><div><span class="eyebrow">Payment history</span><h2>Latest requests</h2></div></div>
        <div class="payment-list">
          <div v-for="payment in payments.slice(0, 6)" :key="payment.id" class="payment-row">
            <div><b>#{{ payment.id }} · {{ formatMoney(payment.amount) }}</b><small>{{ payment.method }} · {{ payment.intent_type === 'plan_renewal' ? `Plan renewal: ${payment.intent_label || payment.intent_id}` : 'Wallet top-up' }} · {{ formatDate(payment.created_at) }}</small></div>
            <span :class="['pay-status', payment.status]">{{ payment.status }}</span>
          </div>
          <p v-if="!payments.length" class="hint">No payment requests yet.</p>
        </div>
      </article>
      <article v-if="portalTab === 'support'" class="card support-card">
        <div class="section-head"><div><span class="eyebrow">Support</span><h2>New ticket</h2></div></div>
        <form class="form-stack" @submit.prevent="createTicket">
          <label>Subject<input v-model.trim="ticketForm.subject" required placeholder="How can we help?" /></label>
          <label>Priority<select v-model="ticketForm.priority"><option value="low">low</option><option value="normal">normal</option><option value="high">high</option></select></label>
          <label>Message<textarea v-model.trim="ticketForm.message" required placeholder="Describe the issue"></textarea></label>
          <button :disabled="busy">{{ busy ? 'Sending…' : 'Create ticket' }}</button>
        </form>
      </article>
      <article v-if="portalTab === 'support'" class="card support-card">
        <div class="section-head"><div><span class="eyebrow">{{ tickets.length }} tickets</span><h2>Support history</h2></div></div>
        <div class="payment-list ticket-list">
          <div v-for="ticket in tickets.slice(0, 6)" :key="ticket.id" class="payment-row" @click="openTicket(ticket.id)">
            <div><b>#{{ ticket.id }} · {{ ticket.subject }}</b><small>{{ ticket.priority }} · {{ formatDate(ticket.updated_at) }}</small></div>
            <span :class="['pay-status', ticket.status === 'open' ? 'approved' : 'rejected']">{{ ticket.status }}</span>
          </div>
          <p v-if="!tickets.length" class="hint">No tickets yet.</p>
        </div>
      </article>
      <article v-if="selectedTicket && portalTab === 'support'" class="card support-card ticket-detail-card">
        <div class="section-head"><div><span class="eyebrow">Ticket #{{ selectedTicket.id }}</span><h2>{{ selectedTicket.subject }}</h2></div><button v-if="selectedTicket.status === 'open'" class="ghost" @click="closeTicket">Close</button></div>
        <div class="ticket-thread portal-thread">
          <div v-for="message in selectedTicket.messages" :key="message.id" :class="['ticket-message', message.sender_type]"><b>{{ message.sender_name }} <small>{{ message.sender_type }} · {{ formatDate(message.created_at) }}</small></b><p>{{ message.message }}</p></div>
        </div>
        <form v-if="selectedTicket.status === 'open'" class="form-stack" @submit.prevent="replyTicket">
          <label>Reply<textarea v-model.trim="ticketReply" placeholder="Write a reply"></textarea></label>
          <button :disabled="busy || !ticketReply.trim()">Send reply</button>
        </form>
      </article>
    </section>
  </main>
</template>
