import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface CannedResponse {
  id: number
  title: string
  body: string
  category: string
  usage_count: number
  created_at: string
  updated_at: string
}

export interface CreateResponseData {
  title: string
  body: string
  category?: string
}

export interface UseCannedResponsesReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<CannedResponse[]>
  listResponses: (category?: string) => Promise<CannedResponse[]>
  createResponse: (data: CreateResponseData) => Promise<CannedResponse>
  updateResponse: (id: number, data: Partial<CreateResponseData>) => Promise<CannedResponse>
  deleteResponse: (id: number) => Promise<void>
  useResponse: (id: number) => Promise<CannedResponse>
  previewResponse: (body: string, vars: Record<string, string>) => Promise<string>
}

export function useCannedResponses(): UseCannedResponsesReturn {
  const { get, post, patch, del, loading, error } = useApi()
  const data = ref<CannedResponse[]>([]) as Ref<CannedResponse[]>

  async function listResponses(category?: string): Promise<CannedResponse[]> {
    const query = category ? `?category=${encodeURIComponent(category)}` : ''
    const result = await get<{ ok: boolean; responses: CannedResponse[] }>(`/api/canned-responses${query}`)
    data.value = result.responses
    return result.responses
  }

  async function createResponse(responseData: CreateResponseData): Promise<CannedResponse> {
    const result = await post<{ ok: boolean; response: CannedResponse }>('/api/canned-responses', responseData)
    return result.response
  }

  async function updateResponse(id: number, responseData: Partial<CreateResponseData>): Promise<CannedResponse> {
    const result = await patch<{ ok: boolean; response: CannedResponse }>(`/api/canned-responses/${id}`, responseData)
    return result.response
  }

  async function deleteResponse(id: number): Promise<void> {
    await del(`/api/canned-responses/${id}`)
  }

  async function useResponse(id: number): Promise<CannedResponse> {
    const result = await post<{ ok: boolean; response: CannedResponse }>(`/api/canned-responses/${id}/use`)
    return result.response
  }

  async function previewResponse(body: string, vars: Record<string, string>): Promise<string> {
    const result = await post<{ ok: boolean; preview: string }>('/api/canned-responses/preview', { body, vars })
    return result.preview
  }

  return {
    loading,
    error,
    data,
    listResponses,
    createResponse,
    updateResponse,
    deleteResponse,
    useResponse,
    previewResponse,
  }
}
