<template>
  <div
    class="k-chart"
    :class="{ 'k-chart--interactive': interactive }"
    :style="{ height: `${height}px` }"
  >
    <!-- Line Chart -->
    <svg
      v-if="type === 'line'"
      class="k-chart__svg"
      :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
      preserveAspectRatio="none"
      role="img"
      :aria-label="`Line chart with ${data.length} data points`"
    >
      <g v-if="options?.showGrid" class="k-chart__grid">
        <line
          v-for="(y, i) in gridLines"
          :key="i"
          :x1="padding.left"
          :y1="y"
          :x2="svgWidth - padding.right"
          :y2="y"
          class="k-chart__grid-line"
        />
      </g>
      <polyline
        v-if="!options?.smoothCurve"
        :points="linePoints"
        class="k-chart__line"
        :class="{ 'k-chart__line--animated': shouldAnimate }"
        fill="none"
        stroke="var(--color-primary)"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        v-else
        :d="smoothPath"
        class="k-chart__line"
        :class="{ 'k-chart__line--animated': shouldAnimate }"
        fill="none"
        stroke="var(--color-primary)"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <g v-if="interactive" class="k-chart__points">
        <circle
          v-for="(pt, i) in computedPoints"
          :key="i"
          :cx="pt.x"
          :cy="pt.y"
          r="4"
          class="k-chart__point"
          :class="{ 'k-chart__point--active': hoveredIndex === i }"
          @mouseenter="handlePointHover(i, $event)"
          @mouseleave="handlePointLeave"
          @click="handlePointClick(i, $event)"
        />
      </g>
    </svg>

    <!-- Area Chart -->
    <svg
      v-else-if="type === 'area'"
      class="k-chart__svg"
      :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
      preserveAspectRatio="none"
      role="img"
      :aria-label="`Area chart with ${data.length} data points`"
    >
      <defs v-if="options?.gradientFill !== false">
        <linearGradient id="k-chart-area-gradient" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--color-primary)" stop-opacity="0.4" />
          <stop offset="100%" stop-color="var(--color-accent)" stop-opacity="0.05" />
        </linearGradient>
      </defs>
      <g v-if="options?.showGrid" class="k-chart__grid">
        <line
          v-for="(y, i) in gridLines"
          :key="i"
          :x1="padding.left"
          :y1="y"
          :x2="svgWidth - padding.right"
          :y2="y"
          class="k-chart__grid-line"
        />
      </g>
      <path
        :d="areaPath"
        class="k-chart__area-fill"
        :class="{ 'k-chart__area-fill--animated': shouldAnimate }"
        :fill="options?.gradientFill !== false ? 'url(#k-chart-area-gradient)' : 'rgba(37, 99, 235, 0.15)'"
      />
      <polyline
        v-if="!options?.smoothCurve"
        :points="linePoints"
        class="k-chart__line"
        :class="{ 'k-chart__line--animated': shouldAnimate }"
        fill="none"
        stroke="var(--color-primary)"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <path
        v-else
        :d="smoothPath"
        class="k-chart__line"
        :class="{ 'k-chart__line--animated': shouldAnimate }"
        fill="none"
        stroke="var(--color-primary)"
        stroke-width="2"
        stroke-linecap="round"
        stroke-linejoin="round"
      />
      <g v-if="interactive" class="k-chart__points">
        <circle
          v-for="(pt, i) in computedPoints"
          :key="i"
          :cx="pt.x"
          :cy="pt.y"
          r="4"
          class="k-chart__point"
          :class="{ 'k-chart__point--active': hoveredIndex === i }"
          @mouseenter="handlePointHover(i, $event)"
          @mouseleave="handlePointLeave"
          @click="handlePointClick(i, $event)"
        />
      </g>
    </svg>

    <!-- Bar Chart -->
    <svg
      v-else-if="type === 'bar'"
      class="k-chart__svg"
      :viewBox="`0 0 ${svgWidth} ${svgHeight}`"
      preserveAspectRatio="none"
      role="img"
      :aria-label="`Bar chart with ${data.length} data points`"
    >
      <g v-if="options?.showGrid" class="k-chart__grid">
        <line
          v-for="(y, i) in gridLines"
          :key="i"
          :x1="padding.left"
          :y1="y"
          :x2="svgWidth - padding.right"
          :y2="y"
          class="k-chart__grid-line"
        />
      </g>
      <g class="k-chart__bars">
        <rect
          v-for="(bar, i) in computedBars"
          :key="i"
          :x="bar.x"
          :y="bar.y"
          :width="bar.width"
          :height="bar.height"
          :fill="bar.color"
          class="k-chart__bar"
          :class="{
            'k-chart__bar--animated': shouldAnimate,
            'k-chart__bar--active': hoveredIndex === i,
          }"
          rx="3"
          @mouseenter="handlePointHover(i, $event)"
          @mouseleave="handlePointLeave"
          @click="handlePointClick(i, $event)"
        />
      </g>
    </svg>

    <!-- Donut Chart -->
    <svg
      v-else-if="type === 'donut'"
      class="k-chart__svg k-chart__svg--donut"
      :viewBox="`0 0 ${donutSize} ${donutSize}`"
      role="img"
      :aria-label="`Donut chart with ${data.length} segments`"
    >
      <circle
        v-for="(segment, i) in computedDonutSegments"
        :key="i"
        :cx="donutSize / 2"
        :cy="donutSize / 2"
        :r="donutRadius"
        fill="none"
        :stroke="segment.color"
        :stroke-width="donutStrokeWidth"
        :stroke-dasharray="segment.dashArray"
        :stroke-dashoffset="segment.dashOffset"
        class="k-chart__donut-segment"
        :class="{
          'k-chart__donut-segment--animated': shouldAnimate,
          'k-chart__donut-segment--active': hoveredIndex === i,
        }"
        stroke-linecap="butt"
        @mouseenter="handlePointHover(i, $event)"
        @mouseleave="handlePointLeave"
        @click="handlePointClick(i, $event)"
      />
      <!-- center: total (default) or hovered value -->
      <text
        :x="donutSize / 2"
        :y="donutSize / 2 - (hoveredIndex !== null ? 6 : 0)"
        text-anchor="middle"
        dominant-baseline="middle"
        class="k-chart__donut-total"
      >
        {{ hoveredIndex !== null ? data[hoveredIndex]?.value : donutTotal }}
      </text>
      <text
        :x="donutSize / 2"
        :y="donutSize / 2 + 14"
        text-anchor="middle"
        dominant-baseline="middle"
        class="k-chart__donut-sub"
      >
        {{ hoveredIndex !== null ? data[hoveredIndex]?.label : "total" }}
      </text>
    </svg>

    <!-- Tooltip -->
    <div
      v-if="interactive && showTooltip && hoveredIndex !== null"
      class="k-chart__tooltip"
      :style="tooltipStyle"
      role="tooltip"
    >
      <span class="k-chart__tooltip-label">{{ tooltipData?.label }}</span>
      <span class="k-chart__tooltip-value">{{ formattedTooltipValue }}</span>
    </div>

    <!-- Legend -->
    <div v-if="options?.showLegend" class="k-chart__legend">
      <span
        v-for="(point, i) in data"
        :key="i"
        class="k-chart__legend-item"
      >
        <span
          class="k-chart__legend-swatch"
          :style="{ backgroundColor: point.color || defaultColors[i % defaultColors.length] }"
        />
        <span class="k-chart__legend-text">{{ point.label }}</span>
      </span>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import type { KChartProps, ChartDataPoint } from '@koris/types/components'

