<template>
  <div class="k-empty-state" role="status">
    <span v-if="icon" class="k-empty-state__icon" aria-hidden="true">
      {{ icon }}
    </span>

    <h3 class="k-empty-state__title">{{ title }}</h3>

    <p v-if="description" class="k-empty-state__description">
      {{ description }}
    </p>

    <button
      v-if="actionText"
      :class="[
        'k-empty-state__action',
        `k-empty-state__action--${actionVariant}`,
      ]"
      @click="emit('action')"
    >
      {{ actionText }}
    </button>
  </div>
</template>

<script setup lang="ts">
interface KEmptyStateProps {
  icon?: string
  title: string
  description?: string
  actionText?: string
  actionVariant?: 'primary' | 'ghost'
}

withDefaults(defineProps<KEmptyStateProps>(), {
  actionVariant: 'primary',
})

const emit = defineEmits<{
  (e: 'action'): void
}>()
</script>

<style scoped>
.k-empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  text-align: center;
  padding: var(--space-12) var(--space-6);
  gap: var(--space-3);
}

.k-empty-state__icon {
  font-size: 48px;
  line-height: 1;
  margin-bottom: var(--space-2);
  opacity: 0.7;
}

.k-empty-state__title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.k-empty-state__description {
  font-size: var(--text-base);
  color: var(--color-muted);
  margin: 0;
  max-width: 360px;
  line-height: var(--leading-normal);
}

.k-empty-state__action {
  margin-top: var(--space-4);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  height: 36px;
  padding: 8px 16px;
  border: none;
  border-radius: var(--radius-md);
  font-family: var(--font-family);
  font-size: var(--text-base);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition:
    background var(--duration-normal) var(--ease-default),
    box-shadow var(--duration-normal) var(--ease-default);
}

.k-empty-state__action--primary {
  background: var(--gradient-brand);
  color: #fff;
  box-shadow: var(--shadow-brand);
}

.k-empty-state__action--primary:hover {
  box-shadow: 0 6px 20px rgba(37, 99, 235, 0.35);
}

.k-empty-state__action--ghost {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}

.k-empty-state__action--ghost:hover {
  background: var(--color-surface-2);
  border-color: var(--color-muted);
}

.k-empty-state__action:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}
</style>
