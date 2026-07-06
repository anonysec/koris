<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useBackups, type BackupRecord, type ManifestData } from '@/composables/useBackups'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import Button from '@koris/ui/Button.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import BackupSettings from '@/components/BackupSettings.vue'
import BackupRestoreDialog from '@/components/BackupRestoreDialog.vue'

const toast = useToast()
const { confirm } = useConfirm()
const {
  backups,
  loading,
  fetchBackups,
  createBackup,
  downloadBackup,
  verifyBackup,
  deleteBackup,
  restoreBackup,
  previewBackup,
} = useBackups()

const creating = ref(false)
const verifying = ref<number | null>(null)
const deleting = ref<number | null>(null)
const previewing = ref<number | null>(null)
const showRestoreDialog = ref(false)
const restoring = ref(false)
const expandedErrors = ref<Set<number>>(new Set())
const expandedErrorFull = ref<Set<number>>(new Set())
const expandedNodes = ref<Set<number>>(new Set())
const showPreviewModal = ref(false)
const previewData = ref<ManifestData | null>(null)
let pollInterval: ReturnType<typeof setInterval> | null = null

const hasInProgress = computed(() =>
  backups.value.some(b => b.status === 'in_progress')
)

function formatSize(bytes: number | null): string {
  if (!bytes) return '—'
  if (bytes < 1024) return `${bytes} B`
  if (bytes < 1024 * 1024) return `${(bytes / 1024).toFixed(1)} KB`
  if (bytes < 1024 * 1024 * 1024) return `${(bytes / (1024 * 1024)).toFixed(1)} MB`
  return `${(bytes / (1024 * 1024 * 1024)).toFixed(2)} GB`
}

function formatDate(iso: string | null): string {
  if (!iso) return '—'
  const d = new Date(iso)
  return d.toLocaleString()
}

function formatDuration(startedAt: string | null, completedAt: string | null, status: string): string {
  if (status === 'in_progress') return 'Running...'
  if (!completedAt) return '—'
  if (!startedAt) return '—'

  const start = new Date(startedAt).getTime()
  const end = new Date(completedAt).getTime()
  const diffSec = Math.floor((end - start) / 1000)

  if (diffSec < 0) return '—'
  if (diffSec < 60) return `${diffSec}s`
  if (diffSec < 3600) {
    const m = Math.floor(diffSec / 60)
    const s = diffSec % 60
    return `${m}m ${s}s`
  }
  const h = Math.floor(diffSec / 3600)
  const m = Math.floor((diffSec % 3600) / 60)
  return `${h}h ${m}m`
}

function statusLabel(status: string): string {
  switch (status) {
    case 'in_progress': return 'pending'
    case 'completed': return 'active'
    case 'failed': return 'failed'
    default: return status
  }
}

function toggleError(id: number) {
  if (expandedErrors.value.has(id)) {
    expandedErrors.value.delete(id)
    expandedErrorFull.value.delete(id)
  } else {
    expandedErrors.value.add(id)
  }
}

function toggleErrorFull(id: number) {
  if (expandedErrorFull.value.has(id)) {
    expandedErrorFull.value.delete(id)
  } else {
    expandedErrorFull.value.add(id)
  }
}

function toggleNodes(id: number) {
  if (expandedNodes.value.has(id)) {
    expandedNodes.value.delete(id)
  } else {
    expandedNodes.value.add(id)
  }
}

function getNodesSummary(backup: BackupRecord): string {
  const included = backup.nodes_included?.length ?? 0
  const skipped = backup.nodes_skipped?.length ?? 0
  if (included === 0 && skipped === 0) return '—'
  return `${included}`
}

function hasNodeDetails(backup: BackupRecord): boolean {
  return (backup.nodes_included?.length ?? 0) > 0 || (backup.nodes_skipped?.length ?? 0) > 0
}

async function handleCreate() {
  creating.value = true
  try {
    const id = await createBackup()
    if (id) {
      toast.success('Backup started')
      await fetchBackups()
      startPolling()
    }
  } catch {
    toast.error('Failed to start backup')
  } finally {
    creating.value = false
  }
}

async function handleVerify(backup: BackupRecord) {
  verifying.value = backup.id
  try {
    const valid = await verifyBackup(backup.id)
    if (valid) {
      toast.success(`Backup "${backup.filename}" integrity verified`)
    } else {
      toast.error(`Backup "${backup.filename}" integrity check failed`)
    }
  } catch {
    toast.error('Verification failed')
  } finally {
    verifying.value = null
  }
}

