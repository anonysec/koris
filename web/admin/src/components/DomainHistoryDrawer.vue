<script setup lang="ts">
import { ref, watch } from 'vue'
import { useDomainsStore, type VpnDomain, type DomainIPHistory } from '@/stores/domains'
import KSlideOver from '@koris/ui/KSlideOver.vue'

const props = defineProps<{
  open: boolean
  domain: VpnDomain | null
}>()

const emit = defineEmits<{
  (e: 'close'): void
}>()

const domainsStore = useDomainsStore()

const historyItems = ref<DomainIPHistory[]>([])
const loading = ref(false)

// Fetch history when the drawer opens
watch(() => props.open, async (isOpen) => {
  if (isOpen && props.domain) {
    loading.value = true
    try {
      historyItems.value = await domainsStore.fetchHistory(props.domain.id)
    } finally {
      loading.value = false
    }
  } else {
    historyItems.value = []
  }
})

function handleClose() {
  emit('close')
}

function formatDate(dateStr: string): string {
  const date = new Date(dateStr)
  return date.toLocaleString(undefined, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
    second: '2-digit',
  })
}
</script>

<template>
  <KSlideOver :open="open" title="IP Rotation History" @close="handleClose">
    <div class="history-drawer">
      <!-- Domain info header -->
      <div v-if="domain" class="history-drawer__header">
        <div class="history-drawer__domain-info">
          <span class="history-drawer__domain-name">{{ domain.name }}</span>
          <span class="history-drawer__current-ip">Current: {{ domain.ip_address }}</span>
        </div>
      </div>

      <!-- Loading state -->
      <div v-if="loading" class="history-drawer__loading">
        <span class="spinner" />
        <span>Loading history...</span>
      </div>

      <!-- Empty state -->
      <div v-else-if="historyItems.length === 0" class="history-drawer__empty">
        <p>No rotation history for this domain.</p>
      </div>

      <!-- Timeline view -->
      <div v-else class="history-drawer__timeline">
        <div
          v-for="item in historyItems"
          :key="item.id"
          class="timeline-item"
        >
          <div class="timeline-item__marker">
            <div class="timeline-item__dot" />
            <div class="timeline-item__line" />
          </div>

          <div class="timeline-item__content">
            <div class="timeline-item__change">
              <code class="timeline-item__ip timeline-item__ip--previous">{{ item.previous_ip }}</code>
              <span class="timeline-item__arrow">→</span>
              <code class="timeline-item__ip timeline-item__ip--new">{{ item.new_ip }}</code>
            </div>

            <div class="timeline-item__meta">
              <span class="timeline-item__admin">{{ item.admin_username }}</span>
              <span class="timeline-item__separator">·</span>
              <span class="timeline-item__time">{{ formatDate(item.rotated_at) }}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  </KSlideOver>
</template>

<style scoped>
.history-drawer {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 1rem);
  padding: var(--space-4, 1rem);
}

.history-drawer__header {
  padding-bottom: var(--space-3, 0.75rem);
  border-bottom: 1px solid var(--color-border, #28333f);
}

.history-drawer__domain-info {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 0.25rem);
}

.history-drawer__domain-name {
  font-size: var(--text-base, 1rem);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text);
}

.history-drawer__current-ip {
  font-size: var(--text-sm, 0.875rem);
  color: var(--color-muted, #888);
  font-family: var(--font-mono, monospace);
}

.history-drawer__loading {
  display: flex;
  align-items: center;
  gap: var(--space-2, 0.5rem);
  padding: var(--space-4, 1rem);
  font-size: var(--text-sm);
  color: var(--color-muted, #888);
}

.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid var(--color-border, #28333f);
  border-top-color: var(--color-primary, #6366f1);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.history-drawer__empty {
  padding: var(--space-6, 1.5rem) var(--space-4, 1rem);
  text-align: center;
}

.history-drawer__empty p {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-muted, #888);
}

/* Timeline */
.history-drawer__timeline {
  display: flex;
  flex-direction: column;
}

.timeline-item {
  display: flex;
  gap: var(--space-3, 0.75rem);
  position: relative;
}

.timeline-item__marker {
  display: flex;
  flex-direction: column;
  align-items: center;
  flex-shrink: 0;
  width: 20px;
}

.timeline-item__dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--color-primary, #6366f1);
  border: 2px solid var(--color-surface, #0b1120);
  box-shadow: 0 0 0 2px var(--color-primary, #6366f1);
  flex-shrink: 0;
  margin-top: 4px;
}

.timeline-item__line {
  width: 2px;
  flex: 1;
  background: var(--color-border, #28333f);
  min-height: 20px;
}

.timeline-item:last-child .timeline-item__line {
  display: none;
}

.timeline-item__content {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 0.25rem);
  padding-bottom: var(--space-4, 1rem);
  flex: 1;
}

.timeline-item__change {
  display: flex;
  align-items: center;
  gap: var(--space-2, 0.5rem);
  flex-wrap: wrap;
}

.timeline-item__ip {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-sm, 0.875rem);
  padding: 2px 6px;
  border-radius: var(--radius-sm, 4px);
}

.timeline-item__ip--previous {
  color: var(--color-muted, #888);
  background: var(--color-surface-2, #1e2630);
  text-decoration: line-through;
}

.timeline-item__ip--new {
  color: var(--color-success, #22c55e);
  background: rgba(34, 197, 94, 0.08);
}

.timeline-item__arrow {
  color: var(--color-muted, #888);
  font-size: var(--text-sm);
}

.timeline-item__meta {
  display: flex;
  align-items: center;
  gap: var(--space-1, 0.25rem);
  font-size: var(--text-xs, 0.75rem);
  color: var(--color-muted, #888);
}

.timeline-item__admin {
  font-weight: var(--font-medium, 500);
}

.timeline-item__separator {
  opacity: 0.5;
}

.timeline-item__time {
  white-space: nowrap;
}
</style>
