<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useFreshData } from '@koris/composables/useFreshData'
import { formatDate } from '@koris/composables/useFormatDate'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'

interface KBArticle {
  id: number
  title: string
  body: string
  category: string
  view_count: number
  created_at: string
  updated_at: string
}

interface ArticlesResponse {
  ok: boolean
  articles: KBArticle[]
  categories?: string[]
}

interface ArticleDetailResponse {
  ok: boolean
  article: KBArticle
}

const { get, loading } = useApi()
const { t } = useI18n()

const articles = ref<KBArticle[]>([])
const categories = ref<string[]>([])
const activeCategory = ref('')
const searchQuery = ref('')
const searchResults = ref<KBArticle[]>([])
const isSearching = ref(false)
const selectedArticle = ref<KBArticle | null>(null)

// Debounce search
let searchTimeout: ReturnType<typeof setTimeout> | null = null

useFreshData(async () => {
  await fetchArticles()
})

async function fetchArticles(category?: string) {
  try {
    const query = category ? `?category=${encodeURIComponent(category)}` : ''
    const res = await get<ArticlesResponse>(`/api/portal/kb${query}`)
    articles.value = res.articles || []
    if (res.categories) {
      categories.value = res.categories
    } else {
      // Extract unique categories from articles
      const cats = new Set(articles.value.map(a => a.category))
      categories.value = Array.from(cats)
    }
  } catch {
    // keep empty state
  }
}

async function handleSearch() {
  if (!searchQuery.value.trim()) {
    searchResults.value = []
    isSearching.value = false
    return
  }
  isSearching.value = true
  try {
    const res = await get<ArticlesResponse>(`/api/portal/kb/search?q=${encodeURIComponent(searchQuery.value)}`)
    searchResults.value = res.articles || []
  } catch {
    searchResults.value = []
  }
}

watch(searchQuery, (val) => {
  if (searchTimeout) clearTimeout(searchTimeout)
  if (!val.trim()) {
    searchResults.value = []
    isSearching.value = false
    return
  }
  searchTimeout = setTimeout(handleSearch, 400)
})

function selectCategory(category: string) {
  activeCategory.value = category
  searchQuery.value = ''
  isSearching.value = false
  fetchArticles(category || undefined)
}

async function viewArticle(article: KBArticle) {
  try {
    const res = await get<ArticleDetailResponse>(`/api/portal/kb/${article.id}`)
    selectedArticle.value = res.article
  } catch {
    selectedArticle.value = article
  }
}

function closeArticle() {
  selectedArticle.value = null
}

const displayedArticles = computed(() => {
  if (isSearching.value) return searchResults.value
  return articles.value
})

function escapeHtml(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
    .replace(/"/g, '&quot;')
    .replace(/\u0027/g, '&#39;')
}

function renderMarkdown(body: string): string {
  // XSS protection: escape HTML entities BEFORE applying markdown transforms
  let html = escapeHtml(body)
    // Code blocks
    .replace(/```([\s\S]*?)```/g, '<pre><code>$1</code></pre>')
    // Inline code
    .replace(/`([^`]+)`/g, '<code>$1</code>')
    // Headings
    .replace(/^### (.+)$/gm, '<h4>$1</h4>')
    .replace(/^## (.+)$/gm, '<h3>$1</h3>')
    .replace(/^# (.+)$/gm, '<h2>$1</h2>')
    // Bold
    .replace(/\*\*(.+?)\*\*/g, '<strong>$1</strong>')
    // Italic
    .replace(/\*(.+?)\*/g, '<em>$1</em>')
    // Links — only allow http/https URLs
    .replace(/\[([^\]]+)\]\((https?:\/\/[^)]+)\)/g, '<a href="$2" target="_blank" rel="noopener noreferrer">$1</a>')
    // Line breaks (double newline = paragraph)
    .replace(/\n\n/g, '</p><p>')
    // Single newline = br
    .replace(/\n/g, '<br>')

  return `<p>${html}</p>`
}
</script>
<template>
  <div class="kb">
    <template v-if="!selectedArticle">
      <h1 class="kb__title">Knowledge Base</h1>

      <!-- Search -->
      <div class="kb__search">
        <Input
          v-model="searchQuery"
          placeholder="Search articles..."
          type="search"
        />
      </div>

      <!-- Category Tabs -->
      <div v-if="categories.length && !isSearching" class="kb__categories">
        <button
          class="kb__category-tab"
          :class="{ 'kb__category-tab--active': !activeCategory }"
          @click="selectCategory('')"
        >
          All
        </button>
        <button
          v-for="cat in categories"
          :key="cat"
          class="kb__category-tab"
          :class="{ 'kb__category-tab--active': activeCategory === cat }"
          @click="selectCategory(cat)"
        >
          {{ cat }}
        </button>
      </div>

      <Skeleton v-if="loading && !articles.length" type="card" :count="3" />

      <template v-else>
        <!-- Search Results Header -->
        <div v-if="isSearching" class="kb__search-info">
          <span class="kb__search-count">{{ searchResults.length }} result{{ searchResults.length !== 1 ? 's' : '' }}</span>
          <Button variant="ghost" size="sm" @click="searchQuery = ''">Clear search</Button>
        </div>

        <!-- Articles List -->
        <div v-if="displayedArticles.length" class="kb__articles-list">
          <div
            v-for="article in displayedArticles"
            :key="article.id"
            class="kb__article-card"
            @click="viewArticle(article)"
          >
            <div class="kb__article-header">
              <h3 class="kb__article-title">{{ article.title }}</h3>
              <span class="kb__article-category">{{ article.category }}</span>
            </div>
            <p class="kb__article-excerpt">
              {{ article.body.substring(0, 150).replace(/[#*`\[\]]/g, '') }}{{ article.body.length > 150 ? '...' : '' }}
            </p>
            <div class="kb__article-meta">
              <span>👁️ {{ article.view_count }}</span>
              <span>{{ formatDate(article.updated_at) }}</span>
            </div>
          </div>
        </div>

        <EmptyState
          v-else
          :title="isSearching ? 'No results found' : 'No articles yet'"
          :description="isSearching ? 'Try a different search term.' : 'Knowledge base articles will appear here.'"
          icon="📚"
        />
      </template>
    </template>

    <!-- Article Detail View -->
    <template v-else>
      <div class="kb__detail">
        <Button variant="ghost" size="sm" @click="closeArticle" class="kb__back-btn">
          ← Back
        </Button>

        <article class="kb__article-detail">
          <header class="kb__article-detail-header">
            <h1 class="kb__article-detail-title">{{ selectedArticle.title }}</h1>
            <div class="kb__article-detail-meta">
              <span class="kb__article-detail-category">{{ selectedArticle.category }}</span>
              <span>{{ formatDate(selectedArticle.updated_at) }}</span>
              <span>👁️ {{ selectedArticle.view_count }}</span>
            </div>
          </header>

          <div class="kb__article-detail-body" v-text="selectedArticle.body" />
        </article>
      </div>
    </template>
  </div>
