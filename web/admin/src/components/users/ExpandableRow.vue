<script setup lang="ts">
/**
 * ExpandableRow — Admin-specific wrapper around KExpandableRow for the Users table.
 *
 * Renders a user row with chevron toggle for expand/collapse (showing summary info),
 * and emits 'row-click' when the row body (not chevron) is clicked to open the detail panel.
 *
 * Requirements: 6.1, 6.2, 6.3, 6.7
 */
import { computed } from 'vue'
import KExpandableRow from '@koris/ui/KExpandableRow.vue'
import UsageBar from '@koris/ui/UsageBar.vue'
import type { Customer } from '@koris/types'
import { formatBytes } from '@koris/core'

export interface ExpandableRowProps {
  user: Customer
  expanded: boolean
}

const props = defineProps<ExpandableRowProps>()

const emit = defineEmits<{
  (e: 'toggle'): void
  (e: 'row-click'): void
  (e: 'edit'): void
}>()

/**
 * Compute "Expires in X days" text from the user's subscription end_date.
 * Falls back to "No expiry" if no subscription or end date is set.
 */
const expiryText = computed(() => {
  const sub = (props.user as any).subscription
  const endDate = sub?.end_date || (props.user as any).expiry_date
  if (!endDate) return 'No expiry'

  const now = new Date()
  const expiry = new Date(endDate)
  const diffMs = expiry.getTime() - now.getTime()
  const diffDays = Math.ceil(diffMs / (1000 * 60 * 60 * 24))

  if (diffDays < 0) return 'Expired'
  if (diffDays === 0) return 'Expires today'
  if (diffDays === 1) return 'Expires in 1 day'
  return `Expires in ${diffDays} days`
})

/**
 * Format the last activity timestamp as a relative or absolute string.
 */
const lastActivityText = computed(() => {
  const lastActivity = (props.user as any).last_activity || (props.user as any).last_connected_at
  if (!lastActivity) return 'No activity'

  const now = new Date()
  const actDate = new Date(lastActivity)
  const diffMs = now.getTime() - actDate.getTime()
  const diffMinutes = Math.floor(diffMs / (1000 * 60))

  if (diffMinutes < 1) return 'Just now'
  if (diffMinutes < 60) return `${diffMinutes}m ago`
  const diffHours = Math.floor(diffMinutes / 60)
  if (diffHours < 24) return `${diffHours}h ago`
  const diffDays = Math.floor(diffHours / 24)
  if (diffDays < 30) return `${diffDays}d ago`
  return actDate.toLocaleDateString('en', { month: 'short', day: 'numeric', year: 'numeric' })
})

/**
 * Total usage bytes formatted as human-readable string.
 */
const totalUsageText = computed(() => {
  const usage = (props.user as any).total_usage_bytes ?? (props.user as any).data_used_bytes ?? 0
  return formatBytes(usage)
})

/**
 * Current period usage in bytes (for the usage bar).
 */
const currentUsed = computed(() => {
  return (props.user as any).current_usage_bytes ?? (props.user as any).data_used_bytes ?? 0
})

/**
 * Data limit in bytes (0 = unlimited).
 */
const dataLimit = computed(() => {
  return (props.user as any).data_limit_bytes ?? (props.user as any).max_data_bytes ?? 0
})

/**
 * Handle row body click — opens the detail panel (Requirement 6.7).
 * Stops propagation so it doesn't interfere with chevron toggle.
 */
function handleRowClick() {
  emit('row-click')
}

/**
 * Handle edit icon click in expanded summary.
 */
function handleEdit() {
  emit('edit')
}

/**
 * Copy subscription link to clipboard.
 */
async function handleCopyLink() {
  const subToken = (props.user as any).sub_token
  if (!subToken) return
  const link = `${window.location.origin}/sub/${subToken}`
  try {
    await navigator.clipboard.writeText(link)
  } catch {
    // Fallback: silently fail
  }
}

/**
 * Generate QR code — emits edit with intent (handled by parent).
 * For now, we just emit 'edit' as QR generation is modal-based.
 */
function handleQR() {
  // QR code generation is handled by the parent via modal
  emit('edit')
}
</script>

