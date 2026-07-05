import { ref, type Ref } from 'vue'

export interface UseClipboardReturn {
  /** Copies the provided text to the system clipboard. */
  copy(text: string): Promise<void>
  /** Reactive flag that is `true` briefly after a successful copy for UI feedback. */
  copied: Ref<boolean>
}

/** Duration in milliseconds to keep `copied` as `true` after a successful copy. */
const COPIED_DURATION = 2000

/**
 * useClipboard composable
 *
 * Provides a simple interface for copying text to the system clipboard
 * with a reactive `copied` flag for UI feedback (e.g. showing a checkmark).
 *
 * Requirements: 17.1, 17.2
 */
export function useClipboard(): UseClipboardReturn {
  const copied = ref(false)
  let timeoutId: ReturnType<typeof setTimeout> | null = null

  /**
   * Writes the provided text to the system clipboard via navigator.clipboard.writeText.
   * On success, sets `copied.value = true` for approximately 2 seconds.
   */
  async function copy(text: string): Promise<void> {
    try {
      await navigator.clipboard.writeText(text)

      // Clear any existing timeout to prevent premature reset if copy is called rapidly
      if (timeoutId !== null) {
        clearTimeout(timeoutId)
      }

      copied.value = true

      timeoutId = setTimeout(() => {
        copied.value = false
        timeoutId = null
      }, COPIED_DURATION)
    } catch {
      // If clipboard write fails (e.g. permissions denied), do not set copied
      copied.value = false
    }
  }

  return {
    copy,
    copied,
  }
}
