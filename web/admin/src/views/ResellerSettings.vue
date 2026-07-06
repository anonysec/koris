<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'

const { t } = useI18n()
const { get, patch, loading } = useApi()
const toast = useToast()

const billingMode = ref('manual')
const loaded = ref(false)

async function loadSettings() {
  try {
    const res = await get<{ ok: boolean; billing_mode: string }>('/api/reseller/settings')
    if (res.ok) {
      billingMode.value = res.billing_mode || 'manual'
    }
    loaded.value = true
  } catch {
    loaded.value = true
  }
}

async function save() {
  try {
    await patch<{ ok: boolean }>('/api/reseller/settings', { billing_mode: billingMode.value })
    toast.success(t('reseller_settings.save_success'))
  } catch {
    toast.error('Failed to save settings')
  }
}

onMounted(loadSettings)
</script>

<template>
  <div class="reseller-settings">
    <h1 class="reseller-settings__title">{{ t('reseller_settings.title') }}</h1>

    <div v-if="loaded" class="reseller-settings__card">
      <h2 class="reseller-settings__section-title">{{ t('reseller_settings.billing_mode') }}</h2>
      <p class="reseller-settings__desc">{{ t('reseller_settings.billing_mode_desc') }}</p>

      <div class="reseller-settings__options">
        <label class="billing-option" :class="{ 'billing-option--active': billingMode === 'manual' }">
          <input type="radio" v-model="billingMode" value="manual" class="billing-option__radio" />
          <div class="billing-option__content">
            <span class="billing-option__label">{{ t('reseller_settings.billing_manual') }}</span>
            <span class="billing-option__desc">{{ t('reseller_settings.billing_manual_desc') }}</span>
          </div>
        </label>

        <label class="billing-option" :class="{ 'billing-option--active': billingMode === 'self_service' }">
          <input type="radio" v-model="billingMode" value="self_service" class="billing-option__radio" />
          <div class="billing-option__content">
            <span class="billing-option__label">{{ t('reseller_settings.billing_self_service') }}</span>
            <span class="billing-option__desc">{{ t('reseller_settings.billing_self_service_desc') }}</span>
          </div>
        </label>
      </div>

      <div class="reseller-settings__actions">
        <Button variant="primary" :loading="loading" @click="save">{{ t('btn.save') }}</Button>
      </div>
    </div>
  </div>
</template>

<style scoped>
.reseller-settings {
  padding: var(--space-6, 24px);
  max-width: 600px;
}

.reseller-settings__title {
  font-size: var(--text-xl, 20px);
  font-weight: var(--font-bold, 700);
  margin: 0 0 var(--space-5, 20px);
}

.reseller-settings__card {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5, 20px);
}

.reseller-settings__section-title {
  font-size: var(--text-md, 16px);
  font-weight: var(--font-semibold, 600);
  margin: 0 0 var(--space-2, 8px);
}

.reseller-settings__desc {
  color: var(--color-muted);
  font-size: var(--text-sm, 14px);
  margin: 0 0 var(--space-4, 16px);
}

.reseller-settings__options {
  display: flex;
  flex-direction: column;
  gap: var(--space-3, 12px);
  margin-bottom: var(--space-5, 20px);
}

.billing-option {
  display: flex;
  align-items: flex-start;
  gap: var(--space-3, 12px);
  padding: var(--space-4, 16px);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color 0.15s, background 0.15s;
}

.billing-option:hover {
  background: var(--color-surface-2);
}

.billing-option--active {
  border-color: var(--color-primary);
  background: rgba(99, 102, 241, 0.05);
}

.billing-option__radio {
  margin-top: 2px;
  accent-color: var(--color-primary);
}

.billing-option__content {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
}

.billing-option__label {
  font-size: var(--text-sm, 14px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text);
}

.billing-option__desc {
  font-size: var(--text-xs, 12px);
  color: var(--color-muted);
}

.reseller-settings__actions {
  display: flex;
  justify-content: flex-end;
}
</style>