</template>
<style scoped>
.kb {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
  padding-bottom: calc(var(--space-8) + env(safe-area-inset-bottom, 20px));
}
.kb__title {
  font-size: var(--text-xl);
  font-weight: 700;
}
.kb__search {
  max-width: 480px;
}
.kb__categories {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.kb__category-tab {
  padding: var(--space-2) var(--space-3);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-text);
  cursor: pointer;
  transition: all 0.15s;
}
.kb__category-tab:hover {
  border-color: var(--color-primary);
}
.kb__category-tab--active {
  background: var(--color-primary);
  color: #fff;
  border-color: var(--color-primary);
}
.kb__search-info {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}
.kb__search-count {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
.kb__articles-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.kb__article-card {
  padding: var(--space-4) var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  cursor: pointer;
  transition: border-color 0.15s, box-shadow 0.15s;
}
.kb__article-card:hover {
  border-color: var(--color-primary);
  box-shadow: 0 2px 8px rgba(0, 0, 0, 0.04);
}
.kb__article-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  margin-bottom: var(--space-2);
}
.kb__article-title {
  font-size: var(--text-sm);
  font-weight: 600;
}
.kb__article-category {
  font-size: var(--text-xs);
  color: var(--color-muted);
  background: var(--color-bg);
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  white-space: nowrap;
}
.kb__article-excerpt {
  font-size: var(--text-sm);
  color: var(--color-muted);
  line-height: 1.5;
  margin-bottom: var(--space-3);
}
.kb__article-meta {
  display: flex;
  gap: var(--space-4);
  font-size: var(--text-xs);
  color: var(--color-muted);
}

/* Detail View */
.kb__detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}
.kb__back-btn {
  align-self: flex-start;
}
.kb__article-detail {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-6);
}
.kb__article-detail-header {
  margin-bottom: var(--space-5);
  padding-bottom: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}
.kb__article-detail-title {
  font-size: var(--text-xl);
  font-weight: 700;
  margin-bottom: var(--space-3);
}
.kb__article-detail-meta {
  display: flex;
  gap: var(--space-4);
  font-size: var(--text-xs);
  color: var(--color-muted);
  align-items: center;
}
.kb__article-detail-category {
  background: var(--color-primary);
  color: #fff;
  padding: var(--space-1) var(--space-2);
  border-radius: var(--radius-sm);
  font-weight: 500;
}
.kb__article-detail-body {
  font-size: var(--text-sm);
  line-height: 1.8;
  color: var(--color-text);
}
.kb__article-detail-body :deep(h2) {
  font-size: var(--text-lg);
  font-weight: 700;
  margin: var(--space-5) 0 var(--space-3);
}
.kb__article-detail-body :deep(h3) {
  font-size: var(--text-md);
  font-weight: 600;
  margin: var(--space-4) 0 var(--space-2);
}
.kb__article-detail-body :deep(h4) {
  font-size: var(--text-sm);
  font-weight: 600;
  margin: var(--space-3) 0 var(--space-2);
}
.kb__article-detail-body :deep(pre) {
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  padding: var(--space-4);
  overflow-x: auto;
  margin: var(--space-3) 0;
}
.kb__article-detail-body :deep(code) {
  font-family: monospace;
  font-size: var(--text-xs);
  background: var(--color-bg);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}
.kb__article-detail-body :deep(pre code) {
  background: none;
  padding: 0;
}
.kb__article-detail-body :deep(a) {
  color: var(--color-primary);
  text-decoration: underline;
}
.kb__article-detail-body :deep(strong) {
  font-weight: 600;
}

/* Mobile */
@media (max-width: 640px) {
  .kb__article-detail {
    padding: var(--space-4);
  }
  .kb__article-header {
    flex-direction: column;
    align-items: flex-start;
  }
}
</style>
