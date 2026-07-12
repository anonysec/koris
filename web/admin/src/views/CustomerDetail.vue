<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useCustomersStore } from '@/stores/customers'
import { useResellersStore } from '@/stores/resellers'
import { useDomainsStore, type MTProtoSecretInfo } from '@/stores/domains'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useApi } from '@koris/composables/useApi'
import { useAuthStore } from '@/stores/auth'
import { formatDate, formatDateTime } from '@koris/composables/useFormatDate'
import Tabs from '@koris/ui/Tabs.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import Textarea from '@koris/ui/Textarea.vue'
import Button from '@koris/ui/Button.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Avatar from '@koris/ui/Avatar.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'

const props = defineProps<{ id: string }>()

const { t } = useI18n()
const router = useRouter()
const store = useCustomersStore()
const domainsStore = useDomainsStore()
const toast = useToast()
const { confirm } = useConfirm()
const { get } = useApi()
const activeTab = ref('profile')
const saving = ref(false)

// ─── Reseller Wallet Adjust ─────────────────────────────────────────────────
const showWalletAdjust = ref(false)
const walletAmount = ref('')
const walletAdjusting = ref(false)

async function adjustWallet() {
  if (!store.detail || !walletAmount.value) return
  walletAdjusting.value = true
  try {
    const { post } = useApi()
    await post(`/api/reseller/users/${store.detail.id}/wallet`, { amount: Number(walletAmount.value) })
    toast.success('Credit adjusted')
    await store.loadDetail(store.detail.id)
    showWalletAdjust.value = false
    walletAmount.value = ''
  } catch {
    toast.error('Failed to adjust credit')
  } finally {
    walletAdjusting.value = false
  }
}

// ─── Traffic Reset State (Requirement 3.4) ───────────────────────────────────
const resettingTraffic = ref(false)

// ─── Connection Limit State (Requirement 4.3) ────────────────────────────────
const editingConnectionLimit = ref(false)
const connectionLimitInput = ref(0)
const savingConnectionLimit = ref(false)

// ─── MTProto Secret State (Requirements 7.6, 5.4) ───────────────────────────
const mtprotoSecret = ref<MTProtoSecretInfo | null>(null)
const mtprotoLoading = ref(false)
const mtprotoRegenerating = ref(false)
const mtprotoCopied = ref(false)

/**
 * Extracts the current connection limit from the customer's radius_checks.
 * Looks for the Simultaneous-Use attribute. Returns 0 (unlimited) if not found.
 * Requirement 4.3
 */
const currentConnectionLimit = computed(() => {
  if (!store.detail?.radius_checks) return 0
  const check = store.detail.radius_checks.find(
    (rc) => rc.attribute === 'Simultaneous-Use'
  )
  return check ? Number(check.value) || 0 : 0
})

/**
 * Reset traffic counters for this customer.
 * Requirement 3.4
 */
async function handleTrafficReset() {
  if (!store.detail) return
  resettingTraffic.value = true
  const success = await store.trafficReset(store.detail.id)
  resettingTraffic.value = false
  if (success) {
    toast.success(t('customer.traffic_reset_success'))
  } else {
    toast.error(t('customer.traffic_reset_error'))
  }
}

/**
 * Start editing the connection limit inline.
 */
function startEditConnectionLimit() {
  connectionLimitInput.value = currentConnectionLimit.value
  editingConnectionLimit.value = true
}

/**
 * Cancel connection limit editing.
 */
function cancelEditConnectionLimit() {
  editingConnectionLimit.value = false
}

/**
 * Save the new connection limit.
 * Requirement 4.3
 */
async function saveConnectionLimit() {
  if (!store.detail) return
  savingConnectionLimit.value = true
  const limit = Math.max(0, Math.floor(connectionLimitInput.value))
  const success = await store.setConnectionLimit(store.detail.id, limit)
  savingConnectionLimit.value = false
  if (success) {
    editingConnectionLimit.value = false
    toast.success(
      limit === 0
        ? t('customer.conn_limit_removed')
        : t('customer.conn_limit_set') + ' ' + limit
    )
  } else {
    toast.error(t('customer.conn_limit_error'))
  }
}

// ─── MTProto Secret Functions (Requirements 7.6, 5.4) ────────────────────────

/**
 * Fetch the customer's MTProto secret and connection info.
 * Requirement 5.4
 */
async function loadMTProtoSecret() {
  if (!props.id || props.id === 'new') return
  mtprotoLoading.value = true
  try {
    mtprotoSecret.value = await domainsStore.fetchMTProtoSecret(Number(props.id))
  } catch {
    // Ignore — customer may not have MTProto enabled
  } finally {
    mtprotoLoading.value = false
  }
}

/**
 * Copy MTProto secret to clipboard.
 * Requirement 7.6
 */
async function copyMTProtoSecret() {
  if (!mtprotoSecret.value?.secret) return
  try {
    await navigator.clipboard.writeText(mtprotoSecret.value.secret)
    mtprotoCopied.value = true
    toast.success('Secret copied to clipboard')
    setTimeout(() => { mtprotoCopied.value = false }, 2000)
  } catch {
    toast.error('Failed to copy secret')
  }
}

/**
 * Regenerate MTProto secret with confirmation dialog.
 * Requirement 5.6 — invalidates old secret and disconnects active sessions.
 */
