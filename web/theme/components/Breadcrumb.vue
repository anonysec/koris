<template>
  <nav class="k-breadcrumb" aria-label="Breadcrumb">
    <ol class="k-breadcrumb__list">
      <li
        v-for="(item, index) in items"
        :key="index"
        class="k-breadcrumb__item"
      >
        <router-link
          v-if="index < items.length - 1 && item.to"
          :to="item.to"
          class="k-breadcrumb__link"
        >
          {{ item.label }}
        </router-link>
        <span
          v-else
          class="k-breadcrumb__current"
          :aria-current="index === items.length - 1 ? 'page' : undefined"
        >
          {{ item.label }}
        </span>

        <svg
          v-if="index < items.length - 1"
          class="k-breadcrumb__separator"
          width="16"
          height="16"
          viewBox="0 0 16 16"
          fill="none"
          aria-hidden="true"
        >
          <path
            d="M6 3l5 5-5 5"
            stroke="currentColor"
            stroke-width="1.5"
            stroke-linecap="round"
            stroke-linejoin="round"
          />
        </svg>
      </li>
    </ol>
  </nav>
</template>

<script setup lang="ts">
import type { Breadcrumb } from '@koris/types/components'

export interface KBreadcrumbProps {
  items: Breadcrumb[]
}

defineProps<KBreadcrumbProps>()
</script>

<style scoped>
.k-breadcrumb {
  font-family: var(--font-family);
  font-size: var(--text-sm);
}

.k-breadcrumb__list {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  list-style: none;
  margin: 0;
  padding: 0;
}

.k-breadcrumb__item {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}

.k-breadcrumb__link {
  color: var(--color-muted);
  text-decoration: none;
  transition: color var(--duration-normal) var(--ease-default);
  border-radius: var(--radius-sm);
  padding: 2px 4px;
  margin: -2px -4px;
}

.k-breadcrumb__link:hover {
  color: var(--color-accent);
}

.k-breadcrumb__link:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.k-breadcrumb__current {
  color: var(--color-text);
  font-weight: var(--font-medium);
}

.k-breadcrumb__separator {
  color: var(--color-muted);
  opacity: 0.5;
  flex-shrink: 0;
}
</style>
