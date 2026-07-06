<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useDebounceFn } from '@vueuse/core'
import Button from '@koris/ui/Button.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
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

interface FilterPreset {
  id: number
  name: string
  filters_json: string
  created_at: string
}

interface FilterState {
  status: string
  plan_id: string
  node_id: string
  group_id: string
  date_from: string
  date_to: string
  expiry_from: string
  expiry_to: string
  bandwidth_min: string
  bandwidth_max: string
  tags: number[]
}

interface UserTag {
  id: number
  name: string
  color: string
}

interface PresetListResponse {
  ok: boolean
  presets: FilterPreset[]
}

interface CountResponse {
  ok: boolean
  count: number
}

interface TagListResponse {
  ok: boolean
  tags: UserTag[]
}

// ═══════════════════════════════════════════════════════════════════════════════
// State
// ═══════════════════════════════════════════════════════════════════════════════

const presets = ref<FilterPreset[]>([])
const tags = ref<UserTag[]>([])
const loading = ref(false)
const matchingCount = ref<number | null>(null)
const countLoading = ref(false)

// Filter controls
const filters = ref<FilterState>({
  status: '',
  plan_id: '',
  node_id: '',
  group_id: '',
  date_from: '',
  date_to: '',
  expiry_from: '',
  expiry_to: '',
  bandwidth_min: '',
  bandwidth_max: '',
  tags: [],
})

// Save preset drawer
const showSaveDrawer = ref(false)
const presetName = ref('')
const savingPreset = ref(false)

// ═══════════════════════════════════════════════════════════════════════════════
// Filter Options
// ═══════════════════════════════════════════════════════════════════════════════

const statusOptions = computed(() => [
  { label: t('filters.all_statuses'), value: '' },
  { label: t('filters.status_active'), value: 'active' },
  { label: t('filters.status_expired'), value: 'expired' },
  { label: t('filters.status_disabled'), value: 'disabled' },
  { label: t('filters.status_suspended'), value: 'suspended' },
])

// Plans and nodes loaded from API
const plans = ref<{ id: number; name: string }[]>([])
const nodes = ref<{ id: number; name: string }[]>([])
const groups = ref<{ id: number; name: string }[]>([])

const planOptions = computed(() => [
  { label: t('filters.all_plans'), value: '' },
  ...plans.value.map(p => ({ label: p.name, value: String(p.id) })),
])

const nodeOptions = computed(() => [
  { label: t('filters.all_nodes'), value: '' },
  ...nodes.value.map(n => ({ label: n.name, value: String(n.id) })),
])

const groupOptions = computed(() => [
  { label: t('filters.all_groups'), value: '' },
  ...groups.value.map(g => ({ label: g.name, value: String(g.id) })),
])

// ═══════════════════════════════════════════════════════════════════════════════
// API Calls
// ═══════════════════════════════════════════════════════════════════════════════

async function fetchPresets() {
  loading.value = true
  try {
    const data = await get<PresetListResponse>('/api/filter-presets')
    if (data?.ok) {
      presets.value = data.presets || []
    }
  } catch {
    presets.value = []
  } finally {
    loading.value = false
  }
}

async function fetchTags() {
  try {
    const data = await get<TagListResponse>('/api/tags')
    if (data?.ok) {
      tags.value = data.tags || []
    }
  } catch {
    tags.value = []
  }
}

async function fetchPlans() {
  try {
    const data = await get<{ ok: boolean; plans: { id: number; name: string }[] }>('/api/admin/plans')
    if (data?.ok) {
      plans.value = data.plans || []
    }
  } catch { /* ignore */ }
}

async function fetchNodes() {
  try {
    const data = await get<{ ok: boolean; nodes: { id: number; name: string }[] }>('/api/admin/nodes')
    if (data?.ok) {
      nodes.value = data.nodes || []
    }
  } catch { /* ignore */ }
}

async function fetchGroups() {
  try {
    const data = await get<{ ok: boolean; groups: { id: number; name: string }[] }>('/api/node-groups')
    if (data?.ok) {
      groups.value = data.groups || []
    }
  } catch { /* ignore */ }
}

async function fetchMatchingCount() {
  countLoading.value = true
  try {
    const params = buildFilterParams()
    const data = await get<CountResponse>(`/api/admin/customers/count?${params.toString()}`)
    if (data?.ok) {
      matchingCount.value = data.count
    }
  } catch {
    matchingCount.value = null
  } finally {
    countLoading.value = false
  }
}

