<script setup lang="ts">
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { useTicketsStore } from '@/stores/tickets'
import { useI18n } from '@koris/composables/useI18n'
import { useToast } from '@koris/composables/useToast'
import KButton from '@koris/ui/KButton.vue'
import KTextarea from '@koris/ui/KTextarea.vue'
import KSelect from '@koris/ui/KSelect.vue'
import KStatusPill from '@koris/ui/KStatusPill.vue'
import KSkeleton from '@koris/ui/KSkeleton.vue'

const { t } = useI18n()
const toast = useToast()

const props = defineProps<{ id: string }>()
const router = useRouter()
const store = useTicketsStore()

// ─── State ───────────────────────────────────────────────────────────────────
const replyText = ref('')
const isInternal = ref(false)
const sending = ref(false)
const showCannedDropdown = ref(false)

// ─── Computed ────────────────────────────────────────────────────────────────
const ticket = computed(() => store.detail)

const visibleMessages = computed(() => {
  if (!ticket.value?.messages) return []
  // Admin can see all messages including internal
  return ticket.value.messages
})

const statusOptions = computed(() => [
  { label: t('tickets.status_open'), value: 'open' },
  { label: t('tickets.status_in_progress'), value: 'in_progress' },
  { label: t('tickets.status_waiting'), value: 'waiting' },
  { label: t('tickets.status_resolved'), value: 'resolved' },
  { label: t('tickets.status_closed'), value: 'closed' },
])

const priorityOptions = computed(() => [
  { label: t('tickets.priority_low'), value: 'low' },
  { label: t('tickets.priority_medium'), value: 'medium' },
  { label: t('tickets.priority_high'), value: 'high' },
])

const categoryOptions = computed(() => [
  { label: t('tickets.category_billing'), value: 'billing' },
  { label: t('tickets.category_technical'), value: 'technical' },
  { label: t('tickets.category_general'), value: 'general' },
])

// ─── Actions ─────────────────────────────────────────────────────────────────
async function sendReply() {
  if (!replyText.value.trim()) return
  sending.value = true
  const ok = await store.replyToTicket(Number(props.id), replyText.value, isInternal.value)
  if (ok) {
    replyText.value = ''
    isInternal.value = false
    toast.success(t('tickets.reply_sent'))
  }
  sending.value = false
}

async function handleStatusChange(newStatus: string) {
  await store.updateTicket(Number(props.id), { status: newStatus as any })
  toast.success(t('tickets.status_updated'))
}

async function handlePriorityChange(newPriority: string) {
  await store.updateTicket(Number(props.id), { priority: newPriority as any })
  toast.success(t('tickets.priority_updated'))
}

async function handleCategoryChange(newCategory: string) {
  await store.updateTicket(Number(props.id), { category: newCategory as any })
}

async function handleAssigneeChange(event: Event) {
  const select = event.target as HTMLSelectElement
  if (select.value !== ticket.value?.assigned_to) {
    await store.updateTicket(Number(props.id), { assigned_to: select.value })
    toast.success(t('tickets.assignee_updated'))
  }
}

function insertCannedResponse(body: string) {
  replyText.value = body
  showCannedDropdown.value = false
}

function formatMessageDate(dateStr: string): string {
  if (!dateStr) return ''
  const d = new Date(dateStr)
  return d.toLocaleString(undefined, {
    month: 'short',
    day: 'numeric',
    hour: '2-digit',
    minute: '2-digit',
  })
}

function timeSinceCreation(dateStr: string): string {
  if (!dateStr) return ''
  const now = Date.now()
  const then = new Date(dateStr).getTime()
  const diffMs = now - then
  const hours = Math.floor(diffMs / 3600000)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  return `${days}d ago`
}

// ─── Lifecycle ───────────────────────────────────────────────────────────────
onMounted(() => {
  store.loadTicketDetail(Number(props.id))
  store.loadCannedResponses()
})

watch(() => props.id, (newId) => {
  store.loadTicketDetail(Number(newId))
})
</script>