const props = withDefaults(defineProps<KChartProps>(), {
  animate: false,
  interactive: false,
  height: 200,
})

const emit = defineEmits<{
  (e: 'point-hover', payload: { point: ChartDataPoint; index: number }): void
  (e: 'point-click', payload: { point: ChartDataPoint; index: number }): void
}>()

// --- Constants ---
const svgWidth = 400
const svgHeight = 200
const padding = { top: 20, right: 20, bottom: 20, left: 20 }
const donutSize = 200
const donutRadius = 70
const donutStrokeWidth = 24
const defaultColors = [
  'var(--color-primary)',
  'var(--color-accent)',
  'var(--color-brand-2)',
  'var(--color-success)',
  'var(--color-warning)',
  'var(--color-danger)',
]

// --- State ---
const hoveredIndex = ref<number | null>(null)
const tooltipPosition = ref({ x: 0, y: 0 })
const prefersReducedMotion = ref(false)

// --- Reduced motion detection ---
let mediaQuery: MediaQueryList | null = null

onMounted(() => {
  mediaQuery = window.matchMedia('(prefers-reduced-motion: reduce)')
  prefersReducedMotion.value = mediaQuery.matches
  mediaQuery.addEventListener('change', handleMotionChange)
})

onUnmounted(() => {
  if (mediaQuery) {
    mediaQuery.removeEventListener('change', handleMotionChange)
  }
})

