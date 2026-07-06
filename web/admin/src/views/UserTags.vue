<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { formatDate } from '@koris/composables/useFormatDate'
import Button from '@koris/ui/Button.vue'
import PageHeader from '@koris/ui/PageHeader.vue'
import Input from '@koris/ui/Input.vue'
import FormField from '@koris/ui/FormField.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import Drawer from '@koris/ui/Drawer.vue'

const { t } = useI18n()
const { get, post, del } = useApi()
const toast = useToast()
const { confirm } = useConfirm()

// ═══════════════════════════════════════════════════════════════════════════════
// Types
// ═══════════════════════════════════════════════════════════════════════════════

interface UserTag {
  id: number
  name: string
  color: string
  created_at: string
}

interface TagListResponse {
  ok: boolean
  tags: UserTag[]
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const tags = ref<UserTag[]>([])
const loading = ref(false)

// Create form
const showCreateDrawer = ref(false)
const newTagName = ref('')
const newTagColor = ref('#3b82f6')
const creating = ref(false)

// Tag assignment (for user detail integration)
const showAssignDrawer = ref(false)
const assignCustomerId = ref<number | null>(null)
const assignCustomerName = ref('')
const customerTags = ref<number[]>([])
const assignLoading = ref(false)

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchTags() {
  loading.value = true
  try {
    const data = await get<TagListResponse>('/api/tags')
    if (data?.ok) {
      tags.value = data.tags || []
    }
  } catch {
    tags.value = []
  } finally {
    loading.value = false
  }
}

async function createTag() {
  if (!newTagName.value.trim()) {
    toast.error(t('tags.name_required'))
    return
  }

  creating.value = true
  try {
    const data = await post<{ ok: boolean; tag: UserTag }>('/api/tags', {
      name: newTagName.value.trim(),
      color: newTagColor.value,
    })
    if (data?.ok) {
      toast.success(t('tags.created'))
      tags.value.push(data.tag)
      newTagName.value = ''
      newTagColor.value = '#3b82f6'
      showCreateDrawer.value = false
    }
  } catch {
    // error toast handled by useApi
  } finally {
    creating.value = false
  }
}

async function deleteTag(tag: UserTag) {
  const confirmed = await confirm({
    title: t('tags.confirm_delete_title'),
    message: t('tags.confirm_delete_msg').replace('{name}', tag.name),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.delete'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  try {
    const data = await del<{ ok: boolean }>(`/api/tags/${tag.id}`)
    if (data?.ok) {
      toast.success(t('tags.deleted'))
      tags.value = tags.value.filter(t => t.id !== tag.id)
    }
  } catch {
    // error toast handled by useApi
  }
}

// ─── Tag Assignment ─────────────────────────────────────────────────────────

function openAssignDrawer(customerId: number, customerName: string, currentTags: number[]) {
  assignCustomerId.value = customerId
  assignCustomerName.value = customerName
  customerTags.value = [...currentTags]
  showAssignDrawer.value = true
}

async function assignTag(tagId: number) {
  if (!assignCustomerId.value) return
  assignLoading.value = true
  try {
    const data = await post<{ ok: boolean }>(`/api/customers/${assignCustomerId.value}/tags`, {
      tag_id: tagId,
    })
    if (data?.ok) {
      customerTags.value.push(tagId)
      toast.success(t('tags.assigned'))
    }
  } catch {
    // error toast handled by useApi
  } finally {
    assignLoading.value = false
  }
}

async function removeTag(tagId: number) {
  if (!assignCustomerId.value) return
  assignLoading.value = true
  try {
    const data = await del<{ ok: boolean }>(`/api/customers/${assignCustomerId.value}/tags/${tagId}`)
    if (data?.ok) {
      customerTags.value = customerTags.value.filter(id => id !== tagId)
      toast.success(t('tags.removed'))
    }
  } catch {
    // error toast handled by useApi
  } finally {
    assignLoading.value = false
  }
}

function isTagAssigned(tagId: number): boolean {
  return customerTags.value.includes(tagId)
}

// Expose for parent views integration
defineExpose({ openAssignDrawer, fetchTags, tags })

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(fetchTags)
</script>

<template>
  <div class="page user-tags-view">
    <!-- Header -->
    <PageHeader :title="t('tags.title')" subtitle="Organize customers with tags">
      <template #actions>
        <Button variant="primary" icon="+" @click="showCreateDrawer = true">
        {{ t('tags.create') }}
      </Button>
      </template>
    </PageHeader>

    <!-- Loading Skeleton -->
    <div v-if="loading" class="tags-skeleton">
      <Skeleton v-for="i in 5" :key="i" height="48px" />
    </div>

    <!-- Empty State -->
    <EmptyState
      v-else-if="tags.length === 0"
      icon="🏷️"
      :title="t('tags.empty_title')"
      :description="t('tags.empty_desc')"
    />

    <!-- Tags List -->
    <div v-else class="tags-list">
      <div v-for="tag in tags" :key="tag.id" class="tag-row">
        <div class="tag-row__info">
          <span class="tag-swatch" :style="{ backgroundColor: tag.color }" />
          <span class="tag-name">{{ tag.name }}</span>
        </div>
        <div class="tag-row__meta">
          <span class="tag-date">{{ formatDate(tag.created_at) }}</span>
          <Button
            variant="danger"
            size="sm"
            @click="deleteTag(tag)"
          >{{ t('btn.delete') }}</Button>
        </div>
      </div>
    </div>

    <!-- Create Tag Drawer -->
    <Drawer :open="showCreateDrawer" :title="t('tags.create_title')" @close="showCreateDrawer = false">
      <form class="drawer-form" @submit.prevent="createTag">
        <FormField name="tag-name" :label="t('tags.name')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="newTagName" :placeholder="t('tags.name_placeholder')" />
          </template>
        </FormField>
        <FormField name="tag-color" :label="t('tags.color')">
          <template #default="{ fieldId }">
            <div class="color-picker-row">
              <input
                :id="fieldId"
                v-model="newTagColor"
                type="color"
                class="color-input"
                :aria-label="t('tags.color')"
              />
              <Input
                v-model="newTagColor"
                placeholder="#3b82f6"
                class="color-hex-input"
              />
              <span class="color-preview" :style="{ backgroundColor: newTagColor }" />
            </div>
          </template>
        </FormField>
        <div class="drawer-form__footer">
          <Button type="button" variant="ghost" @click="showCreateDrawer = false">
            {{ t('btn.cancel') }}
          </Button>
          <Button type="submit" variant="primary" :loading="creating">
            {{ t('btn.create') }}
          </Button>
        </div>
      </form>
    </Drawer>

    <!-- Tag Assignment Drawer (used from user detail) -->
    <Drawer
      :open="showAssignDrawer"
      :title="t('tags.assign_title').replace('{name}', assignCustomerName)"
      @close="showAssignDrawer = false"
    >
      <div class="assign-drawer">
        <p class="assign-drawer__desc">{{ t('tags.assign_desc') }}</p>
        <div class="assign-tags-list">
          <div v-for="tag in tags" :key="tag.id" class="assign-tag-item">
            <div class="assign-tag-item__info">
              <span class="tag-swatch" :style="{ backgroundColor: tag.color }" />
              <span class="tag-name">{{ tag.name }}</span>
            </div>
            <Button
              v-if="isTagAssigned(tag.id)"
              variant="danger"
              size="sm"
              :loading="assignLoading"
              @click="removeTag(tag.id)"
            >{{ t('tags.remove') }}</Button>
            <Button
              v-else
              variant="primary"
              size="sm"
              :loading="assignLoading"
              @click="assignTag(tag.id)"
            >{{ t('tags.add') }}</Button>
          </div>
        </div>
      </div>
    </Drawer>
  </div>
</template>

<style scoped>
.user-tags-view {
  padding: var(--space-6);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-6);
}

.page-header__left {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.page-title {
  font-size: var(--text-xl);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.page-subtitle {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
}

.tags-skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.tags-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.tag-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
  transition: background-color 0.15s;
}

.tag-row:last-child {
  border-bottom: none;
}

.tag-row:hover {
  background-color: var(--color-surface-hover);
}

.tag-row__info {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.tag-swatch {
  width: 16px;
  height: 16px;
  border-radius: var(--radius-sm);
  border: 1px solid var(--color-border);
  flex-shrink: 0;
}

.tag-name {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}

.tag-row__meta {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.tag-date {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}

/* Create Drawer */
.drawer-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
}

.drawer-form__footer {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}

.color-picker-row {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.color-input {
  width: 40px;
  height: 36px;
  padding: 2px;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
  cursor: pointer;
  background: none;
}

.color-hex-input {
  flex: 1;
  max-width: 120px;
}

.color-preview {
  width: 24px;
  height: 24px;
  border-radius: var(--radius-full);
  border: 1px solid var(--color-border);
}

/* Assignment Drawer */
.assign-drawer {
  padding: var(--space-4);
}

.assign-drawer__desc {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin-bottom: var(--space-4);
}

.assign-tags-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.assign-tag-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
}

.assign-tag-item__info {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
</style>