const debouncedFetchCount = useDebounceFn(fetchMatchingCount, 500)

// ═══════════════════════════════════════════════════════════════════════════════
// Helpers
// ═══════════════════════════════════════════════════════════════════════════════

function buildFilterParams(): URLSearchParams {
  const params = new URLSearchParams()
  if (filters.value.status) params.set('status', filters.value.status)
  if (filters.value.plan_id) params.set('plan_id', filters.value.plan_id)
  if (filters.value.node_id) params.set('node_id', filters.value.node_id)
  if (filters.value.group_id) params.set('group_id', filters.value.group_id)
  if (filters.value.date_from) params.set('date_from', filters.value.date_from)
  if (filters.value.date_to) params.set('date_to', filters.value.date_to)
  if (filters.value.expiry_from) params.set('expiry_from', filters.value.expiry_from)
  if (filters.value.expiry_to) params.set('expiry_to', filters.value.expiry_to)
  if (filters.value.bandwidth_min) params.set('bandwidth_min', filters.value.bandwidth_min)
  if (filters.value.bandwidth_max) params.set('bandwidth_max', filters.value.bandwidth_max)
  if (filters.value.tags.length > 0) params.set('tags', filters.value.tags.join(','))
  return params
}

function hasActiveFilters(): boolean {
  return !!(
    filters.value.status ||
    filters.value.plan_id ||
    filters.value.node_id ||
    filters.value.group_id ||
    filters.value.date_from ||
    filters.value.date_to ||
    filters.value.expiry_from ||
    filters.value.expiry_to ||
    filters.value.bandwidth_min ||
    filters.value.bandwidth_max ||
    filters.value.tags.length > 0
  )
}

function clearFilters() {
  filters.value = {
    status: '',
    plan_id: '',
    node_id: '',
    group_id: '',
    date_from: '',
    date_to: '',
    expiry_from: '',
    expiry_to: '',
    bandwidth_min: '',
    bandwidth_max: '',
    tags: [],
  }
}

// ─── Preset Actions ─────────────────────────────────────────────────────────

function openSavePreset() {
  presetName.value = ''
  showSaveDrawer.value = true
}

async function savePreset() {
  if (!presetName.value.trim()) {
    toast.error(t('filters.preset_name_required'))
    return
  }

  savingPreset.value = true
  try {
    const data = await post<{ ok: boolean; preset: FilterPreset }>('/api/filter-presets', {
      name: presetName.value.trim(),
      filters_json: JSON.stringify(filters.value),
    })
    if (data?.ok) {
      toast.success(t('filters.preset_saved'))
      presets.value.push(data.preset)
      showSaveDrawer.value = false
    }
  } catch {
    // error toast handled by useApi
  } finally {
    savingPreset.value = false
  }
}

function loadPreset(preset: FilterPreset) {
  try {
    const parsed = JSON.parse(preset.filters_json) as FilterState
    filters.value = {
      status: parsed.status || '',
      plan_id: parsed.plan_id || '',
      node_id: parsed.node_id || '',
      group_id: parsed.group_id || '',
      date_from: parsed.date_from || '',
      date_to: parsed.date_to || '',
      expiry_from: parsed.expiry_from || '',
      expiry_to: parsed.expiry_to || '',
      bandwidth_min: parsed.bandwidth_min || '',
      bandwidth_max: parsed.bandwidth_max || '',
      tags: parsed.tags || [],
    }
    toast.success(t('filters.preset_loaded').replace('{name}', preset.name))
  } catch {
    toast.error(t('filters.preset_load_error'))
  }
}

