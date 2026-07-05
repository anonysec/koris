<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { getCSSVariables, type ThemeConfig, type ThemeColors } from '@koris/composables/useTheme'

const props = defineProps<{
  /** Initial theme config JSON to populate the editor */
  initialConfig?: ThemeConfig
  /** Initial theme name */
  initialName?: string
  /** Initial theme mode */
  initialMode?: 'light' | 'dark'
  /** Initial theme ID (for updates) */
  initialId?: string
}>()

const emit = defineEmits<{
  saved: [payload: { id: string; name: string; mode: string; config: ThemeConfig }]
}>()

const { post, loading } = useApi({ baseUrl: '/api' })

// ─── Form State ───

const themeName = ref(props.initialName || '')
const themeMode = ref<'light' | 'dark'>(props.initialMode || 'dark')
const themeId = ref(props.initialId || '')

const COLOR_TOKENS: { key: keyof ThemeColors; label: string }[] = [
  { key: 'primary', label: 'Primary' },
  { key: 'primaryHover', label: 'Primary Hover' },
  { key: 'secondary', label: 'Secondary' },
  { key: 'background', label: 'Background' },
  { key: 'surface', label: 'Surface' },
  { key: 'surfaceHover', label: 'Surface Hover' },
  { key: 'text', label: 'Text' },
  { key: 'textMuted', label: 'Text Muted' },
  { key: 'border', label: 'Border' },
  { key: 'success', label: 'Success' },
  { key: 'warning', label: 'Warning' },
  { key: 'error', label: 'Error' },
  { key: 'info', label: 'Info' },
  { key: 'accent', label: 'Accent' },
]

const defaultConfig: ThemeConfig = {
  colors: {
    primary: '#3b82f6',
    primaryHover: '#2563eb',
    secondary: '#64748b',
    background: '#0f172a',
    surface: '#1e293b',
    surfaceHover: '#334155',
    text: '#f1f5f9',
    textMuted: '#94a3b8',
    border: '#334155',
    success: '#4ade80',
    warning: '#fbbf24',
    error: '#f87171',
    info: '#60a5fa',
    accent: '#a78bfa',
  },
  borderRadius: '8px',
  shadows: {
    sm: '0 1px 2px rgba(0,0,0,0.3)',
    md: '0 4px 6px rgba(0,0,0,0.4)',
    lg: '0 10px 15px rgba(0,0,0,0.5)',
  },
}

const jsonText = ref(JSON.stringify(props.initialConfig || defaultConfig, null, 2))
const jsonError = ref('')
const isPreviewing = ref(false)
const saveSuccess = ref(false)

// ─── Parsed Config ───

const parsedConfig = computed<ThemeConfig | null>(() => {
  try {
    const parsed = JSON.parse(jsonText.value) as ThemeConfig
    if (!parsed.colors || !parsed.borderRadius || !parsed.shadows) {
      return null
    }
    return parsed
  } catch {
    return null
  }
})

// ─── Validation ───

function validateJson(): boolean {
  jsonError.value = ''
  try {
    const parsed = JSON.parse(jsonText.value)
    if (!parsed.colors || typeof parsed.colors !== 'object') {
      jsonError.value = 'Missing "colors" object in config'
      return false
    }
    const missingColors = COLOR_TOKENS
      .filter(t => !parsed.colors[t.key])
      .map(t => t.key)
    if (missingColors.length > 0) {
      jsonError.value = `Missing color tokens: ${missingColors.join(', ')}`
      return false
    }
    if (!parsed.borderRadius) {
      jsonError.value = 'Missing "borderRadius" field'
      return false
    }
    if (!parsed.shadows || !parsed.shadows.sm || !parsed.shadows.md || !parsed.shadows.lg) {
      jsonError.value = 'Missing "shadows" object (needs sm, md, lg)'
      return false
    }
    return true
  } catch (e) {
    jsonError.value = e instanceof Error ? e.message : 'Invalid JSON'
    return false
  }
}

// ─── Color Picker Sync ───

