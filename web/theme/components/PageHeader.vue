<script setup lang="ts">
/**
 * PageHeader — standard page/view header used across all admin & portal pages.
 * Provides a consistent title, optional subtitle/eyebrow, and an actions slot.
 *
 * Usage:
 *   <PageHeader title="Customers" subtitle="Manage accounts & subscriptions">
 *     <template #actions><Button>Add</Button></template>
 *   </PageHeader>
 */
defineProps<{
  title: string
  subtitle?: string
  eyebrow?: string
}>()
</script>

<template>
  <header class="k-pagehead">
    <div class="k-pagehead__text">
      <p v-if="eyebrow" class="k-pagehead__eyebrow">{{ eyebrow }}</p>
      <h1 class="k-pagehead__title">
        <slot name="title">{{ title }}</slot>
      </h1>
      <p v-if="subtitle || $slots.subtitle" class="k-pagehead__subtitle">
        <slot name="subtitle">{{ subtitle }}</slot>
      </p>
    </div>
    <div v-if="$slots.actions" class="k-pagehead__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.k-pagehead {
  display: flex;
  align-items: flex-end;
  justify-content: space-between;
  gap: var(--space-4);
  flex-wrap: wrap;
  margin-bottom: var(--space-6);
}
.k-pagehead__text { min-width: 0; }
.k-pagehead__eyebrow {
  margin: 0 0 var(--space-1);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  letter-spacing: var(--tracking-wider);
  text-transform: uppercase;
  color: var(--color-primary);
}
.k-pagehead__title {
  margin: 0;
  font-size: var(--text-3xl);
  font-weight: var(--font-bold);
  letter-spacing: var(--tracking-tight);
  line-height: var(--leading-tight);
  color: var(--color-text);
}
.k-pagehead__subtitle {
  margin: var(--space-2) 0 0;
  font-size: var(--text-md);
  color: var(--color-muted);
  line-height: var(--leading-snug);
  max-width: 60ch;
}
.k-pagehead__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}
@media (max-width: 640px) {
  .k-pagehead { align-items: stretch; }
  .k-pagehead__actions { width: 100%; }
}
</style>
