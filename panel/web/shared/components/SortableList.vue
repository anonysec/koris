<script setup lang="ts">
import { computed } from 'vue'
import { useSortable, type SortableOptions } from '../composables/useSortable'
import type { Ref } from 'vue'

/**
 * SortableList — A reusable drag-and-drop list container.
 *
 * Wraps children in a sortable container with drag handles and visual feedback.
 * Emits 'reorder' event with the new order (array of IDs) after each drag.
 *
 * @example
 * ```vue
 * <SortableList
 *   :items="plans"
 *   id-field="id"
 *   entity="plans"
 *   persist-endpoint="/api/admin/reorder"
 *   @reorder="handleReorder"
 * >
 *   <template #item="{ item, index }">
 *     <div>{{ item.name }}</div>
 *   </template>
 * </SortableList>
 * ```
 */

const props = withDefaults(defineProps<{
  /** The list of items to render and sort */
  items: any[]
  /** Field name used to extract IDs from items */
  idField?: string
  /** Entity name for the reorder API (e.g., 'plans', 'nodes') */
  entity?: string
  /** API endpoint to persist the new order */
  persistEndpoint?: string
  /** Animation duration in ms */
  animation?: number
  /** Whether to show the built-in drag handle. Set false to use a custom handle. */
  showHandle?: boolean
  /** Additional CSS class for the container */
  containerClass?: string
}>(), {
  idField: 'id',
  entity: '',
  persistEndpoint: '/api/admin/reorder',
  animation: 150,
  showHandle: true,
  containerClass: '',
})

const emit = defineEmits<{
  reorder: [order: (string | number)[]]
}>()

// We need a mutable ref that useSortable can write to.
// The parent owns the items array, so we use a computed that syncs both ways.
import { ref, watch } from 'vue'

const localItems = ref([...props.items]) as Ref<any[]>

watch(() => props.items, (newItems) => {
  localItems.value = [...newItems]
}, { deep: true })

watch(localItems, (newItems) => {
  const order = newItems.map((item: any) => item[props.idField])
  emit('reorder', order)
}, { deep: true })

const { containerRef, isDragging } = useSortable(localItems, {
  handle: props.showHandle ? '.sortable-handle' : undefined,
  animation: props.animation,
  persistEndpoint: props.persistEndpoint,
  entity: props.entity,
  idField: props.idField,
})
</script>

<template>
  <div
    ref="containerRef"
    class="sortable-list"
    :class="[containerClass, { 'sortable-list--dragging': isDragging }]"
  >
    <div
      v-for="(item, index) in localItems"
      :key="item[idField]"
      class="sortable-list__item"
    >
      <button
        v-if="showHandle"
        class="sortable-handle"
        type="button"
        :aria-label="'Drag to reorder'"
        tabindex="0"
      >
        <span class="sortable-handle__icon" aria-hidden="true">⋮⋮</span>
      </button>
      <div class="sortable-list__content">
        <slot name="item" :item="item" :index="index" />
      </div>
    </div>
  </div>
</template>

<style scoped>
.sortable-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.sortable-list--dragging {
  cursor: grabbing;
}

.sortable-list__item {
  display: flex;
  align-items: stretch;
  gap: var(--space-2, 8px);
  border-radius: var(--radius-md, 6px);
  transition: box-shadow 0.15s ease;
}

.sortable-list__content {
  flex: 1;
  min-width: 0;
}

.sortable-handle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  min-height: 100%;
  padding: 0;
  border: none;
  background: transparent;
  cursor: grab;
  color: var(--color-muted, #9ca3af);
  border-radius: var(--radius-sm, 4px);
  transition: color 0.15s ease, background 0.15s ease;
  flex-shrink: 0;
}

.sortable-handle:hover {
  color: var(--color-text, #111);
  background: var(--color-surface-hover, rgba(0, 0, 0, 0.05));
}

.sortable-handle:active {
  cursor: grabbing;
}

.sortable-handle__icon {
  font-size: 14px;
  line-height: 1;
  user-select: none;
  letter-spacing: -1px;
}

/* Ghost element (placeholder at the drop position) */
:deep(.sortable-ghost) {
  opacity: 0.4;
  background: var(--color-primary-light, rgba(59, 130, 246, 0.08));
  border-radius: var(--radius-md, 6px);
}

/* The chosen item being dragged */
:deep(.sortable-chosen) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
  border-radius: var(--radius-md, 6px);
}

/* The drag overlay */
:deep(.sortable-drag) {
  opacity: 0.9;
}
</style>
