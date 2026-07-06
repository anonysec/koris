<script setup lang="ts">
/**
 * Revenue trend card for the admin dashboard.
 * Fetches /api/admin/revenue-trend and renders an SVG area chart
 * with MRR + delta headline and a period switcher.
 */
import { computed, ref, watch, onMounted } from 'vue'
import { useApi } from '@koris/composables/useApi'

type Period = '7d' | '30d' | '90d' | '365d'

interface Point {
  label: string
  timestamp: string
  total_cents: number
  new_cents: number
  renew_cents: number
}

interface Trend {
  ok: boolean
  period: Period
  total_cents: number
  new_cents: number
  renew_cents: number
  current_mrr: number
  prior_mrr: number
  points: Point[]
}

const api = useApi({ showErrorToast: false })
const period = ref<Period>('30d')
const trend = ref<Trend | null>(null)
const loading = ref(false)
const hoveredIdx = ref<number | null>(null)

async function load() {
  loading.value = true
  try {
    const data = await api.get<Trend>(`/api/admin/revenue-trend?period=${period.value}`)
    if (data && data.ok) trend.value = data
  } catch {
    // silent — card just renders empty state
  } finally {
    loading.value = false
  }
}

watch(period, load)
onMounted(load)

// ─── Formatting ──────────────────────────────────────────
function fmtCents(c: number): string {
  const dollars = c / 100
  if (Math.abs(dollars) >= 1000) return `$${(dollars / 1000).toFixed(1)}k`
  return `$${dollars.toFixed(0)}`
}

const totalDisplay = computed(() => fmtCents(trend.value?.total_cents ?? 0))
const mrrDisplay = computed(() => fmtCents(trend.value?.current_mrr ?? 0))
const mrrDelta = computed(() => {
  const cur = trend.value?.current_mrr ?? 0
  const prior = trend.value?.prior_mrr ?? 0
  if (prior === 0) return null
  const pct = ((cur - prior) / prior) * 100
  return {
    pct: Math.abs(pct).toFixed(1),
    up: pct >= 0,
  }
})

// ─── SVG geometry ────────────────────────────────────────
const svgW = 700
const svgH = 180
const pad = { top: 12, right: 12, bottom: 24, left: 44 }
const chartW = computed(() => svgW - pad.left - pad.right)
const chartH = computed(() => svgH - pad.top - pad.bottom)

const maxCents = computed(() => {
  const pts = trend.value?.points ?? []
  if (!pts.length) return 100
  const m = Math.max(...pts.map((p) => p.total_cents))
  return m > 0 ? m : 100
})

const path = computed(() => {
  const pts = trend.value?.points ?? []
  if (!pts.length) return ''
  const n = pts.length
  const stepX = chartW.value / Math.max(1, n - 1)
  const points = pts.map((p, i) => {
    const x = pad.left + i * stepX
    const y = pad.top + chartH.value * (1 - p.total_cents / maxCents.value)
    return `${x},${y}`
  })
  return 'M ' + points.join(' L ')
})

const areaPath = computed(() => {
  const pts = trend.value?.points ?? []
  if (!pts.length) return ''
  const n = pts.length
  const stepX = chartW.value / Math.max(1, n - 1)
  let d = `M ${pad.left},${pad.top + chartH.value} `
  pts.forEach((p, i) => {
    const x = pad.left + i * stepX
    const y = pad.top + chartH.value * (1 - p.total_cents / maxCents.value)
    d += `L ${x},${y} `
  })
  d += `L ${pad.left + (n - 1) * stepX},${pad.top + chartH.value} Z`
  return d
})

const yAxisTicks = computed(() => {
  const steps = 4
  const max = maxCents.value
  return Array.from({ length: steps + 1 }, (_, i) => {
    const v = (max * i) / steps
    return {
      y: pad.top + chartH.value * (1 - i / steps),
      label: fmtCents(v),
    }
  })
})

