<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { formatDate } from '@koris/composables/useFormatDate'
import KButton from '@koris/ui/KButton.vue'
import KInput from '@koris/ui/KInput.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KTextarea from '@koris/ui/KTextarea.vue'
import KDrawer from '@koris/ui/KDrawer.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'

const { t } = useI18n()
const { get, post, patch, del } = useApi()
const toast = useToast()
const { confirm } = useConfirm()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface KBArticle {
  id: number
  title: string
  body: string
  category: string
  status: 'draft' | 'published'
  locale: string
  parent_id: number | null
  view_count: number
  created_at: string
  updated_at: string
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const articles = ref<KBArticle[]>([])
const loading = ref(false)

// Filters
const filterCategory = ref('')
const filterStatus = ref('')
const filterLocale = ref('')

// Form state
const showForm = ref(false)
const editingId = ref<number | null>(null)
const formTitle = ref('')
const formBody = ref('')
const formCategory = ref('general')
const formStatus = ref<'draft' | 'published'>('draft')
const formLocale = ref('en')
const formParentId = ref<string>('')
const formSaving = ref(false)

// ═══════════════════════════════════════════════════════════════════════════════
// Computed
// ═══════════════════════════════════════════════════════════════════════════════

const filteredArticles = computed(() => {
  return articles.value.filter(a => {
    if (filterCategory.value && a.category !== filterCategory.value) return false
    if (filterStatus.value && a.status !== filterStatus.value) return false
    if (filterLocale.value && a.locale !== filterLocale.value) return false
    return true
  })
})

const categoryOptions = computed(() => {
  const cats = new Set<string>()
  for (const a of articles.value) {
    cats.add(a.category)
  }
  const options = [{ label: t('kb.all_categories'), value: '' }]
  for (const cat of Array.from(cats).sort()) {
    options.push({ label: cat, value: cat })
  }
  return options
})

const statusOptions = computed(() => [
  { label: t('kb.all_statuses'), value: '' },
  { label: t('kb.status_draft'), value: 'draft' },
  { label: t('kb.status_published'), value: 'published' },
])

const localeOptions = computed(() => [
  { label: t('kb.all_locales'), value: '' },
  { label: 'English', value: 'en' },
  { label: 'فارسی', value: 'fa' },
  { label: '中文', value: 'zh' },
  { label: 'Русский', value: 'ru' },
])

const localeFormOptions = computed(() => [
  { label: 'English', value: 'en' },
  { label: 'فارسی', value: 'fa' },
  { label: '中文', value: 'zh' },
  { label: 'Русский', value: 'ru' },
])

const parentArticleOptions = computed(() => {
  // Only show articles that have no parent_id (root articles) for linking translations
  const roots = articles.value.filter(a => !a.parent_id && a.id !== editingId.value)
  const options = [{ label: t('kb.no_parent'), value: '' }]
  for (const root of roots) {
    options.push({ label: `${root.title} (${root.locale})`, value: String(root.id) })
  }
  return options
})

const statusFormOptions = computed(() => [
  { label: t('kb.status_draft'), value: 'draft' },
  { label: t('kb.status_published'), value: 'published' },
])

const formCategoryOptions = computed(() => [
  { label: 'general', value: 'general' },
  { label: 'getting-started', value: 'getting-started' },
  { label: 'billing', value: 'billing' },
  { label: 'technical', value: 'technical' },
  { label: 'troubleshooting', value: 'troubleshooting' },
])

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchArticles() {
  loading.value = true
  try {
    const data = await get<{ ok: boolean; articles: KBArticle[] }>('/api/kb/articles')
    if (data?.ok) {
      articles.value = data.articles || []
    }
  } catch {
    articles.value = []
  } finally {
    loading.value = false
  }
}

async function saveArticle() {
  if (!formTitle.value.trim() || !formBody.value.trim()) {
    toast.error(t('kb.fill_required'))
    return
  }

  formSaving.value = true
  try {
    const payload: Record<string, unknown> = {
      title: formTitle.value.trim(),
      body: formBody.value,
      category: formCategory.value,
      status: formStatus.value,
      locale: formLocale.value,
    }
    if (formParentId.value) {
      payload.parent_id = Number(formParentId.value)
    } else {
      payload.parent_id = null
    }

    if (editingId.value) {
      await patch<{ ok: boolean }>(`/api/kb/articles/${editingId.value}`, payload)
      toast.success(t('kb.updated'))
    } else {
      await post<{ ok: boolean }>('/api/kb/articles', payload)
      toast.success(t('kb.created'))
    }

    closeForm()
    await fetchArticles()
  } catch {
    // error toast handled by useApi
  } finally {
    formSaving.value = false
  }
}

async function toggleStatus(article: KBArticle) {
  const newStatus = article.status === 'published' ? 'draft' : 'published'
  try {
    await patch<{ ok: boolean }>(`/api/kb/articles/${article.id}`, { status: newStatus })
    toast.success(t('kb.status_changed'))
    await fetchArticles()
  } catch {
    // error toast handled by useApi
  }
}

async function deleteArticle(article: KBArticle) {
  const confirmed = await confirm({
    title: t('kb.delete_title'),
    message: t('kb.delete_confirm', { title: article.title }),
    variant: 'danger',
    confirmText: t('kb.delete'),
  })

  if (!confirmed) return

  try {
    await del<{ ok: boolean }>(`/api/kb/articles/${article.id}`)
    toast.success(t('kb.deleted'))
    await fetchArticles()
  } catch {
    // error toast handled by useApi
  }
}

// ═══════════════════════════════════════════════════════════════════════════════
// Form Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function openCreate() {
  editingId.value = null
  formTitle.value = ''
  formBody.value = ''
  formCategory.value = 'general'
  formStatus.value = 'draft'
  formLocale.value = 'en'
  formParentId.value = ''
  showForm.value = true
}

