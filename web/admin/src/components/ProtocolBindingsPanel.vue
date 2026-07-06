<script setup lang="ts">
import { ref, computed, onMounted, watch, nextTick, onBeforeUnmount } from 'vue'
import Sortable from 'sortablejs'
import { useDomainsStore, type ProtocolBinding, type VpnDomain } from '@/stores/domains'
import { useConfirm } from '@koris/composables/useConfirm'
import { useToast } from '@koris/composables/useToast'
import Button from '@koris/ui/Button.vue'
import Modal from '@koris/ui/Modal.vue'
import StatusPill from '@koris/ui/StatusPill.vue'

/**
 * ProtocolBindingsPanel — Per-node protocol binding management.
 *
 * Displays bindings grouped by protocol with drag-and-drop reordering,
 * add/remove domain actions, and blocked domain warnings.
 *
 * Requirements: 7.4, 7.5, 8.2, 8.4
 */

const props = defineProps<{
  nodeId: number
}>()

const store = useDomainsStore()
const { confirm } = useConfirm()
const toast = useToast()

// ─── State ──────────────────────────────────────────────────────────────────

const loading = ref(false)
const bindings = ref<ProtocolBinding[]>([])
const domains = ref<VpnDomain[]>([])

// Domain picker modal state
const showDomainPicker = ref(false)
const pickerProtocol = ref('')

// Sortable instances per protocol group
const sortableInstances = new Map<string, Sortable>()

// ─── Computed ───────────────────────────────────────────────────────────────

/** All supported protocols */
const protocols = [
  'openvpn-udp',
  'openvpn-tcp',
  'l2tp',
  'ikev2',
  'wireguard',
  'ssh',
  'mtproto',
] as const

/** Group bindings by protocol */
const bindingsByProtocol = computed(() => {
  const grouped: Record<string, ProtocolBinding[]> = {}
  for (const proto of protocols) {
    grouped[proto] = bindings.value
      .filter((b) => b.protocol === proto)
      .sort((a, b) => a.position - b.position)
  }
  return grouped
})

/** All protocols displayed in the panel (user can add domains to any) */
const allProtocols = computed(() => [...protocols])

/** Active domains available in the picker (only status 'active', not already bound to the selected protocol) */
const availableDomains = computed(() => {
  const boundDomainIds = new Set(
    bindings.value
      .filter((b) => b.protocol === pickerProtocol.value)
      .map((b) => b.domain_id)
  )
  return domains.value.filter(
    (d) => d.status === 'active' && !boundDomainIds.has(d.id)
  )
})

// ─── Data Loading ───────────────────────────────────────────────────────────

async function loadData() {
  loading.value = true
  try {
    const [fetchedBindings] = await Promise.all([
      store.fetchBindings(props.nodeId),
      store.fetchDomains(),
    ])
    bindings.value = fetchedBindings
    domains.value = store.domains
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadData()
})

watch(() => props.nodeId, () => {
  loadData()
})

// ─── Drag-and-Drop Reorder ──────────────────────────────────────────────────

function initSortable(protocol: string, el: HTMLElement) {
  // Destroy any existing instance for this protocol
  const existing = sortableInstances.get(protocol)
  if (existing) {
    existing.destroy()
  }

  const instance = Sortable.create(el, {
    handle: '.binding-drag-handle',
    animation: 150,
    ghostClass: 'sortable-ghost',
    chosenClass: 'sortable-chosen',
    dragClass: 'sortable-drag',
    onEnd(evt) {
      const { oldIndex, newIndex } = evt
      if (oldIndex == null || newIndex == null || oldIndex === newIndex) return

      // Reorder the local bindings for this protocol
      const protoBindings = [...bindingsByProtocol.value[protocol]]
      const [moved] = protoBindings.splice(oldIndex, 1)
      protoBindings.splice(newIndex, 0, moved)

      // Update local state
      const otherBindings = bindings.value.filter((b) => b.protocol !== protocol)
      const reorderedBindings = protoBindings.map((b, i) => ({ ...b, position: i + 1 }))
      bindings.value = [...otherBindings, ...reorderedBindings]

      // Persist to backend
      const bindingIds = protoBindings.map((b) => b.id)
      store.reorderBindings(props.nodeId, { binding_ids: bindingIds })
    },
  })

  sortableInstances.set(protocol, instance)
}

