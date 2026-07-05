import { ref, computed, watch, onMounted, type Ref, type ComputedRef } from 'vue'
import { useRouter, useRoute } from 'vue-router'

/**
 * Composable that manages the detail panel open/close state with URL sync.
 *
 * - `selectedUserId`: The currently selected user ID (null when panel is closed)
 * - `isOpen`: Computed boolean, true when a user is selected
 * - `open(userId)`: Opens the panel for the given user and updates the URL query param
 * - `close()`: Closes the panel and removes the URL query param
 * - `switchUser(userId)`: Swaps the panel content to a different user without close/open animation
 *
 * The composable syncs the `selected` query parameter in the URL without triggering
 * a full page reload (uses `router.replace`).
 *
 * Requirements: 2.1, 2.8, 2.9
 */
export function useDetailPanel() {
  const router = useRouter()
  const route = useRoute()

  const selectedUserId: Ref<number | null> = ref(null)
  const isOpen: ComputedRef<boolean> = computed(() => selectedUserId.value !== null)

  /**
   * Opens the detail panel for the given user and updates the URL.
   */
  function open(userId: number): void {
    selectedUserId.value = userId
    syncToUrl(userId)
  }

  /**
   * Closes the detail panel and removes the `selected` query param from the URL.
   */
  function close(): void {
    selectedUserId.value = null
    syncToUrl(null)
  }

  /**
   * Switches the panel content to a different user without triggering a close/open cycle.
   * This updates the URL and the selected user in one step.
   */
  function switchUser(userId: number): void {
    selectedUserId.value = userId
    syncToUrl(userId)
  }

  /**
   * Syncs the selected user ID to the URL query parameter `?selected=:id`.
   * Uses `router.replace` to avoid full page reload and history pollution.
   */
  function syncToUrl(userId: number | null): void {
    const query = { ...route.query }

    if (userId !== null) {
      query.selected = String(userId)
    } else {
      delete query.selected
    }

    // Fire and forget — router.replace is async but we don't need to await
    // in the UI flow since the ref is already updated synchronously
    router.replace({ query })
  }

  // Watch for external route changes (e.g. browser back/forward) and sync state
  watch(() => route.query.selected, (newVal) => {
    if (newVal === undefined || newVal === null) {
      if (selectedUserId.value !== null) {
        selectedUserId.value = null
      }
    } else {
      const parsed = Number(newVal)
      if (!isNaN(parsed) && parsed > 0 && parsed !== selectedUserId.value) {
        selectedUserId.value = parsed
      }
    }
  })

  // On mount, restore state from URL if `?selected=:id` is present
  onMounted(() => {
    const selectedParam = route.query.selected
    if (selectedParam) {
      const parsed = Number(selectedParam)
      if (!isNaN(parsed) && parsed > 0) {
        selectedUserId.value = parsed
      }
    }
  })

  return {
    selectedUserId,
    isOpen,
    open,
    close,
    switchUser,
  }
}
