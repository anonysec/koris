<template>
  <nav class="k-pagination" aria-label="Pagination">
    <button
      class="k-pagination__btn k-pagination__btn--prev"
      :disabled="currentPage <= 1"
      :aria-disabled="currentPage <= 1"
      aria-label="Go to first page"
      @click="goToPage(1)"
    >
      &laquo;
    </button>

    <button
      class="k-pagination__btn k-pagination__btn--prev"
      :disabled="currentPage <= 1"
      :aria-disabled="currentPage <= 1"
      aria-label="Go to previous page"
      @click="goToPage(currentPage - 1)"
    >
      &lsaquo;
    </button>

    <template v-for="item in pageItems" :key="item.key">
      <span v-if="item.type === 'ellipsis'" class="k-pagination__ellipsis" aria-hidden="true">
        &hellip;
      </span>
      <button
        v-else
        :class="[
          'k-pagination__btn',
          'k-pagination__btn--page',
          { 'k-pagination__btn--active': item.page === currentPage },
        ]"
        :aria-label="`Page ${item.page}`"
        :aria-current="item.page === currentPage ? 'page' : undefined"
        @click="goToPage(item.page!)"
      >
        {{ item.page }}
      </button>
    </template>

    <button
      class="k-pagination__btn k-pagination__btn--next"
      :disabled="currentPage >= totalPages"
      :aria-disabled="currentPage >= totalPages"
      aria-label="Go to next page"
      @click="goToPage(currentPage + 1)"
    >
      &rsaquo;
    </button>

    <button
      class="k-pagination__btn k-pagination__btn--next"
      :disabled="currentPage >= totalPages"
      :aria-disabled="currentPage >= totalPages"
      aria-label="Go to last page"
      @click="goToPage(totalPages)"
    >
      &raquo;
    </button>
  </nav>
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface KPaginationProps {
  currentPage: number
  totalPages: number
  siblingCount?: number
}

interface PageItem {
  key: string
  type: 'page' | 'ellipsis'
  page?: number
}

const props = withDefaults(defineProps<KPaginationProps>(), {
  siblingCount: 1,
})

const emit = defineEmits<{
  (e: 'page-change', page: number): void
}>()

const pageItems = computed<PageItem[]>(() => {
  const { currentPage, totalPages, siblingCount } = props
  const items: PageItem[] = []

  if (totalPages <= 0) return items

  // Always show page 1
  items.push({ key: 'page-1', type: 'page', page: 1 })

  const leftSibling = Math.max(2, currentPage - siblingCount)
  const rightSibling = Math.min(totalPages - 1, currentPage + siblingCount)

  // Left ellipsis
  if (leftSibling > 2) {
    items.push({ key: 'ellipsis-left', type: 'ellipsis' })
  }

  // Sibling pages
  for (let i = leftSibling; i <= rightSibling; i++) {
    if (i !== 1 && i !== totalPages) {
      items.push({ key: `page-${i}`, type: 'page', page: i })
    }
  }

  // Right ellipsis
  if (rightSibling < totalPages - 1) {
    items.push({ key: 'ellipsis-right', type: 'ellipsis' })
  }

  // Always show last page if more than 1 page
  if (totalPages > 1) {
    items.push({ key: `page-${totalPages}`, type: 'page', page: totalPages })
  }

  return items
})

function goToPage(page: number) {
  if (page < 1 || page > props.totalPages || page === props.currentPage) return
  emit('page-change', page)
}
</script>

<style scoped>
.k-pagination {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-family: var(--font-family);
}

.k-pagination__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  padding: 0 var(--space-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-text);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  cursor: pointer;
  transition:
    background var(--duration-normal) var(--ease-default),
    border-color var(--duration-normal) var(--ease-default),
    color var(--duration-normal) var(--ease-default);
  outline: none;
  line-height: 1;
}

.k-pagination__btn:hover:not(:disabled):not(.k-pagination__btn--active) {
  background: var(--color-surface-2);
  border-color: var(--color-muted);
}

.k-pagination__btn:focus-visible {
  outline: 2px solid var(--color-accent);
  outline-offset: 2px;
}

.k-pagination__btn:disabled {
  opacity: 0.4;
  cursor: not-allowed;
}

.k-pagination__btn--active {
  background: var(--gradient-brand);
  border-color: transparent;
  color: #fff;
  box-shadow: var(--shadow-brand);
}

.k-pagination__ellipsis {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
  color: var(--color-muted);
  font-size: var(--text-sm);
  user-select: none;
}
</style>
