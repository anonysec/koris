<script setup lang="ts">
/**
 * KExpandableRow — An expandable row wrapper that reveals content below on toggle.
 *
 * Uses CSS grid-template-rows animation for smooth height transitions.
 * - Props: expanded (boolean), transitionDuration (ms, default 200)
 * - Emits: toggle
 * - Respects prefers-reduced-motion: transitions are instant (0ms) when enabled
 * - Accessible: aria-expanded on trigger, aria-controls linking trigger to panel
 */
import { computed, useId } from 'vue'

const props = withDefaults(defineProps<{
  expanded: boolean
  transitionDuration?: number
}>(), {
  transitionDuration: 200,
})

defineEmits<{
  (e: 'toggle'): void
}>()

const panelId = useId()

const transitionStyle = computed(() => ({
  '--k-expandable-duration': `${props.transitionDuration}ms`,
}))
</script>

<template>
  <div class="k-expandable-row" :style="transitionStyle">
    <!-- Trigger: the always-visible row content -->
    <div class="k-expandable-row__trigger">
      <button
        type="button"
        class="k-expandable-row__chevron-btn"
        :aria-expanded="expanded"
        :aria-controls="panelId"
        aria-label="Toggle row details"
        @click.stop="$emit('toggle')"
      >
        <svg
          class="k-expandable-row__chevron"
          :class="{ 'k-expandable-row__chevron--rotated': expanded }"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M6 4L10 8L6 12"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </button>
      <div class="k-expandable-row__content">
        <slot name="row" />
      </div>
    </div>

    <!-- Expandable panel -->
    <div
      :id="panelId"
      class="k-expandable-row__panel"
      :class="{ 'k-expandable-row__panel--open': expanded }"
      role="region"
      :aria-hidden="!expanded"
    >
      <div class="k-expandable-row__panel-inner">
        <slot name="expanded" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.k-expandable-row {
  width: 100%;
}

/* Trigger row */
.k-expandable-row__trigger {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.k-expandable-row__chevron-btn {
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
  flex-shrink: 0;
  transition: background var(--k-expandable-duration, 200ms) ease-out,
              color var(--k-expandable-duration, 200ms) ease-out;
}

.k-expandable-row__chevron-btn:hover {
  background: var(--color-surface-2, #1e2630);
  color: var(--color-text, #e6edf3);
}

.k-expandable-row__chevron-btn:focus-visible {
  outline: 2px solid var(--color-primary, #2563eb);
  outline-offset: 2px;
}

.k-expandable-row__chevron {
  transition: transform var(--k-expandable-duration, 200ms) ease-out;
}

.k-expandable-row__chevron--rotated {
  transform: rotate(90deg);
}

.k-expandable-row__content {
  flex: 1;
  min-width: 0;
}

/* Expandable panel — uses CSS grid for smooth height animation */
.k-expandable-row__panel {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows var(--k-expandable-duration, 200ms) ease-out;
}

.k-expandable-row__panel--open {
  grid-template-rows: 1fr;
}

.k-expandable-row__panel-inner {
  overflow: hidden;
}

/* Respect reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  .k-expandable-row__panel {
    transition-duration: 0ms;
  }

  .k-expandable-row__chevron {
    transition-duration: 0ms;
  }

  .k-expandable-row__chevron-btn {
    transition-duration: 0ms;
  }
}
</style>
