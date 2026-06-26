<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useNodesStore, type CoreStatus } from '@/stores/nodes'
import { useToast } from '@koris/composables/useToast'
import CoreCard from './CoreCard.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'

const props = defineProps<{
  nodeId: number
}>()

const nodesStore = useNodesStore()
const toast = useToast()

const cores = ref<CoreStatus[]>([])
const loading = ref(false)

async function loadCores() {
  loading.value = true
  const apiCores = await nodesStore.listCores(props.nodeId)
  // Only show cores actually returned by the API (installed/configured on the node)
  // Filter out cores with status 'unknown' as those are not actually configured
  cores.value = apiCores.filter((c) => c.status !== 'unknown')
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

onMounted(loadCores)
</script>

<template>
  <div class="node-cores-tab">
    <h4 class="node-cores-tab__title">Cores</h4>

    <KSkeleton v-if="loading" />

    <div v-else class="node-cores-tab__grid">
      <CoreCard
        v-for="core in cores"
        :key="core.coreType"
        :node-id="props.nodeId"
        :core-type="core.coreType"
        :status="core.status"
        :port="core.port"
        :sessions="core.sessions"
        :pid="core.pid"
        @enable="(port) => handleEnable(core.coreType, port)"
        @disable="handleDisable(core.coreType)"
      />
    </div>
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

.node-cores-tab__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-3);
}
</style>
