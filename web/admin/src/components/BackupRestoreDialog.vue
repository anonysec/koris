<script setup lang="ts">
import { ref, watch } from 'vue'
import KButton from '@koris/ui/KButton.vue'

const props = defineProps<{
  open: boolean
  loading?: boolean
}>()

const emit = defineEmits<{
  confirm: [file: File]
  cancel: []
}>()

const selectedFile = ref<File | null>(null)
const fileInputRef = ref<HTMLInputElement | null>(null)

watch(() => props.open, (isOpen) => {
  if (!isOpen) {
    selectedFile.value = null
    if (fileInputRef.value) {
      fileInputRef.value.value = ''
    }
  }
})

function handleFileChange(event: Event) {
  const target = event.target as HTMLInputElement
  if (target.files && target.files.length > 0) {
    selectedFile.value = target.files[0]
  }
}

function formatSize(bytes: number): string {
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
}

function handleConfirm() {
  if (selectedFile.value) {
    emit('confirm', selectedFile.value)
  }
}

function handleCancel() {
  emit('cancel')
}

function handleOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    handleCancel()
  }
}
</script>

<template>
  <Teleport to="body">
    <Transition name="restore-dialog">
      <div
        v-if="open"
        class="restore-dialog__overlay"
        role="dialog"
        aria-modal="true"
        aria-labelledby="restore-dialog-title"
        @click="handleOverlayClick"
      >
        <div class="restore-dialog">
          <h2 id="restore-dialog-title" class="restore-dialog__title">
            Restore from Backup
          </h2>

          <div class="restore-dialog__warning">
            <span class="restore-dialog__warning-icon">⚠️</span>
            <p>This will overwrite the current database. A safety backup will be created first.</p>
          </div>

          <div class="restore-dialog__upload">
            <label class="restore-dialog__label" for="restore-file-input">
              Select backup file (.tar.gz)
            </label>
            <input
              id="restore-file-input"
              ref="fileInputRef"
              type="file"
              accept=".tar.gz,.gz"
              class="restore-dialog__file-input"
              @change="handleFileChange"
            />
          </div>

          <div v-if="selectedFile" class="restore-dialog__file-info">
            <span class="restore-dialog__filename">{{ selectedFile.name }}</span>
            <span class="restore-dialog__filesize">{{ formatSize(selectedFile.size) }}</span>
          </div>

          <div v-if="loading" class="restore-dialog__progress">
            <span class="spinner" />
            <span>Restoring...</span>
          </div>

          <div class="restore-dialog__actions">
            <KButton variant="ghost" :disabled="loading" @click="handleCancel">
              Cancel
            </KButton>
            <KButton
              variant="primary"
              :disabled="!selectedFile || loading"
              :loading="loading"
              @click="handleConfirm"
            >
              Restore
            </KButton>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.restore-dialog__overlay {
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

.restore-dialog {
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-xl, 14px);
  padding: var(--space-6, 24px);
  max-width: 480px;
  width: 100%;
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.restore-dialog__title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.restore-dialog__warning {
  display: flex;
  align-items: flex-start;
  gap: var(--space-2);
  padding: var(--space-3);
  background: rgba(245, 158, 11, 0.08);
  border: 1px solid rgba(245, 158, 11, 0.25);
  border-radius: var(--radius-md);
}

.restore-dialog__warning-icon {
  font-size: 1.2rem;
  flex-shrink: 0;
}

.restore-dialog__warning p {
  margin: 0;
  font-size: var(--text-sm);
  color: var(--color-warning);
  line-height: var(--leading-normal);
}

.restore-dialog__upload {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.restore-dialog__label {
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
  color: var(--color-text);
}

.restore-dialog__file-input {
  font-size: var(--text-sm);
  padding: var(--space-2);
  color: var(--color-text);
}

.restore-dialog__file-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface-2, #1e2630);
  border-radius: var(--radius-md);
}

.restore-dialog__filename {
  font-size: var(--text-sm);
  font-family: var(--font-mono, monospace);
  color: var(--color-text);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.restore-dialog__filesize {
  font-size: var(--text-xs);
  color: var(--color-muted);
  flex-shrink: 0;
  margin-left: var(--space-2);
}

.restore-dialog__progress {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.spinner {
  display: inline-block;
  width: 14px;
  height: 14px;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  border-radius: 50%;
  animation: spin 0.8s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.restore-dialog__actions {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-3);
  padding-top: var(--space-2);
}

/* Transition */
.restore-dialog-enter-active,
.restore-dialog-leave-active {
  transition: opacity 0.2s ease-out;
}

.restore-dialog-enter-active .restore-dialog,
.restore-dialog-leave-active .restore-dialog {
  transition: transform 0.2s ease-out, opacity 0.2s ease-out;
}

.restore-dialog-enter-from,
.restore-dialog-leave-to {
  opacity: 0;
}

.restore-dialog-enter-from .restore-dialog,
.restore-dialog-leave-to .restore-dialog {
  transform: scale(0.95);
  opacity: 0;
}
</style>
