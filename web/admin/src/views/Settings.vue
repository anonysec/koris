<script setup lang="ts">
import { ref, computed, onMounted, watch, reactive } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import { useTheme, availableThemes } from '@koris/composables/useTheme'
import type { ThemeMode, UITheme } from '@koris/composables/useTheme'
import type { Locale } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Tabs from '@koris/ui/Tabs.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import SettingsDatabaseSection from '@/components/settings/SettingsDatabaseSection.vue'
import SettingsTLSSection from '@/components/settings/SettingsTLSSection.vue'
import SettingsWorkersSection from '@/components/settings/SettingsWorkersSection.vue'
import SettingsAlertsSection from '@/components/settings/SettingsAlertsSection.vue'
import SettingsGrpcSection from '@/components/settings/SettingsGrpcSection.vue'
import SettingsPanelInfoSection from '@/components/settings/SettingsPanelInfoSection.vue'
import Backup from '@/views/Backup.vue'

const { t, locale: currentLocale, setLocale } = useI18n()
const { mode: currentMode, theme: currentTheme, setMode, setTheme } = useTheme()
const { get, post, put, patch, del } = useApi()
const toast = useToast()

// ─── Search ──────────────────────────────────────────────────────────────────
const searchQuery = ref('')
const searchFocused = ref(false)

// ─── Category tabs ───────────────────────────────────────────────────────────
const activeTab = ref('general')
const sectionCategories: Record<string, string[]> = {
  general: ['appearance', 'maintenance', 'update'],
  security: ['apikeys', 'audit'],
  notifications: ['notifications'],
  system: ['system', 'backup'],
  integrations: ['links'],
}
const tabs = computed(() => {
  const list = Object.keys(sectionCategories)
  if (!searchQuery.value) return list.map(k => ({ key: k, label: t('settings.cat_' + k) }))
  return list
    .filter(k => sectionCategories[k].some(id => isVisible(id)))
    .map(k => ({ key: k, label: t('settings.cat_' + k) }))
})
watch(searchQuery, () => {
  const visible = tabs.value.map(tb => tb.key)
  if (!visible.includes(activeTab.value) && visible.length) activeTab.value = visible[0]
})

// ─── Dirty tracking ─────────────────────────────────────────────────────────
const dirtySections = reactive<Set<string>>(new Set())
function markDirty(section: string) { dirtySections.add(section) }
function markClean(section: string) { dirtySections.delete(section) }
const hasUnsaved = computed(() => dirtySections.size > 0)

// ─── Panel Settings ─────────────────────────────────────────────────────────
const panelName = ref('')
const panelLang = ref<string>(currentLocale.value)
const selectedTheme = ref<UITheme>(currentTheme.value)
const selectedMode = ref<ThemeMode>(currentMode.value)
const loadingSettings = ref(false)
const savingSettings = ref(false)

watch(selectedTheme, (v) => setTheme(v))
watch(selectedMode, (v) => setMode(v))
watch(panelLang, (v) => { if (v !== currentLocale.value) setLocale(v as Locale) })

const modeOptions: { value: ThemeMode; labelKey: string }[] = [
  { value: 'system', labelKey: 'settings.mode_system' },
  { value: 'dark', labelKey: 'settings.mode_dark' },
  { value: 'light', labelKey: 'settings.mode_light' },
]

async function loadPanelSettings() {
  loadingSettings.value = true
  try {
    const res = await get<any>('/api/panel-settings')
    if (res?.settings) {
      panelName.value = res.settings.panel_name || ''
      const stored = typeof window !== 'undefined' ? localStorage.getItem('koris-lang') : null
      if (!stored) panelLang.value = res.settings.language || 'en'
      if (res.settings.ui_theme && availableThemes.some(t => t.id === res.settings.ui_theme)) {
        selectedTheme.value = res.settings.ui_theme as UITheme
        setTheme(res.settings.ui_theme as UITheme)
      }
      if (res.settings.ui_mode && ['system','dark','light'].includes(res.settings.ui_mode)) {
        selectedMode.value = res.settings.ui_mode as ThemeMode
        setMode(res.settings.ui_mode as ThemeMode)
      }
    }
  } catch {} finally { loadingSettings.value = false }
}

async function savePanelSettings() {
  savingSettings.value = true
  try {
    await patch('/api/panel-settings', {
      panel_name: panelName.value, language: panelLang.value,
      ui_theme: selectedTheme.value, ui_mode: selectedMode.value,
    })
    toast.success(t('settings.save_success'))
    markClean('appearance')
  } catch { toast.error(t('settings.save_error')) }
  finally { savingSettings.value = false }
}

// ─── Maintenance Mode ────────────────────────────────────────────────────────
const maintenance = reactive({ enabled: false, reason: '', enabled_by: '', enabled_at: '' })
const maintenanceReason = ref('')
const savingMaintenance = ref(false)
const loadingMaintenance = ref(false)

async function loadMaintenance() {
  loadingMaintenance.value = true
  try {
    const res = await get<any>('/api/settings/maintenance-mode')
    if (res) {
      maintenance.enabled = res.enabled
      maintenance.reason = res.reason || ''
      maintenance.enabled_by = res.enabled_by || ''
      maintenance.enabled_at = res.enabled_at || ''
      maintenanceReason.value = res.reason || ''
    }
  } catch {} finally { loadingMaintenance.value = false }
}

async function toggleMaintenance() {
  savingMaintenance.value = true
  try {
    await post('/api/settings/maintenance-mode', {
      enabled: !maintenance.enabled,
      reason: maintenanceReason.value || (maintenance.enabled ? '' : 'Scheduled maintenance'),
    })
    maintenance.enabled = !maintenance.enabled
    maintenance.reason = maintenanceReason.value
    toast.success(maintenance.enabled ? t('settings.maintenance_enabled') : t('settings.maintenance_disabled'))
  } catch { toast.error(t('settings.maintenance_error')) }
  finally { savingMaintenance.value = false }
}

