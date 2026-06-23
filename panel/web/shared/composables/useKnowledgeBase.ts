import { ref, type Ref } from 'vue'
import { useApi } from './useApi'

export interface KBArticle {
  id: number
  title: string
  body: string
  category: string
  status: string
  locale: string
  parent_id: number | null
  view_count: number
  created_at: string
  updated_at: string
}

export interface CreateArticleData {
  title: string
  body: string
  category?: string
  status?: string
  locale?: string
  parent_id?: number
}

export interface UseKnowledgeBaseReturn {
  loading: Ref<boolean>
  error: Ref<string>
  data: Ref<KBArticle[]>
  listArticles: (category?: string) => Promise<KBArticle[]>
  createArticle: (data: CreateArticleData) => Promise<KBArticle>
  updateArticle: (id: number, data: Partial<CreateArticleData>) => Promise<KBArticle>
  deleteArticle: (id: number) => Promise<void>
  searchArticles: (query: string) => Promise<KBArticle[]>
  getArticle: (id: number) => Promise<KBArticle>
}

export function useKnowledgeBase(): UseKnowledgeBaseReturn {
  const { get, post, patch, del, loading, error } = useApi()
  const data = ref<KBArticle[]>([]) as Ref<KBArticle[]>

  async function listArticles(category?: string): Promise<KBArticle[]> {
    const query = category ? `?category=${encodeURIComponent(category)}` : ''
    const result = await get<{ ok: boolean; articles: KBArticle[] }>(`/api/kb/articles${query}`)
    data.value = result.articles
    return result.articles
  }

  async function createArticle(articleData: CreateArticleData): Promise<KBArticle> {
    const result = await post<{ ok: boolean; article: KBArticle }>('/api/kb/articles', articleData)
    return result.article
  }

  async function updateArticle(id: number, articleData: Partial<CreateArticleData>): Promise<KBArticle> {
    const result = await patch<{ ok: boolean; article: KBArticle }>(`/api/kb/articles/${id}`, articleData)
    return result.article
  }

  async function deleteArticle(id: number): Promise<void> {
    await del(`/api/kb/articles/${id}`)
  }

  async function searchArticles(query: string): Promise<KBArticle[]> {
    const result = await get<{ ok: boolean; articles: KBArticle[] }>(`/api/portal/kb/search?q=${encodeURIComponent(query)}`)
    return result.articles
  }

  async function getArticle(id: number): Promise<KBArticle> {
    const result = await get<{ ok: boolean; article: KBArticle }>(`/api/portal/kb/${id}`)
    return result.article
  }

  return {
    loading,
    error,
    data,
    listArticles,
    createArticle,
    updateArticle,
    deleteArticle,
    searchArticles,
    getArticle,
  }
}