async function handleDelete(backup: BackupRecord) {
  const confirmed = await confirm({
    title: 'Delete Backup',
    message: `Are you sure you want to delete "${backup.filename}"? This cannot be undone.`,
    variant: 'danger',
    icon: '⚠',
    confirmText: 'Delete',
    cancelText: 'Cancel',
  })
  if (!confirmed) return

  deleting.value = backup.id
  try {
    const ok = await deleteBackup(backup.id)
    if (ok) {
      toast.success('Backup deleted')
      await fetchBackups()
    } else {
      toast.error('Failed to delete backup')
    }
  } catch {
    toast.error('Failed to delete backup')
  } finally {
    deleting.value = null
  }
}

async function handlePreview(backup: BackupRecord) {
  previewing.value = backup.id
  try {
    const manifest = await previewBackup(backup.id)
    if (manifest) {
      previewData.value = manifest
      showPreviewModal.value = true
    }
  } finally {
    previewing.value = null
  }
}

function closePreviewModal() {
  showPreviewModal.value = false
  previewData.value = null
}

function handlePreviewOverlayClick(event: MouseEvent) {
  if (event.target === event.currentTarget) {
    closePreviewModal()
  }
}

async function handleRestore(file: File) {
  restoring.value = true
  try {
    const ok = await restoreBackup(file)
    if (ok) {
      toast.success('Restore started successfully')
      showRestoreDialog.value = false
      await fetchBackups()
      startPolling()
    } else {
      toast.error('Restore failed')
    }
  } catch {
    toast.error('Restore failed')
  } finally {
    restoring.value = false
  }
}

function startPolling() {
  if (pollInterval) return
  pollInterval = setInterval(async () => {
    await fetchBackups()
    if (!hasInProgress.value) {
      stopPolling()
    }
  }, 3000)
}

function stopPolling() {
  if (pollInterval) {
    clearInterval(pollInterval)
    pollInterval = null
  }
}

onMounted(async () => {
  await fetchBackups()
  if (hasInProgress.value) {
    startPolling()
  }
})

onUnmounted(() => {
  stopPolling()
})
</script>

