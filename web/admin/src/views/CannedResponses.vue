<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import Button from '@koris/ui/Button.vue'
import PageHeader from '@koris/ui/PageHeader.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import FormField from '@koris/ui/FormField.vue'
import Tabs from '@koris/ui/Tabs.vue'
import Textarea from '@koris/ui/Textarea.vue'
import Drawer from '@koris/ui/Drawer.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'

const { t } = useI18n()
const { get, post, patch, del } = useApi()
const toast = useToast()
const { confirm } = useConfirm()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface CannedResponse {
  id: number
  title: string
  body: string
  category: string
  usage_count: number
  created_at: string
  updated_at: string
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const responses = ref<CannedResponse[]>([])
const loading = ref(false)
const activeCategory = ref('all')

// Form state
const showForm = ref(false)
const editingId = ref<number | null>(null)
const formTitle = ref('')
const formBody = ref('')
const formCategory = ref('general')
const formSaving = ref(false)

// Placeholder preview
const previewVars = ref('')

// ═══════════════════════════════════════════════════════════════════════════════
// Computed
// ═══════════════════════════════════════════════════════════════════════════════

const categories = computed(() => {
  const cats = new Set<string>()
  for (const r of responses.value) {
    cats.add(r.category)
  }
  return Array.from(cats).sort()
})

const categoryTabs = computed(() => {
  const tabs = [{ key: 'all', label: t('canned.all_categories'), badge: responses.value.length }]
  for (const cat of categories.value) {
    const count = responses.value.filter(r => r.category === cat).length
    tabs.push({ key: cat, label: cat, badge: count })
  }
  return tabs
})

const filteredResponses = computed(() => {
  let list = responses.value
  if (activeCategory.value !== 'all') {
    list = list.filter(r => r.category === activeCategory.value)
  }
  // Sort by usage_count DESC within category
  return [...list].sort((a, b) => b.usage_count - a.usage_count)
})

const previewResult = computed(() => {
  if (!formBody.value) return ''
  let result = formBody.value
  // Parse preview vars (format: key=value, one per line)
  const lines = previewVars.value.split('\n')
  for (const line of lines) {
    const eqIdx = line.indexOf('=')
    if (eqIdx > 0) {
      const key = line.substring(0, eqIdx).trim()
      const value = line.substring(eqIdx + 1).trim()
      result = result.replace(new RegExp(`\\{\\{${key}\\}\\}`, 'g'), value)
    }
  }
  return result
})

const categoryOptions = computed(() => [
  { label: 'general', value: 'general' },
  { label: 'billing', value: 'billing' },
  { label: 'technical', value: 'technical' },
  { label: 'account', value: 'account' },
])

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchResponses() {
  loading.value = true
  try {
    const data = await get<{ ok: boolean; responses: CannedResponse[] }>('/api/canned-responses')
    if (data?.ok) {
      responses.value = data.responses || []
    }
  } catch {
    responses.value = []
  } finally {
    loading.value = false
  }
}

async function saveResponse() {
  if (!formTitle.value.trim() || !formBody.value.trim()) {
    toast.error(t('canned.fill_required'))
    return
  }

  formSaving.value = true
  try {
    const payload = {
      title: formTitle.value.trim(),
      body: formBody.value,
      category: formCategory.value,
    }

    if (editingId.value) {
      await patch<{ ok: boolean }>(`/api/canned-responses/${editingId.value}`, payload)
      toast.success(t('canned.updated'))
    } else {
      await post<{ ok: boolean }>('/api/canned-responses', payload)
      toast.success(t('canned.created'))
    }

    closeForm()
    await fetchResponses()
  } catch {
    // error toast handled by useApi
  } finally {
    formSaving.value = false
  }
}

async function deleteResponse(response: CannedResponse) {
  const confirmed = await confirm({
    title: t('canned.delete_title'),
    message: t('canned.delete_confirm', { title: response.title }),
    variant: 'danger',
    confirmText: t('canned.delete'),
  })

  if (!confirmed) return

  try {
    await del<{ ok: boolean }>(`/api/canned-responses/${response.id}`)
    toast.success(t('canned.deleted'))
    await fetchResponses()
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
  previewVars.value = ''
  showForm.value = true
}

function openEdit(response: CannedResponse) {
  editingId.value = response.id
  formTitle.value = response.title
  formBody.value = response.body
  formCategory.value = response.category
  previewVars.value = ''
  showForm.value = true
}

function closeForm() {
  showForm.value = false
  editingId.value = null
  formTitle.value = ''
  formBody.value = ''
  formCategory.value = 'general'
  previewVars.value = ''
}

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(fetchResponses)
</script>

<template>
  <div class="page canned-view">
    <!-- Header -->
    <PageHeader :title="t('canned.title')" subtitle="Reusable support replies">
      <template #actions>
        <Button variant="primary" size="sm" @click="openCreate">
        + {{ t('canned.create') }}
      </Button>
      </template>
    </PageHeader>

    <!-- Category Tabs -->
    <Tabs
      v-model="activeCategory"
      :tabs="categoryTabs"
      :aria-label="t('canned.categories')"
    />

    <!-- Loading -->
    <div v-if="loading" class="skeleton-wrap">
      <Skeleton variant="table-row" :count="4" />
    </div>

    <!-- Empty State -->
    <EmptyState
      v-else-if="filteredResponses.length === 0"
      icon="💬"
      :title="t('canned.empty')"
      :description="t('canned.empty_desc')"
    />

    <!-- Response List -->
    <div v-else class="responses-list">
      <div
        v-for="response in filteredResponses"
        :key="response.id"
        class="response-card"
      >
        <div class="response-card__header">
          <div class="response-card__title">{{ response.title }}</div>
          <div class="response-card__meta">
            <span class="category-badge">{{ response.category }}</span>
            <span class="usage-badge" :title="t('canned.usage_count')">
              {{ response.usage_count }}×
            </span>
          </div>
        </div>
        <div class="response-card__body">{{ response.body }}</div>
        <div class="response-card__actions">
          <Button variant="ghost" size="sm" @click="openEdit(response)">
            ✏️ {{ t('canned.edit') }}
          </Button>
          <Button variant="ghost" size="sm" @click="deleteResponse(response)">
            🗑️ {{ t('canned.delete') }}
          </Button>
        </div>
      </div>
    </div>

    <!-- Create/Edit Drawer -->
    <Drawer
      :open="showForm"
      :title="editingId ? t('canned.edit_title') : t('canned.create_title')"
      side="right"
      @close="closeForm"
    >
      <form class="canned-form" @submit.prevent="saveResponse">
        <FormField name="title" :label="t('canned.field_title')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="formTitle" :placeholder="t('canned.title_placeholder')" />
          </template>
        </FormField>

        <FormField name="category" :label="t('canned.field_category')">
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="formCategory" :options="categoryOptions" />
          </template>
        </FormField>

        <FormField name="body" :label="t('canned.field_body')">
          <template #default="{ fieldId }">
            <Textarea
              :id="fieldId"
              v-model="formBody"
              :placeholder="t('canned.body_placeholder')"
              :rows="6"
            />
          </template>
        </FormField>

        <p class="hint-text">{{ t('canned.placeholder_hint') }}</p>

        <!-- Placeholder Preview -->
        <div v-if="formBody.includes('{{') " class="preview-section">
          <FormField name="preview-vars" :label="t('canned.preview_vars')">
            <template #default="{ fieldId }">
              <Textarea
                :id="fieldId"
                v-model="previewVars"
                :placeholder="t('canned.preview_vars_placeholder')"
                :rows="3"
              />
            </template>
          </FormField>

          <div class="preview-output">
            <span class="preview-label">{{ t('canned.preview_result') }}</span>
            <div class="preview-content">{{ previewResult }}</div>
          </div>
        </div>

        <div class="form-actions">
          <Button variant="ghost" size="sm" @click="closeForm">
            {{ t('canned.cancel') }}
          </Button>
          <Button type="submit" variant="primary" size="sm" :loading="formSaving">
            {{ editingId ? t('canned.save') : t('canned.create') }}
          </Button>
        </div>
      </form>
    </Drawer>
  </div>
</template>

<style scoped>
.canned-view {
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

/* Response List */
.responses-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.response-card {
  padding: var(--space-4);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  transition: border-color var(--duration-fast);
}
.response-card:hover {
  border-color: var(--color-primary);
}

.response-card__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-2);
}
.response-card__title {
  font-weight: var(--font-semibold);
  font-size: var(--text-sm);
}
.response-card__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.category-badge {
  font-size: var(--text-xs);
  padding: 2px 8px;
  border-radius: var(--radius-full);
  background: var(--color-surface-2);
  color: var(--color-text);
  text-transform: capitalize;
}

.usage-badge {
  font-size: var(--text-xs);
  padding: 2px 6px;
  border-radius: var(--radius-full);
  background: rgba(37, 99, 235, 0.08);
  color: var(--color-primary);
  font-weight: var(--font-semibold);
}

.response-card__body {
  font-size: var(--text-sm);
  color: var(--color-muted);
  white-space: pre-wrap;
  line-height: 1.5;
  margin-bottom: var(--space-3);
  max-height: 80px;
  overflow: hidden;
}

.response-card__actions {
  display: flex;
  gap: var(--space-2);
  border-top: 1px solid var(--color-border);
  padding-top: var(--space-2);
}

/* Form */
.canned-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.hint-text {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin: 0;
}

/* Preview */
.preview-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-3);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
  border: 1px solid var(--color-border);
}

.preview-output {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.preview-label {
  font-size: var(--text-xs);
  color: var(--color-muted);
  text-transform: uppercase;
  letter-spacing: 0.05em;
  font-weight: var(--font-semibold);
}
.preview-content {
  font-size: var(--text-sm);
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  white-space: pre-wrap;
  line-height: 1.5;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Skeleton / Empty */
.skeleton-wrap {
  padding: var(--space-4) 0;
}

@media (max-width: 768px) {
  .page-header { flex-direction: column; align-items: flex-start; gap: var(--space-2); }
  .response-card__header { flex-direction: column; align-items: flex-start; gap: var(--space-1); }
}
</style>
