<script setup lang="ts">
import { onMounted, ref, computed } from 'vue'
import { usePortalTicketsStore } from '@/stores/tickets'
import { formatDate } from '@koris/composables/useFormatDate'
import { useI18n } from '@koris/composables/useI18n'
import Button from '@koris/ui/Button.vue'
import DataTable from '@koris/ui/DataTable.vue'
import FormField from '@koris/ui/FormField.vue'
import Input from '@koris/ui/Input.vue'
import Select from '@koris/ui/Select.vue'
import Textarea from '@koris/ui/Textarea.vue'
import StatusPill from '@koris/ui/StatusPill.vue'
import Skeleton from '@koris/ui/Skeleton.vue'
import EmptyState from '@koris/ui/EmptyState.vue'
import TicketThread from '@/components/TicketThread.vue'

const { t } = useI18n()
const ticketsStore = usePortalTicketsStore()

const categoryOptions = computed(() => [
  { label: t('portal.support.categoryGeneral'), value: 'general' },
  { label: t('portal.support.categoryTechnical'), value: 'technical' },
  { label: t('portal.support.categoryBilling'), value: 'billing' },
])

const priorityOptions = computed(() => [
  { label: t('portal.support.priorityLow'), value: 'low' },
  { label: t('portal.support.priorityMedium'), value: 'medium' },
  { label: t('portal.support.priorityHigh'), value: 'high' },
])

const showCreateForm = ref(false)
const ticketForm = ref({
  subject: '',
  category: 'general',
  priority: 'medium',
  body: '',
})
const replyMessage = ref('')
const notice = ref('')
const ratingValue = ref(0)
const ratingSubmitted = ref(false)

onMounted(() => {
  ticketsStore.loadTickets()
})

const selectedTicket = computed(() => ticketsStore.detail)

/** Whether the selected ticket can be rated (resolved/closed and not yet rated) */
const canRate = computed(() => {
  if (!selectedTicket.value) return false
  const status = selectedTicket.value.status
  return (status === 'resolved' || status === 'closed') && !selectedTicket.value.satisfaction_rating
})

/** Whether the selected ticket is open for replies */
const canReply = computed(() => {
  if (!selectedTicket.value) return false
  const status = selectedTicket.value.status
  return status === 'open' || status === 'in_progress' || status === 'waiting'
})

const ticketColumns = [
  { key: 'subject', label: t('portal.support.subject') },
  { key: 'category', label: t('portal.support.category') },
  { key: 'priority', label: t('portal.support.priority') },
  { key: 'status', label: t('portal.support.status'), sortable: true },
  { key: 'created_at', label: t('portal.support.date'), sortable: true },
  { key: 'actions', label: '' },
]

async function handleCreateTicket() {
  if (!ticketForm.value.subject || !ticketForm.value.body) return
  notice.value = ''
  const id = await ticketsStore.createTicket({
    subject: ticketForm.value.subject,
    category: ticketForm.value.category,
    priority: ticketForm.value.priority,
    body: ticketForm.value.body,
  })
  if (id) {
    notice.value = t('portal.support.ticketCreated')
    ticketForm.value = { subject: '', category: 'general', priority: 'medium', body: '' }
    showCreateForm.value = false
    await ticketsStore.loadTicketDetail(id)
  }
}

async function handleViewTicket(id: number) {
  ratingValue.value = 0
  ratingSubmitted.value = false
  await ticketsStore.loadTicketDetail(id)
}

async function handleReply() {
  if (!selectedTicket.value || !replyMessage.value.trim()) return
  notice.value = ''
  const success = await ticketsStore.replyToTicket(selectedTicket.value.id, replyMessage.value)
  if (success) {
    notice.value = t('portal.support.replySent')
    replyMessage.value = ''
  }
}

async function handleCloseTicket() {
  if (!selectedTicket.value) return
  const success = await ticketsStore.closeTicket(selectedTicket.value.id)
  if (success) {
    notice.value = t('portal.support.ticketClosed')
  }
}

async function handleRate() {
  if (!selectedTicket.value || ratingValue.value < 1 || ratingValue.value > 5) return
  const success = await ticketsStore.rateTicket(selectedTicket.value.id, ratingValue.value)
  if (success) {
    ratingSubmitted.value = true
    notice.value = t('portal.support.ratingSubmitted')
  }
}