async function regenerateMTProtoSecret() {
  if (!store.detail) return

  const confirmed = await confirm({
    title: t('customer.regenerate_mtproto_title'),
    message: t('customer.regenerate_mtproto_warn'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('customer.regenerate'),
    cancelText: t('common.cancel'),
  })
  if (!confirmed) return

  mtprotoRegenerating.value = true
  try {
    const result = await domainsStore.regenerateSecret(store.detail.id)
    if (result) {
      mtprotoSecret.value = result
      toast.success('MTProto secret regenerated')
    } else {
      toast.error('Failed to regenerate secret')
    }
  } catch {
    toast.error('Failed to regenerate secret')
  } finally {
    mtprotoRegenerating.value = false
  }
}

const tabs = computed(() => [
  { key: 'profile', label: t('customer.tab_profile') },
  { key: 'usage', label: t('customer.tab_usage') },
  { key: 'history', label: t('customer.tab_history') },
  { key: 'custom_fields', label: t('customer.tab_custom_fields') },
  { key: 'notes', label: t('customer.tab_notes') },
  { key: 'activity', label: t('customer.tab_activity') },
])

// ─── Custom Fields State ────────────────────────────────────────────────────
interface CustomField {
  key: string
  value: string
  label: string
}
const customFields = ref<CustomField[]>([])
const customFieldsLoading = ref(false)
const customFieldsSaving = ref(false)

async function loadCustomFields() {
  if (!props.id || props.id === 'new') return
  customFieldsLoading.value = true
  try {
    const res = await get<{ ok: boolean; fields: CustomField[] }>(`/api/admin/customers/${props.id}/custom-fields`)
    if (res?.ok && res.fields) {
      customFields.value = res.fields
    }
  } catch { /* ignore */ }
  finally { customFieldsLoading.value = false }
}

async function saveCustomFields() {
  if (!props.id) return
  customFieldsSaving.value = true
  try {
    const { post } = useApi()
    const res = await post<{ ok: boolean }>(`/api/admin/customers/${props.id}/custom-fields`, {
      fields: customFields.value,
    })
    if (res?.ok) {
      toast.success(t('customer.custom_field_saved'))
    } else {
      toast.error(t('customer.custom_field_error'))
    }
  } catch {
    toast.error(t('customer.custom_field_error'))
  } finally {
    customFieldsSaving.value = false
  }
}

// ─── Notes State ────────────────────────────────────────────────────────────
interface AdminNote {
  id: number
  content: string
  created_by: string
  created_at: string
}
const notes = ref<AdminNote[]>([])
const notesLoading = ref(false)
const newNoteContent = ref('')
const addingNote = ref(false)

async function loadNotes() {
  if (!props.id || props.id === 'new') return
  notesLoading.value = true
  try {
    const res = await get<{ ok: boolean; notes: AdminNote[] }>(`/api/admin/customers/${props.id}/notes`)
    if (res?.ok && res.notes) {
      notes.value = res.notes
    }
  } catch { /* ignore */ }
  finally { notesLoading.value = false }
}

async function addNote() {
  if (!props.id || !newNoteContent.value.trim()) return
  addingNote.value = true
  try {
    const { post } = useApi()
    const res = await post<{ ok: boolean; note?: AdminNote }>(`/api/admin/customers/${props.id}/notes`, {
      content: newNoteContent.value.trim(),
    })
    if (res?.ok) {
      toast.success(t('customer.note_saved'))
      newNoteContent.value = ''
      await loadNotes()
    } else {
      toast.error(t('customer.note_error'))
    }
  } catch {
    toast.error(t('customer.note_error'))
  } finally {
    addingNote.value = false
  }
}

// Edit form state
const form = ref({
  username: '',
  password: '',
  display_name: '',
  status: '',
  plan_id: '',
  data_gb: '',
  speed_mbps: '',
  days: '',
  notes: '',
  avatar: '',
  billing_mode: '',
})

const customer = computed(() => store.detail)
const usage = computed(() => store.usage)
const isNew = computed(() => props.id === 'new')

const defaultEmojis = ['🦊', '🐻', '🐼', '🐨', '🦁', '🐯', '🐸', '🐙', '🦋', '🌟', '🔥', '💎', '🎯', '🚀', '⚡', '🌈', '🎪', '🎭', '🏆', '👑']

// Reserved emojis (used by resellers, filtered from user picker)
const authStore = useAuthStore()
const resellersStore = useResellersStore()
const isReseller = computed(() => authStore.user?.role === 'reseller')

// Hide avatar edit for reseller-created users (they inherit reseller's emoji)
const isResellerCreated = computed(() => {
  if (!customer.value?.created_by) return false
  const resellerUsernames = new Set(resellersStore.list.map(r => r.username))
  return resellerUsernames.has(customer.value.created_by)
})

interface ReservedEmojiInfo { emoji: string; reseller: string }
const reservedEmojiList = ref<ReservedEmojiInfo[]>([])

const availableUserEmojis = computed(() => {
  const reservedSet = new Set(reservedEmojiList.value.map(r => r.emoji))
  return defaultEmojis.filter(e => !reservedSet.has(e))
})

async function loadReservedEmojis() {
  if (isReseller.value) return
  try {
    const data = await get<{ ok: boolean; reserved: ReservedEmojiInfo[] }>('/api/reserved-emojis')
    if (data?.ok) {
      reservedEmojiList.value = data.reserved
    }
  } catch { /* ignore */ }
}

function populateForm() {
  if (customer.value) {
    form.value = {
      username: customer.value.username || '',
      password: '',
      display_name: customer.value.display_name || '',
      status: customer.value.status || '',
      plan_id: String(customer.value.plan_id ?? ''),
      data_gb: '',
      speed_mbps: '',
      days: '',
      notes: customer.value.notes || '',
      avatar: customer.value.avatar || '',
      billing_mode: (customer.value as any).billing_mode || '',
    }
  }
}

watch(customer, populateForm)

async function saveProfile() {
  if (!customer.value) return
  saving.value = true
  const payload: Record<string, any> = {
    display_name: form.value.display_name,
    status: form.value.status,
    notes: form.value.notes,
  }
  if (!isReseller.value) {
    payload.avatar = form.value.avatar
  }
  if (isResellerCreated.value || isReseller.value) {
    payload.billing_mode = form.value.billing_mode
  }
  await store.updateCustomer(customer.value.id, payload)
  saving.value = false
}

async function createCustomer() {
  saving.value = true
  const created = await store.createCustomer({
    username: form.value.username,
    password: form.value.password,
    display_name: form.value.display_name,
    plan_id: Number(form.value.plan_id) || 1,
    data_gb: Number(form.value.data_gb) || 0,
    speed_mbps: Number(form.value.speed_mbps) || 0,
    days: Number(form.value.days) || 30,
  })
  saving.value = false
  if (created) {
    router.push({ name: 'users' })
  }
}

function formatBytes(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1048576) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1073741824) return `${(bytes / 1048576).toFixed(1)} MB`
  return `${(bytes / 1073741824).toFixed(2)} GB`
}