<template>
  <div class="page ticket-detail">
    <!-- Loading -->
    <div v-if="store.loading && !ticket" class="loading-state">
      <KSkeleton variant="rect" :width="'100%'" :height="60" />
      <KSkeleton variant="rect" :width="'100%'" :height="300" />
    </div>

    <template v-else-if="ticket">
      <!-- Top bar -->
      <div class="ticket-topbar">
        <KButton variant="ghost" size="sm" @click="router.push({ name: 'tickets' })">
          ← {{ t('tickets.back') }}
        </KButton>
        <span class="ticket-topbar__id">#{{ ticket.id }}</span>
      </div>

      <div class="ticket-layout">
        <!-- Main content (thread) -->
        <main class="ticket-main">
          <!-- Subject header -->
          <header class="ticket-subject-header">
            <h2 class="ticket-subject">{{ ticket.subject }}</h2>
            <div class="ticket-subject-meta">
              <KStatusPill :status="ticket.status" />
              <span class="ticket-created">{{ timeSinceCreation(ticket.created_at) }}</span>
            </div>
          </header>

          <!-- Messages Thread -->
          <section class="messages-thread" aria-label="Ticket conversation">
            <div
              v-for="msg in visibleMessages"
              :key="msg.id"
              :class="[
                'message',
                msg.sender_type === 'admin' ? 'message--admin' : 'message--customer',
                msg.is_internal ? 'message--internal' : '',
              ]"
            >
              <div class="message__avatar">
                {{ msg.sender_type === 'admin' ? '👤' : '🙋' }}
              </div>
              <div class="message__content">
                <div class="message__header">
                  <span class="message__sender">
                    {{ msg.sender_name || msg.sender_id }}
                    <span v-if="msg.is_internal" class="internal-badge">{{ t('tickets.internal_note') }}</span>
                  </span>
                  <span class="message__time">{{ formatMessageDate(msg.created_at) }}</span>
                </div>
                <div class="message__body">{{ msg.body || msg.message }}</div>
              </div>
            </div>
            <p v-if="visibleMessages.length === 0" class="text-muted text-center">
              {{ t('tickets.no_messages') }}
            </p>
          </section>

          <!-- Reply Form -->
          <section class="reply-section">
            <div class="reply-section__header">
              <span class="reply-section__title">
                {{ isInternal ? t('tickets.add_internal_note') : t('tickets.reply_to_customer') }}
              </span>
              <!-- Canned responses dropdown -->
              <div class="canned-dropdown-wrapper">
                <KButton
                  variant="ghost"
                  size="sm"
                  @click="showCannedDropdown = !showCannedDropdown"
                >
                  {{ t('tickets.canned_responses') }}
                </KButton>
                <div v-if="showCannedDropdown && store.cannedResponses.length > 0" class="canned-dropdown">
                  <div
                    v-for="cr in store.cannedResponses"
                    :key="cr.id"
                    class="canned-dropdown__item"
                    role="button"
                    tabindex="0"
                    @click="insertCannedResponse(cr.body)"
                    @keydown.enter="insertCannedResponse(cr.body)"
                  >
                    <span class="canned-dropdown__title">{{ cr.title }}</span>
                    <span class="canned-dropdown__preview">{{ cr.body.slice(0, 60) }}...</span>
                  </div>
                </div>
              </div>
            </div>

            <form class="reply-form" @submit.prevent="sendReply">
              <KTextarea
                v-model="replyText"
                :placeholder="isInternal ? t('tickets.internal_note_placeholder') : t('tickets.type_reply')"
                rows="4"
                :aria-label="isInternal ? t('tickets.add_internal_note') : t('tickets.reply_to_customer')"
              />
              <div class="reply-form__actions">
                <label class="internal-toggle">
                  <input v-model="isInternal" type="checkbox" class="internal-toggle__checkbox" />
                  <span class="internal-toggle__label">{{ t('tickets.mark_internal') }}</span>
                </label>
                <KButton type="submit" :variant="isInternal ? 'secondary' : 'primary'" :loading="sending" :disabled="!replyText.trim()">
                  {{ isInternal ? t('tickets.save_note') : t('tickets.send_reply') }}
                </KButton>
              </div>
            </form>
          </section>
        </main>

        <!-- Sidebar: ticket metadata -->
        <aside class="ticket-sidebar">
          <!-- Status -->
          <div class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.status') }}</label>
            <KSelect
              :model-value="ticket.status"
              :options="statusOptions"
              size="sm"
              @update:model-value="handleStatusChange"
            />
          </div>

          <!-- Priority -->
          <div class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.priority') }}</label>
            <KSelect
              :model-value="ticket.priority"
              :options="priorityOptions"
              size="sm"
              @update:model-value="handlePriorityChange"
            />
          </div>

          <!-- Category -->
          <div class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.category') }}</label>
            <KSelect
              :model-value="ticket.category"
              :options="categoryOptions"
              size="sm"
              @update:model-value="handleCategoryChange"
            />
          </div>

          <!-- Assigned To -->
          <div class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.assigned_to') }}</label>
            <select class="sidebar-input" :value="ticket.assigned_to" @change="handleAssigneeChange">
              <option value="">{{ t('tickets.unassigned') }}</option>
              <option v-for="admin in store.adminsList" :key="admin" :value="admin">{{ admin }}</option>
            </select>
          </div>

          <!-- Customer -->
          <div class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.customer') }}</label>
            <span class="sidebar-value">{{ ticket.username || '—' }}</span>
          </div>

          <!-- Created -->
          <div class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.created') }}</label>
            <span class="sidebar-value">{{ formatMessageDate(ticket.created_at) }}</span>
          </div>

          <!-- Resolved -->
          <div v-if="ticket.resolved_at" class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.resolved_at') }}</label>
            <span class="sidebar-value">{{ formatMessageDate(ticket.resolved_at) }}</span>
          </div>

          <!-- Satisfaction -->
          <div v-if="ticket.satisfaction_rating" class="sidebar-section">
            <label class="sidebar-label">{{ t('tickets.satisfaction') }}</label>
            <span class="sidebar-value sidebar-value--stars">
              <span v-for="n in 5" :key="n" :class="n <= ticket.satisfaction_rating! ? 'star--filled' : 'star--empty'">★</span>
            </span>
          </div>
        </aside>
      </div>
    </template>

    <!-- Not Found -->
    <div v-else class="empty-state">
      <p class="text-muted">{{ t('tickets.not_found') }}</p>
      <KButton variant="ghost" @click="router.push({ name: 'tickets' })">{{ t('tickets.go_back') }}</KButton>
    </div>
  </div>
