<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KInput from '@koris/ui/KInput.vue'

const { t } = useI18n()
const store = useSettingsStore()
const toast = useToast()

const alerts = computed(() => store.settings?.alerts ?? null)

// ─── Form State ──────────────────────────────────────────────────────────────
const cpuThreshold = ref(90)
const ramThreshold = ref(90)
const diskThreshold = ref(85)
const saving = ref(false)
const errors = ref<Record<string, string>>({})

// Pre-fill from store when available
watch(alerts, (val) => {
  if (val) {
    cpuThreshold.value = val.cpuThreshold
    ramThreshold.value = val.ramThreshold
    diskThreshold.value = val.diskThreshold
  }
}, { immediate: true })

function validate(): boolean {
  errors.value = {}
  if (cpuThreshold.value < 1 || cpuThreshold.value > 100) {
    errors.value.cpu = t('settings.alert_range_error')
  }
  if (ramThreshold.value < 1 || ramThreshold.value > 100) {
    errors.value.ram = t('settings.alert_range_error')
  }
  if (diskThreshold.value < 1 || diskThreshold.value > 100) {
    errors.value.disk = t('settings.alert_range_error')
  }
  return Object.keys(errors.value).length === 0
}

async function handleSave() {
  if (!validate()) return
  saving.value = true
  const success = await store.updateAlerts({
    cpu: cpuThreshold.value,
    ram: ramThreshold.value,
    disk: diskThreshold.value,
  })
  saving.value = false
  if (success) {
    toast.success(t('settings.alerts_save_success'))
  } else {
    toast.error(t('settings.alerts_save_error'))
  }
}
</script>

<template>
  <section class="settings-section">
    <h3 class="settings-section__title">{{ t('settings.alerts') }}</h3>
    <p class="settings-section__desc">{{ t('settings.alerts_desc') }}</p>

    <form class="alerts-form" @submit.prevent="handleSave">
      <div class="form-grid">
        <KFormField name="cpu-threshold" :label="t('settings.alert_cpu')" :error="errors.cpu">
          <template #default="{ fieldId }">
            <div class="input-with-unit">
              <KInput
                :id="fieldId"
                v-model.number="cpuThreshold"
                type="number"
                min="1"
                max="100"
              />
              <span class="input-unit">%</span>
            </div>
          </template>
        </KFormField>
        <KFormField name="ram-threshold" :label="t('settings.alert_ram')" :error="errors.ram">
          <template #default="{ fieldId }">
            <div class="input-with-unit">
              <KInput
                :id="fieldId"
                v-model.number="ramThreshold"
                type="number"
                min="1"
                max="100"
              />
              <span class="input-unit">%</span>
            </div>
          </template>
        </KFormField>
        <KFormField name="disk-threshold" :label="t('settings.alert_disk')" :error="errors.disk">
          <template #default="{ fieldId }">
            <div class="input-with-unit">
              <KInput
                :id="fieldId"
                v-model.number="diskThreshold"
                type="number"
                min="1"
                max="100"
              />
              <span class="input-unit">%</span>
            </div>
          </template>
        </KFormField>
      </div>
      <KButton type="submit" variant="primary" size="sm" :loading="saving">
        {{ t('settings.save') }}
      </KButton>
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

.alerts-form {
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
</style>
