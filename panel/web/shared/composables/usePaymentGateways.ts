import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface PaymentGateway {
  id: number
  name: string
  display_name: string
  config_json: string
  is_active: boolean
  created_at: string
}

export interface CreateGatewayData {
  name: string
  display_name: string
  config_json: string
  is_active?: boolean
}

export interface UsePaymentGatewaysReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<PaymentGateway[]>
  listGateways: () => Promise<PaymentGateway[]>
  createGateway: (data: CreateGatewayData) => Promise<PaymentGateway>
  updateGateway: (id: number, data: Partial<CreateGatewayData>) => Promise<PaymentGateway>
  deleteGateway: (id: number) => Promise<void>
}

export function usePaymentGateways(): UsePaymentGatewaysReturn {
  const { get, post, patch, del, loading, error } = useApi()
  const data = ref<PaymentGateway[]>([]) as Ref<PaymentGateway[]>

  async function listGateways(): Promise<PaymentGateway[]> {
    const result = await get<{ ok: boolean; gateways: PaymentGateway[] }>('/api/gateways')
    data.value = result.gateways
    return result.gateways
  }

  async function createGateway(gatewayData: CreateGatewayData): Promise<PaymentGateway> {
    const result = await post<{ ok: boolean; gateway: PaymentGateway }>('/api/gateways', gatewayData)
    return result.gateway
  }

  async function updateGateway(id: number, gatewayData: Partial<CreateGatewayData>): Promise<PaymentGateway> {
    const result = await patch<{ ok: boolean; gateway: PaymentGateway }>(`/api/gateways/${id}`, gatewayData)
    return result.gateway
  }

  async function deleteGateway(id: number): Promise<void> {
    await del(`/api/gateways/${id}`)
  }

  return {
    loading,
    error,
    data,
    listGateways,
    createGateway,
    updateGateway,
    deleteGateway,
  }
}
