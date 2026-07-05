import { ref, onMounted, onBeforeUnmount, watch, type Ref } from 'vue'
import Sortable from 'sortablejs'
import { useApi } from './useApi'

/**
 * Options for the useSortable composable.
 */
export interface SortableOptions {
  /** CSS selector for the drag handle element (e.g., '.drag-handle'). If omitted, entire item is draggable. */
  handle?: string
  /** Animation duration in ms for reorder transitions. Default: 150 */
  animation?: number
  /** API endpoint to persist the new order. POST body: { entity, order: [...ids] } */
  persistEndpoint?: string
  /** Entity name sent to the persist endpoint (e.g., 'plans', 'nodes'). */
  entity?: string
  /** Property name used to extract IDs from items. Default: 'id' */
  idField?: string
  /** Ghost class applied to the ghost element during drag. Default: 'sortable-ghost' */
  ghostClass?: string
  /** Chosen class applied to the item being dragged. Default: 'sortable-chosen' */
  chosenClass?: string
  /** Drag class applied during drag. Default: 'sortable-drag' */
  dragClass?: string
}

/**
 * Return type of the useSortable composable.
 */
export interface UseSortableReturn<T> {
  /** The container ref to bind to a DOM element */
  containerRef: Ref<HTMLElement | null>
  /** Whether a drag is currently in progress */
  isDragging: Ref<boolean>
  /** Callback to manually trigger persist (usually called automatically on drag end) */
  persist: () => Promise<void>
}

/**
 * Composable that wraps sortablejs for drag-and-drop reordering of lists.
 *
 * @param items - Reactive ref to the array of items to sort. Mutated in-place on reorder.
 * @param options - Configuration options for drag behavior and persistence.
 * @returns Object with containerRef (bind to your list element), isDragging state, and persist function.
 *
 * @example
 * ```ts
 * const plans = ref<Plan[]>([])
 * const { containerRef, isDragging } = useSortable(plans, {
 *   handle: '.drag-handle',
 *   animation: 150,
 *   persistEndpoint: '/api/admin/reorder',
 *   entity: 'plans',
 *   idField: 'id',
 * })
 * ```
 */
export function useSortable<T extends Record<string, any>>(
  items: Ref<T[]>,
  options: SortableOptions = {}
): UseSortableReturn<T> {
  const {
    handle,
    animation = 150,
    persistEndpoint = '/api/admin/reorder',
    entity = '',
    idField = 'id',
    ghostClass = 'sortable-ghost',
    chosenClass = 'sortable-chosen',
    dragClass = 'sortable-drag',
  } = options

  const containerRef = ref<HTMLElement | null>(null)
  const isDragging = ref(false)
  const { post } = useApi({ showErrorToast: true })

  let sortableInstance: Sortable | null = null

  function initSortable() {
    if (!containerRef.value) return
    if (sortableInstance) {
      sortableInstance.destroy()
      sortableInstance = null
    }

    sortableInstance = Sortable.create(containerRef.value, {
      handle: handle || undefined,
      animation,
      ghostClass,
      chosenClass,
      dragClass,
      onStart() {
        isDragging.value = true
      },
      onEnd(evt) {
        isDragging.value = false
        const { oldIndex, newIndex } = evt
        if (oldIndex == null || newIndex == null || oldIndex === newIndex) return

        // Reorder items array in-place
        const arr = [...items.value]
        const [moved] = arr.splice(oldIndex, 1)
        arr.splice(newIndex, 0, moved)
        items.value = arr

        // Auto-persist
        persist()
      },
    })
  }

  async function persist(): Promise<void> {
    if (!entity || !persistEndpoint) return
    const order = items.value.map((item) => item[idField])
    try {
      await post(persistEndpoint, { entity, order })
    } catch {
      // Error toast is shown automatically by useApi
    }
  }

  onMounted(() => {
    initSortable()
  })

  // Re-init if the container ref changes (e.g., conditional rendering)
  watch(containerRef, (el) => {
    if (el) initSortable()
  })

  onBeforeUnmount(() => {
    if (sortableInstance) {
      sortableInstance.destroy()
      sortableInstance = null
    }
  })

  return {
    containerRef,
    isDragging,
    persist,
  }
}