<template>
  <KExpandableRow
    :expanded="expanded"
    :transition-duration="200"
    @toggle="emit('toggle')"
  >
    <template #row>
      <div
        class="expandable-row__body"
        role="button"
        tabindex="0"
        @click="handleRowClick"
        @keydown.enter="handleRowClick"
        @keydown.space.prevent="handleRowClick"
      >
        <slot />
      </div>
    </template>

    <template #expanded>
      <div class="expandable-row__summary">
        <!-- Usage bar (current period) -->
        <div class="expandable-row__usage">
          <UsageBar
            :used="currentUsed"
            :limit="dataLimit"
            :show-label="true"
            size="sm"
          />
          <span class="expandable-row__total-usage">
            Total: {{ totalUsageText }}
          </span>
        </div>

        <!-- Action icons -->
        <div class="expandable-row__actions">
          <button
            type="button"
            class="expandable-row__action-btn"
            title="Edit user"
            aria-label="Edit user"
            @click.stop="handleEdit"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <path d="M11.5 2.5L13.5 4.5L5 13H3V11L11.5 2.5Z" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button
            type="button"
            class="expandable-row__action-btn"
            title="Copy subscription link"
            aria-label="Copy subscription link"
            @click.stop="handleCopyLink"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <path d="M10 5H6.5C5.67 5 5 5.67 5 6.5V12.5C5 13.33 5.67 14 6.5 14H10.5C11.33 14 12 13.33 12 12.5V9" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M7 11V8.5C7 7.67 7.67 7 8.5 7H14" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
              <path d="M11 4L14 7" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" stroke-linejoin="round"/>
            </svg>
          </button>
          <button
            type="button"
            class="expandable-row__action-btn"
            title="Generate QR code"
            aria-label="Generate QR code"
            @click.stop="handleQR"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <rect x="2" y="2" width="5" height="5" rx="1" stroke="currentColor" stroke-width="1.5"/>
              <rect x="9" y="2" width="5" height="5" rx="1" stroke="currentColor" stroke-width="1.5"/>
              <rect x="2" y="9" width="5" height="5" rx="1" stroke="currentColor" stroke-width="1.5"/>
              <rect x="10" y="10" width="3" height="3" rx="0.5" stroke="currentColor" stroke-width="1.5"/>
            </svg>
          </button>
        </div>

        <!-- Expiry info -->
        <span class="expandable-row__expiry">
          {{ expiryText }}
        </span>

        <!-- Last activity -->
        <span class="expandable-row__activity">
          {{ lastActivityText }}
        </span>
      </div>
    </template>
  </KExpandableRow>
</template>

<style scoped>
.expandable-row__body {
  flex: 1;
  min-width: 0;
  cursor: pointer;
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 4px 0;
}

.expandable-row__body:focus-visible {
  outline: 2px solid var(--color-primary, #2563eb);
  outline-offset: 2px;
  border-radius: var(--radius-sm, 4px);
}

.expandable-row__summary {
  display: flex;
  align-items: center;
  gap: 16px;
  padding: 12px 16px 12px 36px;
  background: var(--color-surface-1, #151b23);
  border-radius: var(--radius-sm, 6px);
  margin: 4px 0 8px;
  flex-wrap: wrap;
}

.expandable-row__usage {
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 160px;
  max-width: 200px;
}

.expandable-row__total-usage {
  font-size: var(--text-xs, 0.75rem);
  color: var(--color-muted, #8b98a5);
  white-space: nowrap;
}

.expandable-row__actions {
  display: flex;
  align-items: center;
  gap: 4px;
}

.expandable-row__action-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  padding: 0;
  border: none;
  border-radius: var(--radius-sm, 6px);
  background: transparent;
  color: var(--color-muted, #8b98a5);
  cursor: pointer;
  transition: background 100ms ease-out, color 100ms ease-out, transform 100ms ease-out;
}

.expandable-row__action-btn:hover {
  background: var(--color-surface-2, #1e2630);
  color: var(--color-text, #e6edf3);
  transform: scale(1.05);
}

.expandable-row__action-btn:focus-visible {
  outline: 2px solid var(--color-primary, #2563eb);
  outline-offset: 2px;
}

.expandable-row__action-btn:active {
  transform: scale(0.95);
}

.expandable-row__expiry {
  font-size: var(--text-sm, 0.875rem);
  color: var(--color-muted, #8b98a5);
  white-space: nowrap;
}

.expandable-row__activity {
  font-size: var(--text-sm, 0.875rem);
  color: var(--color-muted, #8b98a5);
  white-space: nowrap;
  margin-left: auto;
}

/* Respect reduced motion */
@media (prefers-reduced-motion: reduce) {
  .expandable-row__action-btn {
    transition: none;
  }

  .expandable-row__action-btn:hover {
    transform: none;
  }

  .expandable-row__action-btn:active {
    transform: none;
  }
}
</style>
