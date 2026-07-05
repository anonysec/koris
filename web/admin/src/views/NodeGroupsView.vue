<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import { useNodesStore } from '@/stores/nodes'
import { storeToRefs } from 'pinia'
import KButton from '@koris/ui/KButton.vue'
import KInput from '@koris/ui/KInput.vue'
import KTextarea from '@koris/ui/KTextarea.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'

const { t } = useI18n()
const api = useApi()
const toast = useToast()
const { confirm } = useConfirm()
const nodesStore = useNodesStore()
const { list: nodes } = storeToRefs(nodesStore)

// ─── Types ───────────────────────────────────────────────────────────────────
interface NodeGroup {
  id: number
  name: string
  region: string
  description: string
  load_balancing_enabled: boolean
  max_load_percent: number
  member_count: number
  created_at: string
}

interface GroupLoadInfo {
  group_id: number
  group_name: string
  node_count: number
  total_capacity: number
  active_sessions: number
  load_percent: number
}

// ─── State ───────────────────────────────────────────────────────────────────
const groups = ref<NodeGroup[]>([])
const loadInfo = ref<GroupLoadInfo[]>([])
const loading = ref(false)
const loadingStats = ref(false)

// Modal state
const showModal = ref(false)
const editingGroup = ref<NodeGroup | null>(null)
const saving = ref(false)
const groupForm = ref({
  name: '',
  region: '',
  description: '',
  load_balancing_enabled: false,
})

// Node assignment state
const assigningNodeId = ref<number | string>('')
const assigningGroupId = ref<number | string>('')
const assigning = ref(false)

// ─── Computed ────────────────────────────────────────────────────────────────
const modalTitle = computed(() =>
  editingGroup.value ? t('node_groups.edit_group') : t('node_groups.create_group')
)

const unassignedNodes = computed(() =>
  nodes.value.filter(n => !n.group_id)
)

const groupOptions = computed(() =>
  groups.value.map(g => ({ label: g.name, value: g.id }))
)

const nodeOptions = computed(() =>
  nodes.value.map(n => ({ label: `${n.name} (${n.public_ip})`, value: n.id }))
)

// ─── API Calls ───────────────────────────────────────────────────────────────
async function fetchGroups() {
  loading.value = true
  try {
    const res = await api.get<{ ok: boolean; groups: NodeGroup[] }>('/api/node-groups')
    groups.value = res.groups || []
  } finally {
    loading.value = false
  }
}

async function fetchLoadOverview() {
  loadingStats.value = true
  try {
    const res = await api.get<{ ok: boolean; groups: GroupLoadInfo[] }>('/api/node-groups/load')
    loadInfo.value = res.groups || []
  } finally {
    loadingStats.value = false
  }
}

function openCreateModal() {
  editingGroup.value = null
  groupForm.value = { name: '', region: '', description: '', load_balancing_enabled: false }
  showModal.value = true
}

function openEditModal(group: NodeGroup) {
  editingGroup.value = group
  groupForm.value = {
    name: group.name,
    region: group.region,
    description: group.description,
    load_balancing_enabled: group.load_balancing_enabled,
  }
  showModal.value = true
}

function closeModal() {
  showModal.value = false
  editingGroup.value = null
}

async function saveGroup() {
  saving.value = true
  try {
    if (editingGroup.value) {
      await api.patch<{ ok: boolean }>(`/api/node-groups/${editingGroup.value.id}`, groupForm.value)
      toast.success(t('node_groups.updated'))
    } else {
      await api.post<{ ok: boolean }>('/api/node-groups', groupForm.value)
      toast.success(t('node_groups.created'))
    }
    closeModal()
    await fetchGroups()
    await fetchLoadOverview()
  } catch {
    toast.error(t('node_groups.save_failed'))
  } finally {
    saving.value = false
  }
}

