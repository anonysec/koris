<script setup lang="ts">
import { onMounted } from 'vue'
import { useNodesStore } from '@/stores/nodes'

const store = useNodesStore()

onMounted(() => {
  store.loadNodes()
})
</script>

<template>
  <div class="page">
    <div v-if="store.loading" class="skeleton skeleton-card" style="height:200px"></div>
    <div v-else>
      <h3>Nodes</h3>
      <p style="color:var(--color-muted)">This view will be fully implemented in Phase 6.</p>
    </div>
  </div>
</template>