function updateColorFromPicker(key: keyof ThemeColors, value: string) {
  if (!parsedConfig.value) return
  const updated = { ...parsedConfig.value, colors: { ...parsedConfig.value.colors, [key]: value } }
  jsonText.value = JSON.stringify(updated, null, 2)
}

// ─── Live Preview ───

let originalStyles: Record<string, string> = {}

function applyPreview() {
  if (!validateJson()) return
  const config = parsedConfig.value
  if (!config) return

  const vars = getCSSVariables(config)
  const root = document.documentElement

  // Store original values before overriding
  if (!isPreviewing.value) {
    originalStyles = {}
    for (const key of Object.keys(vars)) {
      originalStyles[key] = root.style.getPropertyValue(key)
    }
  }

  for (const [key, value] of Object.entries(vars)) {
    root.style.setProperty(key, value)
  }
  isPreviewing.value = true
}

function revertPreview() {
  if (!isPreviewing.value) return
  const root = document.documentElement
  for (const [key, value] of Object.entries(originalStyles)) {
    if (value) {
      root.style.setProperty(key, value)
    } else {
      root.style.removeProperty(key)
    }
  }
  originalStyles = {}
  isPreviewing.value = false
}

// ─── Save ───

async function saveTheme() {
  if (!validateJson()) return
  if (!themeName.value.trim()) {
    jsonError.value = 'Theme name is required'
    return
  }

  const config = parsedConfig.value
  if (!config) return

  const id = themeId.value || themeName.value.toLowerCase().replace(/[^a-z0-9]+/g, '-').replace(/(^-|-$)/g, '')

  try {
    await post('/admin/theme', {
      action: props.initialId ? 'update' : 'create',
      id,
      name: themeName.value.trim(),
      mode: themeMode.value,
      config,
    })
    saveSuccess.value = true
    setTimeout(() => { saveSuccess.value = false }, 3000)
    emit('saved', { id, name: themeName.value.trim(), mode: themeMode.value, config })
  } catch {
    // Error handled by useApi toast
  }
}

// ─── Format JSON ───

function formatJson() {
  try {
    const parsed = JSON.parse(jsonText.value)
    jsonText.value = JSON.stringify(parsed, null, 2)
    jsonError.value = ''
  } catch (e) {
    jsonError.value = e instanceof Error ? e.message : 'Invalid JSON'
  }
}

// Revert preview when component unmounts or json changes
watch(jsonText, () => {
  if (isPreviewing.value) {
    revertPreview()
  }
})
</script>

