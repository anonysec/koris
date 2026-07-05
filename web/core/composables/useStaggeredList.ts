import { ref, onMounted, nextTick, type Ref } from 'vue'

/**
 * useStaggeredList composable
 * 
 * Applies staggered animation delays to child elements of a container.
 * Each child gets a progressively increasing animation-delay so items
 * appear to "cascade" into view.
 * 
 * @param containerRef - Ref to the container element
 * @param options - Configuration options
 * @returns Object with trigger function and ready state
 * 
 * Usage:
 * ```vue
 * <div ref="listRef">
 *   <div v-for="item in items" class="stagger-item">{{ item }}</div>
 * </div>
 * 
 * <script setup>
 * const listRef = ref<HTMLElement>()
 * const { trigger } = useStaggeredList(listRef, { delay: 30 })
 * </script>
 * ```
 */
export interface StaggerOptions {
  /** Delay in ms between each child element (default: 30) */
  delay?: number
  /** CSS selector for child elements to animate (default: '.stagger-item') */
  selector?: string
  /** Whether to trigger automatically on mount (default: true) */
  autoTrigger?: boolean
}

export function useStaggeredList(
  containerRef: Ref<HTMLElement | undefined>,
  options: StaggerOptions = {}
) {
  const {
    delay = 30,
    selector = '.stagger-item',
    autoTrigger = true,
  } = options

  const ready = ref(false)

  function trigger() {
    if (!containerRef.value) return

    const children = containerRef.value.querySelectorAll(selector)
    children.forEach((child, index) => {
      const el = child as HTMLElement
      el.style.animationDelay = `${index * delay}ms`
    })
    ready.value = true
  }

  function reset() {
    if (!containerRef.value) return

    const children = containerRef.value.querySelectorAll(selector)
    children.forEach((child) => {
      const el = child as HTMLElement
      el.style.animationDelay = ''
      el.classList.remove('stagger-item')
    })
    ready.value = false
  }

  if (autoTrigger) {
    onMounted(async () => {
      await nextTick()
      trigger()
    })
  }

  return {
    trigger,
    reset,
    ready,
  }
}
