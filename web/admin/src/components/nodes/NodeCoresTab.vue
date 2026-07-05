<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useNodesStore, type CoreStatus } from '@/stores/nodes'
import { useToast } from '@koris/composables/useToast'
import type { CoreInfo } from './types'
import CoresTab from './CoresTab.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'

const props = defineProps<{
  nodeId: number
}>()

const nodesStore = useNodesStore()
const toast = useToast()

const cores = ref<CoreInfo[]>([])
const loading = ref(false)

/** Map store CoreStatus to the component CoreInfo interface. */
function toCoreInfo(c: CoreStatus): CoreInfo {
  let state: CoreInfo['state'] = 'stopped'
  if (c.status === 'running') state = 'running'
  else if (c.status === 'error' || c.status === 'crashed') state = 'crashed'
  else if (c.status === 'stopped') state = 'stopped'
  else state = 'unknown'

  return {
    type: c.coreType,
    state,
    port: c.port ?? 0,
    activeSessions: c.sessions ?? 0,
  }
}

async function loadCores() {
  loading.value = true
  const apiCores = await nodesStore.listCores(props.nodeId)
  cores.value = apiCores.map(toCoreInfo)
  loading.value = false
}

async function handleEnable(coreType: string, port: number) {
  const ok = await nodesStore.enableCore(props.nodeId, coreType, port)
  if (ok) {
    toast.success(`${coreType} enabled`)
    await loadCores()
  } else {
    toast.error(`Failed to enable ${coreType}`)
  }
}

async function handleDisable(coreType: string) {
  const ok = await nodesStore.disableCore(props.nodeId, coreType)
  if (ok) {
    toast.success(`${coreType} disabled`)
    await loadCores()
  } else {
    toast.error(`Failed to disable ${coreType}`)
  }
}

async function handleRestart(coreType: string) {
  // Force restart bypasses auto-restart limit
  const ok = await nodesStore.restartCore(props.nodeId, coreType)
  if (ok) {
    toast.success(`${coreType} restarting`)
    await loadCores()
  } else {
    toast.error(`Failed to restart ${coreType}`)
  }
}

onMounted(loadCores)
</script>

<template>
  <div class="node-cores-tab">
    <h4 class="node-cores-tab__title">Cores</h4>

    <KSkeleton v-if="loading" />

    <CoresTab
      v-else
      :node-id="props.nodeId"
      :cores="cores"
      @enable="handleEnable"
      @disable="handleDisable"
      @restart="handleRestart"
    />
  </div>
</template>

<style scoped>
.node-cores-tab {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

.node-cores-tab__title {
  margin: 0;
  font-size: var(--text-base);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}
</style>