// ─── API Keys ────────────────────────────────────────────────────────────────
interface ApiKey { id: number; name: string; key_prefix: string; scopes: string; last_used_at: string; created_at: string; created_by: string }
const apiKeys = ref<ApiKey[]>([])
const newKeyName = ref('')
const newKeyScopes = ref('read')
const createdKey = ref('')
const loadingKeys = ref(false)
const creatingKey = ref(false)

async function loadApiKeys() {
  loadingKeys.value = true
  try {
    const res = await get<any>('/api/settings/api-keys')
    if (res?.keys) apiKeys.value = res.keys
  } catch {} finally { loadingKeys.value = false }
}

async function createApiKey() {
  if (!newKeyName.value.trim()) return
  creatingKey.value = true
  try {
    const res = await post<any>('/api/settings/api-keys', { name: newKeyName.value.trim(), scopes: newKeyScopes.value })
    if (res?.key) {
      createdKey.value = res.key
      newKeyName.value = ''
      await loadApiKeys()
    }
  } catch { toast.error(t('settings.api_key_error')) }
  finally { creatingKey.value = false }
}

async function deleteApiKey(id: number) {
  if (!confirm(t('settings.revoke_confirm'))) return
  try {
    await del('/api/settings/api-keys', { id })
    await loadApiKeys()
    toast.success(t('settings.api_key_revoked'))
  } catch { toast.error(t('settings.revoke_error')) }
}

function copyKey() {
  navigator.clipboard.writeText(createdKey.value)
  toast.success(t('settings.copied'))
}

// ─── Audit Logs ──────────────────────────────────────────────────────────────
interface AuditEntry { id: number; actor: string; action: string; entity_type: string; entity_id: string; before_json: string; after_json: string; ip: string; created_at: string }
const auditLogs = ref<AuditEntry[]>([])
const auditSearch = ref('')
const auditOffset = ref(0)
const auditLimit = 50
const loadingAudit = ref(false)

async function loadAuditLogs() {
  loadingAudit.value = true
  try {
    const res = await get<any>(`/api/audit-logs?limit=${auditLimit}&offset=${auditOffset.value}`)
    if (res?.logs) auditLogs.value = res.logs
  } catch {} finally { loadingAudit.value = false }
}

const filteredAuditLogs = computed(() => {
  if (!auditSearch.value) return auditLogs.value
  const q = auditSearch.value.toLowerCase()
  return auditLogs.value.filter(l =>
    l.actor.toLowerCase().includes(q) ||
    l.action.toLowerCase().includes(q) ||
    l.entity_type.toLowerCase().includes(q) ||
    l.ip.toLowerCase().includes(q)
  )
})

function auditPage(dir: number) {
  auditOffset.value = Math.max(0, auditOffset.value + dir * auditLimit)
  loadAuditLogs()
}

function formatTime(iso: string) {
  if (!iso) return ''
  try { return new Date(iso).toLocaleString() } catch { return iso }
}

// ─── Update Check ────────────────────────────────────────────────────────────
const updateInfo = reactive({ current_version: '', go_version: '', os: '', arch: '', uptime: '', latest_version: '', update_available: false })
const checkingUpdate = ref(false)

async function checkUpdate() {
  checkingUpdate.value = true
  try {
    const res = await get<any>('/api/settings/update-check')
    if (res) Object.assign(updateInfo, res)
  } catch {} finally { checkingUpdate.value = false }
}

// ─── Data Warnings ───────────────────────────────────────────────────────────
const thresholds = ref<number[]>([80, 95])
const expiryDays = ref<number[]>([7, 3, 1])
const connThresholds = ref<number[]>([80, 95])
const webhookUrl = ref('')
const savingWarnings = ref(false)
const loadingWarnings = ref(false)

async function loadWarnings() {
  loadingWarnings.value = true
  try {
    const res = await get<any>('/api/settings/warning-config')
    if (res?.config) {
      if (res.config.expiry_days?.length) expiryDays.value = res.config.expiry_days
      if (res.config.conn_thresholds?.length) connThresholds.value = res.config.conn_thresholds
      if (res.config.webhook_url) webhookUrl.value = res.config.webhook_url
    }
    const res2 = await get<any>('/api/settings/data-warning-thresholds')
    if (res2?.thresholds?.length) thresholds.value = res2.thresholds
  } catch {} finally { loadingWarnings.value = false }
}

async function saveWarnings() {
  savingWarnings.value = true
  try {
    await put('/api/settings/warning-config', {
      expiry_days: [...new Set(expiryDays.value)].sort((a,b) => b-a),
      conn_thresholds: [...new Set(connThresholds.value)].sort((a,b) => a-b),
      webhook_url: webhookUrl.value.trim(),
    })
    await put('/api/settings/data-warning-thresholds', { thresholds: thresholds.value })
    toast.success(t('settings.warnings_saved'))
    markClean('warnings')
  } catch { toast.error(t('settings.warnings_error')) }
  finally { savingWarnings.value = false }
}

// ─── App Links ───────────────────────────────────────────────────────────────
interface AppLink { name: string; url: string; platform: string; icon: string }
const appLinks = ref<AppLink[]>([])
const savingLinks = ref(false)
const loadingLinks = ref(false)
const platformIcons: Record<string,string> = { ios:'🍎', android:'🤖', windows:'🪟', mac:'💻', other:'🔗' }
const platforms = ['ios','android','windows','mac','other']

