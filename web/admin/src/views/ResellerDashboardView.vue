<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from '@koris/composables/useI18n'
import { useApi } from '@koris/composables/useApi'
import { useAuthStore } from '@/stores/auth'

const { t } = useI18n()
const router = useRouter()
const api = useApi()
const auth = useAuthStore()

interface DashboardStats {
  ok: boolean
  credit: number
  total_users: number
  active_users: number
  total_usage_bytes: number
  daily_usage?: number[]
}

const stats = ref<DashboardStats | null>(null)
const loading = ref(true)

function formatBytes(bytes: number): string {
  if (bytes === 0) return '0 B'
  const tb = bytes / (1024 ** 4)
  if (tb >= 1) return `${tb.toFixed(2)} TB`
  const gb = bytes / (1024 ** 3)
  if (gb >= 1) return `${gb.toFixed(2)} GB`
  const mb = bytes / (1024 ** 2)
  return `${mb.toFixed(1)} MB`
}

/** Simple bar chart heights based on daily usage */
const chartBars = computed(() => {
  const data = stats.value?.daily_usage ?? []
  if (data.length === 0) return Array(7).fill(0)
  const max = Math.max(...data, 1)
  return data.map(v => Math.round((v / max) * 100))
})

const weekDays = ['Mon', 'Tue', 'Wed', 'Thu', 'Fri', 'Sat', 'Sun']

async function loadStats() {
  loading.value = true
  try {
    const data = await api.get<DashboardStats>('/api/reseller/dashboard')
    if (data?.ok) {
      stats.value = data
      if (auth.user) {
        auth.user.credit = data.credit
      }
    }
  } finally {
    loading.value = false
  }
}

onMounted(loadStats)
</script>

<template>
  <div class="reseller-dashboard">
    <h1 class="page-title">{{ t('reseller_dashboard.title') }}</h1>

    <div v-if="loading" class="stats-grid">
      <div v-for="i in 4" :key="i" class="stat-card skeleton" />
    </div>

    <div v-else-if="stats" class="stats-grid">
      <div class="stat-card credit-card stat-card--clickable" @click="router.push({ name: 'reseller-transactions' })">
        <div class="stat-icon">💰</div>
        <div class="stat-content">
          <span class="stat-value credit-value">{{ stats.credit.toLocaleString() }}</span>
          <span class="stat-label">{{ t('reseller_dashboard.credit') }}</span>
        </div>
      </div>

      <div class="stat-card stat-card--clickable" @click="router.push({ name: 'users' })">
        <div class="stat-icon">👥</div>
        <div class="stat-content">
          <span class="stat-value">{{ stats.total_users }}</span>
          <span class="stat-label">{{ t('reseller_dashboard.total_users') }}</span>
        </div>
      </div>

      <div class="stat-card stat-card--clickable" @click="router.push({ name: 'users' })">
        <div class="stat-icon">✅</div>
        <div class="stat-content">
          <span class="stat-value">{{ stats.active_users }}</span>
          <span class="stat-label">{{ t('reseller_dashboard.active_users') }}</span>
        </div>
      </div>

      <div class="stat-card stat-card--clickable" @click="router.push({ name: 'users' })">
        <div class="stat-icon">📊</div>
        <div class="stat-content">
          <span class="stat-value">{{ formatBytes(stats.total_usage_bytes) }}</span>
          <span class="stat-label">{{ t('reseller_dashboard.total_usage') }}</span>
        </div>
      </div>
    </div>

    <!-- Usage Chart -->
    <div v-if="stats" class="usage-chart-card">
      <h3 class="chart-title">{{ t('reseller_dashboard.weekly_usage') }}</h3>
      <div class="chart-bars">
        <div v-for="(height, idx) in chartBars" :key="idx" class="chart-bar-wrap">
          <div class="chart-bar" :style="{ height: `${Math.max(height, 4)}%` }" />
          <span class="chart-bar-label">{{ weekDays[idx] }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.reseller-dashboard {
  padding: var(--space-6, 24px);
}

.page-title {
  font-size: var(--text-2xl, 22px);
  font-weight: var(--font-bold, 700);
  margin: 0 0 var(--space-6, 24px);
}

.stats-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: var(--space-4, 16px);
}

.stat-card {
  background: var(--color-surface-2, #1e2630);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-lg, 12px);
  padding: var(--space-5, 20px);
  display: flex;
  align-items: center;
  gap: var(--space-4, 16px);
  transition: border-color 0.15s;
}

.stat-card:hover {
  border-color: var(--color-primary, #2563eb);
}

.stat-card--clickable {
  cursor: pointer;
  transition: transform 0.15s, border-color 0.15s;
}

.stat-card--clickable:hover {
  transform: translateY(-2px);
  border-color: rgba(37, 99, 235, 0.4);
}

.stat-card.credit-card {
  background: linear-gradient(135deg, rgba(37, 99, 235, 0.15), rgba(124, 92, 255, 0.1));
  border-color: rgba(37, 99, 235, 0.3);
}

.stat-card.skeleton {
  height: 88px;
  animation: pulse 1.5s infinite;
}

@keyframes pulse {
  0%, 100% { opacity: 1; }
  50% { opacity: 0.5; }
}

.stat-icon {
  font-size: 28px;
  flex-shrink: 0;
}

.stat-content {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.stat-value {
  font-size: var(--text-xl, 20px);
  font-weight: var(--font-bold, 700);
  color: var(--color-text, #e6edf3);
}

.credit-value {
  font-size: var(--text-2xl, 24px);
  color: var(--color-primary, #2563eb);
}

.stat-label {
  font-size: var(--text-sm, 12px);
  color: var(--color-muted, #8b98a5);
}

/* Usage Chart */
.usage-chart-card {
  margin-top: var(--space-5, 20px);
  background: var(--color-surface-2, #1e2630);
  border: 1px solid var(--color-border, #28333f);
  border-radius: var(--radius-lg, 12px);
  padding: var(--space-5, 20px);
}

.chart-title {
  font-size: var(--text-sm, 13px);
  font-weight: var(--font-semibold, 600);
  color: var(--color-muted, #8b98a5);
  margin: 0 0 var(--space-4, 16px);
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.chart-bars {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2, 8px);
  height: 120px;
}

.chart-bar-wrap {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  height: 100%;
  justify-content: flex-end;
  gap: 6px;
}

.chart-bar {
  width: 100%;
  max-width: 40px;
  background: linear-gradient(180deg, var(--color-primary, #2563eb), rgba(37, 99, 235, 0.4));
  border-radius: 4px 4px 0 0;
  min-height: 4px;
  transition: height 0.4s cubic-bezier(0.4, 0, 0.2, 1);
}

.chart-bar-label {
  font-size: 10px;
  color: var(--color-muted, #8b98a5);
  text-transform: uppercase;
}
</style>