function openEdit(article: KBArticle) {
  editingId.value = article.id
  formTitle.value = article.title
  formBody.value = article.body
  formCategory.value = article.category
  formStatus.value = article.status
  formLocale.value = article.locale
  formParentId.value = article.parent_id ? String(article.parent_id) : ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingId.value = null
  formTitle.value = ''
  formBody.value = ''
  formCategory.value = 'general'
  formStatus.value = 'draft'
  formLocale.value = 'en'
  formParentId.value = ''
}

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function localeLabel(locale: string): string {
  const labels: Record<string, string> = { en: 'EN', fa: 'FA', zh: 'ZH', ru: 'RU' }
  return labels[locale] || locale.toUpperCase()
}

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(fetchArticles)
</script>

<template>
  <div class="page kb-view">
    <!-- Header -->
    <header class="page-header">
      <h2 class="page-title">{{ t('kb.title') }}</h2>
      <KButton variant="primary" size="sm" @click="openCreate">
        + {{ t('kb.create') }}
      </KButton>
    </header>

    <!-- Filters -->
    <section class="filter-section">
      <div class="filter-row">
        <KSelect
          v-model="filterCategory"
          :options="categoryOptions"
          :aria-label="t('kb.filter_category')"
          class="filter-select"
        />
        <KSelect
          v-model="filterStatus"
          :options="statusOptions"
          :aria-label="t('kb.filter_status')"
          class="filter-select"
        />
        <KSelect
          v-model="filterLocale"
          :options="localeOptions"
          :aria-label="t('kb.filter_locale')"
          class="filter-select"
        />
      </div>
    </section>

    <!-- Loading -->
    <div v-if="loading" class="skeleton-wrap">
      <KSkeleton variant="table-row" :count="5" />
    </div>

    <!-- Empty State -->
    <KEmptyState
      v-else-if="filteredArticles.length === 0"
      icon="📚"
      :title="t('kb.empty')"
      :description="t('kb.empty_desc')"
    />

    <!-- Article Table -->
    <section v-else class="panel">
      <div class="table-wrap">
        <table class="data-table" role="table">
          <thead>
            <tr>
              <th>{{ t('kb.col_title') }}</th>
              <th>{{ t('kb.col_category') }}</th>
              <th>{{ t('kb.col_status') }}</th>
              <th>{{ t('kb.col_locale') }}</th>
              <th>{{ t('kb.col_views') }}</th>
              <th>{{ t('kb.col_updated') }}</th>
              <th>{{ t('kb.col_actions') }}</th>
            </tr>
          </thead>
          <tbody>
            <tr
              v-for="article in filteredArticles"
              :key="article.id"
              class="clickable-row"
              @click="openEdit(article)"
            >
              <td class="article-title-cell">
                <span class="article-title">{{ article.title }}</span>
                <span v-if="article.parent_id" class="translation-badge">
                  🌐 {{ t('kb.translation') }}
                </span>
              </td>
              <td>
                <span class="category-badge">{{ article.category }}</span>
              </td>
              <td>
                <KStatusPill
                  :status="article.status === 'published' ? 'active' : 'default'"
                  size="sm"
                >
                  {{ t(`kb.status_${article.status}`) }}
                </KStatusPill>
              </td>
              <td>
                <span class="locale-badge">{{ localeLabel(article.locale) }}</span>
              </td>
              <td class="text-muted">{{ article.view_count }}</td>
              <td class="text-muted">{{ formatDate(article.updated_at) }}</td>
              <td>
                <div class="action-btns" @click.stop>
                  <KButton
                    variant="ghost"
                    size="sm"
                    @click="toggleStatus(article)"
                  >
                    {{ article.status === 'published' ? '📝' : '🚀' }}
                    {{ article.status === 'published' ? t('kb.unpublish') : t('kb.publish') }}
                  </KButton>
                  <KButton variant="ghost" size="sm" @click="deleteArticle(article)">
                    🗑️
                  </KButton>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </section>

    <!-- Create/Edit Drawer -->
    <KDrawer
      :open="showForm"
      :title="editingId ? t('kb.edit_title') : t('kb.create_title')"
      side="right"
      @close="closeForm"
    >
      <form class="kb-form" @submit.prevent="saveArticle">
        <KFormField name="title" :label="t('kb.field_title')">
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="formTitle" :placeholder="t('kb.title_placeholder')" />
          </template>
        </KFormField>

        <KFormField name="category" :label="t('kb.field_category')">
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="formCategory" :options="formCategoryOptions" />
          </template>
        </KFormField>

        <div class="form-row-2">
          <KFormField name="status" :label="t('kb.field_status')">
            <template #default="{ fieldId }">
              <KSelect :id="fieldId" v-model="formStatus" :options="statusFormOptions" />
            </template>
          </KFormField>

          <KFormField name="locale" :label="t('kb.field_locale')">
            <template #default="{ fieldId }">
              <KSelect :id="fieldId" v-model="formLocale" :options="localeFormOptions" />
            </template>
          </KFormField>
        </div>

        <KFormField name="parent" :label="t('kb.field_parent')">
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="formParentId" :options="parentArticleOptions" />
          </template>
        </KFormField>
        <p class="hint-text">{{ t('kb.parent_hint') }}</p>

        <KFormField name="body" :label="t('kb.field_body')">
          <template #default="{ fieldId }">
            <KTextarea
              :id="fieldId"
              v-model="formBody"
              :placeholder="t('kb.body_placeholder')"
              :rows="12"
              class="markdown-editor"
            />
          </template>
        </KFormField>
        <p class="hint-text">{{ t('kb.markdown_hint') }}</p>

        <div class="form-actions">
          <KButton variant="ghost" size="sm" @click="closeForm">
            {{ t('kb.cancel') }}
          </KButton>
          <KButton
            v-if="editingId"
            variant="secondary"
            size="sm"
            type="button"
            @click="formStatus = formStatus === 'published' ? 'draft' : 'published'"
          >
            {{ formStatus === 'published' ? t('kb.set_draft') : t('kb.set_published') }}
          </KButton>
          <KButton type="submit" variant="primary" size="sm" :loading="formSaving">
            {{ editingId ? t('kb.save') : t('kb.create') }}
          </KButton>
        </div>
      </form>
    </KDrawer>
  </div>
