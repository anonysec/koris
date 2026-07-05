import { ref, type Ref } from 'vue'

/**
 * Composable that tracks which table rows are currently expanded.
 *
 * Allows multiple rows to be expanded simultaneously. Provides toggle,
 * query, and collapse-all functionality for expandable row UIs.
 *
 * Usage:
 *   const { expandedIds, toggle, isExpanded, collapseAll } = useExpandableRows()
 *
 *   // In template: @click="toggle(user.id)"
 *   // In template: v-if="isExpanded(user.id)"
 *
 * Requirements: 6.2, 6.4, 6.6
 */
export function useExpandableRows() {
  const expandedIds: Ref<Set<number>> = ref(new Set())

  /**
   * Toggle the expanded state of a row by its ID.
   * If the row is currently expanded, it will be collapsed, and vice versa.
   */
  function toggle(id: number): void {
    const updated = new Set(expandedIds.value)
    if (updated.has(id)) {
      updated.delete(id)
    } else {
      updated.add(id)
    }
    expandedIds.value = updated
  }

  /**
   * Check whether a row is currently expanded.
   */
  function isExpanded(id: number): boolean {
    return expandedIds.value.has(id)
  }

  /**
   * Collapse all currently expanded rows.
   */
  function collapseAll(): void {
    expandedIds.value = new Set()
  }

  return {
    expandedIds,
    toggle,
    isExpanded,
    collapseAll,
  }
}