</template>

<style scoped>
.ticket-detail { display: flex; flex-direction: column; gap: var(--space-3); }
.loading-state { display: flex; flex-direction: column; gap: var(--space-4); }

/* Top bar */
.ticket-topbar { display: flex; align-items: center; gap: var(--space-2); }
.ticket-topbar__id { font-size: var(--text-sm); color: var(--color-muted); }

/* Layout */
.ticket-layout { display: grid; grid-template-columns: 1fr 280px; gap: var(--space-4); }

/* Main content */
.ticket-main { display: flex; flex-direction: column; gap: var(--space-4); min-width: 0; }

/* Subject header */
.ticket-subject-header { padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); }
.ticket-subject { margin: 0 0 var(--space-2); font-size: var(--text-lg); font-weight: var(--font-bold); }
.ticket-subject-meta { display: flex; align-items: center; gap: var(--space-2); }
.ticket-created { font-size: var(--text-xs); color: var(--color-muted); }

/* Messages */
.messages-thread { display: flex; flex-direction: column; gap: var(--space-3); padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); max-height: 500px; overflow-y: auto; }

.message { display: flex; gap: var(--space-3); padding: var(--space-3); border-radius: var(--radius-md); }
.message--admin { background: rgba(37, 99, 235, 0.04); }
.message--customer { background: var(--color-surface-2); }
.message--internal { background: rgba(245, 158, 11, 0.08); border: 1px dashed rgba(245, 158, 11, 0.3); }

