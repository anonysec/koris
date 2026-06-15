<script setup lang="ts">
import { onMounted } from 'vue'
import { useTicketsStore } from '@/stores/tickets'

const store = useTicketsStore()

onMounted(() => {
  store.loadTickets()
})
</script>

<template>
  <div class="page">
    <div v-if="store.loading" class="skeleton skeleton-card" style="height:200px"></div>
    <div v-else>
      <h3>Tickets</h3>
      <p style="color:var(--color-muted)">This view will be fully implemented in Phase 6.</p>
    </div>
  </div>
</template>