async function deletePreset(preset: FilterPreset) {
  const confirmed = await confirm({
    title: t('filters.confirm_delete_title'),
    message: t('filters.confirm_delete_msg').replace('{name}', preset.name),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.delete'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  try {
    const data = await del<{ ok: boolean }>(`/api/filter-presets/${preset.id}`)
    if (data?.ok) {
      toast.success(t('filters.preset_deleted'))
      presets.value = presets.value.filter(p => p.id !== preset.id)
    }
  } catch {
    // error toast handled by useApi
  }
}

function toggleTag(tagId: number) {
  const idx = filters.value.tags.indexOf(tagId)
  if (idx >= 0) {
    filters.value.tags.splice(idx, 1)
  } else {
    filters.value.tags.push(tagId)
  }
}

// ─── Emit filters for parent to consume ─────────────────────────────────────

const emit = defineEmits<{
  (e: 'apply', params: URLSearchParams): void
}>()

function applyFilters() {
  emit('apply', buildFilterParams())
}

// Expose for parent integration
defineExpose({ filters, buildFilterParams, loadPreset, clearFilters })

// ─── Watch filters for real-time count ──────────────────────────────────────

watch(filters, () => {
  if (hasActiveFilters()) {
    debouncedFetchCount()
  } else {
    matchingCount.value = null
  }
}, { deep: true })

// ═══════════════════════════════════════════════════════════════════════════════
// Lifecycle
// ═══════════════════════════════════════════════════════════════════════════════

onMounted(() => {
  fetchPresets()
  fetchTags()
  fetchPlans()
  fetchNodes()
  fetchGroups()
})
</script>

<template>
  <div class="page filter-presets-view">
    <!-- Header -->
    <header class="page-header">
      <div class="page-header__left">
        <h2 class="page-title">{{ t('filters.title') }}</h2>
        <span class="page-subtitle">{{ t('filters.subtitle') }}</span>
      </div>
    </header>

    <!-- Filter Panel -->
    <section class="filter-panel">
      <div class="panel-header">
        <h3 class="panel-title">{{ t('filters.panel_title') }}</h3>
        <div class="panel-header__actions">
          <span v-if="matchingCount !== null" class="match-count" :class="{ 'match-count--loading': countLoading }">
            {{ countLoading ? '...' : matchingCount }} {{ t('filters.matching_users') }}
          </span>
          <Button variant="ghost" size="sm" @click="clearFilters">
            {{ t('filters.clear') }}
          </Button>
          <Button variant="primary" size="sm" @click="applyFilters">
            {{ t('filters.apply') }}
          </Button>
        </div>
      </div>

      <div class="filter-grid">
        <!-- Status -->
        <FormField name="filter-status" :label="t('filters.status')">
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="filters.status" :options="statusOptions" />
          </template>
        </FormField>

        <!-- Plan -->
        <FormField name="filter-plan" :label="t('filters.plan')">
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="filters.plan_id" :options="planOptions" />
          </template>
        </FormField>

        <!-- Node -->
        <FormField name="filter-node" :label="t('filters.node')">
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="filters.node_id" :options="nodeOptions" />
          </template>
        </FormField>

        <!-- Group -->
        <FormField name="filter-group" :label="t('filters.group')">
          <template #default="{ fieldId }">
            <Select :id="fieldId" v-model="filters.group_id" :options="groupOptions" />
          </template>
        </FormField>

        <!-- Date Range -->
        <FormField name="filter-date-from" :label="t('filters.created_from')">
          <template #default="{ fieldId }">
            <input :id="fieldId" v-model="filters.date_from" type="date" class="date-input" />
          </template>
        </FormField>
        <FormField name="filter-date-to" :label="t('filters.created_to')">
          <template #default="{ fieldId }">
            <input :id="fieldId" v-model="filters.date_to" type="date" class="date-input" />
          </template>
        </FormField>

        <!-- Expiry Range -->
        <FormField name="filter-expiry-from" :label="t('filters.expiry_from')">
          <template #default="{ fieldId }">
            <input :id="fieldId" v-model="filters.expiry_from" type="date" class="date-input" />
          </template>
        </FormField>
        <FormField name="filter-expiry-to" :label="t('filters.expiry_to')">
          <template #default="{ fieldId }">
            <input :id="fieldId" v-model="filters.expiry_to" type="date" class="date-input" />
          </template>
        </FormField>

        <!-- Bandwidth % -->
        <FormField name="filter-bw-min" :label="t('filters.bandwidth_min')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="filters.bandwidth_min" type="number" placeholder="0" />
          </template>
        </FormField>
        <FormField name="filter-bw-max" :label="t('filters.bandwidth_max')">
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="filters.bandwidth_max" type="number" placeholder="100" />
          </template>
        </FormField>
      </div>

      <!-- Tags Filter -->
      <div class="filter-tags-section">
        <label class="filter-label">{{ t('filters.tags') }}</label>
        <div class="filter-tags">
          <button
            v-for="tag in tags"
            :key="tag.id"
            type="button"
            class="filter-tag-btn"
            :class="{ 'filter-tag-btn--active': filters.tags.includes(tag.id) }"
            @click="toggleTag(tag.id)"
          >
            <span class="tag-swatch" :style="{ backgroundColor: tag.color }" />
            {{ tag.name }}
          </button>
          <span v-if="tags.length === 0" class="no-tags-hint">{{ t('filters.no_tags') }}</span>
        </div>
      </div>

      <!-- Save Preset -->
      <div class="filter-panel__footer">
        <Button
          variant="ghost"
          size="sm"
          :disabled="!hasActiveFilters()"
          @click="openSavePreset"
        >
          {{ t('filters.save_preset') }}
        </Button>
      </div>
    </section>

    <!-- Saved Presets -->
    <section class="presets-section">
      <h3 class="section-title">{{ t('filters.saved_presets') }}</h3>

      <div v-if="loading" class="presets-skeleton">
        <Skeleton v-for="i in 3" :key="i" height="40px" />
      </div>

      <EmptyState
        v-else-if="presets.length === 0"
        icon="📋"
        :title="t('filters.no_presets_title')"
        :description="t('filters.no_presets_desc')"
      />

      <div v-else class="presets-list">
        <div v-for="preset in presets" :key="preset.id" class="preset-row">
          <span class="preset-name">{{ preset.name }}</span>
          <div class="preset-actions">
            <Button variant="ghost" size="sm" @click="loadPreset(preset)">
              {{ t('filters.load') }}
            </Button>
            <Button variant="danger" size="sm" @click="deletePreset(preset)">
              {{ t('btn.delete') }}
            </Button>
          </div>
        </div>
      </div>
    </section>

    <!-- Save Preset Drawer -->
    <Drawer :open="showSaveDrawer" :title="t('filters.save_preset_title')" @close="showSaveDrawer = false">
      <form class="drawer-form" @submit.prevent="savePreset">
        <FormField name="preset-name" :label="t('filters.preset_name')" required>
          <template #default="{ fieldId }">
            <Input :id="fieldId" v-model="presetName" :placeholder="t('filters.preset_name_placeholder')" />
          </template>
        </FormField>
        <p class="drawer-hint">{{ t('filters.preset_save_hint') }}</p>
        <div class="drawer-form__footer">
          <Button type="button" variant="ghost" @click="showSaveDrawer = false">
            {{ t('btn.cancel') }}
          </Button>
          <Button type="submit" variant="primary" :loading="savingPreset">
            {{ t('filters.save') }}
          </Button>
        </div>
      </form>
    </Drawer>
  </div>
</template>

<style scoped>
.filter-presets-view {
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

/* Filter Panel */
.filter-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  margin-bottom: var(--space-6);
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.panel-title {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--color-text);
  margin: 0;
}

.panel-header__actions {
  display: flex;
  align-items: center;
  gap: var(--space-3);
}

.match-count {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-primary);
  padding: var(--space-1) var(--space-2);
  background: var(--color-primary-subtle);
  border-radius: var(--radius-sm);
}