.message__avatar { width: 32px; height: 32px; display: flex; align-items: center; justify-content: center; border-radius: 50%; background: var(--color-surface-2); font-size: 14px; flex-shrink: 0; }
.message--admin .message__avatar { background: rgba(37, 99, 235, 0.1); }

.message__content { flex: 1; min-width: 0; }
.message__header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-1); }
.message__sender { font-weight: var(--font-semibold); font-size: var(--text-sm); }
.message__time { font-size: var(--text-xs); color: var(--color-muted); }
.message__body { font-size: var(--text-sm); line-height: 1.6; white-space: pre-wrap; word-break: break-word; }

.internal-badge { font-size: var(--text-xs); font-weight: var(--font-medium); padding: 1px 6px; margin-inline-start: var(--space-2); border-radius: var(--radius-full); background: rgba(245, 158, 11, 0.15); color: #b45309; }

/* Reply Section */
.reply-section { background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); padding: var(--space-4); }
.reply-section__header { display: flex; justify-content: space-between; align-items: center; margin-bottom: var(--space-3); }
.reply-section__title { font-size: var(--text-sm); font-weight: var(--font-semibold); }

.reply-form { display: flex; flex-direction: column; gap: var(--space-3); }
.reply-form__actions { display: flex; justify-content: space-between; align-items: center; }

.internal-toggle { display: flex; align-items: center; gap: var(--space-2); cursor: pointer; font-size: var(--text-sm); color: var(--color-muted); }
.internal-toggle__checkbox { width: 16px; height: 16px; accent-color: var(--color-warning); }
.internal-toggle__label { user-select: none; }

/* Canned Dropdown */
.canned-dropdown-wrapper { position: relative; }
.canned-dropdown { position: absolute; top: 100%; right: 0; z-index: 20; min-width: 260px; max-height: 200px; overflow-y: auto; background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-md); box-shadow: var(--shadow-lg); }
.canned-dropdown__item { padding: var(--space-2) var(--space-3); cursor: pointer; transition: background var(--duration-fast); }
.canned-dropdown__item:hover { background: var(--color-surface-2); }
.canned-dropdown__title { display: block; font-size: var(--text-sm); font-weight: var(--font-medium); }
.canned-dropdown__preview { display: block; font-size: var(--text-xs); color: var(--color-muted); margin-top: 2px; }

/* Sidebar */
.ticket-sidebar { display: flex; flex-direction: column; gap: var(--space-3); padding: var(--space-4); background: var(--color-surface); border: 1px solid var(--color-border); border-radius: var(--radius-lg); height: fit-content; position: sticky; top: var(--space-4); }

.sidebar-section { display: flex; flex-direction: column; gap: var(--space-1); }
.sidebar-label { font-size: var(--text-xs); font-weight: var(--font-semibold); color: var(--color-muted); text-transform: uppercase; letter-spacing: 0.05em; }
.sidebar-value { font-size: var(--text-sm); }
.sidebar-value--stars { font-size: var(--text-base); }
.star--filled { color: #f59e0b; }
.star--empty { color: var(--color-muted); opacity: 0.3; }
.sidebar-input { font-size: var(--text-sm); padding: var(--space-2); background: var(--color-surface-2); border: 1px solid var(--color-border); border-radius: var(--radius-sm); color: var(--color-text); width: 100%; }
.sidebar-input:focus { outline: none; border-color: var(--color-primary); }

/* Utility */
.text-muted { color: var(--color-muted); }
.text-center { text-align: center; }
.empty-state { text-align: center; padding: var(--space-12); }

@media (max-width: 900px) {
  .ticket-layout { grid-template-columns: 1fr; }
  .ticket-sidebar { position: static; }
}
</style>
