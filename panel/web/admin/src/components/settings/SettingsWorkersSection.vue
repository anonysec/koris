<script setup lang="ts">
import { computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useI18n } from '@koris/composables/useI18n'
import KStatusPill from '@koris/ui/KStatusPill.vue'

const { t } = useI18n()
const store = useSettingsStore()

const workers = computed(() => store.settings?.workers ?? null)

const healthStatus = computed(() => {
  if (!workers.value) return 'offline'
  return workers.value.healthStatus === 'healthy' ? 'online' : 'offline'
})
</script>

<template>
  <section class="settings-section">
    <h3 class="settings-section__title">{{ t('settings.workers') }}</h3>

    <div v-if="workers" class="info-grid">
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.workers_configured') }}</span>
        <span class="info-item__value">{{ workers.configured }}</span>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.workers_active') }}</span>
        <span class="info-item__value">{{ workers.active }}</span>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.workers_leader') }}</span>
        <code class="info-item__value info-item__value--mono">{{ workers.leaderId || '—' }}</code>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.workers_current') }}</span>
        <code class="info-item__value info-item__value--mono">{{ workers.currentWorkerId || '—' }}</code>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.workers_health') }}</span>
        <KStatusPill :status="healthStatus" size="sm" />
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
