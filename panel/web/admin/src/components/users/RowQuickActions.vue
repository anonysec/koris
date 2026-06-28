<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import type { Customer } from '@koris/types'

export interface RowQuickActionsProps {
  user: Customer
  visible: boolean
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
 * If the user is disabled, show Enable; otherwise show Disable.
 */
const isDisabled = computed(() => props.user.status === 'disabled')

/**
 * Toggle label and icon based on current status.
 */
const toggleLabel = computed(() =>
  isDisabled.value ? t('customers.enable') : t('customers.disable')
)

const toggleIcon = computed(() => (isDisabled.value ? '✓' : '⏸'))

/**
 * Whether all actions should be disabled (loading state).
 */
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
  <Transition name="row-actions">
    <div
      v-show="visible"
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

      <!-- Action buttons -->
      <template v-else>
        <KButton
          size="sm"
          variant="ghost"
          :icon="toggleIcon"
          :disabled="actionsDisabled"
          :aria-label="toggleLabel"
          :title="toggleLabel"
          class="row-quick-actions__btn"
          @click.stop="handleToggleStatus"
        >
          {{ toggleLabel }}
        </KButton>

        <KButton
          size="sm"
          variant="ghost"
          icon="↺"
          :disabled="actionsDisabled"
          :aria-label="t('customers.traffic_reset')"
          :title="t('customers.traffic_reset')"
          class="row-quick-actions__btn"
          @click.stop="handleResetTraffic"
        >
          {{ t('customers.traffic_reset') }}
        </KButton>

        <KButton
          size="sm"
          variant="danger"
          icon="🗑"
          :disabled="actionsDisabled"
          :aria-label="t('customers.delete')"
          :title="t('customers.delete')"
          class="row-quick-actions__btn row-quick-actions__btn--danger"
          @click.stop="handleDelete"
        >
          {{ t('customers.delete') }}
        </KButton>
      </template>
    </div>
  </Transition>
</template>

<style scoped>
.row-quick-actions {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1, 4px);
  padding: var(--space-1, 4px) var(--space-2, 8px);
  background: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 8px);
  box-shadow: var(--shadow-sm, 0 2px 8px rgba(0, 0, 0, 0.2));
}

.row-quick-actions__btn {
  transition: transform var(--transition-hover, 100ms ease-out);
}

.row-quick-actions__btn:hover:not(:disabled) {
  transform: scale(1.05);
}

.row-quick-actions__spinner {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-2, 8px) var(--space-4, 16px);
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

/* Transition for showing/hiding the actions */
.row-actions-enter-active,
.row-actions-leave-active {
  transition: opacity var(--transition-hover, 100ms ease-out);
}

.row-actions-enter-from,
.row-actions-leave-to {
  opacity: 0;
}

@media (prefers-reduced-motion: reduce) {
  .row-quick-actions__btn:hover:not(:disabled) {
    transform: none;
  }

  .row-quick-actions__spinner-icon {
    animation: none;
  }

  .row-actions-enter-active,
  .row-actions-leave-active {
    transition: none;
  }
}
</style>