// ---- Plan Change ----
interface Plan {
  id: number
  name: string
  data_gb: number
  speed_mbps: number
  duration_days: number
  price: number
  is_active: boolean
}
const plans = ref<Plan[]>([])
const selectedPlanId = ref<number>(0)
const applyingPlan = ref(false)
const switchingPlan = ref(false)

async function loadPlans() {
  try {
    const res = await get<{ ok: boolean; plans: Plan[] }>('/api/plans')
    if (res?.plans) {
      plans.value = res.plans.filter(p => p.is_active)
    }
  } catch {
    // Fallback for resellers: try reseller-specific endpoint
    try {
      const res = await get<{ ok: boolean; plans: any[] }>('/api/reseller/plan-prices')
      if (res?.plans) {
        plans.value = res.plans.map((p: any) => ({
          id: p.id,
          name: p.name,
          data_gb: p.data_gb,
          speed_mbps: p.speed_mbps || 0,
          duration_days: p.duration_days,
          price: p.wholesale_price,
          is_active: true,
        }))
      }
    } catch { /* ignore */ }
  }
}

async function handleApplyPlan() {
  if (!customer.value || !selectedPlanId.value || selectedPlanId.value === 0) return
  applyingPlan.value = true
  try {
    const { post: postApi } = useApi()
    const res = await postApi<{ ok: boolean; error?: string }>(`/api/customers/${customer.value.id}/renew`, {
      plan_id: selectedPlanId.value,
    })
    if (res.ok) {
      toast.success('Plan applied successfully')
      await store.loadDetail(customer.value.id)
    } else {
      console.error('[plan] Apply plan failed:', res.error)
      toast.error(res.error || 'Failed to apply plan')
    }
  } catch (err: any) {
    console.error('[plan] Apply plan error:', err)
    toast.error(err?.message || 'Failed to apply plan')
  } finally {
    applyingPlan.value = false
  }
}

async function handleSwitchPlan() {
  if (!customer.value || !selectedPlanId.value || selectedPlanId.value === 0) return
  switchingPlan.value = true
  try {
    const { post: postApi } = useApi()
    const res = await postApi<{ ok: boolean; refund_amount?: number; new_plan?: string; error?: string }>(`/api/customers/${customer.value.id}/switch-plan`, {
      plan_id: selectedPlanId.value,
    })
    if (res.ok) {
      toast.success(`Plan switched! Refunded $${res.refund_amount?.toFixed(2) || '0.00'} to wallet`)
      await store.loadDetail(customer.value.id)
    } else {
      console.error('[plan] Switch plan failed:', res.error)
      toast.error(res.error || 'Failed to switch plan')
    }
  } catch (err: any) {
    console.error('[plan] Switch plan error:', err)
    toast.error(err?.message || 'Failed to switch plan')
  } finally {
    switchingPlan.value = false
  }
}

onMounted(() => {
  if (props.id && props.id !== 'new') {
    store.loadDetail(Number(props.id))
    loadPlans()
    loadCustomFields()
    loadNotes()
    loadMTProtoSecret()
  }
  loadReservedEmojis()
  if (!isReseller.value) {
    resellersStore.loadResellers()
  }
})
</script>

