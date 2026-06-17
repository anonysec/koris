<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useNodesStore } from '@/stores/nodes'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import KTabs from '@koris/ui/KTabs.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KButton from '@koris/ui/KButton.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'

const props = defineProps<{ tab?: string }>()

const nodesStore = useNodesStore()
const { get, put, patch } = useApi()
const toast = useToast()
const activeTab = ref(props.tab || 'panel-settings')
const saving = ref(false)

const tabs = [
  { key: 'panel-settings', label: 'Panel Settings' },
  { key: 'data-warnings', label: 'Data Warnings' },
  { key: 'telegram', label: 'Telegram Bot' },
  { key: 'certificates', label: 'Certificates' },
  { key: 'backup', label: 'Backup' },
]

// ─── Panel Settings ─────────────────────────────────────────────────────────
const panelName = ref('')
const panelLang = ref('en')
const loadingSettings = ref(false)
const savingSettings = ref(false)

async function loadPanelSettings(): Promise<void> {
  loadingSettings.value = true
  try {
    const res = await get<{ ok: boolean; settings: Record<string, string> }>('/api/panel-settings')
    if (res.settings) {
      panelName.value = res.settings.panel_name || ''
      panelLang.value = res.settings.language || 'en'
    }
  } catch {
    // Use defaults on error
  } finally {
    loadingSettings.value = false
  }
}

async function savePanelSettings(): Promise<void> {
  savingSettings.value = true
  try {
    await patch<{ ok: boolean }>('/api/panel-settings', {
      panel_name: panelName.value,
      language: panelLang.value,
    })
    toast.success('Panel settings saved successfully.')
  } catch {
    toast.error('Failed to save panel settings.')
  } finally {
    savingSettings.value = false
  }
}

// ─── Data Warning Thresholds ────────────────────────────────────────────────
const thresholds = ref<number[]>([80, 95])
const savingThresholds = ref(false)
const loadingThresholds = ref(false)

async function loadThresholds(): Promise<void> {
  loadingThresholds.value = true
  try {
    const res = await get<{ ok: boolean; thresholds: number[] }>('/api/settings/data-warning-thresholds')
    if (res.thresholds && res.thresholds.length > 0) {
      thresholds.value = res.thresholds
    }
  } catch {
    // Use defaults on error
  } finally {
    loadingThresholds.value = false
  }
}

function addThreshold(): void {
  thresholds.value.push(50)
}

function removeThreshold(index: number): void {
  if (thresholds.value.length > 1) {
    thresholds.value.splice(index, 1)
  }
}

function updateThreshold(index: number, value: string | number): void {
  const num = typeof value === 'number' ? value : parseInt(value, 10)
  if (!isNaN(num)) {
    thresholds.value[index] = Math.min(100, Math.max(0, num))
  }
}

async function saveThresholds(): Promise<void> {
  savingThresholds.value = true
  try {
    const sorted = [...new Set(thresholds.value)].sort((a, b) => a - b)
    thresholds.value = sorted
    await put<{ ok: boolean }>('/api/settings/data-warning-thresholds', { thresholds: sorted })
    toast.success('Data warning thresholds saved successfully.')
  } catch {
    toast.error('Failed to save data warning thresholds.')
  } finally {
    savingThresholds.value = false
  }
}

// ─── Backup ─────────────────────────────────────────────────────────────────
const importFileInput = ref<HTMLInputElement | null>(null)

interface ExportItem {
  label: string
  url: string
}

const exportItems: ExportItem[] = [
  { label: 'Customers CSV', url: '/api/export/customers.csv' },
  { label: 'Payments CSV', url: '/api/export/payments.csv' },
  { label: 'Wallet Transactions CSV', url: '/api/export/wallet-transactions.csv' },
  { label: 'RADIUS Accounting CSV', url: '/api/export/radacct.csv' },
]

function downloadExport(url: string): void {
  window.open(url, '_blank')
}

function triggerImport(): void {
  importFileInput.value?.click()
}

