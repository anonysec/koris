import { ref, type Ref } from 'vue'
import type { ConfirmOptions } from '@koris/types/components'

/** Whether the confirm dialog is currently visible */
const isOpen: Ref<boolean> = ref(false)

/** Current confirm dialog options */
const currentOptions: Ref<ConfirmOptions> = ref({
  title: '',
  message: '',
})

/** Stored resolve function for the active promise */
let resolvePromise: ((value: boolean) => void) | null = null

/**
 * Return type of the useConfirm composable
 */
export interface UseConfirmReturn {
  /** Show a confirm dialog and await user response */
  confirm(options: ConfirmOptions): Promise<boolean>
  /** Whether the dialog is currently open */
  isOpen: Ref<boolean>
  /** Current dialog options (read by KConfirmDialog) */
  options: Ref<ConfirmOptions>
  /** Called by KConfirmDialog when user confirms */
  handleConfirm(): void
  /** Called by KConfirmDialog when user cancels */
  handleCancel(): void
}

/**
 * Composable for programmatic confirmation dialogs.
 *
 * Uses a singleton pattern with module-level refs so that a single
 * KConfirmDialog component can serve the entire application.
 *
 * @example
 * ```ts
 * const { confirm } = useConfirm()
 *
 * const confirmed = await confirm({
 *   title: 'Delete Customer',
 *   message: 'Are you sure you want to delete this customer?',
 *   variant: 'danger',
 *   confirmText: 'Delete',
 * })
 *
 * if (confirmed) {
 *   // proceed with deletion
 * }
 * ```
 */
export function useConfirm(): UseConfirmReturn {
  /**
   * Display a confirmation dialog and return a Promise.
   * Resolves true when user confirms, false when user cancels/escapes.
   */
  function confirm(options: ConfirmOptions): Promise<boolean> {
    return new Promise<boolean>((resolve) => {
      currentOptions.value = {
        confirmText: 'Confirm',
        cancelText: 'Cancel',
        variant: 'info',
        ...options,
      }
      resolvePromise = resolve
      isOpen.value = true
    })
  }

  /**
   * Handle confirm action - resolves promise with true and closes dialog.
   */
  function handleConfirm(): void {
    if (resolvePromise) {
      resolvePromise(true)
      resolvePromise = null
    }
    isOpen.value = false
  }

  /**
   * Handle cancel action - resolves promise with false and closes dialog.
   */
  function handleCancel(): void {
    if (resolvePromise) {
      resolvePromise(false)
      resolvePromise = null
    }
    isOpen.value = false
  }

  return {
    confirm,
    isOpen,
    options: currentOptions,
    handleConfirm,
    handleCancel,
  }
}
