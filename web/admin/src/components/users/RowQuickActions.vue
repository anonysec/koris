<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import type { Customer } from '@koris/types'

export interface RowQuickActionsProps {
  user: Customer
  loading?: boolean
  activeAction?: string | null
}

const props = withDefaults(defineProps<RowQuickActionsProps>(), {
  loading: false,
  activeAction: null,
})

const emit = defineEmits<{
  (e: 'enable'): void
  (e: 'disable'): void
  (e: 'reset-traffic'): void
  (e: 'delete'): void
}>()

const { t } = useI18n()

/**
 * Determine whether to show Enable or Disable based on user status.
 */
const isDisabled = computed(() => props.user.status === 'disabled')

const toggleLabel = computed(() =>
  isDisabled.value ? t('customers.enable') : t('customers.disable')
)

const toggleIcon = computed(() => (isDisabled.value ? '✓' : '⏸'))

const actionsDisabled = computed(() => props.loading)

function handleToggleStatus() {
  if (actionsDisabled.value) return
  if (isDisabled.value) {
    emit('enable')
  } else {
    emit('disable')
  }
}

function handleResetTraffic() {
  if (actionsDisabled.value) return
  emit('reset-traffic')
}

function handleDelete() {
  if (actionsDisabled.value) return
  emit('delete')
}
</script>

<template>
  <div
    class="row-quick-actions"
    role="toolbar"
    :aria-label="t('customers.quick_actions') || 'Quick actions'"
  >
    <!-- Loading spinner -->
    <div v-if="loading" class="row-quick-actions__spinner" aria-label="Loading">
      <svg class="row-quick-actions__spinner-icon" viewBox="0 0 24 24" fill="none">
        <circle
          cx="12"
          cy="12"
          r="10"
          stroke="currentColor"
          stroke-width="3"
          stroke-linecap="round"
          stroke-dasharray="50 20"
        />
      </svg>
    </div>

    <!-- Action buttons — plain, minimal -->
    <template v-else>
      <button
        type="button"
        class="row-quick-actions__btn"
        :disabled="actionsDisabled"
        :aria-label="toggleLabel"
        :title="toggleLabel"
        @click.stop="handleToggleStatus"
      >{{ toggleIcon }}</button>

      <button
        type="button"
        class="row-quick-actions__btn"
        :disabled="actionsDisabled"
        :aria-label="t('customers.traffic_reset')"
        :title="t('customers.traffic_reset')"
        @click.stop="handleResetTraffic"
      >↺</button>

      <button
        type="button"
        class="row-quick-actions__btn row-quick-actions__btn--danger"
        :disabled="actionsDisabled"
        :aria-label="t('customers.delete')"
        :title="t('customers.delete')"
        @click.stop="handleDelete"
      >🗑</button>
    </template>
  </div>
</template>

<style scoped>
.row-quick-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1, 4px);
}

.row-quick-actions__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 28px;
  width: 28px;
  height: 28px;
  padding: 4px;
  border: none;
  background: none;
  color: var(--color-muted, #8b98a5);
  font-size: 14px;
  cursor: pointer;
  border-radius: var(--radius-sm, 4px);
  transition: color 100ms ease, background 100ms ease;
}

.row-quick-actions__btn:hover:not(:disabled) {
  color: var(--color-text, #e6edf3);
  background: var(--color-surface-2, #1e2630);
}

.row-quick-actions__btn--danger:hover:not(:disabled) {
  color: var(--color-danger, #ef4444);
  background: rgba(239, 68, 68, 0.08);
}

.row-quick-actions__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.row-quick-actions__spinner {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-1, 4px);
  color: var(--color-muted, #8b98a5);
}

.row-quick-actions__spinner-icon {
  width: 18px;
  height: 18px;
  animation: row-actions-spin 0.75s linear infinite;
}

@keyframes row-actions-spin {
  from {
    transform: rotate(0deg);
  }
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .row-quick-actions__btn:hover:not(:disabled) {
    transform: none;
  }

  .row-quick-actions__spinner-icon {
    animation: none;
  }
}
</style>