/** Called via template ref callback to set up sortable on each protocol list */
function onProtocolListMounted(protocol: string) {
  return (el: HTMLElement | null) => {
    if (el) {
      nextTick(() => initSortable(protocol, el))
    }
  }
}

onBeforeUnmount(() => {
  for (const instance of sortableInstances.values()) {
    instance.destroy()
  }
  sortableInstances.clear()
})

// ─── Add Domain ─────────────────────────────────────────────────────────────

function openDomainPicker(protocol: string) {
  pickerProtocol.value = protocol
  showDomainPicker.value = true
}

async function addDomain(domain: VpnDomain) {
  const protoBindings = bindingsByProtocol.value[pickerProtocol.value]
  const nextPosition = protoBindings.length + 1

  const success = await store.createBinding(props.nodeId, {
    protocol: pickerProtocol.value,
    domain_id: domain.id,
    position: nextPosition,
  })

  if (success) {
    toast.success(`Added ${domain.name} to ${pickerProtocol.value}`)
    showDomainPicker.value = false
    await refreshBindings()
  } else {
    toast.error('Failed to add domain binding')
  }
}

// ─── Remove Domain ──────────────────────────────────────────────────────────

async function removeBinding(binding: ProtocolBinding) {
  const confirmed = await confirm({
    title: 'Remove Domain Binding',
    message: `Remove "${binding.domain_name}" from ${binding.protocol}? Remaining positions will be re-sequenced.`,
    variant: 'danger',
    confirmText: 'Remove',
  })

  if (!confirmed) return

  const success = await store.deleteBinding(props.nodeId, binding.id)
  if (success) {
    toast.success(`Removed ${binding.domain_name} from ${binding.protocol}`)
    await refreshBindings()
  } else {
    toast.error('Failed to remove domain binding')
  }
}

// ─── Helpers ────────────────────────────────────────────────────────────────

async function refreshBindings() {
  const fetched = await store.fetchBindings(props.nodeId)
  bindings.value = fetched
}

function isBlocked(binding: ProtocolBinding): boolean {
  return binding.domain_status === 'blocked'
}

function formatProtocolLabel(protocol: string): string {
  return protocol
    .split('-')
    .map((part) => part.charAt(0).toUpperCase() + part.slice(1))
    .join(' ')
}
</script>