<template>
  <div class="page backup-view">
    <header class="page-header">
      <Button variant="primary" :loading="creating" @click="handleCreate">
        Create Backup Now
      </Button>
    </header>

    <!-- Settings -->
    <BackupSettings />

    <!-- Backup List -->
    <section class="backup-list-section">
      <h3 class="section-title">Backup History</h3>

      <div v-if="loading && backups.length === 0" class="backup-skeleton">
        <Skeleton v-for="i in 3" :key="i" variant="rect" width="100%" :height="48" />
      </div>

      <EmptyState
        v-else-if="backups.length === 0"
        icon="💾"
        title="No backups yet"
        description="Create your first backup to get started."
      />

      <div v-else class="backup-table-wrap">
        <table class="backup-table">
          <thead>
            <tr>
              <th>Filename</th>
              <th>Date</th>
              <th>Size</th>
              <th>Duration</th>
              <th>Status</th>
              <th>Nodes</th>
              <th>Actions</th>
            </tr>
          </thead>
          <tbody>
            <template v-for="backup in backups" :key="backup.id">
              <tr>
                <td class="cell-filename">{{ backup.filename }}</td>
                <td class="cell-date">{{ formatDate(backup.started_at) }}</td>
                <td class="cell-size">{{ formatSize(backup.size_bytes) }}</td>
                <td class="cell-duration">{{ formatDuration(backup.started_at, backup.completed_at, backup.status) }}</td>
                <td class="cell-status">
                  <StatusPill :status="statusLabel(backup.status)" size="sm" />
                  <span v-if="backup.status === 'in_progress'" class="spinner" aria-label="In progress" />
                  <button
                    v-if="backup.status === 'failed' && backup.error_message"
                    class="error-toggle"
                    :aria-expanded="expandedErrors.has(backup.id)"
                    aria-label="Toggle error details"
                    @click="toggleError(backup.id)"
                  >
                    {{ expandedErrors.has(backup.id) ? '▾' : '▸' }}
                  </button>
                </td>
                <td class="cell-nodes">
                  <button
                    v-if="hasNodeDetails(backup)"
                    class="nodes-toggle"
                    :aria-expanded="expandedNodes.has(backup.id)"
                    aria-label="Toggle node details"
                    @click="toggleNodes(backup.id)"
                  >
                    {{ getNodesSummary(backup) }}
                    <span class="nodes-toggle-icon">{{ expandedNodes.has(backup.id) ? '▾' : '▸' }}</span>
                  </button>
                  <span v-else>{{ getNodesSummary(backup) }}</span>
                </td>
                <td class="cell-actions">
                  <Button
                    v-if="backup.status === 'completed'"
                    variant="ghost"
                    size="sm"
                    @click="downloadBackup(backup.id)"
                  >
                    Download
                  </Button>
                  <Button
                    v-if="backup.status === 'completed'"
                    variant="ghost"
                    size="sm"
                    :loading="verifying === backup.id"
                    @click="handleVerify(backup)"
                  >
                    Verify
                  </Button>
                  <Button
                    v-if="backup.status === 'completed'"
                    variant="ghost"
                    size="sm"
                    :loading="previewing === backup.id"
                    @click="handlePreview(backup)"
                  >
                    Preview
                  </Button>
                  <Button
                    variant="ghost"
                    size="sm"
                    :loading="deleting === backup.id"
                    :disabled="backup.status === 'in_progress'"
                    @click="handleDelete(backup)"
                  >
                    Delete
                  </Button>
                </td>
              </tr>

              <!-- Error Details Row -->
              <tr v-if="expandedErrors.has(backup.id) && backup.error_message" class="error-row">
                <td colspan="7">
                  <div class="error-content">
                    <span class="error-label">Error:</span>
                    <span class="error-message">
                      <template v-if="backup.error_message.length > 200 && !expandedErrorFull.has(backup.id)">
                        {{ backup.error_message.slice(0, 200) }}…
                        <button class="show-more-btn" @click="toggleErrorFull(backup.id)">show more</button>
                      </template>
                      <template v-else-if="backup.error_message.length > 200 && expandedErrorFull.has(backup.id)">
                        {{ backup.error_message }}
                        <button class="show-more-btn" @click="toggleErrorFull(backup.id)">show less</button>
                      </template>
                      <template v-else>
                        {{ backup.error_message }}
                      </template>
                    </span>
                  </div>
                </td>
              </tr>

              <!-- Node Details Row -->
              <tr v-if="expandedNodes.has(backup.id)" class="nodes-row">
                <td colspan="7">
                  <div class="nodes-content">
                    <div v-if="backup.nodes_included && backup.nodes_included.length > 0" class="nodes-section">
                      <span class="nodes-label">Included:</span>
                      <span class="nodes-list">{{ backup.nodes_included.join(', ') }}</span>
                    </div>
                    <div v-if="backup.nodes_skipped && backup.nodes_skipped.length > 0" class="nodes-section">
                      <span class="nodes-label">Skipped:</span>
                      <span
                        v-for="(node, idx) in backup.nodes_skipped"
                        :key="idx"
                        class="skipped-node"
                      >
                        {{ node.name }} <span class="skipped-reason">({{ node.reason }})</span>{{ idx < backup.nodes_skipped.length - 1 ? ', ' : '' }}
                      </span>
                    </div>
                  </div>
                </td>
              </tr>
            </template>
          </tbody>
        </table>
      </div>

      <!-- Restore Button -->
      <div class="restore-section">
        <Button variant="ghost" @click="showRestoreDialog = true">
          Restore from File
        </Button>
      </div>
    </section>

    <!-- Restore Dialog -->
    <BackupRestoreDialog
      :open="showRestoreDialog"
      :loading="restoring"
      @confirm="handleRestore"
      @cancel="showRestoreDialog = false"
    />

    <!-- Preview Modal -->
    <Teleport to="body">
      <Transition name="preview-modal">
        <div
          v-if="showPreviewModal && previewData"
          class="preview-modal__overlay"
          role="dialog"
          aria-modal="true"
          aria-labelledby="preview-modal-title"
          @click="handlePreviewOverlayClick"
          @keydown.escape="closePreviewModal"
        >
          <div class="preview-modal">
            <header class="preview-modal__header">
              <h2 id="preview-modal-title" class="preview-modal__title">Backup Preview</h2>
              <button class="preview-modal__close" aria-label="Close preview" @click="closePreviewModal">✕</button>
            </header>
            <div class="preview-modal__content">
              <pre class="preview-modal__json">{{ JSON.stringify(previewData, null, 2) }}</pre>
            </div>
          </div>
        </div>
      </Transition>
    </Teleport>
  </div>
</template>

<style scoped>
.backup-view {
  display: flex;
  flex-direction: column;
  gap: var(--space-5);
}