const xAxisTicks = computed(() => {
  const pts = trend.value?.points ?? []
  if (!pts.length) return []
  const n = pts.length
  const desired = Math.min(6, n)
  const stepIdx = Math.max(1, Math.floor(n / desired))
  const stepX = chartW.value / Math.max(1, n - 1)
  const ticks = []
  for (let i = 0; i < n; i += stepIdx) {
    ticks.push({ x: pad.left + i * stepX, label: pts[i].label })
  }
  return ticks
})

const isEmpty = computed(
  () =>
    !loading.value &&
    (!trend.value || trend.value.points.every((p) => p.total_cents === 0)),
)
</script>

<template>
  <section class="rev-card" aria-label="Revenue trend">
    <header class="rev-card__head">
      <div>
        <div class="rev-card__title">Revenue</div>
        <div class="rev-card__figures">
          <span class="rev-card__mrr" :title="`Last full month · prior month: ${fmtCents(trend?.prior_mrr ?? 0)}`">
            {{ mrrDisplay }} <small>MRR</small>
          </span>
          <span
            v-if="mrrDelta"
            class="rev-card__delta"
            :class="mrrDelta.up ? 'rev-card__delta--up' : 'rev-card__delta--dn'"
          >
            {{ mrrDelta.up ? '▲' : '▼' }} {{ mrrDelta.pct }}%
          </span>
          <span class="rev-card__total">
            <small>{{ period }} total</small> {{ totalDisplay }}
          </span>
        </div>
      </div>
      <div class="rev-card__periods">
        <button
          v-for="p in (['7d','30d','90d','365d'] as Period[])"
          :key="p"
          class="rev-card__period"
          :class="{ 'rev-card__period--active': p === period }"
          @click="period = p"
        >{{ p }}</button>
      </div>
    </header>

    <div v-if="loading" class="rev-card__loading">Loading…</div>
    <div v-else-if="isEmpty" class="rev-card__empty">
      <div class="rev-card__empty-icon">💤</div>
      <div>No revenue in the last {{ period }}.</div>
      <div class="rev-card__empty-hint">Approved payments will show up here.</div>
    </div>
    <svg
      v-else
      :viewBox="`0 0 ${svgW} ${svgH}`"
      preserveAspectRatio="none"
      class="rev-card__svg"
      @mouseleave="hoveredIdx = null"
    >
      <defs>
        <linearGradient id="rev-fill" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stop-color="var(--color-primary)" stop-opacity="0.35" />
          <stop offset="100%" stop-color="var(--color-primary)" stop-opacity="0" />
        </linearGradient>
      </defs>

      <g class="rev-card__grid">
        <line
          v-for="(t, i) in yAxisTicks" :key="'g'+i"
          :x1="pad.left" :x2="svgW - pad.right"
          :y1="t.y" :y2="t.y"
          stroke="var(--color-border)" stroke-width="1" stroke-dasharray="2 4" opacity="0.5"
        />
      </g>

      <g class="rev-card__yaxis">
        <text
          v-for="(t, i) in yAxisTicks" :key="'y'+i"
          :x="pad.left - 8" :y="t.y + 3"
          text-anchor="end" fill="var(--color-muted)" font-size="10"
        >{{ t.label }}</text>
      </g>

      <path :d="areaPath" fill="url(#rev-fill)" />
      <path :d="path" fill="none" stroke="var(--color-primary)" stroke-width="2" stroke-linejoin="round" stroke-linecap="round" />

      <g class="rev-card__xaxis">
        <text
          v-for="(t, i) in xAxisTicks" :key="'x'+i"
          :x="t.x" :y="svgH - 6"
          text-anchor="middle" fill="var(--color-muted)" font-size="10"
        >{{ t.label }}</text>
      </g>

      <g v-if="trend" class="rev-card__hover-targets">
        <rect
          v-for="(p, i) in trend.points" :key="'h'+i"
          :x="pad.left + (i * chartW / Math.max(1, trend.points.length - 1)) - 8"
          :y="pad.top"
          width="16" :height="chartH"
          fill="transparent"
          @mouseenter="hoveredIdx = i"
        />
      </g>

      <g v-if="hoveredIdx !== null && trend">
        <circle
          :cx="pad.left + (hoveredIdx * chartW / Math.max(1, trend.points.length - 1))"
          :cy="pad.top + chartH * (1 - trend.points[hoveredIdx].total_cents / maxCents)"
          r="4" fill="var(--color-primary)" stroke="var(--color-bg)" stroke-width="2"
        />
      </g>
    </svg>

    <div v-if="hoveredIdx !== null && trend" class="rev-card__tooltip">
      <strong>{{ trend.points[hoveredIdx].label }}</strong> ·
      Total {{ fmtCents(trend.points[hoveredIdx].total_cents) }}
      <span class="rev-card__tt-split">
        New {{ fmtCents(trend.points[hoveredIdx].new_cents) }} ·
        Renew {{ fmtCents(trend.points[hoveredIdx].renew_cents) }}
      </span>
    </div>
  </section>