function handleMotionChange(e: MediaQueryListEvent) {
  prefersReducedMotion.value = e.matches
}

// --- Computed ---
const shouldAnimate = computed(() => props.animate && !prefersReducedMotion.value)

const showTooltip = computed(() => props.options?.showTooltip !== false)

const chartWidth = computed(() => svgWidth - padding.left - padding.right)
const chartHeight = computed(() => svgHeight - padding.top - padding.bottom)

const maxValue = computed(() => {
  if (props.data.length === 0) return 1
  return Math.max(...props.data.map((d) => d.value), 1)
})

const computedPoints = computed(() => {
  if (props.data.length === 0) return []
  const stepX = chartWidth.value / Math.max(props.data.length - 1, 1)
  return props.data.map((point, i) => ({
    x: padding.left + i * stepX,
    y: padding.top + chartHeight.value - (point.value / maxValue.value) * chartHeight.value,
    ...point,
  }))
})

const linePoints = computed(() => {
  return computedPoints.value.map((p) => `${p.x},${p.y}`).join(' ')
})

const smoothPath = computed(() => {
  const pts = computedPoints.value
  if (pts.length < 2) return ''
  let d = `M ${pts[0].x} ${pts[0].y}`
  for (let i = 1; i < pts.length; i++) {
    const prev = pts[i - 1]
    const curr = pts[i]
    const cpx1 = prev.x + (curr.x - prev.x) / 3
    const cpy1 = prev.y
    const cpx2 = curr.x - (curr.x - prev.x) / 3
    const cpy2 = curr.y
    d += ` C ${cpx1} ${cpy1}, ${cpx2} ${cpy2}, ${curr.x} ${curr.y}`
  }
  return d
})

const areaPath = computed(() => {
  const pts = computedPoints.value
  if (pts.length === 0) return ''
  const baseline = padding.top + chartHeight.value
  let d: string
  if (props.options?.smoothCurve && pts.length >= 2) {
    d = `M ${pts[0].x} ${baseline} L ${pts[0].x} ${pts[0].y}`
    for (let i = 1; i < pts.length; i++) {
      const prev = pts[i - 1]
      const curr = pts[i]
      const cpx1 = prev.x + (curr.x - prev.x) / 3
      const cpy1 = prev.y
      const cpx2 = curr.x - (curr.x - prev.x) / 3
      const cpy2 = curr.y
      d += ` C ${cpx1} ${cpy1}, ${cpx2} ${cpy2}, ${curr.x} ${curr.y}`
    }
    d += ` L ${pts[pts.length - 1].x} ${baseline} Z`
  } else {
    d = `M ${pts[0].x} ${baseline}`
    pts.forEach((p) => {
      d += ` L ${p.x} ${p.y}`
    })
    d += ` L ${pts[pts.length - 1].x} ${baseline} Z`
  }
  return d
})