<template>
  <div class="theme-editor">
    <!-- Header fields -->
    <div class="theme-editor__header">
      <div class="theme-editor__field">
        <label class="theme-editor__label" for="theme-name">Theme Name</label>
        <input
          id="theme-name"
          v-model="themeName"
          type="text"
          class="theme-editor__input"
          placeholder="My Custom Theme"
        />
      </div>
      <div class="theme-editor__field theme-editor__field--small">
        <label class="theme-editor__label" for="theme-mode">Mode</label>
        <select id="theme-mode" v-model="themeMode" class="theme-editor__select">
          <option value="light">Light</option>
          <option value="dark">Dark</option>
        </select>
      </div>
    </div>

    <!-- Color Tokens -->
    <div class="theme-editor__colors">
      <span class="theme-editor__section-title">Color Tokens</span>
      <div class="theme-editor__color-grid">
        <div
          v-for="token in COLOR_TOKENS"
          :key="token.key"
          class="theme-editor__color-item"
        >
          <label class="theme-editor__color-label" :for="`color-${token.key}`">
            {{ token.label }}
          </label>
          <div class="theme-editor__color-input-wrap">
            <input
              :id="`color-${token.key}`"
              type="color"
              class="theme-editor__color-picker"
              :value="parsedConfig?.colors[token.key] || '#000000'"
              @input="updateColorFromPicker(token.key, ($event.target as HTMLInputElement).value)"
            />
            <input
              type="text"
              class="theme-editor__color-hex"
              :value="parsedConfig?.colors[token.key] || ''"
              placeholder="#000000"
              @change="updateColorFromPicker(token.key, ($event.target as HTMLInputElement).value)"
            />
          </div>
        </div>
      </div>
    </div>

    <!-- JSON Editor -->
    <div class="theme-editor__json-section">
      <div class="theme-editor__json-header">
        <span class="theme-editor__section-title">Config JSON</span>
        <button class="theme-editor__btn theme-editor__btn--ghost" @click="formatJson">
          Format
        </button>
      </div>
      <textarea
        v-model="jsonText"
        class="theme-editor__textarea"
        spellcheck="false"
        rows="18"
        @blur="validateJson"
      />
      <p v-if="jsonError" class="theme-editor__error" role="alert">
        {{ jsonError }}
      </p>
    </div>

    <!-- Preview Swatch -->
    <div class="theme-editor__preview-section">
      <span class="theme-editor__section-title">Preview</span>
      <div
        v-if="parsedConfig"
        class="theme-editor__swatch"
        :style="{
          background: parsedConfig.colors.background,
          borderRadius: parsedConfig.borderRadius,
          boxShadow: parsedConfig.shadows.md,
        }"
      >
        <div
          class="theme-editor__swatch-surface"
          :style="{
            background: parsedConfig.colors.surface,
            borderRadius: parsedConfig.borderRadius,
            border: `1px solid ${parsedConfig.colors.border}`,
          }"
        >
          <p class="theme-editor__swatch-title" :style="{ color: parsedConfig.colors.text }">
            {{ themeName || 'Custom Theme' }}
          </p>
          <p class="theme-editor__swatch-muted" :style="{ color: parsedConfig.colors.textMuted }">
            Sample muted text content
          </p>
          <div class="theme-editor__swatch-row">
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.primary, color: '#fff' }"
            >Primary</span>
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.accent, color: '#fff' }"
            >Accent</span>
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.success, color: '#fff' }"
            >Success</span>
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.error, color: '#fff' }"
            >Error</span>
          </div>
          <div class="theme-editor__swatch-row">
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.warning, color: '#000' }"
            >Warning</span>
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.info, color: '#fff' }"
            >Info</span>
            <span
              class="theme-editor__swatch-pill"
              :style="{ background: parsedConfig.colors.secondary, color: '#fff' }"
            >Secondary</span>
          </div>
        </div>
      </div>
      <p v-else class="theme-editor__error">Fix JSON errors to see preview</p>
    </div>

    <!-- Actions -->
    <div class="theme-editor__actions">
      <button
        class="theme-editor__btn theme-editor__btn--outline"
        :disabled="!parsedConfig"
        @click="isPreviewing ? revertPreview() : applyPreview()"
      >
        {{ isPreviewing ? 'Revert Preview' : 'Preview Live' }}
      </button>
      <button
        class="theme-editor__btn theme-editor__btn--primary"
        :disabled="loading || !parsedConfig"
        @click="saveTheme"
      >
        {{ loading ? 'Saving...' : saveSuccess ? 'Saved ✓' : 'Save Theme' }}
      </button>
    </div>
  </div>
</template>