function handleImportFile(event: Event): void {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    toast.success('Import functionality coming soon.')
    // Reset file input so the same file can be selected again
    target.value = ''
  }
}

onMounted(async () => {
  await Promise.all([loadPanelSettings(), loadThresholds()])
})
</script>

<template>
  <div class="page settings-view">
    <KTabs v-model="activeTab" :tabs="tabs" aria-label="Settings sections">
      <!-- Panel Settings -->
      <template #panel-settings>
        <div class="settings-panel">
          <h4 class="section-title">Panel Settings</h4>
          <form class="settings-form" @submit.prevent="savePanelSettings">
            <KFormField name="panel-name" label="Panel Name">
              <template #default="{ fieldId }">
                <KInput
                  :id="fieldId"
                  v-model="panelName"
                  placeholder="KorisPanel"
                  :disabled="loadingSettings"
                />
              </template>
            </KFormField>
            <KFormField name="panel-lang" label="Language">
              <template #default="{ fieldId }">
                <KSelect
                  :id="fieldId"
                  v-model="panelLang"
                  :options="[
                    { label: 'English', value: 'en' },
                    { label: 'Persian', value: 'fa' },
                    { label: 'Chinese', value: 'zh' },
                  ]"
                  :disabled="loadingSettings"
                />
              </template>
            </KFormField>
            <KButton type="submit" variant="primary" :loading="savingSettings" :disabled="loadingSettings">
              Save Settings
            </KButton>
          </form>
        </div>
      </template>

      <!-- Data Usage Warnings -->
      <template #data-warnings>
        <div class="settings-panel">
          <h4 class="section-title">Data Usage Warnings</h4>
          <p class="text-muted text-sm">
            Configure percentage thresholds at which customers receive data usage warnings.
            When a customer's traffic reaches any of these thresholds, a warning notification will be sent.
          </p>
          <form class="settings-form" @submit.prevent="saveThresholds">
            <div class="thresholds-list">
              <div
                v-for="(threshold, index) in thresholds"
                :key="index"
                class="threshold-row"
              >
                <KFormField :name="`threshold-${index}`" :label="`Threshold ${index + 1}`">
                  <template #default="{ fieldId }">
                    <div class="threshold-input-group">
                      <KInput
                        :id="fieldId"
                        :model-value="String(threshold)"
                        type="number"
                        min="0"
                        max="100"
                        placeholder="e.g. 80"
                        @update:model-value="updateThreshold(index, $event)"
                      />
                      <span class="threshold-unit">%</span>
                      <KButton
                        variant="ghost"
                        size="sm"
                        type="button"
                        :disabled="thresholds.length <= 1"
                        @click="removeThreshold(index)"
                      >
                        Remove
                      </KButton>
                    </div>
                  </template>
                </KFormField>
              </div>
            </div>
            <div class="threshold-actions">
              <KButton variant="ghost" size="sm" type="button" @click="addThreshold">
                + Add Threshold
              </KButton>
            </div>
            <KButton type="submit" variant="primary" :loading="savingThresholds">
              Save Thresholds
            </KButton>
          </form>
        </div>
      </template>

      <!-- Telegram Bot -->
      <template #telegram>
        <div class="settings-panel">
          <h4 class="section-title">Telegram Bot</h4>
          <form class="settings-form">
            <KFormField name="tg-token" label="Bot Token" hint="Get this from @BotFather">
              <template #default="{ fieldId }">
                <KInput :id="fieldId" placeholder="123456:ABC-DEF..." type="password" />
              </template>
            </KFormField>
            <KFormField name="tg-chat" label="Chat ID">
              <template #default="{ fieldId }">
                <KInput :id="fieldId" placeholder="-1001234567890" />
              </template>
            </KFormField>
            <KButton variant="primary" size="sm">Save Telegram Settings</KButton>
          </form>
        </div>
      </template>

      <!-- Certificates -->
      <template #certificates>
        <div class="settings-panel">
          <h4 class="section-title">SSL/TLS Certificates</h4>
          <div class="cert-info">
            <div class="cert-item">
              <span class="cert-item__label">CA Certificate</span>
              <KStatusPill :status="nodesStore.vpnSettings?.ca_exists ? 'active' : 'disabled'" size="sm" />
            </div>
            <div class="cert-item">
              <span class="cert-item__label">TLS Crypt Key</span>
              <KStatusPill :status="nodesStore.vpnSettings?.tls_crypt_exists ? 'active' : 'disabled'" size="sm" />
            </div>
          </div>
          <KButton variant="primary" size="sm" class="mt-3">Regenerate Certificates</KButton>
        </div>
      </template>

      <!-- Backup -->
      <template #backup>
        <div class="settings-panel">
          <h4 class="section-title">Backup &amp; Restore</h4>
          <p class="text-muted text-sm">Export your panel data or import a backup.</p>

          <div class="backup-section">
            <h5 class="subsection-title">Export Data</h5>
            <p class="text-muted text-sm">Download your data as CSV files.</p>
            <div class="export-list">
              <div
                v-for="item in exportItems"
                :key="item.url"
                class="export-item"
              >
                <span class="export-item__label">{{ item.label }}</span>
                <KButton variant="ghost" size="sm" @click="downloadExport(item.url)">
                  Download
                </KButton>
              </div>
            </div>
          </div>

          <div class="backup-section">
            <h5 class="subsection-title">Import Data</h5>
            <p class="text-muted text-sm">Restore data from a previously exported file.</p>
            <input
              ref="importFileInput"
              type="file"
              accept=".csv,.json"
              class="hidden-input"
              @change="handleImportFile"
            />
            <KButton variant="ghost" size="sm" @click="triggerImport">
              Import Backup
            </KButton>
          </div>
        </div>
      </template>
    </KTabs>
  </div>