async function loadAppLinks() {
  loadingLinks.value = true
  try {
    const res = await get<any>('/api/panel-settings')
    if (res?.settings?.app_links) {
      try { const p = JSON.parse(res.settings.app_links); if (Array.isArray(p)) appLinks.value = p } catch {}
    }
  } catch {} finally { loadingLinks.value = false }
}

function addLink() { appLinks.value.push({ name: '', url: '', platform: 'other', icon: '🔗' }) }
function removeLink(i: number) { appLinks.value.splice(i, 1) }
function updateLinkPlatform(i: number) { appLinks.value[i].icon = platformIcons[appLinks.value[i].platform] || '🔗' }

async function saveLinks() {
  savingLinks.value = true
  try {
    await patch('/api/panel-settings', { app_links: JSON.stringify(appLinks.value.filter(l => l.name && l.url)) })
    toast.success(t('settings.links_saved'))
    markClean('links')
  } catch { toast.error(t('settings.links_error')) }
  finally { savingLinks.value = false }
}

// ─── Telegram ────────────────────────────────────────────────────────────────
const telegramToken = ref('')
const telegramChats = ref('')
const telegramEnabled = ref(false)
const savingTelegram = ref(false)

async function loadTelegram() {
  try {
    const res = await get<any>('/api/panel-settings')
    if (res?.settings) {
      telegramToken.value = res.settings.telegram_bot_token || ''
      telegramChats.value = res.settings.telegram_admin_chats || ''
      telegramEnabled.value = !!telegramToken.value
    }
  } catch {}
}

async function saveTelegram() {
  savingTelegram.value = true
  try {
    await patch('/api/panel-settings', {
      telegram_bot_token: telegramToken.value,
      telegram_admin_chats: telegramChats.value,
    })
    toast.success(t('settings.telegram_saved'))
    markClean('telegram')
  } catch { toast.error(t('settings.telegram_error')) }
  finally { savingTelegram.value = false }
}

// ─── Filtered sections for search ────────────────────────────────────────────
const sections = [
  { id: 'appearance', keywords: 'appearance theme mode dark light language panel name ui' },
  { id: 'maintenance', keywords: 'maintenance mode toggle downtime offline' },
  { id: 'apikeys', keywords: 'api keys tokens integration external access revoke' },
  { id: 'audit', keywords: 'audit log history activity track actor action' },
  { id: 'update', keywords: 'update version upgrade check release' },
  { id: 'notifications', keywords: 'notifications telegram webhook alerts thresholds warnings expiry data' },
  { id: 'links', keywords: 'app links download ios android windows mac' },
  { id: 'system', keywords: 'system database tls certificates workers grpc panel info' },
  { id: 'backup', keywords: 'backup restore export import database' },
]

const visibleSections = computed(() => {
  if (!searchQuery.value) return sections.map(s => s.id)
  const q = searchQuery.value.toLowerCase()
  return sections.filter(s => s.id.includes(q) || s.keywords.includes(q)).map(s => s.id)
})

function isVisible(id: string) { return visibleSections.value.includes(id) }

// ─── Init ────────────────────────────────────────────────────────────────────
onMounted(() => {
  loadPanelSettings()
  loadMaintenance()
  loadApiKeys()
  loadAuditLogs()
  checkUpdate()
  loadWarnings()
  loadAppLinks()
  loadTelegram()
})
</script>