<style scoped>
.theme-editor {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

/* ─── Header ─── */

.theme-editor__header {
  display: flex;
  gap: var(--space-3, 12px);
  align-items: flex-end;
}

.theme-editor__field {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
}

.theme-editor__field--small {
  flex: 0 0 120px;
}

.theme-editor__label {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-muted, #94a3b8);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.theme-editor__input,
.theme-editor__select {
  padding: var(--space-2, 8px) var(--space-3, 12px);
  border: 1px solid var(--color-border, #334155);
  border-radius: var(--radius-md, 6px);
  background: var(--color-surface, #1e293b);
  color: var(--color-text, #f1f5f9);
  font-size: 14px;
  transition: border-color 0.15s ease;
}

.theme-editor__input:focus,
.theme-editor__select:focus {
  outline: none;
  border-color: var(--color-primary, #60a5fa);
}

/* ─── Color Grid ─── */

.theme-editor__colors {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.theme-editor__section-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--color-muted, #94a3b8);
  text-transform: uppercase;
  letter-spacing: 0.03em;
}

.theme-editor__color-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: var(--space-2, 8px);
}

.theme-editor__color-item {
  display: flex;
  align-items: center;
  gap: var(--space-2, 8px);
}

.theme-editor__color-label {
  font-size: 12px;
  color: var(--color-text, #f1f5f9);
  min-width: 80px;
}

.theme-editor__color-input-wrap {
  display: flex;
  align-items: center;
  gap: 4px;
}

.theme-editor__color-picker {
  width: 28px;
  height: 28px;
  border: 1px solid var(--color-border, #334155);
  border-radius: 4px;
  cursor: pointer;
  padding: 0;
  background: none;
}

.theme-editor__color-picker::-webkit-color-swatch-wrapper {
  padding: 2px;
}

.theme-editor__color-picker::-webkit-color-swatch {
  border-radius: 2px;
  border: none;
}

.theme-editor__color-hex {
  width: 76px;
  padding: 4px 6px;
  font-size: 12px;
  font-family: monospace;
  border: 1px solid var(--color-border, #334155);
  border-radius: 4px;
  background: var(--color-surface, #1e293b);
  color: var(--color-text, #f1f5f9);
}

.theme-editor__color-hex:focus {
  outline: none;
  border-color: var(--color-primary, #60a5fa);
}

/* ─── JSON Editor ─── */

.theme-editor__json-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.theme-editor__json-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.theme-editor__textarea {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 1.5;
  padding: var(--space-3, 12px);
  border: 1px solid var(--color-border, #334155);
  border-radius: var(--radius-md, 6px);
  background: var(--color-surface, #1e293b);
  color: var(--color-text, #f1f5f9);
  resize: vertical;
  min-height: 200px;
  tab-size: 2;
}

.theme-editor__textarea:focus {
  outline: none;
  border-color: var(--color-primary, #60a5fa);
}

.theme-editor__error {
  font-size: 12px;
  color: var(--color-error, #f87171);
  margin: 0;
}

/* ─── Preview Swatch ─── */

.theme-editor__preview-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.theme-editor__swatch {
  padding: var(--space-4, 16px);
  border: 1px solid var(--color-border, #334155);
}

.theme-editor__swatch-surface {
  padding: var(--space-3, 12px);
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.theme-editor__swatch-title {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
}

.theme-editor__swatch-muted {
  margin: 0;
  font-size: 13px;
}

.theme-editor__swatch-row {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.theme-editor__swatch-pill {
  padding: 3px 10px;
  border-radius: 12px;
  font-size: 11px;
  font-weight: 500;
}

/* ─── Actions ─── */

.theme-editor__actions {
  display: flex;
  gap: var(--space-3, 12px);
  justify-content: flex-end;
}

.theme-editor__btn {
  padding: var(--space-2, 8px) var(--space-4, 16px);
  border-radius: var(--radius-md, 6px);
  font-size: 13px;
  font-weight: 500;
  cursor: pointer;
  transition: all 0.15s ease;
  border: none;
}

.theme-editor__btn:disabled {
  opacity: 0.5;
  cursor: not-allowed;
}

.theme-editor__btn--primary {
  background: var(--color-primary, #60a5fa);
  color: #fff;
}

.theme-editor__btn--primary:hover:not(:disabled) {
  opacity: 0.9;
}

.theme-editor__btn--outline {
  background: transparent;
  border: 1px solid var(--color-border, #334155);
  color: var(--color-text, #f1f5f9);
}

.theme-editor__btn--outline:hover:not(:disabled) {
  background: var(--color-surface-2, #334155);
}

.theme-editor__btn--ghost {
  background: transparent;
  color: var(--color-muted, #94a3b8);
  padding: var(--space-1, 4px) var(--space-2, 8px);
  font-size: 12px;
}

.theme-editor__btn--ghost:hover {
  color: var(--color-text, #f1f5f9);
}
</style>