.match-count--loading {
  opacity: 0.6;
}

.filter-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(200px, 1fr));
  gap: var(--space-4);
  margin-bottom: var(--space-4);
}

.date-input {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-sm);
}

.date-input:focus {
  outline: none;
  border-color: var(--color-primary);
  box-shadow: 0 0 0 2px var(--color-primary-subtle);
}

/* Tags Filter */
.filter-tags-section {
  margin-bottom: var(--space-4);
}

.filter-label {
  display: block;
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
  margin-bottom: var(--space-2);
}

.filter-tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.filter-tag-btn {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  padding: var(--space-1) var(--space-3);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-full);
  background: var(--color-surface);
  color: var(--color-text);
  font-size: var(--text-xs);
  cursor: pointer;
  transition: all 0.15s;
}

.filter-tag-btn:hover {
  border-color: var(--color-primary);
}

.filter-tag-btn--active {
  background: var(--color-primary-subtle);
  border-color: var(--color-primary);
  color: var(--color-primary);
}

.tag-swatch {
  width: 10px;
  height: 10px;
  border-radius: var(--radius-full);
  flex-shrink: 0;
}

.no-tags-hint {
  font-size: var(--text-xs);
  color: var(--color-text-muted);
}

.filter-panel__footer {
  display: flex;
  justify-content: flex-end;
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Presets Section */
.presets-section {
  margin-top: var(--space-6);
}

.section-title {
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--color-text);
  margin: 0 0 var(--space-4);
}

.presets-skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.presets-list {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
}

.preset-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.preset-row:last-child {
  border-bottom: none;
}

.preset-name {
  font-size: var(--text-sm);
  font-weight: 500;
  color: var(--color-text);
}

.preset-actions {
  display: flex;
  gap: var(--space-2);
}

/* Drawer */
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

.drawer-hint {
  font-size: var(--text-sm);
  color: var(--color-text-muted);
  margin: 0;
}
</style>