async function deleteGroup(group: NodeGroup) {
  const confirmed = await confirm({
    title: t('node_groups.confirm_delete_title'),
    message: t('node_groups.confirm_delete_msg').replace('{name}', group.name),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.delete'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  try {
    await api.del<{ ok: boolean }>(`/api/node-groups/${group.id}`)
    toast.success(t('node_groups.deleted'))
    await fetchGroups()
    await fetchLoadOverview()
  } catch {
    toast.error(t('node_groups.delete_failed'))
  }
}

async function assignNodeToGroup() {
  if (!assigningNodeId.value || !assigningGroupId.value) return
  assigning.value = true
  try {
    await api.post<{ ok: boolean }>(`/api/nodes/${assigningNodeId.value}/assign-group`, {
      group_id: Number(assigningGroupId.value),
    })
    toast.success(t('node_groups.node_assigned'))
    assigningNodeId.value = ''
    assigningGroupId.value = ''
    await fetchGroups()
    await fetchLoadOverview()
    await nodesStore.loadNodes()
  } catch {
    toast.error(t('node_groups.assign_failed'))
  } finally {
    assigning.value = false
  }
}

function getLoadColor(percent: number): string {
  if (percent >= 90) return 'var(--color-danger, #ef4444)'
  if (percent >= 70) return 'var(--color-warning, #f59e0b)'
  return 'var(--color-success, #10b981)'
}

onMounted(async () => {
  await Promise.all([fetchGroups(), fetchLoadOverview(), nodesStore.loadNodes()])
})
</script>

<template>
  <div class="page node-groups-view">
    <!-- Header -->
    <header class="page-header">
      <h3 class="section-title">{{ t('node_groups.title') }}</h3>
      <KButton variant="primary" @click="openCreateModal">
        {{ t('node_groups.create_group') }}
      </KButton>
    </header>

    <!-- Node Assignment Section -->
    <section class="assign-section">
      <h4 class="subsection-title">{{ t('node_groups.assign_node') }}</h4>
      <div class="assign-form">
        <KFormField name="assign-node" :label="t('node_groups.node')">
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="assigningNodeId" :options="nodeOptions" :placeholder="t('node_groups.select_node')" />
          </template>
        </KFormField>
        <KFormField name="assign-group" :label="t('node_groups.group')">
          <template #default="{ fieldId }">
            <KSelect :id="fieldId" v-model="assigningGroupId" :options="groupOptions" :placeholder="t('node_groups.select_group')" />
          </template>
        </KFormField>
        <KButton
          variant="primary"
          size="sm"
          :loading="assigning"
          :disabled="!assigningNodeId || !assigningGroupId"
          @click="assignNodeToGroup"
        >
          {{ t('node_groups.assign') }}
        </KButton>
      </div>
    </section>

    <!-- Groups List -->
    <section class="groups-section">
      <div v-if="loading && groups.length === 0">
        <KSkeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="72" />
      </div>

      <KEmptyState
        v-else-if="groups.length === 0"
        icon="📦"
        :title="t('node_groups.no_groups')"
        :description="t('node_groups.no_groups_desc')"
      />

      <div v-else class="groups-grid">
        <div v-for="group in groups" :key="group.id" class="group-card">
          <div class="group-card__header">
            <div class="group-card__title">
              <h4 class="group-card__name">{{ group.name }}</h4>
              <span v-if="group.region" class="group-card__region">{{ group.region }}</span>
            </div>
            <div class="group-card__meta">
              <span class="group-card__members">{{ group.member_count }} {{ t('node_groups.nodes') }}</span>
              <span v-if="group.load_balancing_enabled" class="group-card__lb-badge">LB</span>
            </div>
          </div>
          <p v-if="group.description" class="group-card__desc">{{ group.description }}</p>
          <div class="group-card__actions">
            <KButton variant="ghost" size="sm" @click="openEditModal(group)">{{ t('btn.edit') }}</KButton>
            <KButton variant="danger" size="sm" @click="deleteGroup(group)">{{ t('btn.delete') }}</KButton>
          </div>
        </div>
      </div>
    </section>

    <!-- Load Overview Dashboard -->
    <section class="load-section">
      <h4 class="subsection-title">{{ t('node_groups.load_overview') }}</h4>

      <div v-if="loadingStats">
        <KSkeleton v-for="i in 2" :key="i" variant="rect" width="100%" :height="48" />
      </div>

      <KEmptyState
        v-else-if="loadInfo.length === 0"
        icon="📊"
        :title="t('node_groups.no_load_data')"
        :description="t('node_groups.no_load_data_desc')"
      />

      <div v-else class="load-bars">
        <div v-for="info in loadInfo" :key="info.group_id" class="load-bar-item">
          <div class="load-bar-item__header">
            <span class="load-bar-item__name">{{ info.group_name }}</span>
            <span class="load-bar-item__stats">
              {{ info.active_sessions }}/{{ info.total_capacity }} ({{ info.load_percent }}%)
            </span>
          </div>
          <div class="load-bar-item__bar">
            <div
              class="load-bar-item__fill"
              :style="{ width: `${Math.min(info.load_percent, 100)}%`, backgroundColor: getLoadColor(info.load_percent) }"
            />
          </div>
          <span class="load-bar-item__nodes text-muted">{{ info.node_count }} {{ t('node_groups.nodes') }}</span>
        </div>
      </div>
    </section>

    <!-- Create/Edit Modal -->
    <Teleport to="body">
      <Transition name="modal">
        <div
          v-if="showModal"
          class="modal-overlay"
          role="dialog"
          aria-modal="true"
          @click.self="closeModal"
          @keydown.escape="closeModal"
        >
          <div class="modal">
            <header class="modal__header">
              <h2 class="modal__title">{{ modalTitle }}</h2>
              <button class="modal__close" aria-label="Close" @click="closeModal">✕</button>
            </header>
            <form class="modal__body" @submit.prevent="saveGroup">
              <KFormField name="group-name" :label="t('node_groups.name')" required>
                <template #default="{ fieldId }">
                  <KInput :id="fieldId" v-model="groupForm.name" :placeholder="t('node_groups.name_placeholder')" />
                </template>
              </KFormField>
              <KFormField name="group-region" :label="t('node_groups.region')">
                <template #default="{ fieldId }">
                  <KInput :id="fieldId" v-model="groupForm.region" :placeholder="t('node_groups.region_placeholder')" />
                </template>
              </KFormField>
              <KFormField name="group-desc" :label="t('node_groups.description')">
                <template #default="{ fieldId }">
                  <KTextarea :id="fieldId" v-model="groupForm.description" :rows="3" />
                </template>
              </KFormField>
              <KFormField name="group-lb" :label="t('node_groups.load_balancing')">
                <template #default>
                  <label class="toggle-switch">
                    <input
                      type="checkbox"
                      :checked="groupForm.load_balancing_enabled"
                      @change="groupForm.load_balancing_enabled = ($event.target as HTMLInputElement).checked"
                    />
                    <span class="toggle-switch__slider" />
                  </label>
                </template>
              </KFormField>
              <div class="modal__actions">
                <KButton variant="ghost" @click="closeModal">{{ t('btn.cancel') }}</KButton>
                <KButton type="submit" variant="primary" :loading="saving">{{ t('btn.save') }}</KButton>
              </div>
            </form>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.node-groups-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-6);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.section-title {
  margin: 0;
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
}

