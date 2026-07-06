<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'

const { t } = useI18n()
const store = useSettingsStore()
const toast = useToast()

const grpc = computed(() => store.settings?.grpc ?? null)

// ─── Form State ──────────────────────────────────────────────────────────────
const connectTimeout = ref(10)
const keepaliveInterval = ref(30)
const metricsInterval = ref(15)
const saving = ref(false)
const restartNotice = ref(false)

// Pre-fill from store
watch(grpc, (val) => {
  if (val) {
    connectTimeout.value = val.connectTimeout
    keepaliveInterval.value = val.keepaliveInterval
    metricsInterval.value = val.metricsInterval
  }
}, { immediate: true })

async function handleSave() {
  saving.value = true
  restartNotice.value = false
  const result = await store.updateGrpc({
    connectTimeout: connectTimeout.value,
    keepaliveInterval: keepaliveInterval.value,
    metricsInterval: metricsInterval.value,
  })
  saving.value = false
  if (result.success) {
    toast.success(t('settings.grpc_save_success'))
    if (result.restartRequired) {
      restartNotice.value = true
    }
  } else {
    toast.error(t('settings.grpc_save_error'))
  }
}
</script>

<template>
  <section class="settings-section">
    <h3 class="settings-section__title">{{ t('settings.grpc') }}</h3>
    <p class="settings-section__desc">{{ t('settings.grpc_desc') }}</p>

    <form class="grpc-form" autocomplete="off" @submit.prevent="handleSave">
      <div class="form-grid">
        <FormField name="connect-timeout" :label="t('settings.grpc_connect_timeout')">
          <template #default="{ fieldId }">
            <div class="input-with-unit">
              <Input
                :id="fieldId"
                v-model.number="connectTimeout"
                type="number"
                min="1"
              />
              <span class="input-unit">s</span>
            </div>
          </template>
        </FormField>
        <FormField name="keepalive-interval" :label="t('settings.grpc_keepalive')">
          <template #default="{ fieldId }">
            <div class="input-with-unit">
              <Input
                :id="fieldId"
                v-model.number="keepaliveInterval"
                type="number"
                min="1"
              />
              <span class="input-unit">s</span>
            </div>
          </template>
        </FormField>
        <FormField name="metrics-interval" :label="t('settings.grpc_metrics_interval')">
          <template #default="{ fieldId }">
            <div class="input-with-unit">
              <Input
                :id="fieldId"
                v-model.number="metricsInterval"
                type="number"
                min="1"
              />
              <span class="input-unit">s</span>
            </div>
          </template>
        </FormField>
      </div>

      <!-- Restart Notice -->
      <div v-if="restartNotice" class="restart-notice">
        <span class="restart-notice__icon">🔄</span>
        <span>{{ t('settings.grpc_restart_required') }}</span>
      </div>

      <Button type="submit" variant="primary" size="sm" :loading="saving">
        {{ t('settings.save') }}
      </Button>
    </form>
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
  margin: 0 0 var(--space-2);
}

.settings-section__desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0 0 var(--space-4);
}

.grpc-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--space-4);
}

.input-with-unit {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.input-unit {
  font-size: var(--text-sm);
  color: var(--color-muted);
  font-weight: var(--font-medium);
}

.restart-notice {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid var(--color-primary);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-primary);
}

.restart-notice__icon {
  font-size: var(--text-lg);
}
</style>
