import { ref, watch, type Ref } from 'vue'
import { useApi } from '@koris/composables/useApi'
import type { ConnectedClient } from '@koris/types/entities'

interface ConnectionsResponse {
  ok: boolean
  connections: ConnectedClient[]
}

export function useConnectedClients(userId: Ref<number | null>) {
  const { get } = useApi({ showErrorToast: false })
  const clients = ref<ConnectedClient[]>([])
  const loading = ref(false)

  async function refresh(): Promise<void> {
    if (!userId.value) {
      clients.value = []
      return
    }

    loading.value = true
    try {
      const res = await get<ConnectionsResponse>(`/api/customers/${userId.value}/connections`)
      clients.value = res.connections || []
    } catch {
      clients.value = []
    } finally {
      loading.value = false
    }
  }

  watch(userId, () => {
    refresh()
  }, { immediate: true })

  return {
    clients,
    loading,
    refresh,
  }
}