<template>
  <div class="protocol-bindings-panel">
    <!-- Header -->
    <header class="panel-header">
      <h3 class="panel-title">Protocol Bindings</h3>
      <p class="panel-subtitle">Assign domains to protocols with failover priority. Drag to reorder.</p>
    </header>

    <!-- Loading State -->
    <div v-if="loading" class="panel-loading">
      <span class="panel-loading__spinner" aria-hidden="true" />
      Loading bindings…
    </div>

    <!-- Protocol Groups -->
    <div v-else class="protocol-groups">
      <div
        v-for="protocol in allProtocols"
        :key="protocol"
        class="protocol-group"
      >
        <div class="protocol-group__header">
          <span class="protocol-group__label">{{ formatProtocolLabel(protocol) }}</span>
          <span class="protocol-group__count">
            {{ bindingsByProtocol[protocol].length }} domain{{ bindingsByProtocol[protocol].length !== 1 ? 's' : '' }}
          </span>
          <Button
            variant="ghost"
            size="sm"
            icon="+"
            :aria-label="`Add domain to ${protocol}`"
            @click="openDomainPicker(protocol)"
          >
            Add
          </Button>
        </div>

        <!-- Bindings List (sortable) -->
        <div
          v-if="bindingsByProtocol[protocol].length > 0"
          :ref="onProtocolListMounted(protocol)"
          class="binding-list"
          role="list"
          :aria-label="`${formatProtocolLabel(protocol)} domain bindings`"
        >
          <div
            v-for="binding in bindingsByProtocol[protocol]"
            :key="binding.id"
            class="binding-item"
            :class="{ 'binding-item--blocked': isBlocked(binding) }"
            role="listitem"
          >
            <!-- Drag Handle -->
            <button
              class="binding-drag-handle"
              type="button"
              :aria-label="`Drag to reorder ${binding.domain_name}`"
              tabindex="0"
            >
              <span class="binding-drag-handle__icon" aria-hidden="true">⋮⋮</span>
            </button>

            <!-- Position Badge -->
            <span class="binding-position" :aria-label="`Priority ${binding.position}`">
              {{ binding.position }}
            </span>

            <!-- Domain Info -->
            <div class="binding-info">
              <span class="binding-domain-name">{{ binding.domain_name }}</span>
              <span class="binding-domain-ip">{{ binding.domain_ip }}</span>
            </div>

            <!-- Status / Warning -->
            <div class="binding-status">
              <StatusPill
                v-if="isBlocked(binding)"
                status="blocked"
                size="sm"
              />
              <span
                v-if="isBlocked(binding)"
                class="binding-warning"
                role="alert"
                aria-label="Domain is blocked — clients may experience connectivity issues"
                title="Domain is blocked — clients may experience connectivity issues"
              >
                ⚠️
              </span>
            </div>

            <!-- Remove Button -->
            <Button
              variant="ghost"
              size="sm"
              icon="×"
              :aria-label="`Remove ${binding.domain_name} from ${protocol}`"
              @click="removeBinding(binding)"
            />
          </div>
        </div>

        <!-- Empty State -->
        <div v-else class="binding-list-empty">
          <span class="binding-list-empty__text">No domains assigned. Falls back to node IP.</span>
        </div>
      </div>
    </div>

    <!-- Domain Picker Modal -->
    <Modal
      :open="showDomainPicker"
      :title="`Add Domain to ${formatProtocolLabel(pickerProtocol)}`"
      width="440px"
      @close="showDomainPicker = false"
    >
      <div class="domain-picker">
        <p v-if="availableDomains.length === 0" class="domain-picker__empty">
          No active domains available to add.
        </p>
        <div v-else class="domain-picker__list" role="list" aria-label="Available domains">
          <button
            v-for="domain in availableDomains"
            :key="domain.id"
            type="button"
            class="domain-picker__item"
            role="listitem"
            @click="addDomain(domain)"
          >
            <span class="domain-picker__name">{{ domain.name }}</span>
            <span class="domain-picker__ip">{{ domain.ip_address }}</span>
          </button>
        </div>
      </div>
    </Modal>
  </div>
</template>

<style scoped>
.protocol-bindings-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-5, 20px);
}

/* ─── Header ─── */

.panel-header {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
}