const gridLines = computed(() => {
  const count = 4
  const lines: number[] = []
  for (let i = 0; i <= count; i++) {
    lines.push(padding.top + (chartHeight.value / count) * i)
  }
  return lines
})

const computedBars = computed(() => {
  if (props.data.length === 0) return []
  const barGap = 8
  const totalGaps = (props.data.length - 1) * barGap
  const barWidth = Math.max(
    (chartWidth.value - totalGaps) / props.data.length,
    4
  )
  return props.data.map((point, i) => {
    const barHeight = (point.value / maxValue.value) * chartHeight.value
    return {
      x: padding.left + i * (barWidth + barGap),
      y: padding.top + chartHeight.value - barHeight,
      width: barWidth,
      height: barHeight,
      color: point.color || defaultColors[i % defaultColors.length],
    }
  })
})

const computedDonutSegments = computed(() => {
  const total = props.data.reduce((sum, d) => sum + d.value, 0)
  if (total === 0) return []
  const circumference = 2 * Math.PI * donutRadius
  let offset = 0
  return props.data.map((point, i) => {
    const proportion = point.value / total
    const length = proportion * circumference
    const segment = {
      color: point.color || defaultColors[i % defaultColors.length],
      dashArray: `${length} ${circumference - length}`,
      dashOffset: -offset,
    }
    offset += length
    return segment
  })
})

const donutTotal = computed(() =>
  props.data.reduce((sum, d) => sum + (d.value || 0), 0)
)

const tooltipData = computed(() => {
  if (hoveredIndex.value === null) return null
  return props.data[hoveredIndex.value] || null
})

const formattedTooltipValue = computed(() => {
  if (!tooltipData.value) return ''
  if (props.options?.yAxisFormat) {
    return props.options.yAxisFormat(tooltipData.value.value)
  }
  return tooltipData.value.value.toLocaleString()
})

const tooltipStyle = computed(() => ({
  left: `${tooltipPosition.value.x}px`,
  top: `${tooltipPosition.value.y}px`,
}))

// --- Handlers ---
function handlePointHover(index: number, event: MouseEvent) {
  if (!props.interactive) return
  hoveredIndex.value = index
  const target = event.currentTarget as Element
  const rect = target.closest('.k-chart')?.getBoundingClientRect()
  if (rect) {
    tooltipPosition.value = {
      x: event.clientX - rect.left,
      y: event.clientY - rect.top - 40,
    }
  }
  emit('point-hover', { point: props.data[index], index })
}

function handlePointLeave() {
  hoveredIndex.value = null
}

function handlePointClick(index: number, event: MouseEvent) {
  if (!props.interactive) return
  event.stopPropagation()
  emit('point-click', { point: props.data[index], index })
}
</script>

<style scoped>
.k-chart {
  position: relative;
  width: 100%;
  font-family: var(--font-family);
}

.k-chart__svg {
  width: 100%;
  height: 100%;
  display: block;
}

.k-chart__svg--donut {
  max-width: 200px;
  max-height: 200px;
  margin: 0 auto;
}

/* ─── Grid ─── */

.k-chart__grid-line {
  stroke: var(--color-border);
  stroke-width: 0.5;
  stroke-dasharray: 4 4;
  opacity: 0.5;
}

/* ─── Line ─── */

.k-chart__line {
  vector-effect: non-scaling-stroke;
}

.k-chart__line--animated {
  stroke-dasharray: 1000;
  stroke-dashoffset: 1000;
  animation: k-chart-draw 1.2s var(--ease-out) forwards;
}

/* ─── Area Fill ─── */

.k-chart__area-fill {
  opacity: 1;
}

.k-chart__area-fill--animated {
  opacity: 0;
  animation: k-chart-fade-in 0.8s var(--ease-out) 0.4s forwards;
}

