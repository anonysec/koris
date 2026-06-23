<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KInput from '@koris/ui/KInput.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KSlideOver from '@koris/ui/KSlideOver.vue'
import KEmptyState from '@koris/ui/KEmptyState.vue'

export interface XrayTemplate {
  id: number
  name: string
  config_json: string
  created_at: string
  updated_at: string
}

const props = defineProps<{
  nodeId?: number
}>()

const emit = defineEmits<{
  'apply-template': [template: XrayTemplate]
}>()

const { t } = useI18n()
const api = useApi({ baseUrl: '/api/admin' })
const toast = useToast()
const { confirm } = useConfirm()

const templates = ref<XrayTemplate[]>([])
const loading = ref(false)
const showForm = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const form = ref({
  name: '',
  config_json: '{\n  "inbounds": [],\n  "routing": {}\n}',
})

async function loadTemplates() {
  loading.value = true
  try {
    const res = await api.get<{ ok: boolean; templates: XrayTemplate[] }>('/xray/templates')
    if (res.ok) {
      templates.value = res.templates || []
    }
  } catch {
    // Handled by useApi
  } finally {
    loading.value = false
  }
}

function openCreateForm() {
  editingId.value = null
  form.value = { name: '', config_json: '{\n  "inbounds": [],\n  "routing": {}\n}' }
  showForm.value = true
}

function openEditForm(tmpl: XrayTemplate) {
  editingId.value = tmpl.id
  form.value = { name: tmpl.name, config_json: tmpl.config_json }
  showForm.value = true
}

async function handleSave() {
  if (!form.value.name.trim()) return

  // Validate JSON
  try {
    JSON.parse(form.value.config_json)
  } catch {
    toast.error(t('xray.invalid_json'))
    return
  }

  saving.value = true
  try {
    if (editingId.value) {
      const res = await api.put<{ ok: boolean }>(`/xray/templates/${editingId.value}`, {
        name: form.value.name,
        config_json: form.value.config_json,
      })
      if (res.ok) {
        toast.success(t('xray.template_saved'))
        showForm.value = false
        await loadTemplates()
      }
    } else {
      const res = await api.post<{ ok: boolean }>('/xray/templates', {
        name: form.value.name,
        config_json: form.value.config_json,
      })
      if (res.ok) {
        toast.success(t('xray.template_created'))
        showForm.value = false
        await loadTemplates()
      }
    }
  } catch {
    // Handled by useApi
  } finally {
    saving.value = false
  }
}

async function handleDelete(tmpl: XrayTemplate) {
  const confirmed = await confirm({
    title: t('xray.delete_template_title'),
    message: t('xray.delete_template_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('btn.delete'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  try {
    const res = await api.del<{ ok: boolean }>(`/xray/templates/${tmpl.id}`)
    if (res.ok) {
      toast.success(t('xray.template_deleted'))
      await loadTemplates()
    }
  } catch {
    // Handled by useApi
  }
}

function handleApply(tmpl: XrayTemplate) {
  emit('apply-template', tmpl)
}

onMounted(() => {
  loadTemplates()
})
</script>

<template>
  <div class="xray-templates">
    <div class="templates-header">
      <h4>{{ t('xray.templates') }}</h4>
      <KButton variant="primary" size="sm" icon="+" @click="openCreateForm">
        {{ t('xray.new_template') }}
      </KButton>
    </div>

    <KEmptyState
      v-if="!loading && templates.length === 0"
      icon="📄"
      :title="t('xray.no_templates')"
      :description="t('xray.no_templates_desc')"
    />

    <div v-else class="templates-list">
      <div
        v-for="tmpl in templates"
        :key="tmpl.id"
        class="template-card"
      >
        <div class="template-card__info">
          <span class="template-name">{{ tmpl.name }}</span>
          <span class="template-date">{{ new Date(tmpl.created_at).toLocaleDateString() }}</span>
        </div>
        <div class="template-card__actions">
          <KButton
            v-if="nodeId"
            variant="ghost"
            size="sm"
            @click="handleApply(tmpl)"
          >
            {{ t('xray.apply') }}
          </KButton>
          <KButton variant="ghost" size="sm" @click="openEditForm(tmpl)">
            {{ t('btn.edit') }}
          </KButton>
          <KButton variant="danger" size="sm" @click="handleDelete(tmpl)">
            {{ t('btn.delete') }}
          </KButton>
        </div>
      </div>
    </div>

    <!-- Create/Edit Slide-Over -->
    <KSlideOver
      :open="showForm"
      :title="editingId ? t('xray.edit_template') : t('xray.new_template')"
      @close="showForm = false"
    >
      <form class="template-form" @submit.prevent="handleSave">
        <KFormField name="template-name" :label="t('xray.template_name')" required>
          <template #default="{ fieldId }">
            <KInput :id="fieldId" v-model="form.name" :placeholder="t('xray.template_name_placeholder')" />
          </template>
        </KFormField>

        <KFormField name="template-config" :label="t('xray.template_config')">
          <template #default>
            <textarea
              v-model="form.config_json"
              class="config-textarea"
              spellcheck="false"
              rows="15"
            />
          </template>
        </KFormField>

        <div class="form-actions">
          <KButton variant="ghost" @click="showForm = false">{{ t('btn.cancel') }}</KButton>
          <KButton type="submit" variant="primary" :loading="saving">
            {{ editingId ? t('btn.save') : t('btn.create') }}
          </KButton>
        </div>
      </form>
    </KSlideOver>
  </div>
</template>

<style scoped>
.templates-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-4);
}

.templates-header h4 {
  margin: 0;
  font-size: var(--text-base);
  font-weight: 600;
  color: var(--color-text);
}

.templates-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.template-card {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  background: var(--color-surface);
  transition: border-color 0.15s;
}

.template-card:hover {
  border-color: var(--color-primary);
}

.template-card__info {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.template-name {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}

.template-date {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.template-card__actions {
  display: flex;
  gap: var(--space-1);
}

.template-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  padding: var(--space-4);
}

.config-textarea {
  width: 100%;
  padding: var(--space-3);
  font-family: var(--font-mono, monospace);
  font-size: var(--text-sm);
  line-height: 1.5;
  color: var(--color-text);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  resize: vertical;
  outline: none;
}

.config-textarea:focus {
  border-color: var(--color-primary);
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-2);
  margin-top: var(--space-4);
}
</style>
