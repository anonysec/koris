<script setup lang="ts">
import { computed } from 'vue'
import { useSettingsStore } from '@/stores/settings'
import { useI18n } from '@koris/composables/useI18n'

const { t } = useI18n()
const store = useSettingsStore()

const info = computed(() => store.settings?.panelInfo ?? null)

function formatUptime(seconds: number): string {
  if (!seconds) return '—'
  const days = Math.floor(seconds / 86400)
  const hours = Math.floor((seconds % 86400) / 3600)
  const minutes = Math.floor((seconds % 3600) / 60)
  if (days > 0) return `${days}d ${hours}h ${minutes}m`
  if (hours > 0) return `${hours}h ${minutes}m`
  return `${minutes}m`
}
</script>

<template>
  <section class="settings-section">
    <h3 class="settings-section__title">{{ t('settings.panel_info') }}</h3>

    <div v-if="info" class="info-grid">
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.panel_version') }}</span>
        <span class="info-item__value">{{ info.version }}</span>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.panel_edition') }}</span>
        <span class="info-item__value info-item__value--badge">{{ info.edition }}</span>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.panel_uptime') }}</span>
        <span class="info-item__value">{{ formatUptime(info.uptime) }}</span>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.panel_go_version') }}</span>
        <code class="info-item__value info-item__value--mono">{{ info.goVersion }}</code>
      </div>
      <div class="info-item">
        <span class="info-item__label">{{ t('settings.panel_migration') }}</span>
        <span class="info-item__value">{{ info.migrationVersion }}</span>
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
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
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

.info-item__value--badge {
  display: inline-block;
  background: var(--color-primary);
  color: #fff;
  padding: 2px 8px;
  border-radius: var(--radius-full);
  font-size: var(--text-xs);
  font-weight: var(--font-bold);
  text-transform: uppercase;
  width: fit-content;
}

.info-empty {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
</style>