<template>
  <div class="page customer-detail">
    <!-- Create New Customer Form -->
    <template v-if="isNew">
      <header class="detail-header">
        <div class="detail-header__left">
          <div class="detail-header__info">
            <h2 class="detail-header__username">{{ t('customer.new_customer') }}</h2>
          </div>
        </div>
        <Button variant="ghost" @click="router.back()">{{ t('customer.back') }}</Button>
      </header>

      <form class="profile-form" @submit.prevent="createCustomer">
        <div class="form-grid">
          <FormField name="username" :label="t('login.username')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.username" placeholder="username" />
            </template>
          </FormField>

          <FormField name="password" :label="t('login.password')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.password" type="password" :placeholder="t('customer.placeholder_password')" />
            </template>
          </FormField>

          <FormField name="display_name" :label="t('customer.display_name')" required>
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.display_name" :placeholder="t('customer.placeholder_display_name')" />
            </template>
          </FormField>

          <FormField name="days" :label="t('customer.duration_days')">
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.days" type="number" placeholder="30" />
            </template>
          </FormField>

          <FormField name="data_gb" :label="t('customer.data_gb')">
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.data_gb" type="number" :placeholder="t('customer.placeholder_plan_default')" />
            </template>
          </FormField>

          <FormField name="speed_mbps" :label="t('customer.speed_mbps')">
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="form.speed_mbps" type="number" :placeholder="t('customer.placeholder_plan_default')" />
            </template>
          </FormField>
        </div>

        <FormField name="notes" :label="t('customer.notes')">
          <template #default="{ fieldId }">
            <Textarea :id="fieldId" v-model="form.notes" rows="3" />
          </template>
        </FormField>

        <div class="form-actions">
          <Button variant="ghost" @click="router.back()">{{ t('btn.cancel') }}</Button>
          <Button type="submit" variant="primary" :loading="saving">{{ t('customer.create_customer') }}</Button>
        </div>
      </form>
    </template>

    <!-- Loading State -->
    <div v-else-if="store.detailLoading" class="loading-state">
      <Skeleton variant="rect" :width="'100%'" :height="80" />
      <Skeleton variant="rect" :width="'100%'" :height="300" />
    </div>

    <template v-else-if="customer">
      <!-- Header -->
      <header class="detail-header">
        <div class="detail-header__left">
          <Avatar :name="customer.display_name || customer.username" size="lg" :emoji="customer.avatar || undefined" />
          <div class="detail-header__info">
            <h2 class="detail-header__username">{{ customer.username }}</h2>
            <div class="detail-header__meta">
              <StatusPill :status="customer.status" />
              <span class="detail-header__balance">${{ customer.credit.toFixed(2) }}</span>
              <span class="detail-header__plan">{{ customer.plan || 'No plan' }}</span>
              <Button v-if="isReseller" type="button" variant="ghost" size="sm" @click="showWalletAdjust = true">+ Credit</Button>
            </div>
          </div>
        </div>
        <Button variant="ghost" @click="router.back()">{{ t('customer.back') }}</Button>
      </header>

      <!-- Reseller Wallet Adjust Modal -->
      <div v-if="showWalletAdjust && isReseller" class="wallet-adjust-panel">
        <div class="wallet-adjust-form">
          <FormField name="wallet-amount" label="Amount">
            <template #default="{ fieldId }">
              <Input :id="fieldId" v-model="walletAmount" type="number" placeholder="10.00" />
            </template>
          </FormField>
          <div class="wallet-adjust-actions">
            <Button type="button" variant="ghost" size="sm" @click="showWalletAdjust = false">Cancel</Button>
            <Button type="button" variant="primary" size="sm" :loading="walletAdjusting" @click="adjustWallet">Save</Button>
          </div>
        </div>
      </div>

      <!-- Tabs -->
      <Tabs v-model="activeTab" :tabs="tabs" aria-label="Customer details">
        <!-- Profile Tab -->
        <template #profile>
          <form class="profile-form" @submit.prevent="saveProfile">
            <div class="form-grid">
              <FormField name="display_name" :label="t('customer.display_name')" required>
                <template #default="{ fieldId, describedBy }">
                  <Input :id="fieldId" v-model="form.display_name" :aria-describedby="describedBy" />
                </template>
              </FormField>

              <FormField name="status" :label="t('customer.status')">
                <template #default="{ fieldId }">
                  <Select
                    :id="fieldId"
                    v-model="form.status"
                    :options="[
                      { label: t('status.active'), value: 'active' },
                      { label: t('status.disabled'), value: 'disabled' },
                      { label: t('status.limited'), value: 'limited' },
                      { label: t('status.expired'), value: 'expired' },
                    ]"
                  />
                </template>
              </FormField>

              <FormField name="data_gb" :label="t('customer.data_gb')">
                <template #default="{ fieldId }">
                  <Input :id="fieldId" v-model="form.data_gb" type="number" :placeholder="t('customer.placeholder_plan_default')" />
                </template>
              </FormField>

              <FormField name="speed_mbps" :label="t('customer.speed_mbps')">
                <template #default="{ fieldId }">
                  <Input :id="fieldId" v-model="form.speed_mbps" type="number" :placeholder="t('customer.placeholder_plan_default')" />
                </template>
              </FormField>
            </div>

            <FormField name="notes" :label="t('customer.notes')">
              <template #default="{ fieldId }">
                <Textarea :id="fieldId" v-model="form.notes" rows="3" />
              </template>
            </FormField>

            <FormField v-if="!isReseller && !isResellerCreated" name="user-avatar" :label="t('user.avatar')">
              <template #default>
                <div class="emoji-picker">
                  <button
                    v-for="em in availableUserEmojis"
                    :key="em"
                    type="button"
                    class="emoji-btn"
                    :class="{ 'emoji-btn--active': form.avatar === em }"
                    @click="form.avatar = form.avatar === em ? '' : em"
                  >{{ em }}</button>
                  <button
                    v-for="em in reservedEmojiList"
                    :key="'reserved-' + em.emoji"
                    type="button"
                    class="emoji-btn emoji-btn--reserved"
                    disabled
                    :title="`Used by reseller: ${em.reseller}`"
                  >{{ em.emoji }}</button>
                </div>
              </template>
            </FormField>

            <FormField v-if="isResellerCreated || isReseller" name="billing_mode" :label="t('customer.billing_mode')">
              <template #default="{ fieldId }">
                <Select
                  :id="fieldId"
                  v-model="form.billing_mode"
                  :options="[
                    { label: t('customer.billing_inherit'), value: '' },
                    { label: t('customer.billing_manual'), value: 'manual' },
                    { label: t('customer.billing_self_service'), value: 'self_service' },
                  ]"
                />
              </template>
            </FormField>

            <div class="form-actions">
              <Button type="submit" variant="primary" :loading="saving">{{ t('customer.save_changes') }}</Button>
            </div>
          </form>

          <!-- Change Plan Section -->
          <div v-if="!isNew && customer" class="plan-cards-section">
            <h4 class="section-title">Plan</h4>
            <div class="plan-cards">
              <div
                v-for="plan in plans"
                :key="plan.id"
                class="plan-card"
                :class="{
                  'plan-card--active': customer.plan_id === plan.id,
                  'plan-card--selected': selectedPlanId === plan.id && customer.plan_id !== plan.id,
                }"
                @click="selectedPlanId = plan.id"
              >
                <div class="plan-card__name">{{ plan.name }}</div>
                <div class="plan-card__price">${{ plan.price }}</div>
                <div class="plan-card__details">
                  <span v-if="plan.data_gb > 0">{{ plan.data_gb }} GB</span>
                  <span v-else>Unlimited</span>
                  <span>·</span>
                  <span v-if="plan.duration_days > 0">{{ plan.duration_days }} days</span>
                  <span v-else>Pay as you go</span>
                </div>
                <div v-if="customer.plan_id === plan.id" class="plan-card__badge">Current</div>
              </div>
            </div>
            <div v-if="selectedPlanId && selectedPlanId !== customer.plan_id" class="plan-actions">
              <Button
                variant="primary"
                size="sm"
                :loading="applyingPlan"
                @click="handleApplyPlan"
              >
                Apply Plan
              </Button>
              <Button
                variant="ghost"
                size="sm"
                :loading="switchingPlan"
                @click="handleSwitchPlan"
              >
                Switch (Refund to Wallet)
              </Button>
            </div>
          </div>
        </template>

        <!-- Usage Tab -->
        <template #usage>
          <div class="usage-tab">
            <div v-if="usage" class="usage-stats">
              <div class="usage-stat">
                <span class="usage-stat__label">{{ t('customer.status') }}</span>
                <StatusPill :status="usage.online ? 'online' : 'offline'" size="sm" />
              </div>
              <div class="usage-stat">
                <span class="usage-stat__label">{{ t('customer.active_sessions') }}</span>
                <span class="usage-stat__value">{{ usage.active_sessions }}</span>
              </div>
              <div class="usage-stat">
                <span class="usage-stat__label">{{ t('customer.total_download') }}</span>
                <span class="usage-stat__value">{{ formatBytes(usage.total_input_bytes) }}</span>
              </div>
              <div class="usage-stat">
                <span class="usage-stat__label">{{ t('customer.total_upload') }}</span>
                <span class="usage-stat__value">{{ formatBytes(usage.total_output_bytes) }}</span>
              </div>
              <div class="usage-stat">
                <span class="usage-stat__label">{{ t('customer.data_used') }}</span>
                <span class="usage-stat__value">{{ formatBytes(usage.total_usage_bytes) }}</span>
              </div>
            </div>

            <!-- Traffic Management Section (Requirements 3.4, 4.3) -->
            <div class="traffic-management">
              <!-- Traffic Reset Button (Requirement 3.4) -->
              <div class="traffic-management__row">
                <div class="traffic-management__info">
                  <h4 class="section-title">{{ t('customer.traffic_reset') }}</h4>
                  <p class="traffic-management__desc">{{ t('customer.traffic_reset_desc') }}</p>
                </div>
                <Button
                  variant="ghost"
                  size="sm"
                  :loading="resettingTraffic"
                  @click="handleTrafficReset"
                >
                  {{ t('customer.reset_traffic') }}
                </Button>
              </div>

              <!-- Connection Limit Inline Editor (Requirement 4.3) -->
              <div class="traffic-management__row">
                <div class="traffic-management__info">
                  <h4 class="section-title">{{ t('customer.connection_limit') }}</h4>
                  <p class="traffic-management__desc">{{ t('customer.connection_limit_desc') }}</p>
                </div>
                <div class="connection-limit-editor">
                  <template v-if="!editingConnectionLimit">
                    <span class="connection-limit-editor__value">
                      {{ currentConnectionLimit === 0 ? t('templates.unlimited') : currentConnectionLimit }}
                    </span>
                    <Button variant="ghost" size="sm" @click="startEditConnectionLimit">
                      {{ t('btn.edit') }}
                    </Button>
                  </template>
                  <template v-else>
                    <input
                      v-model.number="connectionLimitInput"
                      type="number"
                      min="0"
                      class="connection-limit-editor__input"
                      :aria-label="t('customer.connection_limit')"
                    />
                    <Button
                      variant="primary"
                      size="sm"
                      :loading="savingConnectionLimit"
                      @click="saveConnectionLimit"
                    >
                      {{ t('btn.save') }}
                    </Button>
                    <Button variant="ghost" size="sm" @click="cancelEditConnectionLimit">
                      {{ t('btn.cancel') }}
                    </Button>
                  </template>
                </div>
              </div>
            </div>

            <!-- MTProto Secret Section (Requirements 7.6, 5.4) -->
            <div v-if="mtprotoSecret" class="mtproto-section">
              <h4 class="section-title">MTProto Proxy</h4>

              <!-- Secret Display -->
              <div class="mtproto-secret-row">
                <div class="mtproto-secret-label">Secret</div>
                <div class="mtproto-secret-value">
                  <code class="mtproto-secret-code">{{ mtprotoSecret.secret }}</code>
                  <Button
                    variant="ghost"
                    size="sm"
                    :aria-label="mtprotoCopied ? 'Copied' : 'Copy secret to clipboard'"
                    @click="copyMTProtoSecret"
                  >
                    {{ mtprotoCopied ? '✓ Copied' : '📋 Copy' }}
                  </Button>
                </div>
              </div>

              <!-- Connection Info -->
              <div class="mtproto-secret-row">
                <div class="mtproto-secret-label">Connections</div>
                <div class="mtproto-connections">
                  <span class="mtproto-connections__count">
                    {{ mtprotoSecret.connections }} / {{ mtprotoSecret.connection_limit === 0 ? '∞' : mtprotoSecret.connection_limit }}
                  </span>
                  <span class="mtproto-connections__label">active</span>
                </div>
              </div>

              <!-- Regenerate -->
              <div class="mtproto-secret-row">
                <div class="mtproto-secret-label">Actions</div>
                <Button
                  variant="ghost"
                  size="sm"
                  :loading="mtprotoRegenerating"
                  @click="regenerateMTProtoSecret"
                >
                  🔄 Regenerate Secret
                </Button>
              </div>
            </div>
            <div v-else-if="mtprotoLoading" class="mtproto-section">
              <h4 class="section-title">MTProto Proxy</h4>
              <Skeleton variant="rect" :width="'100%'" :height="60" />
            </div>

            <!-- Sessions Table -->
            <h4 class="section-title">{{ t('customer.sessions') }}</h4>
            <table class="mini-table" role="table">
              <thead>
                <tr><th>IP</th><th>{{ t('customer.th_start') }}</th><th>{{ t('customer.th_duration') }}</th><th>{{ t('customer.th_traffic') }}</th><th>{{ t('customer.th_status') }}</th></tr>
              </thead>
              <tbody>
                <tr v-for="s in usage?.sessions?.slice(0, 10)" :key="s.id">
                  <td>{{ s.framed_ip }}</td>
                  <td class="text-muted">{{ formatDateTime(s.start_time) }}</td>
                  <td>{{ Math.floor(s.session_seconds / 60) }}m</td>
                  <td>{{ formatBytes(s.total_bytes) }}</td>
                  <td><StatusPill :status="s.online ? 'online' : 'offline'" size="sm" /></td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <!-- History Tab -->
        <template #history>
          <div class="history-tab">
            <h4 class="section-title">{{ t('customer.wallet_transactions') }}</h4>
            <div v-if="!customer.wallet_transactions?.length" class="text-muted text-sm">No transactions yet.</div>
            <table v-else class="mini-table" role="table">
              <thead>
                <tr><th>{{ t('customer.th_date') }}</th><th>{{ t('customer.th_type') }}</th><th>{{ t('customer.th_amount') }}</th><th>{{ t('customer.th_description') }}</th></tr>
              </thead>
              <tbody>
                <tr v-for="tx in customer.wallet_transactions" :key="tx.id">
                  <td class="text-muted">{{ formatDate(tx.created_at) }}</td>
                  <td>{{ tx.type }}</td>
                  <td :class="{ 'text-success': tx.amount > 0, 'text-danger': tx.amount < 0 }">
                    ${{ tx.amount.toFixed(2) }}
                  </td>
                  <td>{{ tx.description }}</td>
                </tr>
              </tbody>
            </table>

            <h4 class="section-title">{{ t('customer.subscriptions') }}</h4>
            <div v-if="!customer.subscriptions?.length" class="text-muted text-sm">No subscriptions yet.</div>
            <table v-else class="mini-table" role="table">
              <thead>
                <tr><th>{{ t('customer.th_plan') }}</th><th>{{ t('customer.th_start') }}</th><th>{{ t('customer.th_end') }}</th><th>{{ t('customer.th_status') }}</th></tr>
              </thead>
              <tbody>
                <tr v-for="sub in customer.subscriptions" :key="sub.id">
                  <td>{{ sub.plan_name }}</td>
                  <td class="text-muted">{{ sub.started_at ? formatDate(sub.started_at) : 'Pending' }}</td>
                  <td class="text-muted">{{ sub.expires_at ? formatDate(sub.expires_at) : 'Unlimited' }}</td>
                  <td><StatusPill :status="sub.status" size="sm" /></td>
                </tr>
              </tbody>
            </table>
          </div>
        </template>

        <!-- Custom Fields Tab -->
        <template #custom_fields>
          <div class="custom-fields-tab">
            <div v-if="customFieldsLoading" class="loading-state">
              <Skeleton variant="rect" :width="'100%'" :height="200" />
            </div>
            <div v-else-if="customFields.length === 0" class="text-muted text-sm" style="padding: var(--space-4) 0;">
              {{ t('customer.custom_fields_empty') }}
            </div>
            <template v-else>
              <form class="profile-form" @submit.prevent="saveCustomFields">
                <div class="form-grid">
                  <FormField
                    v-for="field in customFields"
                    :key="field.key"
                    :name="`cf-${field.key}`"
                    :label="field.label || field.key"
                  >
                    <template #default="{ fieldId }">
                      <Input :id="fieldId" v-model="field.value" />
                    </template>
                  </FormField>
                </div>
                <div class="form-actions">
                  <Button type="submit" variant="primary" :loading="customFieldsSaving">{{ t('btn.save') }}</Button>
                </div>
              </form>
            </template>
          </div>
        </template>

        <!-- Notes Tab -->
        <template #notes>
          <div class="notes-tab">
            <!-- Add Note Form -->
            <form class="note-form" @submit.prevent="addNote">
              <FormField name="new-note" :label="t('customer.add_note')">
                <template #default="{ fieldId }">
                  <textarea
                    :id="fieldId"
                    v-model="newNoteContent"
                    class="note-textarea"
                    rows="3"
                    :placeholder="t('customer.note_placeholder')"
                  />
                </template>
              </FormField>
              <div class="form-actions">
                <Button type="submit" variant="primary" size="sm" :loading="addingNote" :disabled="!newNoteContent.trim()">{{ t('customer.add_note') }}</Button>
              </div>
            </form>

            <!-- Notes List -->
            <div v-if="notesLoading" class="loading-state">
              <Skeleton variant="rect" :width="'100%'" :height="100" />
            </div>
            <div v-else-if="notes.length === 0" class="text-muted text-sm" style="padding: var(--space-4) 0;">
              {{ t('customer.notes_empty') }}
            </div>
            <div v-else class="notes-list">
              <div v-for="note in notes" :key="note.id" class="note-card">
                <div class="note-card__header">
                  <span class="note-card__author">{{ note.created_by }}</span>
                  <span class="note-card__date text-muted">{{ formatDate(note.created_at) }}</span>
                </div>
                <p class="note-card__content">{{ note.content }}</p>
              </div>
            </div>
          </div>
        </template>

        <!-- Activity Tab (Placeholder) -->
        <template #activity>
          <div class="activity-tab">
            <EmptyState
              icon="📋"
              :title="t('customer.tab_activity')"
              description="Activity and audit log coming soon."
            />
          </div>
        </template>

      </Tabs>
    </template>

    <!-- Not Found -->
    <div v-else class="empty-state">
      <p class="text-muted">Customer not found.</p>
      <Button variant="ghost" @click="router.back()">{{ t('common.go_back') }}</Button>
    </div>
  </div>