<template>
  <div class="settings">
    <!-- Unsaved changes banner -->
    <Transition name="slide">
      <div v-if="hasUnsaved" class="unsaved-banner">
        <span class="unsaved-dot"></span>
        <span>{{ t('settings.unsaved_changes') }} ({{ dirtySections.size }} {{ dirtySections.size === 1 ? t('settings.section') : t('settings.sections') }})</span>
      </div>
    </Transition>

    <!-- Search -->
    <div class="settings-search" :class="{ focused: searchFocused }">
      <svg class="search-icon" width="18" height="18" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"><circle cx="11" cy="11" r="8"/><path d="m21 21-4.3-4.3"/></svg>
      <input
        v-model="searchQuery"
        type="text"
        :placeholder="t('settings.search_placeholder')"
        class="search-input"
        @focus="searchFocused = true"
        @blur="searchFocused = false"
      />
      <button v-if="searchQuery" class="search-clear" @click="searchQuery = ''">✕</button>
    </div>

    <!-- Category tabs -->
    <Tabs v-model="activeTab" :tabs="tabs" aria-label="Settings categories" class="settings-tabs">
      <!-- ─── General ─── -->
      <template #general>
        <!-- Appearance -->
        <section v-if="isVisible('appearance')" class="card" id="appearance">
          <div class="card-header">
            <div class="card-icon">🎨</div>
            <div>
              <h3 class="card-title">{{ t('settings.appearance') }}</h3>
              <p class="card-desc">{{ t('settings.appearance_desc') }}</p>
            </div>
            <span v-if="dirtySections.has('appearance')" class="dirty-badge">{{ t('settings.modified') }}</span>
          </div>
          <Skeleton v-if="loadingSettings" variant="rect" width="100%" :height="240" />
          <div v-else class="card-body">
            <div class="field-row">
              <label class="field-label">{{ t('settings.panel_name') }}</label>
              <input v-model="panelName" class="field-input" :placeholder="t('settings.panel_name')" @input="markDirty('appearance')" />
            </div>
            <div class="field-row">
              <label class="field-label">{{ t('settings.language') }}</label>
              <select v-model="panelLang" class="field-select" @change="markDirty('appearance')">
                <option value="en">English</option>
                <option value="fa">فارسی</option>
                <option value="ru">Русский</option>
                <option value="zh">中文</option>
              </select>
            </div>
            <div class="field-row">
              <label class="field-label">{{ t('settings.display_mode') }}</label>
              <div class="mode-group">
                <button v-for="m in modeOptions" :key="m.value" class="mode-btn" :class="{ active: selectedMode === m.value }" @click="selectedMode = m.value; markDirty('appearance')">
                  {{ t(m.labelKey) }}
                </button>
              </div>
            </div>
            <div class="field-row">
              <label class="field-label">{{ t('settings.theme') }}</label>
              <div class="theme-grid">
                <button v-for="th in availableThemes" :key="th.id" class="theme-card" :class="{ active: selectedTheme === th.id }" @click="selectedTheme = th.id; markDirty('appearance')">
                  <div class="theme-swatches">
                    <span v-for="(c, ci) in (th as any).swatches || []" :key="ci" class="swatch" :style="{ background: c }"></span>
                  </div>
                  <span class="theme-name">{{ (th as any).name || th.id }}</span>
                </button>
              </div>
            </div>
            <div class="card-actions">
              <Button variant="primary" :loading="savingSettings" @click="savePanelSettings">{{ t('settings.save') }}</Button>
              <span v-if="dirtySections.has('appearance')" class="reset-link" @click="loadPanelSettings(); markClean('appearance')">{{ t('settings.reset') }}</span>
            </div>
          </div>
        </section>

        <!-- Maintenance Mode -->
        <section v-if="isVisible('maintenance')" class="card" id="maintenance">
          <div class="card-header">
            <div class="card-icon">🔧</div>
            <div>
              <h3 class="card-title">{{ t('settings.maintenance') }}</h3>
              <p class="card-desc">{{ t('settings.maintenance_desc') }}</p>
            </div>
            <StatusPill :status="maintenance.enabled ? 'active' : 'inactive'" :label="maintenance.enabled ? t('settings.active') : t('settings.inactive')" />
          </div>
          <Skeleton v-if="loadingMaintenance" variant="rect" width="100%" :height="170" />
          <div v-else class="card-body">
            <div v-if="maintenance.enabled" class="maintenance-info">
              <div class="info-row"><span class="info-label">{{ t('settings.reason') }}</span><span>{{ maintenance.reason || '—' }}</span></div>
              <div class="info-row"><span class="info-label">{{ t('settings.enabled_by') }}</span><span>{{ maintenance.enabled_by }}</span></div>
              <div class="info-row"><span class="info-label">{{ t('settings.since') }}</span><span>{{ formatTime(maintenance.enabled_at) }}</span></div>
            </div>
            <div class="field-row">
              <label class="field-label">{{ t('settings.reason') }}</label>
              <input v-model="maintenanceReason" class="field-input" :placeholder="t('settings.maintenance_reason_ph')" />
            </div>
            <div class="card-actions">
              <Button :variant="maintenance.enabled ? 'danger' : 'warning'" :loading="savingMaintenance" @click="toggleMaintenance">
                {{ maintenance.enabled ? t('settings.disable_maintenance') : t('settings.enable_maintenance') }}
              </Button>
            </div>
          </div>
        </section>

        <!-- Software Update -->
        <section v-if="isVisible('update')" class="card" id="update">
          <div class="card-header">
            <div class="card-icon">🔄</div>
            <div>
              <h3 class="card-title">{{ t('settings.software_update') }}</h3>
              <p class="card-desc">{{ t('settings.software_update_desc') }}</p>
            </div>
          </div>
          <Skeleton v-if="checkingUpdate" variant="rect" width="100%" :height="150" />
          <div v-else class="card-body">
            <div class="update-grid">
              <div class="info-row"><span class="info-label">{{ t('settings.lbl_version') }}</span><code>{{ updateInfo.current_version || 'dev' }}</code></div>
              <div class="info-row"><span class="info-label">{{ t('settings.lbl_go') }}</span><span>{{ updateInfo.go_version }}</span></div>
              <div class="info-row"><span class="info-label">{{ t('settings.lbl_platform') }}</span><span>{{ updateInfo.os }}/{{ updateInfo.arch }}</span></div>
              <div class="info-row"><span class="info-label">{{ t('settings.lbl_uptime') }}</span><span>{{ updateInfo.uptime }}</span></div>
            </div>
            <div class="card-actions">
              <Button variant="secondary" :loading="checkingUpdate" @click="checkUpdate">{{ t('settings.check_updates') }}</Button>
            </div>
          </div>
        </section>
      </template>

      <!-- ─── Security ─── -->
      <template #security>
        <!-- API Keys -->
        <section v-if="isVisible('apikeys')" class="card" id="apikeys">
          <div class="card-header">
            <div class="card-icon">🔑</div>
            <div>
              <h3 class="card-title">{{ t('settings.api_keys') }}</h3>
              <p class="card-desc">{{ t('settings.api_keys_desc') }}</p>
            </div>
            <span class="card-count">{{ apiKeys.length }} {{ apiKeys.length === 1 ? t('settings.key') : t('settings.keys') }}</span>
          </div>
          <Skeleton v-if="loadingKeys" variant="rect" width="100%" :height="200" />
          <div v-else class="card-body">
            <Transition name="slide">
              <div v-if="createdKey" class="key-created-banner">
                <div class="key-created-header">
                  <strong>🔑 {{ t('settings.key_created_title') }}</strong>
                  <span class="key-created-hint">{{ t('settings.key_created_hint') }}</span>
                </div>
                <div class="key-created-value">
                  <code>{{ createdKey }}</code>
                  <button class="copy-btn" @click="copyKey">📋 {{ t('settings.copy') }}</button>
                </div>
                <button class="dismiss-btn" @click="createdKey = ''">{{ t('settings.dismiss') }}</button>
              </div>
            </Transition>

            <div v-if="apiKeys.length" class="keys-list">
              <div v-for="key in apiKeys" :key="key.id" class="key-row">
                <div class="key-info">
                  <span class="key-name">{{ key.name }}</span>
                  <code class="key-prefix">{{ key.key_prefix }}</code>
                  <span class="key-meta">{{ key.scopes }} · {{ formatTime(key.created_at) }}{{ key.last_used_at ? ' · ' + formatTime(key.last_used_at) : '' }}</span>
                </div>
                <button class="revoke-btn" @click="deleteApiKey(key.id)">{{ t('settings.revoke') }}</button>
              </div>
            </div>
            <EmptyState v-else-if="!createdKey" icon="🔑" :title="t('settings.no_api_keys')" />

            <div class="create-key-form">
              <input v-model="newKeyName" class="field-input" :placeholder="t('settings.key_name_ph')" @keyup.enter="createApiKey" />
              <select v-model="newKeyScopes" class="field-select" style="max-width: 140px">
                <option value="read">{{ t('settings.scope_read') }}</option>
                <option value="read,write">{{ t('settings.scope_read_write') }}</option>
                <option value="admin">{{ t('settings.scope_admin') }}</option>
              </select>
              <Button variant="primary" :loading="creatingKey" :disabled="!newKeyName.trim()" @click="createApiKey">{{ t('settings.create_key') }}</Button>
            </div>
          </div>
        </section>

        <!-- Audit Log -->
        <section v-if="isVisible('audit')" class="card" id="audit">
          <div class="card-header">
            <div class="card-icon">📋</div>
            <div>
              <h3 class="card-title">{{ t('settings.audit_log') }}</h3>
              <p class="card-desc">{{ t('settings.audit_log_desc') }}</p>
            </div>
          </div>
          <Skeleton v-if="loadingAudit" variant="rect" width="100%" :height="220" />
          <div v-else class="card-body">
            <div class="audit-toolbar">
              <input v-model="auditSearch" class="field-input audit-search" :placeholder="t('settings.filter_by_actor')" />
              <div class="audit-pager">
                <button class="pager-btn" :disabled="auditOffset === 0" @click="auditPage(-1)">← {{ t('settings.prev') }}</button>
                <span class="pager-info">{{ auditOffset + 1 }}–{{ auditOffset + filteredAuditLogs.length }}</span>
                <button class="pager-btn" :disabled="filteredAuditLogs.length < auditLimit" @click="auditPage(1)">{{ t('settings.next') }} →</button>
              </div>
            </div>
            <div class="audit-table">
              <div class="audit-row audit-header-row">
                <span class="audit-col audit-time">{{ t('settings.audit_time') }}</span>
                <span class="audit-col audit-actor">{{ t('settings.audit_actor') }}</span>
                <span class="audit-col audit-action">{{ t('settings.audit_action') }}</span>
                <span class="audit-col audit-entity">{{ t('settings.audit_entity') }}</span>
                <span class="audit-col audit-ip">{{ t('settings.audit_ip') }}</span>
              </div>
              <div v-for="log in filteredAuditLogs" :key="log.id" class="audit-row">
                <span class="audit-col audit-time">{{ formatTime(log.created_at) }}</span>
                <span class="audit-col audit-actor">{{ log.actor }}</span>
                <span class="audit-col audit-action"><code>{{ log.action }}</code></span>
                <span class="audit-col audit-entity">{{ log.entity_type }}{{ log.entity_id ? ' #' + log.entity_id : '' }}</span>
                <span class="audit-col audit-ip">{{ log.ip }}</span>
              </div>
              <div v-if="!filteredAuditLogs.length" class="audit-empty">{{ t('settings.no_audit_entries') }}</div>
            </div>
          </div>
        </section>
      </template>

      <!-- ─── Notifications ─── -->
      <template #notifications>
        <section v-if="isVisible('notifications')" class="card" id="notifications">
          <div class="card-header">
            <div class="card-icon">🔔</div>
            <div>
              <h3 class="card-title">{{ t('settings.notifications_title') }}</h3>
              <p class="card-desc">{{ t('settings.notifications_desc') }}</p>
            </div>
            <span v-if="dirtySections.has('warnings')" class="dirty-badge">{{ t('settings.modified') }}</span>
          </div>
          <Skeleton v-if="loadingWarnings" variant="rect" width="100%" :height="340" />
          <div v-else class="card-body">
            <SettingsAlertsSection />

            <div class="divider"></div>

            <h4 class="subsection-title">{{ t('settings.data_warnings') }}</h4>
            <div class="threshold-list">
              <div v-for="(th, i) in thresholds" :key="i" class="threshold-row">
                <input type="number" :value="th" class="threshold-input" min="1" max="100" @input="thresholds[i] = parseInt(($event.target as HTMLInputElement).value) || 0; markDirty('warnings')" />
                <span class="threshold-unit">%</span>
                <button class="remove-btn" @click="thresholds.splice(i, 1); markDirty('warnings')" v-if="thresholds.length > 1">✕</button>
              </div>
              <button class="add-btn" @click="thresholds.push(50); markDirty('warnings')">{{ t('settings.add_threshold') }}</button>
            </div>

            <h4 class="subsection-title">{{ t('settings.expiry_warnings') }}</h4>
            <div class="threshold-list">
              <div v-for="(d, i) in expiryDays" :key="i" class="threshold-row">
                <input type="number" :value="d" class="threshold-input" min="1" @input="expiryDays[i] = parseInt(($event.target as HTMLInputElement).value) || 1; markDirty('warnings')" />
                <span class="threshold-unit">{{ t('settings.days_unit') }}</span>
                <button class="remove-btn" @click="expiryDays.splice(i, 1); markDirty('warnings')" v-if="expiryDays.length > 1">✕</button>
              </div>
              <button class="add-btn" @click="expiryDays.push(7); markDirty('warnings')">{{ t('settings.add') }}</button>
            </div>

            <h4 class="subsection-title">{{ t('settings.webhook_url') }}</h4>
            <input v-model="webhookUrl" class="field-input" :placeholder="t('settings.webhook_ph')" @input="markDirty('warnings')" />

            <div class="divider"></div>

            <h4 class="subsection-title">{{ t('settings.telegram_bot') }}</h4>
            <div class="field-row">
              <label class="field-label">{{ t('settings.bot_token') }}</label>
              <input v-model="telegramToken" type="password" class="field-input" :placeholder="t('settings.bot_token')" @input="markDirty('telegram')" />
            </div>
            <div class="field-row">
              <label class="field-label">{{ t('settings.admin_chat_ids') }}</label>
              <input v-model="telegramChats" class="field-input" :placeholder="'-100123456789, -100987654321'" @input="markDirty('telegram')" />
            </div>

            <div class="card-actions">
              <Button variant="primary" :loading="savingWarnings" @click="saveWarnings(); saveTelegram()">{{ t('settings.save_all') }}</Button>
              <span v-if="dirtySections.has('warnings')" class="reset-link" @click="loadWarnings(); markClean('warnings')">{{ t('settings.reset') }}</span>
            </div>
          </div>
        </section>
      </template>

      <!-- ─── System ─── -->
      <template #system>
        <section v-if="isVisible('system')" class="card" id="system">
          <div class="card-header">
            <div class="card-icon">⚙️</div>
            <div>
              <h3 class="card-title">{{ t('settings.system') }}</h3>
              <p class="card-desc">{{ t('settings.system_desc') }}</p>
            </div>
          </div>
          <div class="card-body system-grid">
            <SettingsPanelInfoSection />
            <SettingsDatabaseSection />
            <SettingsTLSSection />
            <SettingsWorkersSection />
            <SettingsAlertsSection />
            <SettingsGrpcSection />
          </div>
        </section>

        <!-- Backup -->
        <section v-if="isVisible('backup')" class="card" id="backup">
          <div class="card-header">
            <div class="card-icon">💾</div>
            <div>
              <h3 class="card-title">{{ t('settings.backup_restore') }}</h3>
              <p class="card-desc">{{ t('settings.backup_restore_desc') }}</p>
            </div>
          </div>
          <div class="card-body">
            <Backup />
          </div>
        </section>
      </template>

      <!-- ─── Integrations ─── -->
      <template #integrations>
        <section v-if="isVisible('links')" class="card" id="links">
          <div class="card-header">
            <div class="card-icon">📱</div>
            <div>
              <h3 class="card-title">{{ t('settings.client_app_links') }}</h3>
              <p class="card-desc">{{ t('settings.client_app_links_desc') }}</p>
            </div>
            <span v-if="dirtySections.has('links')" class="dirty-badge">{{ t('settings.modified') }}</span>
          </div>
          <Skeleton v-if="loadingLinks" variant="rect" width="100%" :height="170" />
          <div v-else class="card-body">
            <div v-for="(link, i) in appLinks" :key="i" class="link-row">
              <span class="link-icon">{{ link.icon }}</span>
              <select v-model="link.platform" class="field-select link-platform" @change="updateLinkPlatform(i); markDirty('links')">
                <option v-for="p in platforms" :key="p" :value="p">{{ p }}</option>
              </select>
              <input v-model="link.name" class="field-input link-name" :placeholder="t('settings.app_name_ph')" @input="markDirty('links')" />
              <input v-model="link.url" class="field-input link-url" :placeholder="t('settings.app_url_ph')" @input="markDirty('links')" />
              <button class="remove-btn" @click="removeLink(i); markDirty('links')">✕</button>
            </div>
            <button class="add-btn" @click="addLink(); markDirty('links')">{{ t('settings.add_link') }}</button>
            <div class="card-actions">
              <Button variant="primary" :loading="savingLinks" @click="saveLinks">{{ t('settings.save') }}</Button>
              <span v-if="dirtySections.has('links')" class="reset-link" @click="loadAppLinks(); markClean('links')">{{ t('settings.reset') }}</span>
            </div>
          </div>
        </section>
      </template>
    </Tabs>
  </div>
