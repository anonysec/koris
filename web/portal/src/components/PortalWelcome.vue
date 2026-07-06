<script setup lang="ts">
/**
 * Portal welcome / first-connect checklist for customers.
 *
 * Guides new customers through the "download config → install client → connect"
 * flow. Auto-hides once we detect an active VPN session (usage stats > 0
 * suggests they've connected), or if the customer explicitly dismisses.
 *
 * Zero backend changes — reads existing stores.
 */
import { computed, ref } from 'vue'
import { useUsageStore } from '@/stores/usage'

const usage = useUsageStore()

const DISMISS_KEY = 'koris.portal.welcome.dismissed'
const dismissed = ref<boolean>(
  typeof localStorage !== 'undefined' && localStorage.getItem(DISMISS_KEY) === '1',
)

// Any traffic recorded = the customer has connected at least once.
const hasConnected = computed(() => {
  const u = usage.usage
  if (!u) return false
  const rx = (u as any).rx_bytes || (u as any).download_bytes || 0
  const tx = (u as any).tx_bytes || (u as any).upload_bytes || 0
  return rx + tx > 0
})

const visible = computed(() => !dismissed.value && !hasConnected.value)

function dismiss() {
  dismissed.value = true
  if (typeof localStorage !== 'undefined') {
    localStorage.setItem(DISMISS_KEY, '1')
  }
}

// Scroll to the configs section (inside SinglePage) — respect existing DOM
function scrollToConfigs() {
  const el =
    document.querySelector('[data-section="configs"]') ||
    document.querySelector('.configs, .vpn-profiles, .profiles-section')
  if (el) {
    el.scrollIntoView({ behavior: 'smooth', block: 'start' })
  }
}

interface Step {
  n: number
  title: string
  desc: string
  icon: string
}

const steps: Step[] = [
  {
    n: 1,
    title: 'Download your VPN config',
    desc: 'Pick your device below — Windows, macOS, iOS, Android, or Linux.',
    icon: '📥',
  },
  {
    n: 2,
    title: 'Install the client app',
    desc: 'OpenVPN Connect, WireGuard, or your OS-native VPN — links in the config download.',
    icon: '📦',
  },
  {
    n: 3,
    title: 'Import & connect',
    desc: "Open the config in the client app and hit Connect. That's it.",
    icon: '⚡',
  },
]
</script>

<template>
  <section v-if="visible" class="p-welcome" aria-label="First-connect guide">
    <header class="p-welcome__head">
      <div>
        <h2 class="p-welcome__title">Welcome — let's get you online</h2>
        <p class="p-welcome__sub">Three quick steps. Under 2 minutes.</p>
      </div>
      <button class="p-welcome__dismiss" @click="dismiss" title="Dismiss">
        <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2">
          <path d="M18 6L6 18M6 6l12 12" />
        </svg>
      </button>
    </header>

    <ol class="p-welcome__steps">
      <li v-for="step in steps" :key="step.n" class="p-step">
        <span class="p-step__n">{{ step.n }}</span>
        <span class="p-step__icon">{{ step.icon }}</span>
        <div class="p-step__body">
          <div class="p-step__title">{{ step.title }}</div>
          <div class="p-step__desc">{{ step.desc }}</div>
        </div>
      </li>
    </ol>

    <div class="p-welcome__cta">
      <button class="p-welcome__btn" @click="scrollToConfigs">
        Go to my configs
        <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4">
          <path d="M12 5v14M5 12l7 7 7-7" />
        </svg>
      </button>
    </div>
  </section>
</template>

<style scoped>
.p-welcome {
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
.p-welcome::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 4px;
  background: var(--gradient-brand);
}
.p-welcome::after {
  content: '';
  position: absolute;
  top: -30%;
  left: 40%;
  width: 50%;
  height: 160%;
  background: radial-gradient(
    closest-side,
    color-mix(in srgb, var(--color-accent) 10%, transparent),
    transparent
  );
  filter: blur(20px);
  pointer-events: none;
  z-index: -1;
}

.p-welcome__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  margin-bottom: 18px;
}
.p-welcome__title {
  font-size: var(--text-xl);
  font-weight: var(--font-semibold);
  letter-spacing: var(--tracking-tight);
  margin: 0 0 2px;
}
.p-welcome__sub {
  font-size: var(--text-sm);
  color: var(--color-muted);
  margin: 0;
}
.p-welcome__dismiss {
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
.p-welcome__dismiss:hover {
  background: var(--color-surface-2);
  color: var(--color-text);
}

.p-welcome__steps {
  list-style: none;
  padding: 0;
  margin: 0 0 16px;
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 12px;
}
@media (max-width: 720px) {
  .p-welcome__steps { grid-template-columns: 1fr; }
}

.p-step {
  padding: 14px 16px;
  border-radius: var(--radius-md);
  background: color-mix(in srgb, var(--color-surface-2) 55%, transparent);
  border: 1px solid transparent;
  transition: border-color var(--duration-fast);
  display: flex;
  align-items: flex-start;
  gap: 12px;
}
.p-step:hover {
  border-color: color-mix(in srgb, var(--color-primary) 30%, transparent);
}
.p-step__n {
  display: grid;
  place-items: center;
  width: 24px;
  height: 24px;
  border-radius: 50%;
  background: var(--gradient-brand);
  color: #fff;
  font-size: var(--text-xs);
  font-weight: var(--font-bold);
  flex-shrink: 0;
}
.p-step__icon {
  font-size: 18px;
  flex-shrink: 0;
}
.p-step__body { min-width: 0; }
.p-step__title {
  font-size: var(--text-md);
  font-weight: var(--font-semibold);
  color: var(--color-text);
  line-height: 1.3;
  margin-bottom: 3px;
}
.p-step__desc {
  font-size: var(--text-sm);
  color: var(--color-muted);
  line-height: 1.4;
}

.p-welcome__cta {
  display: flex;
  justify-content: flex-end;
}
.p-welcome__btn {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 9px 16px;
  border-radius: var(--radius-md);
  border: none;
  background: var(--gradient-brand);
  color: #fff;
  font-weight: var(--font-medium);
  font-size: var(--text-sm);
  cursor: pointer;
  box-shadow: var(--shadow-brand);
  transition: transform var(--duration-fast), filter var(--duration-fast);
}
.p-welcome__btn:hover {
  transform: translateY(-1px);
  filter: brightness(1.08);
}
</style>
