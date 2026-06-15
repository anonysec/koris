<script setup lang="ts">
import { useRealtimeStore } from '@/stores/realtime'

const store = useRealtimeStore()
</script>

<template>
  <div class="page">
    <div v-if="!store.stats.customers && !store.connected" class="skeleton skeleton-card" style="height:200px"></div>
    <div v-else>
      <h3>Dashboard</h3>
      <p style="color:var(--color-muted)">This view will be fully implemented in Phase 6.</p>
    </div>
  </div>
</template>