</template>

<style scoped>
.settings { display: flex; flex-direction: column; gap: var(--space-5); padding-bottom: var(--space-8); }

/* Unsaved banner */
.unsaved-banner { display: flex; align-items: center; gap: var(--space-2); padding: var(--space-3) var(--space-4); background: color-mix(in srgb, var(--color-warning) 12%, var(--color-surface)); border: 1px solid color-mix(in srgb, var(--color-warning) 30%, var(--color-border)); border-radius: var(--radius-lg); font-size: var(--text-sm); font-weight: var(--font-medium); color: var(--color-warning); }
.unsaved-dot { width: 8px; height: 8px; border-radius: 50%; background: var(--color-warning); animation: pulse 2s infinite; }
@keyframes pulse { 0%,100% { opacity: 1; } 50% { opacity: 0.4; } }

/* Search */
.settings-search { display: flex; align-items: center; gap: var(--space-2); padding: var(--space-3) var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); transition: border-color 0.15s, box-shadow 0.15s; }
.settings-search.focused { border-color: var(--color-primary); box-shadow: 0 0 0 3px color-mix(in srgb, var(--color-primary) 15%, transparent); }
.search-icon { color: var(--color-muted); flex-shrink: 0; }
.search-input { flex: 1; background: none; border: none; outline: none; color: var(--color-text); font-size: var(--text-base); }
.search-input::placeholder { color: var(--color-muted); }
.search-clear { background: none; border: none; color: var(--color-muted); cursor: pointer; font-size: var(--text-sm); padding: 2px 6px; }
.search-clear:hover { color: var(--color-text); }