</template>

<style scoped>
.customer-detail { display: flex; flex-direction: column; gap: var(--space-5); }
.loading-state { display: flex; flex-direction: column; gap: var(--space-4); }

.detail-header { display: flex; justify-content: space-between; align-items: center; padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); }
.detail-header__left { display: flex; align-items: center; gap: var(--space-4); }
.detail-header__info { display: flex; flex-direction: column; gap: var(--space-1); }
.detail-header__username { margin: 0; font-size: var(--text-lg); font-weight: var(--font-bold); }
.detail-header__meta { display: flex; align-items: center; gap: var(--space-3); }
.detail-header__balance { font-size: var(--text-sm); font-weight: var(--font-semibold); color: var(--color-accent); }

.wallet-adjust-panel { padding: var(--space-3) var(--space-4); background: var(--color-surface-2); border: 1px solid var(--color-border); border-radius: var(--radius-md); margin-bottom: var(--space-4); }
.wallet-adjust-form { display: flex; align-items: flex-end; gap: var(--space-3); }
.wallet-adjust-actions { display: flex; gap: var(--space-2); }

.profile-form { display: flex; flex-direction: column; gap: var(--space-4); padding: var(--space-4) 0; }
.form-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4); }
.form-actions { display: flex; justify-content: flex-end; padding-top: var(--space-3); }

