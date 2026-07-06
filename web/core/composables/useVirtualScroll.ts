import { ref, computed, type Ref, type ComputedRef } from 'vue'

/**
 * Options for configuring the useVirtualScroll composable
 */
export interface UseVirtualScrollOptions {
  /** Total number of items in the dataset */
  totalItems: Ref<number> | ComputedRef<number>
  /** Fixed row height in pixels (default: 44) */
  rowHeight: number
  /** Viewport container height in pixels */
  containerHeight: Ref<number>
  /** Number of rows above/below the visible area to render as buffer (default: 5) */
  bufferSize?: number
}

/**
 * Return type of the useVirtualScroll composable
 */
export interface UseVirtualScrollReturn {
  /** First index in the rendered slice (inclusive) */
  startIndex: ComputedRef<number>
  /** Last index in the rendered slice (inclusive) */
  endIndex: ComputedRef<number>
  /** CSS translateY offset in pixels for positioning the visible slice */
  offsetY: ComputedRef<number>
  /** Total scrollable height in pixels (totalItems * rowHeight) */
  totalHeight: ComputedRef<number>
  /** Number of items visible in the viewport */
  visibleItems: ComputedRef<number>
  /** Current scroll position from the top */
  scrollTop: Ref<number>
  /** Scroll event handler to attach to the scroll container */
  onScroll(event: Event): void
}

/**
 * Composable providing virtual scroll calculation logic for DataTable.
 *
 * Only renders visible rows plus a configurable buffer above and below the viewport,
 * enabling smooth scrolling for datasets with 1000+ rows while maintaining
 * a minimal DOM footprint.
 *
 * @param options - Configuration for the virtual scroll engine
 * @returns Reactive computed values for rendering the visible slice
 *
 * @example
 * ```ts
 * const containerHeight = ref(600)
 * const totalItems = computed(() => data.value.length)
 *
 * const {
 *   startIndex,
 *   endIndex,
 *   offsetY,
 *   totalHeight,
 *   onScroll
 * } = useVirtualScroll({
 *   totalItems,
 *   rowHeight: 44,
 *   containerHeight,
 *   bufferSize: 5
 * })
 *
 * // Slice the data for rendering
 * const visibleData = computed(() =>
 *   data.value.slice(startIndex.value, endIndex.value + 1)
 * )
 * ```
 */
export function useVirtualScroll(options: UseVirtualScrollOptions): UseVirtualScrollReturn {
  const { totalItems, rowHeight, containerHeight, bufferSize = 5 } = options

  const scrollTop: Ref<number> = ref(0)

  /**
   * Number of items that fit in the visible viewport.
   */
  const visibleItems = computed(() => {
    if (containerHeight.value <= 0) return 0
    return Math.ceil(containerHeight.value / rowHeight)
  })

  /**
   * First index of the rendered slice (inclusive).
   * Includes buffer rows above the viewport for smooth scrolling.
   *
   * Algorithm: max(0, floor(scrollTop / rowHeight) - bufferSize)
   */
  const startIndex = computed(() => {
    const rawStart = Math.floor(scrollTop.value / rowHeight)
    return Math.max(0, rawStart - bufferSize)
  })

  /**
   * Last index of the rendered slice (inclusive).
   * Includes buffer rows below the viewport for smooth scrolling.
   *
   * Algorithm: min(totalItems - 1, floor(scrollTop / rowHeight) + visibleItems + bufferSize)
   */
  const endIndex = computed(() => {
    const total = totalItems.value
    if (total <= 0) return 0
    const rawStart = Math.floor(scrollTop.value / rowHeight)
    return Math.min(total - 1, rawStart + visibleItems.value + bufferSize)
  })

  /**
   * CSS translateY offset to position the visible slice correctly
   * within the scrollable container.
   *
   * Algorithm: startIndex * rowHeight
   */
  const offsetY = computed(() => {
    return startIndex.value * rowHeight
  })

  /**
   * Total scrollable height for the scroll container sizing.
   * This creates the correct scrollbar proportions.
   *
   * Algorithm: totalItems * rowHeight
   */
  const totalHeight = computed(() => {
    return totalItems.value * rowHeight
  })

  /**
   * Scroll event handler to be attached to the scroll container element.
   * Updates the scrollTop ref which triggers recalculation of all computed values.
   */
  function onScroll(event: Event): void {
    const target = event.target as HTMLElement
    if (target) {
      scrollTop.value = target.scrollTop
    }
  }

  return {
    startIndex,
    endIndex,
    offsetY,
    totalHeight,
    visibleItems,
    scrollTop,
    onScroll,
  }
}
