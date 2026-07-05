<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useWireGuard } from '@/composables/useWireGuard'
import { useNodesStore } from '@/stores/nodes'
import { useToast } from '@koris/composables/useToast'
import { useI18n } from '@koris/composables/useI18n'
import KButton from '@koris/ui/KButton.vue'
import KFormField from '@koris/ui/KFormField.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KInput from '@koris/ui/KInput.vue'

const emit = defineEmits<{
  close: []
  created: []
}>()

const { t } = useI18n()
const toast = useToast()
const { createPeer } = useWireGuard()
const nodesStore = useNodesStore()

const selectedNode = ref<string>('')
const customerSearch = ref('')
const selectedCustomerId = ref<number | undefined>(undefined)
const creating = ref(false)
const assignedIp = ref<string | null>(null)

// Only show WireGuard-enabled nodes
const wireGuardNodes = computed(() => {
  return nodesStore.list.filter(node => {
    const configs = nodesStore.vpnConfigs[node.id]
    if (!configs) return false
    return configs.some(c => c.protocol === 'wireguard' && c.enabled)
  })
})

const nodeOptions = computed(() =>
  wireGuardNodes.value.map(n => ({ label: n.name, value: String(n.id) }))
)

async function handleCreate() {
  if (!selectedNode.value) {
    toast.error(t('wireguard.select_node_required'))
    return
  }
  creating.value = true
  const data: { node_id: number; customer_id?: number } = {
    node_id: Number(selectedNode.value),
  }
  if (selectedCustomerId.value) {
    data.customer_id = selectedCustomerId.value
  }
  const peer = await createPeer(data)
  creating.value = false
  if (peer) {
    assignedIp.value = peer.allowed_ips
    toast.success(t('wireguard.peer_created'))
    emit('created')
  } else {
    toast.error(t('wireguard.peer_create_error'))
  }
}

function handleClose() {
  emit('close')
}

onMounted(() => {
  // Ensure node configs are loaded so we can filter by WireGuard-enabled
  nodesStore.list.forEach(node => nodesStore.loadNodeVpnConfigs(node.id))
})
</script>

<template>
  <div class="dialog-overlay" @click.self="handleClose">
    <div class="dialog" role="dialog" :aria-label="t('wireguard.create_peer')">
      <div class="dialog-header">
        <h3>{{ t('wireguard.create_peer') }}</h3>
        <button class="dialog-close" @click="handleClose" :aria-label="t('btn.close')">×</button>
      </div>

      <div class="dialog-body">
        <!-- Success state: show assigned IP -->
        <div v-if="assignedIp" class="success-panel">
          <p class="success-text">{{ t('wireguard.peer_created_success') }}</p>
          <div class="assigned-ip">
            <span class="assigned-ip__label">{{ t('wireguard.assigned_ip') }}:</span>
            <code class="assigned-ip__value">{{ assignedIp }}</code>
          </div>
          <KButton variant="primary" @click="handleClose">{{ t('btn.close') }}</KButton>
        </div>

        <!-- Create form -->
        <form v-else @submit.prevent="handleCreate">
          <div class="form-stack">
            <KFormField name="peer-node" :label="t('wireguard.select_node')" required>
              <template #default="{ fieldId }">
                <KSelect
                  :id="fieldId"
                  v-model="selectedNode"
                  :options="nodeOptions"
                  :placeholder="t('wireguard.select_node_placeholder')"
                />
              </template>
            </KFormField>

            <KFormField name="peer-customer" :label="t('wireguard.customer_optional')">
              <template #default="{ fieldId }">
                <KInput
                  :id="fieldId"
                  v-model="customerSearch"
                  :placeholder="t('wireguard.customer_search_placeholder')"
                />
              </template>
            </KFormField>

            <p class="hint-text">{{ t('wireguard.ip_auto_assigned_hint') }}</p>
          </div>

          <div class="dialog-actions">
            <KButton variant="ghost" @click="handleClose">{{ t('btn.cancel') }}</KButton>
            <KButton type="submit" variant="primary" :loading="creating">
              {{ t('wireguard.create_peer') }}
            </KButton>
          </div>
        </form>
      </div>
    </div>
  </div>
</template>

<style scoped>
.dialog-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.dialog {
  background: var(--color-surface);
  border-radius: var(--radius-lg);
  width: 100%;
  max-width: 440px;
  box-shadow: var(--shadow-lg);
}

.dialog-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-4);
  border-bottom: 1px solid var(--color-border);
}

.dialog-header h3 {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}

.dialog-close {
  border: none;
  background: none;
  font-size: 1.5rem;
  cursor: pointer;
  color: var(--color-muted);
  padding: 0;
  line-height: 1;
}

.dialog-close:hover {
  color: var(--color-text);
}

.dialog-body {
  padding: var(--space-4);
}

.form-stack {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
}

.hint-text {
  font-size: var(--text-xs);
  color: var(--color-muted);
  margin: 0;
}

.dialog-actions {
  display: flex;
  justify-content: flex-end;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}

.success-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: var(--space-4);
  padding: var(--space-4) 0;
}

.success-text {
  font-size: var(--text-sm);
  color: var(--color-success, #22c55e);
  font-weight: var(--font-medium);
  margin: 0;
}

.assigned-ip {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3);
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
}

.assigned-ip__label {
  font-size: var(--text-sm);
  color: var(--color-muted);
}

.assigned-ip__value {
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
}
</style>