.usage-tab { display: flex; flex-direction: column; gap: var(--space-4); padding: var(--space-4) 0; }
.usage-stats { display: grid; grid-template-columns: repeat(auto-fit, minmax(150px, 1fr)); gap: var(--space-3); }
.usage-stat { display: flex; flex-direction: column; gap: var(--space-1); padding: var(--space-3); background: var(--color-surface-2); border-radius: var(--radius-md); }
.usage-stat__label { font-size: var(--text-xs); color: var(--color-muted); text-transform: uppercase; }
.usage-stat__value { font-size: var(--text-lg); font-weight: var(--font-bold); }

.history-tab { display: flex; flex-direction: column; gap: var(--space-4); padding: var(--space-4) 0; }
.section-title { margin: 0; font-size: var(--text-sm); font-weight: var(--font-semibold); color: var(--color-text); }

.mini-table { width: 100%; border-collapse: collapse; font-size: var(--text-sm); }
.mini-table th { text-align: left; padding: var(--space-2) var(--space-3); color: var(--color-muted); font-size: var(--text-xs); text-transform: uppercase; border-bottom: 1px solid var(--color-border); }
.mini-table td { padding: var(--space-2) var(--space-3); border-bottom: 1px solid var(--color-border); color: var(--color-text); }

.text-muted { color: var(--color-muted); }
.text-success { color: var(--color-success); }
.text-danger { color: var(--color-danger); }
.empty-state { text-align: center; padding: var(--space-12); }