.page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.section-title {
  margin: 0 0 var(--space-3);
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

.backup-list-section {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.backup-skeleton {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.backup-table-wrap {
  overflow-x: auto;
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
}

.backup-table {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--text-sm);
}

.backup-table th {
  text-align: left;
  padding: var(--space-3) var(--space-3);
  background: var(--color-surface);
  border-bottom: 1px solid var(--color-border);
  font-weight: var(--font-semibold);
  color: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.backup-table td {
  padding: var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
  color: var(--color-text);
}

.backup-table tr:last-child td {
  border-bottom: none;
}

.cell-filename {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-xs);
}

.cell-date {
  white-space: nowrap;
}

.cell-size {
  white-space: nowrap;
}

.cell-duration {
  white-space: nowrap;
  color: var(--color-muted);
}

.cell-status {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.cell-nodes {
  white-space: nowrap;
}

.cell-actions {
  display: flex;
  gap: var(--space-1);
  flex-wrap: wrap;
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

/* Error toggle button */
.error-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  background: none;
  border: none;
  color: var(--color-danger, #ef4444);
  cursor: pointer;
  font-size: var(--text-xs);
  padding: 2px 4px;
  border-radius: var(--radius-sm);
}

.error-toggle:hover {
  background: rgba(239, 68, 68, 0.1);
}

/* Error row */
.error-row td {
  padding: 0 var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.error-content {
  padding: var(--space-2) var(--space-3);
  background: rgba(239, 68, 68, 0.06);
  border: 1px solid rgba(239, 68, 68, 0.2);
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  line-height: var(--leading-normal);
}

.error-label {
  font-weight: var(--font-semibold);
  color: var(--color-danger, #ef4444);
  margin-right: var(--space-2);
}

.error-message {
  color: var(--color-text);
  word-break: break-word;
}

.show-more-btn {
  background: none;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  font-size: var(--text-xs);
  padding: 0;
  text-decoration: underline;
  margin-left: var(--space-1);
}

.show-more-btn:hover {
  opacity: 0.8;
}

/* Node toggle button */
.nodes-toggle {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  background: none;
  border: none;
  color: var(--color-primary);
  cursor: pointer;
  font-size: var(--text-sm);
  padding: 2px 4px;
  border-radius: var(--radius-sm);
}

.nodes-toggle:hover {
  background: rgba(37, 99, 235, 0.1);
}

.nodes-toggle-icon {
  font-size: var(--text-xs);
}

/* Node details row */
.nodes-row td {
  padding: 0 var(--space-3) var(--space-3);
  border-bottom: 1px solid var(--color-border);
}

.nodes-content {
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface-2, #1e2630);
  border-radius: var(--radius-md);
  font-size: var(--text-xs);
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.nodes-section {
  display: flex;
  align-items: baseline;
  gap: var(--space-2);
  flex-wrap: wrap;
}

.nodes-label {
  font-weight: var(--font-semibold);
  color: var(--color-muted);
  flex-shrink: 0;
}

.nodes-list {
  color: var(--color-text);
}

.skipped-node {
  color: var(--color-text);
}

.skipped-reason {
  color: var(--color-muted);
  font-style: italic;
}

/* Restore section */
.restore-section {
  padding-top: var(--space-3);
  border-top: 1px solid var(--color-border);
}

/* Preview Modal */
.preview-modal__overlay {
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

.preview-modal {
  background-color: var(--color-surface, #0b1120);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-xl, 14px);
  max-width: 640px;
  width: 100%;
  max-height: 80vh;
  display: flex;
  flex-direction: column;
  box-shadow: var(--shadow-xl, 0 30px 80px rgba(0, 0, 0, 0.6));
}

.preview-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--color-border);
  flex-shrink: 0;
}

.preview-modal__title {
  font-size: var(--text-lg);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  margin: 0;
}

.preview-modal__close {
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

.preview-modal__close:hover {
  background: var(--color-surface-2, #1e2630);
  color: var(--color-text);
}

.preview-modal__content {
  overflow-y: auto;
  padding: var(--space-4) var(--space-5);
  flex: 1;
}

.preview-modal__json {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-xs);
  line-height: var(--leading-relaxed, 1.6);
  color: var(--color-text);
  white-space: pre-wrap;
  word-break: break-word;
  margin: 0;
}

/* Preview modal transition */
.preview-modal-enter-active,
.preview-modal-leave-active {
  transition: opacity 0.2s ease-out;
}

.preview-modal-enter-active .preview-modal,
.preview-modal-leave-active .preview-modal {
  transition: transform 0.2s ease-out, opacity 0.2s ease-out;
}

.preview-modal-enter-from,
.preview-modal-leave-to {
  opacity: 0;
}

.preview-modal-enter-from .preview-modal,
.preview-modal-leave-to .preview-modal {
  transform: scale(0.95);
  opacity: 0;
}

@media (max-width: 768px) {
  .page-header {
    flex-direction: column;
    align-items: stretch;
  }

  .cell-actions {
    flex-direction: column;
  }
}
</style>
