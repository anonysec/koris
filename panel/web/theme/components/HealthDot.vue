<template>
  <span
    class="health-dot"
    :class="[`health-dot--${level}`]"
    :title="`Health: ${score.toFixed(2)}`"
    :aria-label="`Health ${levelLabel}: ${score.toFixed(2)}`"
    role="img"
  />
</template>

<script setup lang="ts">
import { computed } from 'vue'

export interface HealthDotProps {
  score: number
}

const props = defineProps<HealthDotProps>()

const level = computed<'green' | 'yellow' | 'red'>(() => {
  if (props.score >= 0.8) return 'green'
  if (props.score >= 0.4) return 'yellow'
  return 'red'
})

const levelLabel = computed(() => {
  if (props.score >= 0.8) return 'good'
  if (props.score >= 0.4) return 'degraded'
  return 'critical'
})
</script>

<style scoped>
.health-dot {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 50%;
  flex-shrink: 0;
  vertical-align: middle;
}

.health-dot--green {
  background-color: #22c55e;
  animation: pulse-green 2s ease-in-out infinite;
}

.health-dot--yellow {
  background-color: #f59e0b;
}

.health-dot--red {
  background-color: #ef4444;
}

@keyframes pulse-green {
  0%, 100% {
    box-shadow: 0 0 0 0 rgba(34, 197, 94, 0.5);
  }
  50% {
    box-shadow: 0 0 0 4px rgba(34, 197, 94, 0);
  }
}

@media (prefers-reduced-motion: reduce) {
  .health-dot--green {
    animation: none;
  }
}
</style>