/* Tabs */
.settings-tabs { background: transparent; }
.settings-tabs :deep(.k-tabs__list) { overflow-x: auto; scrollbar-width: thin; }
.settings-tabs :deep(.k-tabs__tab) { text-transform: capitalize; }

/* Cards */
.card { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-xl); overflow: hidden; transition: border-color 0.15s; }
.card:hover { border-color: color-mix(in srgb, var(--color-border) 80%, var(--color-text)); }
.card-header { display: flex; align-items: center; gap: var(--space-3); padding: var(--space-5) var(--space-5) 0; }
.card-icon { font-size: 1.4rem; width: 36px; height: 36px; display: flex; align-items: center; justify-content: center; background: var(--color-surface-2); border-radius: var(--radius-md); flex-shrink: 0; }
.card-title { margin: 0; font-size: var(--text-md); font-weight: var(--font-semibold); color: var(--color-text); }
.card-desc { margin: 2px 0 0; font-size: var(--text-sm); color: var(--color-muted); }
.card-count { font-size: var(--text-xs); color: var(--color-muted); background: var(--color-surface-2); padding: 2px 8px; border-radius: var(--radius-full); margin-left: auto; }
.dirty-badge { font-size: var(--text-xs); color: var(--color-warning); background: color-mix(in srgb, var(--color-warning) 12%, transparent); padding: 2px 8px; border-radius: var(--radius-full); margin-left: auto; font-weight: var(--font-medium); }
.card-body { padding: var(--space-4) var(--space-5) var(--space-5); display: flex; flex-direction: column; gap: var(--space-3); }
.card-actions { display: flex; align-items: center; gap: var(--space-3); margin-top: var(--space-2); }