function handleBack() {
  ticketsStore.clearDetail()
  ratingValue.value = 0
  ratingSubmitted.value = false
}

function statusVariant(status: string) {
  switch (status) {
    case 'open':
    case 'in_progress':
      return 'active'
    case 'resolved':
      return 'expired'
    case 'closed':
      return 'disabled'
    default:
      return 'expired'
  }
}
</script>
<template>
  <div class="support">
    <div class="support__header">
      <h1 class="support__title">{{ t('portal.support.title') }}</h1>
      <Button
        v-if="!selectedTicket && !showCreateForm"
        variant="primary"
        size="sm"
        @click="showCreateForm = true"
      >
        + {{ t('portal.support.newTicket') }}
      </Button>
      <Button
        v-if="selectedTicket"
        variant="ghost"
        size="sm"
        @click="handleBack"
      >
        ← {{ t('portal.support.backToList') }}
      </Button>
    </div>

    <div v-if="notice" class="support__notice" role="status">{{ notice }}</div>

    <Skeleton v-if="ticketsStore.loading && !ticketsStore.list.length && !selectedTicket" variant="card" :count="2" />

    <!-- Create Ticket Form -->
    <section v-else-if="showCreateForm && !selectedTicket" class="support__section">
      <h2 class="support__section-title">{{ t('portal.support.createTitle') }}</h2>
      <form class="support__form" @submit.prevent="handleCreateTicket">
        <FormField :label="t('portal.support.subject')" :required="true">
          <Input v-model="ticketForm.subject" :placeholder="t('portal.support.subjectPlaceholder')" />
        </FormField>

        <FormField :label="t('portal.support.category')">
          <Select v-model="ticketForm.category" :options="categoryOptions" />
        </FormField>

        <FormField :label="t('portal.support.priority')">
          <Select v-model="ticketForm.priority" :options="priorityOptions" />
        </FormField>

        <FormField :label="t('portal.support.message')" :required="true">
          <Textarea v-model="ticketForm.body" :placeholder="t('portal.support.messagePlaceholder')" :rows="5" />
        </FormField>

        <div class="support__form-actions">
          <Button variant="ghost" @click="showCreateForm = false">{{ t('portal.support.cancel') }}</Button>
          <Button type="submit" variant="primary" :loading="ticketsStore.loading" :disabled="!ticketForm.subject || !ticketForm.body">
            {{ t('portal.support.create') }}
          </Button>
        </div>
      </form>
    </section>

    <!-- Ticket Detail View -->
    <template v-else-if="selectedTicket">
      <section class="support__section">
        <div class="support__ticket-header">
          <h2 class="support__section-title">#{{ selectedTicket.id }}: {{ selectedTicket.subject }}</h2>
          <div class="support__ticket-meta">
            <StatusPill :status="statusVariant(selectedTicket.status)">
              {{ selectedTicket.status }}
            </StatusPill>
            <span class="support__priority">{{ selectedTicket.priority }}</span>
            <span v-if="selectedTicket.category" class="support__category">{{ selectedTicket.category }}</span>
            <Button
              v-if="canReply"
              variant="ghost"
              size="sm"
              @click="handleCloseTicket"
            >
              {{ t('portal.support.closeTicket') }}
            </Button>
          </div>
        </div>

        <TicketThread :messages="selectedTicket.messages || []" />

        <!-- Reply form (only for open/in_progress/waiting tickets) -->
        <form v-if="canReply" class="support__reply-form" @submit.prevent="handleReply">
          <FormField :label="t('portal.support.yourReply')">
            <Textarea v-model="replyMessage" :placeholder="t('portal.support.replyPlaceholder')" :rows="3" />
          </FormField>
          <Button type="submit" variant="primary" :loading="ticketsStore.loading" :disabled="!replyMessage.trim()">
            {{ t('portal.support.send') }}
          </Button>
        </form>

        <!-- Satisfaction Survey (for resolved/closed tickets without rating) -->
        <div v-if="canRate && !ratingSubmitted" class="support__rating">
          <h3 class="support__rating-title">{{ t('portal.support.ratingTitle') }}</h3>
          <p class="support__rating-desc">{{ t('portal.support.ratingDesc') }}</p>
          <div class="support__stars">
            <button
              v-for="star in 5"
              :key="star"
              type="button"
              class="support__star"
              :class="{ 'support__star--active': star <= ratingValue }"
              :aria-label="`${star} star${star > 1 ? 's' : ''}`"
              @click="ratingValue = star"
            >
              ★
            </button>
          </div>
          <Button
            variant="primary"
            size="sm"
            :disabled="ratingValue < 1"
            :loading="ticketsStore.loading"
            @click="handleRate"
          >
            {{ t('portal.support.submitRating') }}
          </Button>
        </div>

        <!-- Already rated -->
        <div v-if="selectedTicket.satisfaction_rating" class="support__rated">
          <span class="support__rated-label">{{ t('portal.support.yourRating') }}:</span>
          <span class="support__rated-stars">
            <span v-for="star in 5" :key="star" :class="{ 'support__star--active': star <= selectedTicket.satisfaction_rating }">★</span>
          </span>
        </div>
      </section>
    </template>

    <!-- Ticket List -->
    <template v-else>
      <section class="support__section">
        <h2 class="support__section-title">{{ t('portal.support.myTickets') }}</h2>

        <EmptyState
          v-if="!ticketsStore.list.length"
          :title="t('portal.support.noTickets')"
          :description="t('portal.support.noTicketsDesc')"
          icon="🎫"
        />

        <DataTable
          v-else
          :columns="ticketColumns"
          :data="ticketsStore.list"
          :loading="ticketsStore.loading"
        >
          <template #cell-category="{ row }">
            {{ row.category || 'general' }}
          </template>
          <template #cell-priority="{ row }">
            <StatusPill status="expired">{{ row.priority }}</StatusPill>
          </template>
          <template #cell-status="{ row }">
            <StatusPill :status="statusVariant(row.status)">
              {{ row.status }}
            </StatusPill>
          </template>
          <template #cell-created_at="{ row }">
            {{ formatDate(row.created_at) }}
          </template>
          <template #cell-actions="{ row }">
            <Button variant="ghost" size="sm" @click="handleViewTicket(row.id)">
              {{ t('portal.support.view') }}
            </Button>
          </template>
        </DataTable>
      </section>
    </template>
  </div>
