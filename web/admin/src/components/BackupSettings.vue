<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useBackups } from '@/composables/useBackups'
import { useToast } from '@koris/composables/useToast'
import Button from '@koris/ui/Button.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'

const toast = useToast()
const { settings, fetchSettings, updateSettings } = useBackups()

const saving = ref(false)
const loadingSettings = ref(false)

// Local form state derived from settings
const scheduleType = ref<'disabled' | 'daily' | 'weekly'>('daily')
const scheduleHour = ref(2)
const scheduleDay = ref('sun')
const retentionCount = ref(7)

const hourOptions = computed(() =>
  Array.from({ length: 24 }, (_, i) => ({
    label: `${String(i).padStart(2, '0')}:00`,
    value: String(i),
  }))
)

const dayOptions = [
  { label: 'Sunday', value: 'sun' },
  { label: 'Monday', value: 'mon' },
  { label: 'Tuesday', value: 'tue' },
  { label: 'Wednesday', value: 'wed' },
  { label: 'Thursday', value: 'thu' },
  { label: 'Friday', value: 'fri' },
  { label: 'Saturday', value: 'sat' },
]

const scheduleTypeOptions = [
  { label: 'Disabled', value: 'disabled' },
  { label: 'Daily', value: 'daily' },
  { label: 'Weekly', value: 'weekly' },
]

function parseSchedule(schedule: string) {
  if (schedule === 'disabled') {
    scheduleType.value = 'disabled'
    return
  }
  const parts = schedule.split(':')
  if (parts[0] === 'daily') {
    scheduleType.value = 'daily'
    scheduleHour.value = parseInt(parts[1] || '2', 10)
  } else if (parts[0] === 'weekly') {
    scheduleType.value = 'weekly'
    scheduleDay.value = parts[1] || 'sun'
    scheduleHour.value = parseInt(parts[2] || '2', 10)
  }
}

function buildScheduleString(): string {
  if (scheduleType.value === 'disabled') return 'disabled'
  if (scheduleType.value === 'daily') return `daily:${String(scheduleHour.value).padStart(2, '0')}`
  return `weekly:${scheduleDay.value}:${String(scheduleHour.value).padStart(2, '0')}`
}

async function handleSave() {
  saving.value = true
  try {
    const schedule = buildScheduleString()
    const count = Math.max(1, Math.min(30, retentionCount.value))
    const ok = await updateSettings(schedule, count)
    if (ok) {
      toast.success('Backup settings saved')
    } else {
      toast.error('Failed to save settings')
    }
  } catch {
    toast.error('Failed to save settings')
  } finally {
    saving.value = false
  }
}

onMounted(async () => {
  loadingSettings.value = true
  try {
    await fetchSettings()
    parseSchedule(settings.value.schedule)
    retentionCount.value = settings.value.retention_count
  } catch {
    // Use defaults
  } finally {
    loadingSettings.value = false
  }
})
</script>

<template>
  <section class="backup-settings">
    <h3 class="section-title">Schedule & Retention</h3>
    <form class="settings-form" @submit.prevent="handleSave">
      <FormField name="schedule-type" label="Schedule">
        <template #default="{ fieldId }">
          <Select
            :id="fieldId"
            v-model="scheduleType"
            :options="scheduleTypeOptions"
            :disabled="loadingSettings"
          />
        </template>
      </FormField>

      <FormField v-if="scheduleType === 'weekly'" name="schedule-day" label="Day of Week">
        <template #default="{ fieldId }">
          <Select
            :id="fieldId"
            v-model="scheduleDay"
            :options="dayOptions"
            :disabled="loadingSettings"
          />
        </template>
      </FormField>

      <FormField v-if="scheduleType !== 'disabled'" name="schedule-hour" label="Hour (UTC)">
        <template #default="{ fieldId }">
          <Select
            :id="fieldId"
            :model-value="String(scheduleHour)"
            :options="hourOptions"
            :disabled="loadingSettings"
            @update:model-value="scheduleHour = parseInt($event, 10)"
          />
        </template>
      </FormField>

      <FormField name="retention-count" label="Retention Count" hint="Number of backups to keep (1–30)">
        <template #default="{ fieldId }">
          <Input
            :id="fieldId"
            :model-value="String(retentionCount)"
            type="number"
            min="1"
            max="30"
            :disabled="loadingSettings"
            @update:model-value="retentionCount = Math.max(1, Math.min(30, parseInt($event, 10) || 7))"
          />
        </template>
      </FormField>

      <Button type="submit" variant="primary" :loading="saving" :disabled="loadingSettings">
        Save Settings
      </Button>
    </form>
  </section>
</template>

<style scoped>
.backup-settings {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.section-title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

.settings-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  max-width: 360px;
}
</style>
