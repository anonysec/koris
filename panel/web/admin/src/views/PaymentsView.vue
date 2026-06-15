<script setup lang="ts">
import { onMounted } from 'vue'
import { usePaymentsStore } from '@/stores/payments'

const store = usePaymentsStore()

onMounted(() => {
  store.loadPayments()
})
</script>

<template>
  <div class="page">
    <div v-if="store.loading" class="skeleton skeleton-card" style="height:200px"></div>
    <div v-else>
      <h3>Payments</h3>
      <p style="color:var(--color-muted)">This view will be fully implemented in Phase 6.</p>
    </div>
  </div>
</template>