.traffic-management { display: flex; flex-direction: column; gap: var(--space-4); padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); }
.traffic-management__row { display: flex; align-items: center; justify-content: space-between; gap: var(--space-4); }
.traffic-management__info { display: flex; flex-direction: column; gap: var(--space-1); }
.traffic-management__desc { margin: 0; font-size: var(--text-xs); color: var(--color-muted); }

.connection-limit-editor { display: flex; align-items: center; gap: var(--space-2); }
.connection-limit-editor__value { font-size: var(--text-sm); font-weight: var(--font-semibold); color: var(--color-text); min-width: 60px; }
.connection-limit-editor__input { width: 80px; padding: var(--space-1) var(--space-2); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text); font-size: var(--text-sm); outline: none; transition: border-color var(--duration-normal); }
.connection-limit-editor__input:focus { border-color: var(--color-primary); }

/* MTProto Secret Section (Requirements 7.6, 5.4) */
.mtproto-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}
.mtproto-secret-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.mtproto-secret-label {
  min-width: 100px;
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  font-weight: var(--font-semibold);
}
.mtproto-secret-value {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex: 1;
  min-width: 0;
}
.mtproto-secret-code {
  font-family: 'JetBrains Mono', 'Fira Code', 'Cascadia Code', monospace;
  font-size: var(--text-xs);
  padding: var(--space-1) var(--space-2);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  word-break: break-all;
  color: var(--color-text);
  max-width: 480px;
  overflow: hidden;
  text-overflow: ellipsis;
}
.mtproto-connections {
  display: flex;
  align-items: baseline;
  gap: var(--space-1);
}
.mtproto-connections__count {
  font-size: var(--text-base);
  font-weight: var(--font-bold);
  color: var(--color-text);
}
.mtproto-connections__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