.subsection-title {
  margin: 0 0 var(--space-3);
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

/* Assign section */
.assign-section {
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
}

.assign-form {
  display: flex;
  align-items: flex-end;
  gap: var(--space-3);
  flex-wrap: wrap;
}

/* Groups Grid */
.groups-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.groups-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: var(--space-4);
}

.group-card {
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.group-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
}

.group-card__title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.group-card__name {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

.group-card__region {
  font-size: var(--text-xs);
  color: var(--color-muted);
  background: var(--color-surface-2, #1e2630);
  padding: 2px 8px;
  border-radius: var(--radius-full);
}

.group-card__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.group-card__members {
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.group-card__lb-badge {
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--color-primary);
  background: rgba(59, 130, 246, 0.1);
  padding: 2px 6px;
  border-radius: var(--radius-sm);
}

.group-card__desc {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-muted);
  line-height: var(--leading-normal);
}

.group-card__actions {
  display: flex;
  gap: var(--space-2);
  margin-top: var(--space-2);
}

/* Load Overview */
.load-section {
  padding: var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  background: var(--color-surface);
}

.load-bars {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.load-bar-item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.load-bar-item__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.load-bar-item__name {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}

.load-bar-item__stats {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.load-bar-item__bar {
  height: 8px;
  background: var(--color-surface-2, #1e2630);
  border-radius: var(--radius-full);
  overflow: hidden;
}

.load-bar-item__fill {
  height: 100%;
  border-radius: var(--radius-full);
  transition: width 0.3s ease;
}

.load-bar-item__nodes {
  font-size: var(--text-xs);
}

/* Modal */
.modal-overlay {
  position: fixed;
  inset: 0;
  z-index: var(--z-modal, 200);
  display: flex;
  align-items: center;
  justify-content: center;
  background-color: rgba(0, 0, 0, 0.6);
  backdrop-filter: blur(2px);
  padding: var(--space-4);
}

.modal {
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-xl, 14px);
  max-width: 480px;
  width: 100%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
}

.modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
}

.modal__title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.modal__close {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: var(--radius-sm);
  background: transparent;
  color: var(--color-muted);
  cursor: pointer;
  font-size: var(--text-base);
}

.modal__close:hover {
  background: var(--color-surface-2, #1e2630);
  color: var(--color-text);
}

.modal__body {
  padding: var(--space-4) var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  overflow-y: auto;
}

.modal__actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Toggle switch */
.toggle-switch {
  position: relative;
  display: inline-flex;
  align-items: center;
  cursor: pointer;
}

.toggle-switch input {
  position: absolute;
  opacity: 0;
  width: 0;
  height: 0;
}

.toggle-switch__slider {
  width: 36px;
  height: 20px;
  background: var(--color-border);
  border-radius: var(--radius-full);
  transition: background 0.2s;
  position: relative;
}

.toggle-switch__slider::before {
  content: '';
  position: absolute;
  top: 2px;
  left: 2px;
  width: 16px;
  height: 16px;
  border-radius: 50%;
  background: white;
  transition: transform 0.2s;
}

.toggle-switch input:checked + .toggle-switch__slider {
  background: var(--color-primary);
}

.toggle-switch input:checked + .toggle-switch__slider::before {
  transform: translateX(16px);
}

/* Modal transition */
.modal-enter-active,
.modal-leave-active {
  transition: opacity 0.2s ease-out;
}

.modal-enter-active .modal,
.modal-leave-active .modal {
  transition: transform 0.2s ease-out, opacity 0.2s ease-out;
}

.modal-enter-from,
.modal-leave-to {
  opacity: 0;
}

.modal-enter-from .modal,
.modal-leave-to .modal {
  transform: scale(0.95);
  opacity: 0;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }

  .assign-form {
    flex-direction: column;
    align-items: stretch;
  }

  .groups-grid {
    grid-template-columns: 1fr;
  }
}
</style>