</template>
<style scoped>
.support__header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: var(--space-6);
}
.support__title {
  font-size: var(--text-2xl);
  font-weight: 700;
}
.support__notice {
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-md);
  background: rgba(34, 197, 94, 0.1);
  color: var(--color-success);
  font-size: var(--text-sm);
  margin-bottom: var(--space-4);
  border: 1px solid rgba(34, 197, 94, 0.2);
}
.support__section {
  padding: var(--space-5);
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  margin-bottom: var(--space-4);
}
.support__section-title {
  font-size: var(--text-md);
  font-weight: 600;
  margin-bottom: var(--space-4);
}
.support__form {
  display: flex;
  flex-direction: column;
  gap: var(--space-4);
  max-width: 500px;
}
.support__form-actions {
  display: flex;
  gap: var(--space-3);
  justify-content: flex-end;
}
.support__ticket-header {
  margin-bottom: var(--space-4);
}
.support__ticket-meta {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  margin-top: var(--space-2);
  flex-wrap: wrap;
}
.support__priority {
  font-size: var(--text-xs);
  color: var(--color-muted);
}
.support__category {
  font-size: var(--text-xs);
  color: var(--color-muted);
  padding: 2px var(--space-2);
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-sm);
}
.support__reply-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
}

/* Satisfaction Survey */
.support__rating {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  align-items: flex-start;
}
.support__rating-title {
  font-size: var(--text-sm);
  font-weight: 600;
  margin: 0;
}
.support__rating-desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0;
}
.support__stars {
  display: flex;
  gap: var(--space-1);
}
.support__star {
  background: none;
  border: none;
  font-size: 1.8rem;
  cursor: pointer;
  color: var(--color-border);
  transition: color 0.15s, transform 0.1s;
  padding: 0;
  line-height: 1;
}
.support__star:hover,
.support__star--active {
  color: #f59e0b;
  transform: scale(1.1);
}
.support__rated {
  margin-top: var(--space-4);
  padding-top: var(--space-4);
  border-top: 1px solid var(--color-border);
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.support__rated-label {
  font-size: var(--text-sm);
  color: var(--color-muted);
}
.support__rated-stars {
  display: flex;
  gap: 2px;
  font-size: 1.2rem;
  color: var(--color-border);
}
.support__rated-stars .support__star--active {
  color: #f59e0b;
}
</style>
