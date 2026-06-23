<script setup lang="ts">
import { ref } from 'vue'
import { useApi } from '@koris/composables/useApi'
import { useToast } from '@koris/composables/useToast'
import { useConfirm } from '@koris/composables/useConfirm'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'

const props = defineProps<{
  customerId: number
  uuid: string
  syncStatus?: 'synced' | 'pending'
}>()

const emit = defineEmits<{
  'uuid-regenerated': [newUuid: string]
}>()

const { t } = useI18n()
const api = useApi({ baseUrl: '/api/admin' })
const toast = useToast()
const { confirm } = useConfirm()

const regenerating = ref(false)
const copied = ref(false)

async function copyUuid() {
  try {
    await navigator.clipboard.writeText(props.uuid)
    copied.value = true
    toast.success(t('xray.uuid_copied'))
    setTimeout(() => { copied.value = false }, 2000)
  } catch {
    toast.error(t('xray.copy_failed'))
  }
}

async function handleRegenerate() {
  const confirmed = await confirm({
    title: t('xray.regenerate_uuid_title'),
    message: t('xray.regenerate_uuid_msg'),
    variant: 'danger',
    icon: '⚠',
    confirmText: t('xray.regenerate'),
    cancelText: t('btn.cancel'),
  })
  if (!confirmed) return

  regenerating.value = true
  try {
    const res = await api.post<{ ok: boolean; uuid: string }>(
      `/customers/${props.customerId}/xray-uuid/regenerate`
    )
    if (res.ok) {
      toast.success(t('xray.uuid_regenerated'))
      emit('uuid-regenerated', res.uuid)
    }
  } catch {
    // Handled by useApi
  } finally {
    regenerating.value = false
  }
}
</script>

<template>
  <div class="xray-uuid">
    <div class="uuid-header">
      <h4>{{ t('xray.user_uuid') }}</h4>
      <span
        class="sync-badge"
        :class="`sync-badge--${syncStatus || 'pending'}`"
      >
        {{ syncStatus === 'synced' ? t('xray.synced') : t('xray.pending_sync') }}
      </span>
    </div>

    <div class="uuid-display">
      <code class="uuid-value">{{ uuid || t('xray.no_uuid') }}</code>
      <div class="uuid-actions">
        <KButton
          variant="ghost"
          size="sm"
          :disabled="!uuid"
          @click="copyUuid"
        >
          {{ copied ? '✓' : '📋' }} {{ t('xray.copy') }}
        </KButton>
        <KButton
          variant="ghost"
          size="sm"
          :loading="regenerating"
          @click="handleRegenerate"
        >
          🔄 {{ t('xray.regenerate') }}
        </KButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.xray-uuid {
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  padding: var(--space-4);
  background: var(--color-surface);
}

.uuid-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: var(--space-3);
}

.uuid-header h4 {
  margin: 0;
  font-size: var(--text-sm);
  font-weight: 600;
  color: var(--color-text);
}

.sync-badge {
  font-size: var(--text-xs);
  font-weight: 500;
  padding: 2px 8px;
  border-radius: var(--radius-full, 9999px);
}

.sync-badge--synced {
  color: #22c55e;
  background: rgba(34, 197, 94, 0.1);
}

.sync-badge--pending {
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.1);
}

.uuid-display {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.uuid-value {
  font-family: var(--font-mono, monospace);
  font-size: var(--text-sm);
  color: var(--color-text);
  background: var(--color-surface-2, rgba(0, 0, 0, 0.1));
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  word-break: break-all;
}

.uuid-actions {
  display: flex;
  gap: var(--space-1);
  flex-shrink: 0;
}
</style>
