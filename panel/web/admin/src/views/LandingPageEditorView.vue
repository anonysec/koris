<script setup lang="ts">
import { ref } from 'vue'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import { useApi } from '@koris/composables/useApi'
import KButton from '@koris/ui/KButton.vue'
import KTextarea from '@koris/ui/KTextarea.vue'
import KFormField from '@koris/ui/KFormField.vue'

const { t } = useI18n()
const toast = useToast()
const { get, post } = useApi()

const content = ref('')
const loading = ref(false)
const saving = ref(false)

async function loadContent() {
  loading.value = true
  try {
    const res = await get<{ ok: boolean; content: string }>('/api/admin/landing-page')
    if (res?.ok) {
      content.value = res.content || ''
    }
  } catch {
    // Use default
  } finally {
    loading.value = false
  }
}

async function handleSave() {
  saving.value = true
  try {
    await post('/api/admin/landing-page', { content: content.value })
    toast.success(t('landing_editor.save_success'))
  } catch {
    toast.error(t('landing_editor.save_error'))
  } finally {
    saving.value = false
  }
}

loadContent()
</script>

<template>
  <div class="page landing-editor-view">
    <header class="page-header">
      <h2 class="page-title">{{ t('landing_editor.title') }}</h2>
    </header>

    <div class="editor-panel">
      <p class="editor-desc">{{ t('landing_editor.description') }}</p>
      <KFormField name="landing-content" :label="t('landing_editor.content_label')">
        <template #default="{ fieldId }">
          <KTextarea
            :id="fieldId"
            v-model="content"
            :rows="20"
            :placeholder="t('landing_editor.placeholder')"
            :disabled="loading"
          />
        </template>
      </KFormField>
      <KButton variant="primary" :loading="saving" @click="handleSave">
        {{ t('landing_editor.save') }}
      </KButton>
    </div>
  </div>
</template>

<style scoped>
.landing-editor-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.page-title {
  font-size: var(--text-2xl);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.editor-panel {
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-5);
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.editor-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0;
}
</style>
