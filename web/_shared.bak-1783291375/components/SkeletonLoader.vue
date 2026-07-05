<script setup lang="ts">
withDefaults(defineProps<{
  type?: 'text' | 'card' | 'table' | 'chart' | 'avatar'
  lines?: number
  animated?: boolean
  width?: string
  height?: string
}>(), {
  type: 'text',
  lines: 3,
  animated: true,
  width: '100%',
  height: 'auto',
})
</script>

<template>
  <div
    class="skeleton-loader"
    :class="[`skeleton-${type}`, { 'skeleton-animated': animated }]"
    :style="{ width, height: height !== 'auto' ? height : undefined }"
    role="status"
    aria-label="Loading..."
  >
    <!-- Text skeleton -->
    <template v-if="type === 'text'">
      <div
        v-for="i in lines"
        :key="i"
        class="skeleton-line"
        :style="{ width: i === lines ? '60%' : '100%' }"
      />
    </template>

    <!-- Card skeleton -->
    <template v-else-if="type === 'card'">
      <div class="skeleton-card-image" />
      <div class="skeleton-card-body">
        <div class="skeleton-line" style="width: 70%" />
        <div class="skeleton-line" style="width: 100%" />
        <div class="skeleton-line" style="width: 40%" />
      </div>
    </template>

    <!-- Table skeleton -->
    <template v-else-if="type === 'table'">
      <div class="skeleton-table-header">
        <div v-for="i in 5" :key="i" class="skeleton-cell" />
      </div>
      <div v-for="row in lines" :key="row" class="skeleton-table-row">
        <div v-for="col in 5" :key="col" class="skeleton-cell" />
      </div>
    </template>

    <!-- Chart skeleton -->
    <template v-else-if="type === 'chart'">
      <div class="skeleton-chart">
        <div class="skeleton-chart-bar" v-for="i in 7" :key="i" :style="{ height: `${20 + Math.random() * 60}%` }" />
      </div>
    </template>

    <!-- Avatar skeleton -->
    <template v-else-if="type === 'avatar'">
      <div class="skeleton-avatar" />
    </template>
  </div>
</template>

<style scoped>
.skeleton-loader {
  --skeleton-bg: var(--koris-border, #e2e8f0);
  --skeleton-shine: var(--koris-surface-hover, #f1f5f9);
}

.skeleton-animated .skeleton-line,
.skeleton-animated .skeleton-card-image,
.skeleton-animated .skeleton-cell,
.skeleton-animated .skeleton-chart-bar,
.skeleton-animated .skeleton-avatar {
  background: linear-gradient(
    90deg,
    var(--skeleton-bg) 25%,
    var(--skeleton-shine) 50%,
    var(--skeleton-bg) 75%
  );
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}

@keyframes shimmer {
  0% { background-position: 200% 0; }
  100% { background-position: -200% 0; }
}

/* Text */
.skeleton-line {
  height: 14px;
  border-radius: 4px;
  background: var(--skeleton-bg);
  margin-bottom: 10px;
}
.skeleton-line:last-child {
  margin-bottom: 0;
}

/* Card */
.skeleton-card-image {
  width: 100%;
  height: 140px;
  border-radius: 8px 8px 0 0;
  background: var(--skeleton-bg);
}
.skeleton-card-body {
  padding: 12px;
}
.skeleton-card-body .skeleton-line {
  height: 12px;
  margin-bottom: 8px;
}

/* Table */
.skeleton-table-header,
.skeleton-table-row {
  display: flex;
  gap: 12px;
  padding: 10px 0;
  border-bottom: 1px solid var(--skeleton-bg);
}
.skeleton-table-header .skeleton-cell {
  height: 12px;
  opacity: 0.7;
}
.skeleton-cell {
  flex: 1;
  height: 14px;
  border-radius: 4px;
  background: var(--skeleton-bg);
}

/* Chart */
.skeleton-chart {
  display: flex;
  align-items: flex-end;
  gap: 8px;
  height: 160px;
  padding: 12px 0;
}
.skeleton-chart-bar {
  flex: 1;
  border-radius: 4px 4px 0 0;
  background: var(--skeleton-bg);
  min-height: 20px;
}

/* Avatar */
.skeleton-avatar {
  width: 48px;
  height: 48px;
  border-radius: 50%;
  background: var(--skeleton-bg);
}

/* Respect reduced motion */
@media (prefers-reduced-motion: reduce) {
  .skeleton-animated .skeleton-line,
  .skeleton-animated .skeleton-card-image,
  .skeleton-animated .skeleton-cell,
  .skeleton-animated .skeleton-chart-bar,
  .skeleton-animated .skeleton-avatar {
    animation: none;
  }
}
</style>
