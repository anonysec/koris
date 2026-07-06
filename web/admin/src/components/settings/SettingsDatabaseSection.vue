<script setup lang="ts">
import { computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useI18n } from '@koris/composables/useI18n'
import StatusPill from '@koris/ui/StatusPill.vue'

const { t } = useI18n()
const store = useSettingsStore()

const db = computed(() => store.settings?.database ?? null)

const backendLabel = computed(() => {
  if (!db.value) return '—'
  const labels: Record<string, string> = {
    timescaledb: 'TimescaleDB',
    postgresql: 'PostgreSQL',
    mariadb: 'MariaDB',
    sqlite: 'SQLite',
  }
  return labels[db.value.backend] ?? db.value.backend
})
</script>

<template>
  <section class="settings-section">
    <h3 class="settings-section__title">{{ t('settings.database') }}</h3>

    <div v-if="db" class="info-grid">
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.db_backend') }}</span>
        <span class="info-item__value">{{ backendLabel }}</span>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.db_status') }}</span>
        <StatusPill :status="db.connected ? 'online' : 'offline'" size="sm" />
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.db_version') }}</span>
        <code class="info-item__value info-item__value--mono">{{ db.version }}</code>
      </div>
      <template v-if="db.backend === 'timescaledb'">
        <div class="info-item">
          <span class="info-item__label">{{ t('settings.timescale_version') }}</span>
          <code class="info-item__value info-item__value--mono">{{ db.timescaleVersion ?? '—' }}</code>
        </div>
        <div class="info-item">
          <span class="info-item__label">{{ t('settings.hypertable_status') }}</span>
          <StatusPill :status="db.hypertableEnabled ? 'online' : 'offline'" size="sm" />
        </div>
      </template>
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

.info-empty {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
</style>