@media (max-width: 768px) {
  .form-grid { grid-template-columns: 1fr; }
  .traffic-management__row { flex-direction: column; align-items: flex-start; }
  .mtproto-secret-row { flex-direction: column; align-items: flex-start; }
  .mtproto-secret-code { max-width: 100%; }
}

.detail-header__plan {
  font-size: var(--text-sm);
  color: var(--color-muted);
  padding: 2px 8px;
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
}

.plan-cards-section {
  margin-top: var(--space-6);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}
.plan-cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(160px, 1fr));
  gap: var(--space-3);
  margin-top: var(--space-3);
}
.plan-card {
  position: relative;
  padding: var(--space-4);
  background: var(--color-surface-2);
  border: 2px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: all 0.2s ease;
  text-align: center;
}
.plan-card:hover {
  border-color: var(--color-primary);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(37, 99, 235, 0.1);
}
.plan-card--active {
  border-color: var(--color-success, #22c55e);
  background: rgba(34, 197, 94, 0.06);
  transform: scale(1.03);
}
.plan-card--selected {
  border-color: var(--color-primary);
  background: rgba(37, 99, 235, 0.06);
}
.plan-card__name {
  font-size: var(--text-sm);
  font-weight: 700;
  margin-bottom: var(--space-1);
}
.plan-card__price {
  font-size: var(--text-xl);
  font-weight: 800;
  color: var(--color-primary);
  margin-bottom: var(--space-2);
}
.plan-card--active .plan-card__price {
  color: var(--color-success, #22c55e);
}
.plan-card__details {
  font-size: var(--text-xs);
  color: var(--color-muted);
  display: flex;
  gap: var(--space-1);
  justify-content: center;
}
.plan-card__badge {
  position: absolute;
  top: -8px;
  right: -8px;
  padding: 2px 8px;
  background: var(--color-success, #22c55e);
  color: #fff;
  font-size: 10px;
  font-weight: 700;
  border-radius: var(--radius-full);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}
.plan-actions {
  display: flex;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Emoji Picker for user avatar */
.emoji-picker {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.emoji-btn {
  width: 36px;
  height: 36px;
  font-size: 20px;
  border: 2px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s ease;
}
.emoji-btn:hover {
  border-color: var(--color-primary, #2563eb);
  background: rgba(37, 99, 235, 0.08);
}
.emoji-btn--active {
  border-color: var(--color-primary, #2563eb);
  background: rgba(37, 99, 235, 0.15);
  transform: scale(1.1);
}

.emoji-btn--reserved {
  opacity: 0.35;
  cursor: not-allowed;
  filter: grayscale(0.7);
}

.emoji-btn--reserved:hover {
  border-color: var(--color-border, #28333f);
  background: var(--color-surface, #0b1120);
}

/* Custom Fields Tab */
.custom-fields-tab { padding: var(--space-4) 0; }

/* Notes Tab */
.notes-tab { display: flex; flex-direction: column; gap: var(--space-4); padding: var(--space-4) 0; }

.note-form { display: flex; flex-direction: column; gap: var(--space-3); padding-bottom: var(--space-4); border-bottom: 1px solid var(--color-border); }

.note-textarea {
  width: 100%;
  padding: var(--space-3);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  color: var(--color-text);
  font-size: var(--text-sm);
  font-family: inherit;
  resize: vertical;
  min-height: 80px;
}

.note-textarea:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px rgba(37, 99, 235, 0.15);
}

.notes-list { display: flex; flex-direction: column; gap: var(--space-3); }

.note-card {
  padding: var(--space-3) var(--space-4);
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.note-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-2);
}

.note-card__author {
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--color-primary);
}

.note-card__date {
  font-size: var(--text-xs);
}

.note-card__content {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-text);
  white-space: pre-wrap;
  line-height: 1.5;
}

/* Activity Tab */
.activity-tab { padding: var(--space-8) 0; }

.text-sm { font-size: var(--text-sm); }
</style>