/* Fields */
.field-row { display: flex; flex-direction: column; gap: var(--space-1); }
.field-label { font-size: var(--text-sm); font-weight: var(--font-medium); color: var(--color-text); }
.field-input { padding: var(--space-2) var(--space-3); background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius-md); color: var(--color-text); font-size: var(--text-sm); outline: none; transition: border-color 0.15s; }
.field-input:focus { border-color: var(--color-primary); }
.field-select { padding: var(--space-2) var(--space-3); background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius-md); color: var(--color-text); font-size: var(--text-sm); outline: none; }
.reset-link { font-size: var(--text-sm); color: var(--color-muted); cursor: pointer; text-decoration: underline; }
.reset-link:hover { color: var(--color-text); }

/* Mode buttons */
.mode-group { display: flex; gap: var(--space-2); }
.mode-btn { padding: var(--space-2) var(--space-4); background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius-md); color: var(--color-muted); font-size: var(--text-sm); font-weight: var(--font-medium); cursor: pointer; transition: all 0.15s; }
.mode-btn:hover { border-color: var(--color-primary); color: var(--color-text); }
.mode-btn.active { border-color: var(--color-primary); background: color-mix(in srgb, var(--color-primary) 10%, var(--color-bg)); color: var(--color-primary); }

/* Theme grid */
.theme-grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(120px, 1fr)); gap: var(--space-2); }
.theme-card { display: flex; flex-direction: column; gap: var(--space-1); padding: var(--space-3); background: var(--color-bg); border: 2px solid var(--color-border); border-radius: var(--radius-md); cursor: pointer; transition: all 0.15s; text-align: left; }
.theme-card:hover { border-color: var(--color-primary); }
.theme-card.active { border-color: var(--color-primary); box-shadow: 0 0 0 2px color-mix(in srgb, var(--color-primary) 20%, transparent); }
.theme-swatches { display: flex; gap: 3px; }
.swatch { width: 16px; height: 16px; border-radius: 50%; border: 1px solid rgba(255,255,255,0.1); }
.theme-name { font-size: var(--text-xs); font-weight: var(--font-medium); color: var(--color-text); }

/* Maintenance */
.maintenance-info { display: flex; flex-direction: column; gap: var(--space-2); padding: var(--space-3); background: var(--color-bg); border-radius: var(--radius-md); }
.info-row { display: flex; justify-content: space-between; font-size: var(--text-sm); }
.info-label { color: var(--color-muted); }

/* API Keys */
.key-created-banner { padding: var(--space-3); background: color-mix(in srgb, var(--color-success) 10%, var(--color-bg)); border: 1px solid color-mix(in srgb, var(--color-success) 30%, var(--color-border)); border-radius: var(--radius-md); display: flex; flex-direction: column; gap: var(--space-2); }
.key-created-header { display: flex; align-items: center; gap: var(--space-2); }
.key-created-header strong { font-size: var(--text-sm); color: var(--color-success); }
.key-created-hint { font-size: var(--text-xs); color: var(--color-muted); }
.key-created-value { display: flex; align-items: center; gap: var(--space-2); }
.key-created-value code { flex: 1; padding: var(--space-2) var(--space-3); background: var(--color-bg); border-radius: var(--radius-sm); font-size: var(--text-xs); font-family: var(--font-mono); color: var(--color-text); word-break: break-all; }
.copy-btn { padding: var(--space-1) var(--space-2); background: var(--color-primary); color: white; border: none; border-radius: var(--radius-sm); font-size: var(--text-xs); cursor: pointer; white-space: nowrap; }
.dismiss-btn { align-self: flex-end; background: none; border: none; color: var(--color-muted); font-size: var(--text-xs); cursor: pointer; }

