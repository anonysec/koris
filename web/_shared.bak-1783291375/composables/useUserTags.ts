import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface UserTag {
  id: number
  name: string
  color: string
  created_at: string
}

export interface UseUserTagsReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<UserTag[]>
  listTags: () => Promise<UserTag[]>
  createTag: (name: string, color: string) => Promise<UserTag>
  deleteTag: (id: number) => Promise<void>
  assignTags: (customerId: number, tagIds: number[]) => Promise<void>
  removeTag: (customerId: number, tagId: number) => Promise<void>
}

export function useUserTags(): UseUserTagsReturn {
  const { get, post, del, loading, error } = useApi()
  const data = ref<UserTag[]>([]) as Ref<UserTag[]>

  async function listTags(): Promise<UserTag[]> {
    const result = await get<{ ok: boolean; tags: UserTag[] }>('/api/tags')
    data.value = result.tags
    return result.tags
  }

  async function createTag(name: string, color: string): Promise<UserTag> {
    const result = await post<{ ok: boolean; tag: UserTag }>('/api/tags', { name, color })
    return result.tag
  }

  async function deleteTag(id: number): Promise<void> {
    await del(`/api/tags/${id}`)
  }

  async function assignTags(customerId: number, tagIds: number[]): Promise<void> {
    await post(`/api/customers/${customerId}/tags`, { tag_ids: tagIds })
  }

  async function removeTag(customerId: number, tagId: number): Promise<void> {
    await del(`/api/customers/${customerId}/tags/${tagId}`)
  }

  return {
    loading,
    error,
    data,
    listTags,
    createTag,
    deleteTag,
    assignTags,
    removeTag,
  }
}
