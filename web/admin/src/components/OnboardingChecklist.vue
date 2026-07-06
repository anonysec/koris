<script setup lang="ts">
/**
 * Progressive-disclosure onboarding for new admins.
 *
 * Renders only when the panel has zero of anything essential
 * (nodes, plans, or customers). Auto-hides once the admin has
 * added at least one of each, or after they explicitly dismiss.
 * Dismissal is stored in localStorage per-browser.
 */
import { computed, ref } from 'vue'
import { useRouter } from 'vue-router'
import { useNodesStore } from '@/stores/nodes'
import { useCustomersStore } from '@/stores/customers'
import { usePlansStore } from '@/stores/plans'

const router = useRouter()
const nodes = useNodesStore()
const customers = useCustomersStore()
const plans = usePlansStore()

const DISMISS_KEY = 'koris.onboarding.dismissed'
const dismissed = ref<boolean>(
  typeof localStorage !== 'undefined' && localStorage.getItem(DISMISS_KEY) === '1',
)

interface Step {
  id: 'node' | 'plan' | 'customer'
  title: string
  desc: string
  done: boolean
  cta: string
  goto: () => void
  icon: string
}

const steps = computed<Step[]>(() => [
  {
    id: 'node',
    title: 'Add your first VPN node',
    desc: 'Install knode on a server to start terminating VPN traffic.',
    done: (nodes.nodes?.length ?? 0) > 0,
    cta: 'Add node',
    goto: () => router.push({ name: 'nodes' }),
    icon: '🖥️',
  },
  {
    id: 'plan',
    title: 'Create a subscription plan',
    desc: 'Define quota, price, and duration so customers can subscribe.',
    done: (plans.plans?.length ?? 0) > 0,
    cta: 'Create plan',
    goto: () => router.push({ name: 'plans' }),
    icon: '💳',
  },
  {
    id: 'customer',
    title: 'Add your first customer',
    desc: 'Or share the portal signup URL and let them self-onboard.',
    done: (customers.customers?.length ?? 0) > 0,
    cta: 'Add customer',
    goto: () => router.push({ name: 'users' }),
    icon: '👤',
  },
])

const completedCount = computed(() => steps.value.filter((s) => s.done).length)
const totalCount = computed(() => steps.value.length)
const allDone = computed(() => completedCount.value === totalCount.value)

// Show if not dismissed AND at least one step is incomplete
const visible = computed(() => !dismissed.value && !allDone.value)

function dismiss() {
  dismissed.value = true
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(DISMISS_KEY, '1')
  }
}

const progressPct = computed(() =>
  Math.round((completedCount.value / totalCount.value) * 100),
)
</script>

<template>
  <section v-if="visible" class="onboarding" aria-label="Setup checklist">
    <header class="onboarding__head">
      <div>
        <h2 class="onboarding__title">Welcome — let's get you set up</h2>
        <p class="onboarding__sub">
          {{ completedCount }} of {{ totalCount }} done · takes about 3 minutes
        </p>
      </div>
      <button class="onboarding__dismiss" @click="dismiss" title="Dismiss checklist">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
    </header>

    <div class="onboarding__progress">
      <div class="onboarding__progress-bar" :style="{ width: progressPct + '%' }"></div>
    </div>

    <ol class="onboarding__steps">
      <li
        v-for="(step, i) in steps"
        :key="step.id"
        class="step"
        :class="{ 'step--done': step.done }"
      >
        <span class="step__check" :aria-label="step.done ? 'Completed' : `Step ${i + 1}`">
          <svg v-if="step.done" width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.5">
            <path d="M20 6L9 17l-5-5" />
          </svg>
          <span v-else>{{ i + 1 }}</span>
        </span>
        <span class="step__icon">{{ step.icon }}</span>
        <div class="step__body">
          <div class="step__title">{{ step.title }}</div>
          <div class="step__desc">{{ step.desc }}</div>
        </div>
        <button
          v-if="!step.done"
          class="step__cta"
          @click="step.goto"
        >
          {{ step.cta }}
          <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4">
            <path d="M5 12h14M13 5l7 7-7 7" />
          </svg>
        </button>
        <span v-else class="step__done-label">Done</span>
      </li>
    </ol>
  </section>
</template>

<style scoped>
.onboarding {
  position: relative;
  margin-bottom: 24px;
  padding: 22px 24px;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  overflow: hidden;
  isolation: isolate;
}
.onboarding::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: var(--gradient-brand);
  opacity: 0.9;
}
.onboarding::after {
  content: '';
  position: absolute;
  top: -50%;
  right: -20%;
  width: 40%;
  height: 200%;
  background: radial-gradient(
    closest-side,
    color-mix(in srgb, var(--color-accent) 12%, transparent),
    transparent
  );
  filter: blur(20px);
  pointer-events: none;
  z-index: -1;
}

.onboarding__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 14px;
}
.onboarding__title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  letter-spacing: var(--tracking-tight);
  margin: 0 0 2px;
  color: var(--color-text);
}
.onboarding__sub {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0;
}

.onboarding__dismiss {
  display: grid;
  place-items: center;
  width: 30px;
  height: 30px;
  padding: 0;
  border: none;
  background: transparent;
  color: var(--color-muted);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: background var(--duration-fast), color var(--duration-fast);
}
.onboarding__dismiss:hover {
  background: var(--color-surface-2);
  color: var(--color-text);
}

.onboarding__progress {
  height: 4px;
  border-radius: 999px;
  background: var(--color-surface-2);
  overflow: hidden;
  margin-bottom: 18px;
}
.onboarding__progress-bar {
  height: 100%;
  background: var(--gradient-brand);
  transition: width var(--duration-slow, 0.2s) var(--ease-out);
}

.onboarding__steps {
  list-style: none;
  padding: 0;
  margin: 0;
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.step {
  display: flex;
  align-items: center;
  gap: 14px;
  padding: 12px 14px;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--color-surface-2) 55%, transparent);
  border: 1px solid transparent;
  transition: border-color var(--duration-fast);
}
.step:hover:not(.step--done) {
  border-color: color-mix(in srgb, var(--color-primary) 30%, transparent);
}
.step--done {
  background: color-mix(in srgb, var(--color-success) 5%, transparent);
}

.step__check {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  background: var(--color-surface);
  border: 1px solid var(--color-border);
  font-size: var(--text-sm);
  font-weight: var(--font-semibold);
  color: var(--color-muted);
  flex-shrink: 0;
}
.step--done .step__check {
  background: var(--color-success);
  border-color: var(--color-success);
  color: #fff;
}

.step__icon {
  font-size: 20px;
  flex-shrink: 0;
}

.step__body {
  flex: 1;
  min-width: 0;
}
.step__title {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  line-height: 1.3;
}
.step--done .step__title {
  color: var(--color-muted);
  text-decoration: line-through;
  text-decoration-color: color-mix(in srgb, var(--color-muted) 40%, transparent);
}
.step__desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin-top: 2px;
}

.step__cta {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 14px;
  border-radius: var(--radius-md);
  border: none;
  background: var(--gradient-brand);
  color: #fff;
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
  cursor: pointer;
  box-shadow: var(--shadow-brand);
  transition: transform var(--duration-fast), filter var(--duration-fast);
  flex-shrink: 0;
}
.step__cta:hover {
  transform: translateY(-1px);
  filter: brightness(1.08);
}

.step__done-label {
  font-size: var(--text-xs);
  font-weight: var(--font-semibold);
  color: var(--color-success);
  text-transform: uppercase;
  letter-spacing: var(--tracking-wide);
  flex-shrink: 0;
}
</style>
