<script setup lang="ts">
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { useTicketsStore, type Ticket } from '@/stores/tickets'
import { useI18n } from '@koris/composables/useI18n'
import { formatDate } from '@koris/composables/useFormatDate'
import Button from '@koris/ui/Button.vue'
import Select from '@koris/ui/Select.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import Skeleton from '@koris/ui/Skeleton.vue'

const { t } = useI18n()
const router = useRouter()
const store = useTicketsStore()

// ─── View mode toggle ────────────────────────────────────────────────────────
const viewMode = ref<'kanban' | 'list'>('kanban')

// ─── Filters ─────────────────────────────────────────────────────────────────
const filterCategory = ref('')
const filterPriority = ref('')

const categoryOptions = computed(() => [
  { label: t('tickets.filter_all'), value: '' },
  { label: t('tickets.category_billing'), value: 'billing' },
  { label: t('tickets.category_technical'), value: 'technical' },
  { label: t('tickets.category_general'), value: 'general' },
])

const priorityOptions = computed(() => [
  { label: t('tickets.filter_all'), value: '' },
  { label: t('tickets.priority_low'), value: 'low' },
  { label: t('tickets.priority_medium'), value: 'medium' },
  { label: t('tickets.priority_high'), value: 'high' },
])

// ─── Kanban columns ──────────────────────────────────────────────────────────
const kanbanStatuses = ['open', 'in_progress', 'waiting', 'resolved', 'closed'] as const

function statusLabel(status: string): string {
  return t(`tickets.status_${status}`)
}

function statusColor(status: string): string {
  const colors: Record<string, string> = {
    open: 'var(--color-warning)',
    in_progress: 'var(--color-primary)',
    waiting: 'var(--color-muted)',
    resolved: 'var(--color-success)',
    closed: 'var(--color-muted)',
  }
  return colors[status] || 'var(--color-muted)'
}

// ─── Filtered tickets ────────────────────────────────────────────────────────
const filteredTickets = computed(() => {
  return store.list.filter((t) => {
    if (filterCategory.value && t.category !== filterCategory.value) return false
    if (filterPriority.value && t.priority !== filterPriority.value) return false
    return true
  })
})

const filteredByStatus = computed(() => {
  const groups: Record<string, Ticket[]> = {
    open: [],
    in_progress: [],
    waiting: [],
    resolved: [],
    closed: [],
  }
  for (const ticket of filteredTickets.value) {
    if (groups[ticket.status]) {
      groups[ticket.status].push(ticket)
    }
  }
  return groups
})

// ─── Helpers ─────────────────────────────────────────────────────────────────
function timeSince(dateStr: string): string {
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const minutes = Math.floor(diffMs / 60000)
  if (minutes < 60) return `${minutes}m`
  const hours = Math.floor(minutes / 60)
  if (hours < 24) return `${hours}h`
  const days = Math.floor(hours / 24)
  return `${days}d`
}

function priorityClass(priority: string): string {
  if (priority === 'high') return 'priority--high'
  if (priority === 'medium') return 'priority--medium'
  return 'priority--low'
}

function handleTicketClick(ticket: Ticket) {
  router.push({ name: 'ticket-detail', params: { id: String(ticket.id) } })
}

async function applyFilters() {
  await store.loadTickets({
    category: filterCategory.value || undefined,
    priority: filterPriority.value || undefined,
  })
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  store.loadTickets()
})
</script>

