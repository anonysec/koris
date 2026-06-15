<script setup lang="ts">
import { onMounted } from 'vue'
import { useCustomersStore } from '@/stores/customers'

const props = defineProps<{
  id: string
}>()

const store = useCustomersStore()

onMounted(() => {
  store.loadDetail(Number(props.id))
})
</script>

<template>
  <div class="page">
    <div v-if="store.detailLoading" class="skeleton skeleton-card" style="height:200px"></div>
    <div v-else>
      <h3>Customer Detail</h3>
      <p style="color:var(--color-muted)">This view will be fully implemented in Phase 6.</p>
    </div>
  </div>
</template>
