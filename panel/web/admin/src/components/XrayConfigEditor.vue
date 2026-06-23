<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'

const props = defineProps<{
  modelValue: string
  readonly?: boolean
}>()

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const { t } = useI18n()

const localValue = ref(props.modelValue)
const jsonError = ref('')
const isVisualMode = ref(true)

watch(() => props.modelValue, (val) => {
  localValue.value = val
  validateJson(val)
})

function validateJson(val: string): boolean {
  if (!val.trim()) {
    jsonError.value = ''
    return true
  }
  try {
    JSON.parse(val)
    jsonError.value = ''
    return true
  } catch (e: any) {
    jsonError.value = e.message || t('xray.invalid_json')
    return false
  }
}

function handleInput(event: Event) {
  const target = event.target as HTMLTextAreaElement
  localValue.value = target.value
  validateJson(target.value)
  emit('update:modelValue', target.value)
}

function handleFormat() {
  try {
    const parsed = JSON.parse(localValue.value)
    const formatted = JSON.stringify(parsed, null, 2)
    localValue.value = formatted
    jsonError.value = ''
    emit('update:modelValue', formatted)
  } catch {
    // Error already shown via validateJson
  }
}

function toggleMode() {
  isVisualMode.value = !isVisualMode.value
}

const lineCount = computed(() => {
  return (localValue.value || '').split('\n').length
})
</script>

<template>
  <div class="xray-config-editor">
    <div class="editor-toolbar">
      <div class="editor-toolbar__left">
        <span class="editor-label">{{ t('xray.config_editor') }}</span>
        <span v-if="lineCount > 0" class="line-count">{{ lineCount }} {{ t('xray.lines') }}</span>
      </div>
      <div class="editor-toolbar__right">
        <KButton variant="ghost" size="sm" @click="toggleMode">
          {{ isVisualMode ? t('xray.raw_json') : t('xray.visual_editor') }}
        </KButton>
        <KButton variant="ghost" size="sm" :disabled="!!jsonError || readonly" @click="handleFormat">
          {{ t('xray.format') }}
        </KButton>
      </div>
    </div>

    <div class="editor-body">
      <textarea
        class="json-textarea"
        :class="{ 'has-error': !!jsonError }"
        :value="localValue"
        :readonly="readonly"
        spellcheck="false"
        @input="handleInput"
      />
    </div>

    <div v-if="jsonError" class="editor-error">
      ⚠ {{ jsonError }}
    </div>
  </div>
</template>

<style scoped>
.xray-config-editor {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  overflow: hidden;
  background: var(--color-surface);
}

.editor-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  background: var(--color-surface-2, rgba(0, 0, 0, 0.1));
}

.editor-toolbar__left {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.editor-toolbar__right {
  display: flex;
  gap: var(--space-1);
}

.editor-label {
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}

.line-count {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.editor-body {
  position: relative;
}

.json-textarea {
  width: 100%;
  min-height: 300px;
  padding: var(--space-3);
  font-family: var(--font-mono, 'JetBrains Mono', monospace);
  font-size: var(--text-sm);
  line-height: 1.6;
  color: var(--color-text);
  background: var(--color-surface);
  border: none;
  outline: none;
  resize: vertical;
  tab-size: 2;
}

.json-textarea.has-error {
  border-left: 3px solid #ef4444;
}

.json-textarea:focus {
  box-shadow: inset 0 0 0 1px var(--color-primary);
}

.editor-error {
  padding: var(--space-2) var(--space-3);
  font-size: var(--text-xs);
  color: #ef4444;
  background: rgba(239, 68, 68, 0.08);
  border-top: 1px solid var(--color-border);
}
</style>
