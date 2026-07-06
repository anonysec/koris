<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import FormField from '@koris/ui/FormField.vue'
import Skeleton from '@koris/ui/Skeleton.vue'

const { t } = useI18n()
const { get, patch } = useApi()
const toast = useToast()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface SLATarget {
  priority: string
  response_minutes: number
}

interface SLAStats {
  priority: string
  met: number
  breached: number
  percent_met: number
  avg_response_minutes: number
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const loadingConfig = ref(false)
const loadingStats = ref(false)
const saving = ref(false)

// SLA targets per priority
const urgentMinutes = ref(60)
const highMinutes = ref(240)
const normalMinutes = ref(1440)
const lowMinutes = ref(4320)

// SLA statistics
const stats = ref<SLAStats[]>([])

// ═══════════════════════════════════════════════════════════════════════════════
// Computed
// ═══════════════════════════════════════════════════════════════════════════════

const overallPercentMet = computed(() => {
  if (stats.value.length === 0) return 0
  const totalMet = stats.value.reduce((sum, s) => sum + s.met, 0)
  const totalAll = stats.value.reduce((sum, s) => sum + s.met + s.breached, 0)
  if (totalAll === 0) return 100
  return Math.round((totalMet / totalAll) * 100)
})

const overallAvgResponse = computed(() => {
  if (stats.value.length === 0) return 0
  const validStats = stats.value.filter(s => s.avg_response_minutes > 0)
  if (validStats.length === 0) return 0
  const sum = validStats.reduce((acc, s) => acc + s.avg_response_minutes, 0)
  return Math.round(sum / validStats.length)
})

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchConfig() {
  loadingConfig.value = true
  try {
    const data = await get<{ ok: boolean; targets: SLATarget[] }>('/api/sla/config')
    if (data?.ok && data.targets) {
      for (const target of data.targets) {
        switch (target.priority) {
          case 'urgent': urgentMinutes.value = target.response_minutes; break
          case 'high': highMinutes.value = target.response_minutes; break
          case 'normal': normalMinutes.value = target.response_minutes; break
          case 'low': lowMinutes.value = target.response_minutes; break
        }
      }
    }
  } catch {
    // error toast handled by useApi
  } finally {
    loadingConfig.value = false
  }
}

async function fetchStats() {
  loadingStats.value = true
  try {
    const data = await get<{ ok: boolean; stats: SLAStats[] }>('/api/sla/stats')
    if (data?.ok) {
      stats.value = data.stats || []
    }
  } catch {
    stats.value = []
  } finally {
    loadingStats.value = false
  }
}

async function saveConfig() {
  saving.value = true
  try {
    const targets = [
      { priority: 'urgent', response_minutes: Number(urgentMinutes.value) },
      { priority: 'high', response_minutes: Number(highMinutes.value) },
      { priority: 'normal', response_minutes: Number(normalMinutes.value) },
      { priority: 'low', response_minutes: Number(lowMinutes.value) },
    ]

    await patch<{ ok: boolean }>('/api/sla/config', { targets })
    toast.success(t('sla.saved'))
  } catch {
    // error toast handled by useApi
  } finally {
    saving.value = false
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function formatMinutes(minutes: number): string {
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  const mins = minutes % 60
  if (hours < 24) return mins > 0 ? `${hours}h ${mins}m` : `${hours}h`
  const days = Math.floor(hours / 24)
  const remainHours = hours % 24
  return remainHours > 0 ? `${days}d ${remainHours}h` : `${days}d`
}

function priorityColor(priority: string): string {
  switch (priority) {
    case 'urgent': return 'var(--color-danger)'
    case 'high': return 'var(--color-warning)'
    case 'normal': return 'var(--color-primary)'
    case 'low': return 'var(--color-muted)'
    default: return 'var(--color-muted)'
  }
}

function complianceColor(percent: number): string {
  if (percent >= 90) return 'var(--color-success)'
  if (percent >= 70) return 'var(--color-warning)'
  return 'var(--color-danger)'
}

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(() => {
  fetchConfig()
  fetchStats()
})
</script>

<template>
  <div class="page sla-view">
    <h2 class="page-title">{{ t('sla.title') }}</h2>

    <div class="sla-grid">
      <!-- SLA Configuration Panel -->
      <section class="panel config-panel">
        <div class="panel-header">
          <h3 class="panel-title">{{ t('sla.config_title') }}</h3>
        </div>

        <div v-if="loadingConfig" class="skeleton-wrap">
          <Skeleton variant="rect" :height="200" />
        </div>

        <form v-else class="sla-form" @submit.prevent="saveConfig">
          <p class="sla-description">{{ t('sla.config_description') }}</p>

          <div class="priority-inputs">
            <!-- Urgent -->
            <div class="priority-row">
              <span class="priority-dot" :style="{ background: priorityColor('urgent') }"></span>
              <span class="priority-label">{{ t('sla.priority_urgent') }}</span>
              <FormField name="urgent" :label="t('sla.minutes')">
                <template #default="{ fieldId }">
                  <Input
                    :id="fieldId"
                    v-model="urgentMinutes"
                    type="number"
                    min="1"
                    class="minutes-input"
                  />
                </template>
              </FormField>
              <span class="minutes-display">{{ formatMinutes(Number(urgentMinutes)) }}</span>
            </div>

            <!-- High -->
            <div class="priority-row">
              <span class="priority-dot" :style="{ background: priorityColor('high') }"></span>
              <span class="priority-label">{{ t('sla.priority_high') }}</span>
              <FormField name="high" :label="t('sla.minutes')">
                <template #default="{ fieldId }">
                  <Input
                    :id="fieldId"
                    v-model="highMinutes"
                    type="number"
                    min="1"
                    class="minutes-input"
                  />
                </template>
              </FormField>
              <span class="minutes-display">{{ formatMinutes(Number(highMinutes)) }}</span>
            </div>

            <!-- Normal -->
            <div class="priority-row">
              <span class="priority-dot" :style="{ background: priorityColor('normal') }"></span>
              <span class="priority-label">{{ t('sla.priority_normal') }}</span>
              <FormField name="normal" :label="t('sla.minutes')">
                <template #default="{ fieldId }">
                  <Input
                    :id="fieldId"
                    v-model="normalMinutes"
                    type="number"
                    min="1"
                    class="minutes-input"
                  />
                </template>
              </FormField>
              <span class="minutes-display">{{ formatMinutes(Number(normalMinutes)) }}</span>
            </div>

            <!-- Low -->
            <div class="priority-row">
              <span class="priority-dot" :style="{ background: priorityColor('low') }"></span>
              <span class="priority-label">{{ t('sla.priority_low') }}</span>
              <FormField name="low" :label="t('sla.minutes')">
                <template #default="{ fieldId }">
                  <Input
                    :id="fieldId"
                    v-model="lowMinutes"
                    type="number"
                    min="1"
                    class="minutes-input"
                  />
                </template>
              </FormField>
              <span class="minutes-display">{{ formatMinutes(Number(lowMinutes)) }}</span>
            </div>
          </div>

          <div class="form-actions">
            <Button type="submit" variant="primary" :loading="saving">
              {{ t('sla.save') }}
            </Button>
          </div>
        </form>
      </section>

      <!-- SLA Compliance Stats Panel -->
      <section class="panel stats-panel">
        <div class="panel-header">
          <h3 class="panel-title">{{ t('sla.stats_title') }}</h3>
        </div>

        <div v-if="loadingStats" class="skeleton-wrap">
          <Skeleton variant="rect" :height="200" />
        </div>

        <div v-else-if="stats.length === 0" class="empty-stats">
          <p class="text-muted">{{ t('sla.no_stats') }}</p>
        </div>

        <div v-else class="stats-content">
          <!-- Overall Summary -->
          <div class="stats-summary">
            <div class="stat-card">
              <span class="stat-value" :style="{ color: complianceColor(overallPercentMet) }">
                {{ overallPercentMet }}%
              </span>
              <span class="stat-label">{{ t('sla.overall_compliance') }}</span>
            </div>
            <div class="stat-card">
              <span class="stat-value">{{ formatMinutes(overallAvgResponse) }}</span>
              <span class="stat-label">{{ t('sla.avg_response_time') }}</span>
            </div>
          </div>

          <!-- Per-Priority Stats -->
          <div class="priority-stats">
            <div
              v-for="stat in stats"
              :key="stat.priority"
              class="priority-stat-row"
            >
              <div class="priority-stat-header">
                <span class="priority-dot" :style="{ background: priorityColor(stat.priority) }"></span>
                <span class="priority-label">{{ t(`sla.priority_${stat.priority}`) }}</span>
                <span
                  class="compliance-percent"
                  :style="{ color: complianceColor(stat.percent_met) }"
                >
                  {{ stat.percent_met }}%
                </span>
              </div>
              <div class="priority-stat-bar">
                <div
                  class="priority-stat-bar__fill"
                  :style="{
                    width: `${stat.percent_met}%`,
                    background: complianceColor(stat.percent_met),
                  }"
                ></div>
              </div>
              <div class="priority-stat-details">
                <span class="stat-detail">
                  <span class="stat-detail__label">{{ t('sla.met') }}:</span>
                  <span class="stat-detail__value">{{ stat.met }}</span>
                </span>
                <span class="stat-detail">
                  <span class="stat-detail__label">{{ t('sla.breached') }}:</span>
                  <span class="stat-detail__value text-danger">{{ stat.breached }}</span>
                </span>
                <span class="stat-detail">
                  <span class="stat-detail__label">{{ t('sla.avg') }}:</span>
                  <span class="stat-detail__value">{{ formatMinutes(stat.avg_response_minutes) }}</span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </section>
    </div>
  </div>
</template>

<style scoped>
.sla-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
}

.sla-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-4);
}

