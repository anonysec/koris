<script setup lang="ts">
import { ref, computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KTextarea from '@koris/ui/KTextarea.vue'

const { t } = useI18n()
const store = useSettingsStore()
const toast = useToast()

const tls = computed(() => store.settings?.tls ?? null)

const modeLabel = computed(() => {
  if (!tls.value) return '—'
  const labels: Record<string, string> = {
    acme: 'ACME (Let\'s Encrypt)',
    'self-signed': 'Self-Signed',
    manual: 'Manual',
  }
  return labels[tls.value.mode] ?? tls.value.mode
})

const daysUntilExpiry = computed(() => {
  if (!tls.value?.expiresAt) return null
  const expires = new Date(tls.value.expiresAt).getTime()
  const now = Date.now()
  return Math.ceil((expires - now) / (1000 * 60 * 60 * 24))
})

const expiryWarning = computed(() => {
  if (daysUntilExpiry.value === null) return false
  return daysUntilExpiry.value <= 14
})

// ─── Upload Form ─────────────────────────────────────────────────────────────
const certPem = ref('')
const keyPem = ref('')
const uploading = ref(false)

async function handleUpload() {
  if (!certPem.value.trim() || !keyPem.value.trim()) {
    toast.error(t('settings.tls_files_required'))
    return
  }
  uploading.value = true
  const success = await store.uploadTlsCert(certPem.value, keyPem.value)
  uploading.value = false
  if (success) {
    toast.success(t('settings.tls_upload_success'))
    certPem.value = ''
    keyPem.value = ''
    await store.loadSettings()
  } else {
    toast.error(t('settings.tls_upload_error'))
  }
}
</script>

<template>
  <section class="settings-section">
    <h3 class="settings-section__title">{{ t('settings.tls') }}</h3>

    <div v-if="tls" class="tls-content">
      <div class="info-grid">
        <div class="info-item">
          <span class="info-item__label">{{ t('settings.tls_mode') }}</span>
          <span class="info-item__value">{{ modeLabel }}</span>
        </div>
        <div class="info-item">
          <span class="info-item__label">{{ t('settings.tls_domain') }}</span>
          <code class="info-item__value info-item__value--mono">{{ tls.domain || '—' }}</code>
        </div>
        <div class="info-item">
          <span class="info-item__label">{{ t('settings.tls_expires') }}</span>
          <span class="info-item__value" :class="{ 'info-item__value--danger': expiryWarning }">
            {{ tls.expiresAt ? new Date(tls.expiresAt).toLocaleDateString() : '—' }}
            <span v-if="daysUntilExpiry !== null" class="expiry-days">({{ daysUntilExpiry }}d)</span>
          </span>
        </div>
        <div class="info-item">
          <span class="info-item__label">{{ t('settings.tls_issuer') }}</span>
          <span class="info-item__value">{{ tls.issuer || '—' }}</span>
        </div>
      </div>

      <!-- Expiry Warning Banner -->
      <div v-if="expiryWarning" class="warning-banner">
        <span class="warning-banner__icon">⚠️</span>
        <span>{{ t('settings.tls_expiry_warning') }}</span>
      </div>

      <!-- Manual Upload Form -->
      <div v-if="tls.mode === 'manual'" class="upload-form">
        <h4 class="upload-form__title">{{ t('settings.tls_upload_title') }}</h4>
        <form @submit.prevent="handleUpload">
          <KFormField name="cert-pem" :label="t('settings.tls_cert_pem')">
            <template #default="{ fieldId }">
              <KTextarea
                :id="fieldId"
                v-model="certPem"
                :placeholder="'-----BEGIN CERTIFICATE-----\n...'"
                :rows="4"
              />
            </template>
          </KFormField>
          <KFormField name="key-pem" :label="t('settings.tls_key_pem')">
            <template #default="{ fieldId }">
              <KTextarea
                :id="fieldId"
                v-model="keyPem"
                :placeholder="'-----BEGIN PRIVATE KEY-----\n...'"
                :rows="4"
              />
            </template>
          </KFormField>
          <KButton type="submit" variant="primary" size="sm" :loading="uploading">
            {{ t('settings.tls_upload') }}
          </KButton>
        </form>
      </div>
    </div>
    <div v-else class="info-empty">{{ t('settings.loading') }}</div>
  </section>
</template>

<style scoped>
.settings-section {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
}

.settings-section__title {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0 0 var(--space-4);
}

.tls-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.info-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-4);
}

.info-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.info-item__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

.info-item__value {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}

.info-item__value--mono {
  font-family: var(--font-mono);
}

.info-item__value--danger {
  color: var(--color-danger);
}

.expiry-days {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.warning-banner {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: rgba(var(--color-warning-rgb, 245, 158, 11), 0.1);
  border: 1px solid var(--color-warning);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-warning);
}

.warning-banner__icon {
  font-size: var(--text-lg);
}

.upload-form {
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.upload-form__title {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.info-empty {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
</style>