<template>
  <div class="page tickets-view">
    <!-- Header bar -->
    <header class="tickets-header">
      <div>
        <h2 class="page-title">{{ t('tickets.title') }}</h2>
        <p class="tickets-subtitle">Customer support tickets and SLAs</p>
      </div>
      <div class="tickets-header__controls">
        <!-- Filters -->
        <div class="tickets-filters">
          <Select
            v-model="filterCategory"
            :options="categoryOptions"
            size="sm"
            :aria-label="t('tickets.filter_category')"
            @update:model-value="applyFilters"
          />
          <Select
            v-model="filterPriority"
            :options="priorityOptions"
            size="sm"
            :aria-label="t('tickets.filter_priority')"
            @update:model-value="applyFilters"
          />
        </div>
        <!-- View toggle -->
        <div class="view-toggle" role="group" :aria-label="t('tickets.view_mode')">
          <button
            :class="['view-toggle__btn', { active: viewMode === 'kanban' }]"
            :aria-pressed="viewMode === 'kanban'"
            @click="viewMode = 'kanban'"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <rect x="1" y="1" width="4" height="14" rx="1" stroke="currentColor" stroke-width="1.5" />
              <rect x="6" y="1" width="4" height="10" rx="1" stroke="currentColor" stroke-width="1.5" />
              <rect x="11" y="1" width="4" height="7" rx="1" stroke="currentColor" stroke-width="1.5" />
            </svg>
          </button>
          <button
            :class="['view-toggle__btn', { active: viewMode === 'list' }]"
            :aria-pressed="viewMode === 'list'"
            @click="viewMode = 'list'"
          >
            <svg width="16" height="16" viewBox="0 0 16 16" fill="none" aria-hidden="true">
              <line x1="1" y1="3" x2="15" y2="3" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
              <line x1="1" y1="8" x2="15" y2="8" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
              <line x1="1" y1="13" x2="15" y2="13" stroke="currentColor" stroke-width="1.5" stroke-linecap="round" />
            </svg>
          </button>
        </div>
      </div>
    </header>

    <!-- Loading state -->
    <div v-if="store.loading && store.list.length === 0" class="loading-state">
      <Skeleton variant="rect" :width="'100%'" :height="300" />
    </div>

    <!-- Kanban View -->
    <div v-else-if="viewMode === 'kanban'" class="kanban-board">
      <div
        v-for="status in kanbanStatuses"
        :key="status"
        class="kanban-column"
      >
        <div class="kanban-column__header">
          <span class="kanban-column__dot" :style="{ background: statusColor(status) }"></span>
          <span class="kanban-column__title">{{ statusLabel(status) }}</span>
          <span class="kanban-column__count">{{ filteredByStatus[status].length }}</span>
        </div>
        <div class="kanban-column__cards">
          <div
            v-for="ticket in filteredByStatus[status]"
            :key="ticket.id"
            class="kanban-card"
            role="button"
            tabindex="0"
            @click="handleTicketClick(ticket)"
            @keydown.enter="handleTicketClick(ticket)"
          >
            <div class="kanban-card__subject">{{ ticket.subject }}</div>
            <div class="kanban-card__meta">
              <span :class="['priority-badge', priorityClass(ticket.priority)]">
                {{ t(`tickets.priority_${ticket.priority}`) }}
              </span>
              <span class="kanban-card__customer">{{ ticket.username }}</span>
            </div>
            <div class="kanban-card__footer">
              <span class="kanban-card__time">{{ timeSince(ticket.created_at) }}</span>
              <span v-if="ticket.assigned_to" class="kanban-card__assignee">{{ ticket.assigned_to }}</span>
            </div>
          </div>
          <EmptyState
            v-if="filteredByStatus[status].length === 0"
            icon="📋"
            :title="t('tickets.column_empty')"
            class="kanban-column__empty"
          />
        </div>
      </div>
    </div>

    <!-- List View -->
    <div v-else class="tickets-list-view">
      <EmptyState
        v-if="filteredTickets.length === 0"
        icon="🎫"
        :title="t('tickets.no_open_tickets')"
        :description="t('tickets.all_caught_up')"
      />
      <div v-else class="tickets-list">
        <div class="tickets-list__header">
          <span class="col-subject">{{ t('tickets.subject') }}</span>
          <span class="col-customer">{{ t('tickets.customer') }}</span>
          <span class="col-category">{{ t('tickets.category') }}</span>
          <span class="col-priority">{{ t('tickets.priority') }}</span>
          <span class="col-status">{{ t('tickets.status') }}</span>
          <span class="col-date">{{ t('tickets.created') }}</span>
        </div>
        <div
          v-for="ticket in filteredTickets"
          :key="ticket.id"
          class="tickets-list__row"
          role="button"
          tabindex="0"
          @click="handleTicketClick(ticket)"
          @keydown.enter="handleTicketClick(ticket)"
        >
          <span class="col-subject">{{ ticket.subject }}</span>
          <span class="col-customer text-muted">{{ ticket.username }}</span>
          <span class="col-category">
            <span class="category-badge">{{ t(`tickets.category_${ticket.category}`) }}</span>
          </span>
          <span class="col-priority">
            <span :class="['priority-badge', priorityClass(ticket.priority)]">
              {{ t(`tickets.priority_${ticket.priority}`) }}
            </span>
          </span>
          <span class="col-status">
            <StatusPill :status="ticket.status" size="sm" />
          </span>
          <span class="col-date text-muted">{{ formatDate(ticket.created_at) }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.tickets-view { display: flex; flex-direction: column; gap: var(--space-4); }

.tickets-header { display: flex; justify-content: space-between; align-items: center; flex-wrap: wrap; gap: var(--space-3); }
.page-title { margin: 0; font-size: var(--text-lg); font-weight: var(--font-bold); }
.tickets-header__controls { display: flex; align-items: center; gap: var(--space-3); }
.tickets-filters { display: flex; gap: var(--space-2); }

/* View Toggle */
.view-toggle { display: flex; background: var(--color-surface-2); border-radius: var(--radius-md); padding: 2px; }
.view-toggle__btn { display: flex; align-items: center; justify-content: center; width: 32px; height: 28px; border: none; background: transparent; color: var(--color-muted); border-radius: var(--radius-sm); cursor: pointer; transition: all var(--duration-fast); }
.view-toggle__btn:hover { color: var(--color-text); }
.view-toggle__btn.active { background: var(--color-surface); color: var(--color-primary); box-shadow: var(--shadow-sm); }

/* Kanban Board */
.kanban-board { display: grid; grid-template-columns: repeat(5, 1fr); gap: var(--space-3); overflow-x: auto; min-height: 400px; }

.kanban-column { display: flex; flex-direction: column; min-width: 220px; }
.kanban-column__header { display: flex; align-items: center; gap: var(--space-2); padding: var(--space-2) var(--space-3); margin-bottom: var(--space-2); }
.kanban-column__dot { width: 8px; height: 8px; border-radius: 50%; flex-shrink: 0; }
.kanban-column__title { font-size: var(--text-xs); font-weight: var(--font-semibold); text-transform: uppercase; letter-spacing: 0.05em; color: var(--color-text); }
.kanban-column__count { font-size: var(--text-xs); color: var(--color-muted); margin-inline-start: auto; background: var(--color-surface-2); padding: 0 6px; border-radius: var(--radius-full); }
.kanban-column__cards { display: flex; flex-direction: column; gap: var(--space-2); flex: 1; }
.kanban-column__empty { padding: var(--space-4); }

/* Kanban Card */
.kanban-card { padding: var(--space-3); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); cursor: pointer; transition: all var(--duration-fast); display: flex; flex-direction: column; gap: var(--space-2); }
.kanban-card:hover { border-color: var(--color-primary); transform: translateY(-1px); box-shadow: var(--shadow-sm); }
.kanban-card:focus-visible { outline: 2px solid var(--color-accent); outline-offset: -2px; }
.kanban-card__subject { font-size: var(--text-sm); font-weight: var(--font-medium); line-height: 1.3; display: -webkit-box; -webkit-line-clamp: 2; -webkit-box-orient: vertical; overflow: hidden; }
.kanban-card__meta { display: flex; align-items: center; gap: var(--space-2); }
.kanban-card__customer { font-size: var(--text-xs); color: var(--color-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.kanban-card__footer { display: flex; justify-content: space-between; align-items: center; }
.kanban-card__time { font-size: var(--text-xs); color: var(--color-muted); }
.kanban-card__assignee { font-size: var(--text-xs); color: var(--color-primary); background: rgba(37, 99, 235, 0.08); padding: 1px 6px; border-radius: var(--radius-full); }

/* Priority Badges */
.priority-badge { font-size: var(--text-xs); font-weight: var(--font-medium); padding: 1px 6px; border-radius: var(--radius-full); text-transform: capitalize; }
.priority--high { background: rgba(239, 68, 68, 0.1); color: var(--color-danger); }
.priority--medium { background: rgba(245, 158, 11, 0.1); color: var(--color-warning); }
.priority--low { background: rgba(139, 152, 165, 0.1); color: var(--color-muted); }

/* Category Badge */
.category-badge { font-size: var(--text-xs); padding: 1px 6px; border-radius: var(--radius-full); background: var(--color-surface-2); color: var(--color-text); }

/* List View */
.tickets-list-view { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); overflow: hidden; }
.tickets-list__header { display: grid; grid-template-columns: 2fr 1fr 1fr 80px 100px 100px; gap: var(--space-3); padding: var(--space-3) var(--space-4); border-bottom: 1px solid var(--color-border); font-size: var(--text-xs); font-weight: var(--font-semibold); color: var(--color-muted); text-transform: uppercase; letter-spacing: 0.05em; }
.tickets-list__row { display: grid; grid-template-columns: 2fr 1fr 1fr 80px 100px 100px; gap: var(--space-3); padding: var(--space-3) var(--space-4); border-bottom: 1px solid var(--color-border); font-size: var(--text-sm); cursor: pointer; transition: background var(--duration-fast); align-items: center; }
.tickets-list__row:last-child { border-bottom: none; }
.tickets-list__row:hover { background: var(--color-surface-2); }
.tickets-list__row:focus-visible { outline: 2px solid var(--color-accent); outline-offset: -2px; }

.col-subject { font-weight: var(--font-medium); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }

/* Utility */
.text-muted { color: var(--color-muted); }
.loading-state { display: flex; flex-direction: column; gap: var(--space-4); }

@media (max-width: 1200px) {
  .kanban-board { grid-template-columns: repeat(3, 1fr); }
}
@media (max-width: 768px) {
  .kanban-board { grid-template-columns: 1fr; }
  .tickets-list__header,
  .tickets-list__row { grid-template-columns: 2fr 1fr 80px; }
  .col-customer, .col-category, .col-date { display: none; }
  .tickets-filters { flex-wrap: wrap; }
}

.tickets-subtitle { margin: 4px 0 0; font-size: var(--text-md); color: var(--color-muted); }
</style>