/* ─── Points ─── */

.k-chart__point {
  fill: var(--color-primary);
  stroke: var(--color-surface);
  stroke-width: 2;
  cursor: pointer;
  transition: r var(--duration-normal) var(--ease-default);
}

.k-chart__point--active {
  r: 6;
  fill: var(--color-accent);
}

/* ─── Bars ─── */

.k-chart__bar {
  transition:
    opacity var(--duration-normal) var(--ease-default),
    height var(--duration-slow) var(--ease-out),
    y var(--duration-slow) var(--ease-out);
  cursor: pointer;
}

.k-chart__bar--animated {
  animation: k-chart-bar-grow 0.6s var(--ease-out) forwards;
  transform-origin: bottom;
}

.k-chart__bar--active {
  opacity: 0.8;
  filter: brightness(1.2);
}

/* ─── Donut ─── */

.k-chart__donut-segment {
  transition:
    stroke-width var(--duration-normal) var(--ease-default),
    opacity var(--duration-normal) var(--ease-default);
  cursor: pointer;
  transform: rotate(-90deg);
  transform-origin: center;
}

.k-chart__donut-segment--animated {
  stroke-dashoffset: 0 !important;
  animation: k-chart-donut-draw 1s var(--ease-out) forwards;
}

.k-chart__donut-segment--active {
  stroke-width: 28;
  opacity: 0.9;
}

.k-chart__donut-label {
  fill: var(--color-text);
  font-size: var(--text-sm);
  font-weight: var(--font-medium);
}
.k-chart__donut-total {
  fill: var(--color-text);
  font-size: var(--text-2xl);
  font-weight: var(--font-extrabold);
  font-variant-numeric: tabular-nums;
}
.k-chart__donut-sub {
  fill: var(--color-muted);
  font-size: var(--text-xs);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wide);
}

/* ─── Tooltip ─── */

.k-chart__tooltip {
  position: absolute;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  pointer-events: none;
  z-index: var(--z-tooltip);
  transform: translateX(-50%);
  white-space: nowrap;
}

.k-chart__tooltip-label {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

.k-chart__tooltip-value {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-text);
}

/* ─── Legend ─── */

.k-chart__legend {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-3);
  margin-top: var(--space-3);
  justify-content: center;
}

.k-chart__legend-item {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
}

.k-chart__legend-swatch {
  width: 10px;
  height: 10px;
  border-radius: var(--radius-full);
}

.k-chart__legend-text {
  font-size: var(--text-xs);
  color: var(--color-muted);
}

/* ─── Animations ─── */

@keyframes k-chart-draw {
  to {
    stroke-dashoffset: 0;
  }
}

@keyframes k-chart-fade-in {
  to {
    opacity: 1;
  }
}

@keyframes k-chart-bar-grow {
  from {
    transform: scaleY(0);
  }
  to {
    transform: scaleY(1);
  }
}

@keyframes k-chart-donut-draw {
  from {
    stroke-dasharray: 0 1000;
  }
}

/* ─── Reduced Motion ─── */

@media (prefers-reduced-motion: reduce) {
  .k-chart__line--animated {
    animation: none;
    stroke-dasharray: none;
    stroke-dashoffset: 0;
  }

  .k-chart__area-fill--animated {
    animation: none;
    opacity: 1;
  }

  .k-chart__bar--animated {
    animation: none;
    transform: scaleY(1);
  }

  .k-chart__donut-segment--animated {
    animation: none;
  }

  .k-chart__point {
    transition: none;
  }

  .k-chart__bar {
    transition: none;
  }

  .k-chart__donut-segment {
    transition: none;
  }
}

/* ─── Interactive cursor ─── */

.k-chart--interactive .k-chart__point,
.k-chart--interactive .k-chart__bar,
.k-chart--interactive .k-chart__donut-segment {
  cursor: pointer;
}
</style>