</template>

<style scoped>
.kb-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.page-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.page-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-bold);
}

/* Filters */
.filter-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.filter-row {
  display: flex;
  gap: var(--space-2);
  flex-wrap: wrap;
}
.filter-select {
  width: 180px;
}

/* Panel */
.panel {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

/* Table */
.table-wrap {
  overflow-x: auto;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}
.data-table th {
  text-align: left;
  padding: var(--space-2) var(--space-3);
  color: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wider);
  border-bottom: 1px solid var(--color-border);
  white-space: nowrap;
}
.data-table td {
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
  white-space: nowrap;
}
.clickable-row {
  cursor: pointer;
  transition: background 0.15s;
}
.clickable-row:hover {
  background: var(--color-surface-2, rgba(0, 0, 0, 0.02));
}

.article-title-cell {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.article-title {
  font-weight: var(--font-medium);
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 250px;
}

.translation-badge {
  font-size: var(--text-xs);
  padding: 1px 6px;
  border-radius: var(--radius-full);
  background: rgba(37, 99, 235, 0.08);
  color: var(--color-primary);
  white-space: nowrap;
}

.category-badge {
  font-size: var(--text-xs);
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-2);
  color: var(--color-text);
  text-transform: capitalize;
}

.locale-badge {
  font-size: var(--text-xs);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
  background: var(--color-surface-2);
  color: var(--color-muted);
  font-weight: var(--font-semibold);
  letter-spacing: 0.05em;
}

.action-btns {
  display: flex;
  gap: var(--space-1);
}

/* Form */
.kb-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.form-row-2 {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.hint-text {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin: calc(-1 * var(--space-2)) 0 0;
}

.markdown-editor {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-sm);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Utility */
.text-muted { color: var(--color-muted); }
.skeleton-wrap { padding: var(--space-4) 0; }

@media (max-width: 768px) {
  .page-header { flex-direction: column; align-items: flex-start; gap: var(--space-2); }
  .filter-row { flex-direction: column; }
  .filter-select { width: 100%; }
  .form-row-2 { grid-template-columns: 1fr; }
  .article-title { max-width: 150px; }
}
</style>
