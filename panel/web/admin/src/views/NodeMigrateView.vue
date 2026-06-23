<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useNodesStore } from '@/stores/nodes'
import { storeToRefs } from 'pinia'
import KButton from '@koris/ui/KButton.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KFormField from '@koris/ui/KFormField.vue'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()
const { list: nodes } = storeToRefs(nodesStore)

// ─── Types ───────────────────────────────────────────────────────────────────
interface MigrationResult {
  total: number
  migrated: number
  failed: number
  failures: Array<{ username: string; error: string }>
}

// ─── State ───────────────────────────────────────────────────────────────────
const sourceNodeId = ref<number | string>('')
const destNodeId = ref<number | string>('')
const migrating = ref(false)
const migrationStarted = ref(false)
const migrationResult = ref<MigrationResult | null>(null)
const migrationError = ref('')
const progress = ref({ migrated: 0, total: 0 })
let pollTimer: ReturnType<typeof setInterval> | null = null

// ─── Computed ────────────────────────────────────────────────────────────────
const nodeOptions = computed(() =>
  nodes.value.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

const destNodeOptions = computed(() =>
  nodes.value
    .filter(n => n.id !== Number(sourceNodeId.value))
    .map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

const canMigrate = computed(() =>
  sourceNodeId.value && destNodeId.value && sourceNodeId.value !== destNodeId.value
)

const progressPercent = computed(() => {
  if (progress.value.total === 0) return 0
  return Math.round((progress.value.migrated / progress.value.total) * 100)
})

// ─── Actions ─────────────────────────────────────────────────────────────────
async function startMigration() {
  if (!canMigrate.value) return

  const sourceNode = nodes.value.find(n => n.id === Number(sourceNodeId.value))
  const destNode = nodes.value.find(n => n.id === Number(destNodeId.value))

  const confirmed = await confirm({
    title: t('migrate.confirm_title'),
    message: t('migrate.confirm_msg')
      .replace('{source}', sourceNode?.name || '')
      .replace('{dest}', destNode?.name || ''),
    variant: 'info',
    icon: '🔄',
    confirmText: t('migrate.start'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  migrating.value = true
  migrationStarted.value = true
  migrationResult.value = null
  migrationError.value = ''
  progress.value = { migrated: 0, total: 0 }

  try {
    const res = await api.post<{
      ok: boolean
      total: number
      migrated: number
      failed: number
      failures?: Array<{ username: string; error: string }>
    }>(`/api/nodes/${sourceNodeId.value}/migrate`, {
      destination_node_id: Number(destNodeId.value),
    })

    migrationResult.value = {
      total: res.total,
      migrated: res.migrated,
      failed: res.failed,
      failures: res.failures || [],
    }
    progress.value = { migrated: res.migrated, total: res.total }

    if (res.failed === 0) {
      toast.success(t('migrate.success'))
    } else {
      toast.success(t('migrate.partial_success')
        .replace('{migrated}', String(res.migrated))
        .replace('{failed}', String(res.failed)))
    }
  } catch (err: any) {
    migrationError.value = err.message || t('migrate.failed')
    toast.error(t('migrate.failed'))
  } finally {
    migrating.value = false
  }
}

function resetMigration() {
  migrationStarted.value = false
  migrationResult.value = null
  migrationError.value = ''
  progress.value = { migrated: 0, total: 0 }
  sourceNodeId.value = ''
  destNodeId.value = ''
}

function getProgressColor(): string {
  if (migrationError.value) return 'var(--color-danger, #ef4444)'
  if (migrationResult.value && migrationResult.value.failed > 0) return 'var(--color-warning, #f59e0b)'
  return 'var(--color-success, #10b981)'
}

onMounted(() => {
  nodesStore.loadNodes()
})
</script>

<template>
  <div class="page migrate-view">
    <h3 class="section-title">{{ t('migrate.title') }}</h3>

    <!-- Source/Destination Selection -->
    <section class="migrate-form-section">
      <div class="migrate-form">
        <KFormField name="migrate-source" :label="t('migrate.source_node')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="sourceNodeId"
              :options="nodeOptions"
              :placeholder="t('migrate.select_source')"
              :disabled="migrationStarted"
            />
          </template>
        </KFormField>

        <div class="migrate-arrow">→</div>

        <KFormField name="migrate-dest" :label="t('migrate.dest_node')" required>
          <template #default="{ fieldId }">
            <KSelect
              :id="fieldId"
              v-model="destNodeId"
              :options="destNodeOptions"
              :placeholder="t('migrate.select_dest')"
              :disabled="migrationStarted || !sourceNodeId"
            />
          </template>
        </KFormField>
      </div>

      <div v-if="!migrationStarted" class="migrate-actions">
        <KButton
          variant="primary"
          :disabled="!canMigrate"
          :loading="migrating"
          @click="startMigration"
        >
          {{ t('migrate.start') }}
        </KButton>
      </div>
    </section>

    <!-- Migration Progress -->
    <section v-if="migrationStarted" class="migrate-progress-section">
      <div class="progress-card">
        <div class="progress-card__header">
          <h4 class="progress-card__title">{{ t('migrate.progress') }}</h4>
          <span class="progress-card__percent">{{ progressPercent }}%</span>
        </div>

        <!-- Progress Bar -->
        <div class="progress-bar">
          <div
            class="progress-bar__fill"
            :style="{ width: `${progressPercent}%`, backgroundColor: getProgressColor() }"
          />
        </div>

        <div class="progress-stats">
          <span>{{ t('migrate.migrated') }}: <strong>{{ progress.migrated }}</strong></span>
          <span>{{ t('migrate.total') }}: <strong>{{ progress.total }}</strong></span>
        </div>

        <!-- Error -->
        <div v-if="migrationError" class="migrate-error">
          <span class="migrate-error__icon">⚠</span>
          <span class="migrate-error__text">{{ migrationError }}</span>
        </div>

        <!-- Summary Report -->
        <div v-if="migrationResult" class="migrate-summary">
          <h5 class="migrate-summary__title">{{ t('migrate.summary') }}</h5>
          <div class="migrate-summary__stats">
            <div class="summary-stat summary-stat--success">
              <span class="summary-stat__label">{{ t('migrate.successful') }}</span>
              <span class="summary-stat__value">{{ migrationResult.migrated }}</span>
            </div>
            <div class="summary-stat summary-stat--danger">
              <span class="summary-stat__label">{{ t('migrate.failed_count') }}</span>
              <span class="summary-stat__value">{{ migrationResult.failed }}</span>
            </div>
            <div class="summary-stat">
              <span class="summary-stat__label">{{ t('migrate.total') }}</span>
              <span class="summary-stat__value">{{ migrationResult.total }}</span>
            </div>
          </div>

          <!-- Failures List -->
          <div v-if="migrationResult.failures.length > 0" class="failures-list">
            <h6 class="failures-list__title">{{ t('migrate.failures') }}</h6>
            <div class="failures-table-wrap">
              <table class="failures-table">
                <thead>
                  <tr>
                    <th>{{ t('migrate.username') }}</th>
                    <th>{{ t('migrate.error') }}</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="failure in migrationResult.failures" :key="failure.username">
                    <td>{{ failure.username }}</td>
                    <td class="text-muted">{{ failure.error }}</td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>

          <!-- Reset -->
          <div class="migrate-summary__actions">
            <KButton variant="primary" @click="resetMigration">
              {{ t('migrate.new_migration') }}
            </KButton>
          </div>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.migrate-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.section-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

/* Form */
.migrate-form-section {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.migrate-form {
  display: flex;
  align-items: flex-end;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.migrate-arrow {
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: var(--text-xl);
  color: var(--color-muted);
  padding-bottom: var(--space-2);
}

.migrate-actions {
  display: flex;
  justify-content: flex-end;
}

/* Progress */
.migrate-progress-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.progress-card {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.progress-card__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.progress-card__title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

.progress-card__percent {
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
  color: var(--color-primary);
}

.progress-bar {
  height: 12px;
  background: var(--color-surface-2, #1e2630);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.progress-bar__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.4s ease;
}

.progress-stats {
  display: flex;
  gap: var(--space-4);
  font-size: var(--text-sm);
  color: var(--color-muted);
}

/* Error */
.migrate-error {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: var(--radius-md);
  background: rgba(239, 68, 68, 0.06);
}

.migrate-error__icon {
  color: var(--color-danger, #ef4444);
  flex-shrink: 0;
}

.migrate-error__text {
  font-size: var(--text-sm);
  color: var(--color-text);
}

/* Summary */
.migrate-summary {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}

.migrate-summary__title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

.migrate-summary__stats {
  display: flex;
  gap: var(--space-4);
  flex-wrap: wrap;
}

.summary-stat {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  min-width: 100px;
}

.summary-stat--success {
  border-color: rgba(16, 185, 129, 0.3);
  background: rgba(16, 185, 129, 0.06);
}

.summary-stat--danger {
  border-color: rgba(239, 68, 68, 0.3);
  background: rgba(239, 68, 68, 0.06);
}

.summary-stat__label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.summary-stat__value {
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
  color: var(--color-text);
}

/* Failures */
.failures-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.failures-list__title {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-danger, #ef4444);
}

.failures-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.failures-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.failures-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface-2, #1e2630);
  border-bottom: 1px solid var(--color-border);
  font-weight: var(--font-semibold);
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
}

.failures-table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.failures-table tr:last-child td {
  border-bottom: none;
}

.migrate-summary__actions {
  display: flex;
  gap: var(--space-2);
}

@media (max-width: 768px) {
  .migrate-form {
    flex-direction: column;
    align-items: stretch;
  }

  .migrate-arrow {
    transform: rotate(90deg);
    padding: 0;
  }

  .migrate-summary__stats {
    flex-direction: column;
  }
}
</style>