.keys-list { display: flex; flex-direction: column; gap: 1px; background: var(--color-border); border-radius: var(--radius-md); overflow: hidden; }
.key-row { display: flex; align-items: center; justify-content: space-between; padding: var(--space-3); background: var(--color-bg); }
.key-info { display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap; }
.key-name { font-size: var(--text-sm); font-weight: var(--font-medium); color: var(--color-text); }
.key-prefix { font-size: var(--text-xs); font-family: var(--font-mono); color: var(--color-muted); background: var(--color-surface-2); padding: 1px 6px; border-radius: var(--radius-sm); }
.key-meta { font-size: var(--text-xs); color: var(--color-muted); }
.revoke-btn { padding: var(--space-1) var(--space-3); background: none; border: 1px solid var(--color-danger); color: var(--color-danger); border-radius: var(--radius-sm); font-size: var(--text-xs); cursor: pointer; transition: all 0.15s; }
.revoke-btn:hover { background: var(--color-danger); color: white; }

.create-key-form { display: flex; gap: var(--space-2); align-items: center; flex-wrap: wrap; padding-top: var(--space-2); border-top: 1px solid var(--color-border); }
.create-key-form .field-input { flex: 1; min-width: 150px; }

.empty-state { text-align: center; padding: var(--space-4); color: var(--color-muted); font-size: var(--text-sm); }

/* Audit Log */
.audit-toolbar { display: flex; justify-content: space-between; align-items: center; gap: var(--space-3); flex-wrap: wrap; }
.audit-search { max-width: 300px; }
.audit-pager { display: flex; align-items: center; gap: var(--space-2); }
.pager-btn { padding: var(--space-1) var(--space-3); background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text); font-size: var(--text-xs); cursor: pointer; }
.pager-btn:disabled { opacity: 0.4; cursor: default; }
.pager-info { font-size: var(--text-xs); color: var(--color-muted); }

.audit-table { display: flex; flex-direction: column; gap: 1px; background: var(--color-border); border-radius: var(--radius-md); overflow: hidden; }
.audit-row { display: grid; grid-template-columns: 160px 100px 1fr 1fr 120px; gap: var(--space-2); padding: var(--space-2) var(--space-3); background: var(--color-bg); font-size: var(--text-xs); align-items: center; }
.audit.header-row { background: var(--color-surface-2); font-weight: var(--font-semibold); color: var(--color-muted); text-transform: uppercase; letter-spacing: var(--tracking-wide); font-size: var(--text-xs); }
.audit-time { color: var(--color-muted); font-family: var(--font-mono); }
.audit-actor { color: var(--color-text); font-weight: var(--font-medium); }
.audit-action code { font-size: var(--text-xs); background: var(--color-surface-2); padding: 1px 4px; border-radius: 3px; }
.audit-entity { color: var(--color-muted); }
.audit-ip { color: var(--color-muted); font-family: var(--font-mono); }
.audit-empty { text-align: center; padding: var(--space-4); color: var(--color-muted); font-size: var(--text-sm); background: var(--color-bg); }

/* Thresholds */
.threshold-list { display: flex; flex-direction: column; gap: var(--space-2); }
.threshold-row { display: flex; align-items: center; gap: var(--space-2); }
.threshold-input { width: 70px; padding: var(--space-1) var(--space-2); background: var(--color-bg); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text); font-size: var(--text-sm); text-align: center; }
.threshold-unit { font-size: var(--text-sm); color: var(--color-muted); }
.add-btn { background: none; border: 1px dashed var(--color-border); color: var(--color-muted); padding: var(--space-2); border-radius: var(--radius-md); font-size: var(--text-sm); cursor: pointer; transition: all 0.15s; }
.add-btn:hover { border-color: var(--color-primary); color: var(--color-primary); }
.remove-btn { background: none; border: none; color: var(--color-muted); cursor: pointer; font-size: var(--text-sm); padding: 2px 6px; border-radius: var(--radius-sm); }
.remove-btn:hover { color: var(--color-danger); background: color-mix(in srgb, var(--color-danger) 10%, transparent); }

/* Links */
.link-row { display: flex; align-items: center; gap: var(--space-2); flex-wrap: wrap; }
.link-icon { font-size: 1.2rem; width: 28px; text-align: center; }
.link-platform { max-width: 110px; }
.link-name { flex: 1; min-width: 100px; }
.link-url { flex: 2; min-width: 150px; }

/* Update */
.update-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-2); }

/* System grid */
.system-grid { display: grid; grid-template-columns: 1fr 1fr; gap: var(--space-4); }
.system-grid > * { min-width: 0; }

/* Divider */
.divider { height: 1px; background: var(--color-border); margin: var(--space-2) 0; }
.subsection-title { margin: 0; font-size: var(--text-sm); font-weight: var(--font-semibold); color: var(--color-text); }

/* Transitions */
.slide-enter-active, .slide-leave-active { transition: all 0.2s ease; }
.slide-enter-from, .slide-leave-to { opacity: 0; transform: translateY(-8px); }

/* Responsive */
@media (max-width: 768px) {
  .audit-row { grid-template-columns: 1fr 1fr; }
  .audit-col.audit-entity, .audit-col.audit-ip { display: none; }
  .update-grid { grid-template-columns: 1fr; }
  .system-grid { grid-template-columns: 1fr; }
  .theme-grid { grid-template-columns: repeat(auto-fill, minmax(100px, 1fr)); }
  .link-row { flex-direction: column; align-items: stretch; }
  .settings-tabs :deep(.k-tabs__list) { flex-wrap: nowrap; }
}
</style>