.panel {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}
.panel-header {
  margin-bottom: var(--space-3);
}
.panel-title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

/* Config Form */
.sla-description {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0 0 var(--space-3);
}

.sla-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.priority-inputs {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.priority-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.priority-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  flex-shrink: 0;
}

.priority-label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  min-width: 70px;
  text-transform: capitalize;
}

.minutes-input {
  width: 100px;
}

.minutes-display {
  font-size: var(--text-xs);
  color: var(--color-muted);
  min-width: 60px;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Stats */
.empty-stats {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100px;
}

.stats-content {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.stats-summary {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.stat-card {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: var(--space-3);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
}
.stat-value {
  font-size: var(--text-xl);
  font-weight: var(--font-bold);
}
.stat-label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin-top: var(--space-1);
}

/* Per-Priority Stats */
.priority-stats {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.priority-stat-row {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.priority-stat-header {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.compliance-percent {
  margin-inline-start: auto;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
}

.priority-stat-bar {
  width: 100%;
  height: 6px;
  background: var(--color-surface-2);
  border-radius: var(--radius-full);
  overflow: hidden;
}
.priority-stat-bar__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.4s ease;
}

.priority-stat-details {
  display: flex;
  gap: var(--space-3);
}
.stat-detail {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  font-size: var(--text-xs);
}
.stat-detail__label {
  color: var(--color-muted);
}
.stat-detail__value {
  font-weight: var(--font-semibold);
}

/* Utility */
.text-muted { color: var(--color-muted); }
.text-danger { color: var(--color-danger); }
.skeleton-wrap { padding: var(--space-2) 0; }

@media (max-width: 768px) {
  .sla-grid {
    grid-template-columns: 1fr;
  }
  .priority-row {
    flex-wrap: wrap;
  }
  .stats-summary {
    grid-template-columns: 1fr;
  }
}
</style>