</template>

<style scoped>
.settings-view { display: flex; flex-direction: column; gap: var(--space-5); }

.settings-panel { padding: var(--space-5) 0; display: flex; flex-direction: column; gap: var(--space-4); }
.section-title { margin: 0; font-size: var(--text-base); font-weight: var(--font-semibold); }
.subsection-title { margin: 0; font-size: var(--text-sm); font-weight: var(--font-semibold); }

.settings-form { display: flex; flex-direction: column; gap: var(--space-3); max-width: 480px; }

.cert-info { display: flex; flex-direction: column; gap: var(--space-2); }
.cert-item { display: flex; justify-content: space-between; align-items: center; padding: var(--space-3); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); max-width: 400px; }
.cert-item__label { font-size: var(--text-sm); }

.thresholds-list { display: flex; flex-direction: column; gap: var(--space-2); }
.threshold-row { display: flex; align-items: flex-end; gap: var(--space-2); }
.threshold-input-group { display: flex; align-items: center; gap: var(--space-2); }
.threshold-unit { font-size: var(--text-sm); color: var(--color-muted); font-weight: var(--font-medium); }
.threshold-actions { display: flex; align-items: center; }

.backup-section { display: flex; flex-direction: column; gap: var(--space-2); padding-top: var(--space-3); border-top: 1px solid var(--color-border); }
.backup-section:first-of-type { border-top: none; padding-top: 0; }

.export-list { display: flex; flex-direction: column; gap: var(--space-2); max-width: 400px; }
.export-item { display: flex; justify-content: space-between; align-items: center; padding: var(--space-2) var(--space-3); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); }
.export-item__label { font-size: var(--text-sm); font-weight: var(--font-medium); }

.hidden-input { display: none; }

.text-muted { color: var(--color-muted); }
.text-sm { font-size: var(--text-sm); }
.mt-3 { margin-top: var(--space-3); }

@media (max-width: 768px) {
  .settings-form { max-width: 100%; }
  .cert-item { max-width: 100%; }
  .export-list { max-width: 100%; }
  .export-item { flex-direction: column; align-items: flex-start; gap: var(--space-2); }
  .threshold-input-group { flex-wrap: wrap; }
}
</style>