</template>

<style scoped>
.rev-card {
  padding: 20px 22px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  margin-bottom: 24px;
}

.rev-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
  flex-wrap: wrap;
}

.rev-card__title {
  font-size: var(--text-sm);
  color: var(--color-muted);
  font-weight: var(--font-semibold);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wide);
  margin-bottom: 6px;
}

.rev-card__figures {
  display: flex;
  align-items: baseline;
  gap: 14px;
  flex-wrap: wrap;
}
.rev-card__mrr {
  font-size: 26px;
  font-weight: var(--font-extrabold);
  letter-spacing: var(--tracking-tight);
  font-variant-numeric: tabular-nums;
  color: var(--color-text);
}
.rev-card__mrr small {
  font-size: 12px;
  font-weight: var(--font-semibold);
  color: var(--color-muted);
  letter-spacing: var(--tracking-wide);
  margin-left: 4px;
}
.rev-card__delta {
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  font-variant-numeric: tabular-nums;
}
.rev-card__delta--up { color: var(--color-success); }
.rev-card__delta--dn { color: var(--color-danger); }
.rev-card__total {
  font-size: var(--text-sm);
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
  font-weight: var(--font-medium);
}
.rev-card__total small {
  color: var(--color-muted);
  font-weight: var(--font-medium);
  margin-right: 4px;
}

.rev-card__periods {
  display: inline-flex;
  gap: 2px;
  padding: 2px;
  background: var(--color-surface-2);
  border-radius: var(--radius-md);
}
.rev-card__period {
  padding: 4px 10px;
  border: none;
  background: transparent;
  color: var(--color-muted);
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  border-radius: 6px;
  cursor: pointer;
  transition: color var(--duration-fast), background var(--duration-fast);
}
.rev-card__period:hover { color: var(--color-text); }
.rev-card__period--active {
  background: var(--color-surface);
  color: var(--color-text);
  box-shadow: var(--shadow-sm);
}

.rev-card__svg { width: 100%; height: 180px; display: block; }
.rev-card__hover-targets rect { cursor: crosshair; }

.rev-card__loading,
.rev-card__empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 20px;
  color: var(--color-muted);
  font-size: var(--text-sm);
  gap: 6px;
  min-height: 180px;
}
.rev-card__empty-icon { font-size: 32px; margin-bottom: 4px; opacity: 0.6; }
.rev-card__empty-hint { font-size: var(--text-xs); }

.rev-card__tooltip {
  margin-top: 8px;
  padding: 8px 12px;
  background: var(--color-surface-2);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-size: var(--text-sm);
  color: var(--color-text);
  font-variant-numeric: tabular-nums;
}
.rev-card__tt-split {
  color: var(--color-muted);
  margin-left: 8px;
  font-size: var(--text-xs);
}
</style>