.panel-title {
  margin: 0;
  font-size: var(--text-lg, 16px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-text, #e6edf3);
}

.panel-subtitle {
  margin: 0;
  font-size: var(--text-sm, 12px);
  color: var(--color-muted, #8b98a5);
}

/* ─── Loading ─── */

.panel-loading {
  display: flex;
  align-items: center;
  gap: var(--space-2, 8px);
  padding: var(--space-6, 24px);
  color: var(--color-muted, #8b98a5);
  font-size: var(--text-sm, 12px);
}

.panel-loading__spinner {
  width: 16px;
  height: 16px;
  border: 2px solid var(--color-border, #28333f);
  border-top-color: var(--color-primary, #2563eb);
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

/* ─── Protocol Groups ─── */

.protocol-groups {
  display: flex;
  flex-direction: column;
  gap: var(--space-4, 16px);
}

.protocol-group {
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-lg, 10px);
  overflow: hidden;
}

.protocol-group__header {
  display: flex;
  align-items: center;
  gap: var(--space-3, 12px);
  padding: var(--space-3, 12px) var(--space-4, 16px);
  background: var(--color-surface-2, #1e2630);
  border-bottom: 1px solid var(--color-border, #28333f);
}

.protocol-group__label {
  font-size: var(--text-base, 13.5px);
  font-weight: var(--font-medium, 500);
  color: var(--color-text, #e6edf3);
}

.protocol-group__count {
  font-size: var(--text-xs, 11px);
  color: var(--color-muted, #8b98a5);
  margin-right: auto;
}

/* ─── Binding List ─── */

.binding-list {
  display: flex;
  flex-direction: column;
}

.binding-item {
  display: flex;
  align-items: center;
  gap: var(--space-3, 12px);
  padding: var(--space-3, 12px) var(--space-4, 16px);
  border-bottom: 1px solid var(--color-border, #28333f);
  transition: background 0.15s ease;
}

.binding-item:last-child {
  border-bottom: none;
}

.binding-item:hover {
  background: var(--color-surface-2, #1e2630);
}

.binding-item--blocked {
  background: rgba(239, 68, 68, 0.04);
  border-left: 3px solid var(--color-danger, #ef4444);
}

.binding-item--blocked:hover {
  background: rgba(239, 68, 68, 0.08);
}

/* ─── Drag Handle ─── */

.binding-drag-handle {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 24px;
  height: 24px;
  padding: 0;
  border: none;
  background: transparent;
  cursor: grab;
  color: var(--color-muted, #9ca3af);
  border-radius: var(--radius-sm, 4px);
  transition: color 0.15s ease, background 0.15s ease;
  flex-shrink: 0;
}

.binding-drag-handle:hover {
  color: var(--color-text, #e6edf3);
  background: var(--color-surface-hover, rgba(0, 0, 0, 0.05));
}

.binding-drag-handle:active {
  cursor: grabbing;
}

.binding-drag-handle__icon {
  font-size: 12px;
  line-height: 1;
  user-select: none;
  letter-spacing: -1px;
}

/* ─── Position Badge ─── */

.binding-position {
  display: flex;
  align-items: center;
  justify-content: center;
  width: 22px;
  height: 22px;
  border-radius: var(--radius-full, 9999px);
  background: var(--color-surface-2, #1e2630);
  border: 1px solid var(--color-border, #28333f);
  font-size: var(--text-xs, 11px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-muted, #8b98a5);
  flex-shrink: 0;
}

/* ─── Binding Info ─── */

.binding-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.binding-domain-name {
  font-size: var(--text-base, 13.5px);
  font-weight: var(--font-medium, 500);
  color: var(--color-text, #e6edf3);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.binding-domain-ip {
  font-size: var(--text-xs, 11px);
  color: var(--color-muted, #8b98a5);
  font-family: var(--font-mono, monospace);
}

/* ─── Status / Warning ─── */

.binding-status {
  display: flex;
  align-items: center;
  gap: var(--space-2, 8px);
  flex-shrink: 0;
}

.binding-warning {
  font-size: 16px;
  line-height: 1;
}

/* ─── Empty State ─── */

.binding-list-empty {
  padding: var(--space-4, 16px);
}

.binding-list-empty__text {
  font-size: var(--text-sm, 12px);
  color: var(--color-muted, #8b98a5);
  font-style: italic;
}

/* ─── Domain Picker ─── */

.domain-picker {
  display: flex;
  flex-direction: column;
  gap: var(--space-2, 8px);
}

.domain-picker__empty {
  color: var(--color-muted, #8b98a5);
  font-size: var(--text-base, 13.5px);
  text-align: center;
  padding: var(--space-6, 24px) 0;
}

.domain-picker__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1, 4px);
  max-height: 320px;
  overflow-y: auto;
}

.domain-picker__item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3, 12px);
  padding: var(--space-3, 12px) var(--space-4, 16px);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-md, 8px);
  background: transparent;
  cursor: pointer;
  transition: background 0.15s ease, border-color 0.15s ease;
  text-align: left;
  width: 100%;
}

.domain-picker__item:hover {
  background: var(--color-surface-2, #1e2630);
  border-color: var(--color-primary, #2563eb);
}

.domain-picker__item:focus-visible {
  outline: 2px solid var(--color-accent, #22d3ee);
  outline-offset: 2px;
}

.domain-picker__name {
  font-size: var(--text-base, 13.5px);
  font-weight: var(--font-medium, 500);
  color: var(--color-text, #e6edf3);
}

.domain-picker__ip {
  font-size: var(--text-xs, 11px);
  color: var(--color-muted, #8b98a5);
  font-family: var(--font-mono, monospace);
}

/* ─── Sortable States ─── */

:deep(.sortable-ghost) {
  opacity: 0.4;
  background: var(--color-primary-light, rgba(59, 130, 246, 0.08));
}

:deep(.sortable-chosen) {
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.15);
}

:deep(.sortable-drag) {
  opacity: 0.9;
}
</style>
