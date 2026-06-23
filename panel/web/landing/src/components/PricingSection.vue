<script setup lang="ts">
import { ref, onMounted, computed } from 'vue'
import { useI18n } from '../i18n'

const { t } = useI18n()

interface Plan {
  id: number
  name: string
  price: number
  data_gb: number
  speed_mbps: number
  duration_days: number
  features: string[]
  popular?: boolean
}

const plans = ref<Plan[]>([])
const loading = ref(true)
const error = ref(false)

async function fetchPlans() {
  try {
    const res = await fetch('/api/public-plans')
    if (!res.ok) throw new Error('Failed to fetch plans')
    const data = await res.json()
    if (data.ok && Array.isArray(data.plans)) {
      plans.value = data.plans
    } else {
      plans.value = []
    }
  } catch {
    error.value = true
  } finally {
    loading.value = false
  }
}

const highlightedPlans = computed(() => {
  if (plans.value.length === 0) return []
  const hasPopular = plans.value.some((p) => p.popular)
  if (hasPopular) return plans.value
  // Mark the middle plan as popular
  const midIndex = Math.floor(plans.value.length / 2)
  return plans.value.map((p, i) => ({ ...p, popular: i === midIndex }))
})

function formatDuration(days: number): string {
  if (days >= 365) return `${Math.round(days / 365)} year${days >= 730 ? 's' : ''}`
  if (days >= 30) return `${Math.round(days / 30)} month${days >= 60 ? 's' : ''}`
  return `${days} day${days !== 1 ? 's' : ''}`
}

onMounted(fetchPlans)
</script>

<template>
  <section class="pricing" id="pricing">
    <div class="pricing-container">
      <h2 class="pricing-title">{{ t('pricing.title') }}</h2>
      <p class="pricing-subtitle">
        {{ t('pricing.subtitle') }}
      </p>

      <!-- Loading skeleton -->
      <div v-if="loading" class="pricing-grid">
        <div v-for="n in 3" :key="n" class="pricing-card skeleton-card">
          <div class="skeleton-line skeleton-name"></div>
          <div class="skeleton-line skeleton-price"></div>
          <div class="skeleton-line skeleton-meta"></div>
          <div class="skeleton-line skeleton-meta"></div>
          <div class="skeleton-features">
            <div v-for="f in 3" :key="f" class="skeleton-line skeleton-feature"></div>
          </div>
          <div class="skeleton-line skeleton-btn"></div>
        </div>
      </div>

      <!-- Error state -->
      <div v-else-if="error" class="pricing-empty">
        <span class="pricing-empty-icon">⚠️</span>
        <p class="pricing-empty-text">{{ t('pricing.error') }}</p>
        <button class="pricing-retry-btn" @click="loading = true; error = false; fetchPlans()">
          {{ t('pricing.retry') }}
        </button>
      </div>

      <!-- Empty state -->
      <div v-else-if="highlightedPlans.length === 0" class="pricing-empty">
        <span class="pricing-empty-icon">📋</span>
        <p class="pricing-empty-text">{{ t('pricing.empty') }}</p>
      </div>

      <!-- Plans grid -->
      <div v-else class="pricing-grid">
        <div
          v-for="plan in highlightedPlans"
          :key="plan.id"
          class="pricing-card"
          :class="{ popular: plan.popular }"
        >
          <span v-if="plan.popular" class="popular-badge">{{ t('pricing.popular') }}</span>
          <h3 class="plan-name">{{ plan.name }}</h3>
          <div class="plan-price">
            <span class="price-amount">${{ plan.price }}</span>
            <span class="price-period">/ {{ formatDuration(plan.duration_days) }}</span>
          </div>
          <ul class="plan-meta">
            <li><span class="meta-label">{{ t('pricing.data') }}</span> {{ plan.data_gb }} GB</li>
            <li><span class="meta-label">{{ t('pricing.speed') }}</span> {{ plan.speed_mbps }} Mbps</li>
            <li><span class="meta-label">{{ t('pricing.duration') }}</span> {{ formatDuration(plan.duration_days) }}</li>
          </ul>
          <ul v-if="plan.features && plan.features.length" class="plan-features">
            <li v-for="(feature, i) in plan.features" :key="i">
              <span class="feature-check" aria-hidden="true">✓</span>
              {{ feature }}
            </li>
          </ul>
          <a
            :href="`/portal/?plan=${plan.id}`"
            class="plan-cta"
            :class="{ 'cta-popular': plan.popular }"
          >
            {{ t('pricing.cta') }}
          </a>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.pricing {
  padding: 6rem 2rem;
  background: #070a12;
}

.pricing-container {
  max-width: 1200px;
  margin: 0 auto;
}

.pricing-title {
  text-align: center;
  font-size: clamp(1.75rem, 3.5vw, 2.5rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
  letter-spacing: -0.01em;
}

.pricing-subtitle {
  text-align: center;
  font-size: clamp(0.95rem, 1.5vw, 1.1rem);
  color: #94a3b8;
  margin: 0 0 3.5rem;
  max-width: 560px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.6;
}

/* Grid */
.pricing-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
  gap: 1.5rem;
  align-items: stretch;
}

/* Card */
.pricing-card {
  position: relative;
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 16px;
  padding: 2rem 1.5rem;
  display: flex;
  flex-direction: column;
  transition: transform 0.3s ease, box-shadow 0.3s ease, border-color 0.3s ease;
}

.pricing-card:hover {
  transform: translateY(-4px);
  border-color: rgba(99, 102, 241, 0.2);
  box-shadow: 0 8px 32px rgba(99, 102, 241, 0.06);
}

.pricing-card.popular {
  border-color: rgba(99, 102, 241, 0.4);
  background: rgba(99, 102, 241, 0.05);
  box-shadow: 0 4px 24px rgba(99, 102, 241, 0.1);
}

.popular-badge {
  position: absolute;
  top: -12px;
  left: 50%;
  transform: translateX(-50%);
  background: linear-gradient(135deg, #6366f1, #818cf8);
  color: #fff;
  font-size: 0.75rem;
  font-weight: 600;
  padding: 0.25rem 1rem;
  border-radius: 999px;
  letter-spacing: 0.02em;
  text-transform: uppercase;
}

/* Plan name */
.plan-name {
  font-size: 1.25rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 0.5rem 0 1rem;
  text-align: center;
}

/* Price */
.plan-price {
  text-align: center;
  margin-bottom: 1.25rem;
}

.price-amount {
  font-size: 2.25rem;
  font-weight: 700;
  color: #f1f5f9;
  letter-spacing: -0.02em;
}

.price-period {
  font-size: 0.9rem;
  color: #94a3b8;
  margin-left: 0.25rem;
}

/* Meta (data, speed, duration) */
.plan-meta {
  list-style: none;
  padding: 0;
  margin: 0 0 1.25rem;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
}

.plan-meta li {
  font-size: 0.9rem;
  color: #cbd5e1;
}

.meta-label {
  color: #94a3b8;
  font-weight: 500;
}

/* Features */
.plan-features {
  list-style: none;
  padding: 0;
  margin: 0 0 1.5rem;
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  border-top: 1px solid rgba(255, 255, 255, 0.06);
  padding-top: 1.25rem;
}

.plan-features li {
  font-size: 0.85rem;
  color: #cbd5e1;
  display: flex;
  align-items: center;
  gap: 0.5rem;
}

.feature-check {
  color: #6366f1;
  font-weight: 700;
  font-size: 0.9rem;
  flex-shrink: 0;
}

/* CTA button */
.plan-cta {
  display: block;
  text-align: center;
  padding: 0.75rem 1.5rem;
  border-radius: 8px;
  font-size: 0.95rem;
  font-weight: 600;
  text-decoration: none;
  color: #e2e8f0;
  background: rgba(255, 255, 255, 0.06);
  border: 1px solid rgba(255, 255, 255, 0.1);
  transition: background 0.2s ease, border-color 0.2s ease, transform 0.2s ease;
  margin-top: auto;
}

.plan-cta:hover {
  background: rgba(99, 102, 241, 0.15);
  border-color: rgba(99, 102, 241, 0.3);
  transform: translateY(-1px);
}

.plan-cta.cta-popular {
  background: linear-gradient(135deg, #6366f1, #818cf8);
  border-color: transparent;
  color: #fff;
}

.plan-cta.cta-popular:hover {
  opacity: 0.9;
  transform: translateY(-1px);
}

/* Empty / Error states */
.pricing-empty {
  text-align: center;
  padding: 3rem 1rem;
}

.pricing-empty-icon {
  display: block;
  font-size: 2.5rem;
  margin-bottom: 1rem;
}

.pricing-empty-text {
  color: #94a3b8;
  font-size: 1rem;
  margin: 0 0 1.5rem;
}

.pricing-retry-btn {
  background: rgba(99, 102, 241, 0.15);
  border: 1px solid rgba(99, 102, 241, 0.3);
  color: #a5b4fc;
  font-size: 0.9rem;
  font-weight: 500;
  padding: 0.5rem 1.25rem;
  border-radius: 8px;
  cursor: pointer;
  transition: background 0.2s ease;
}

.pricing-retry-btn:hover {
  background: rgba(99, 102, 241, 0.25);
}

/* Skeleton loading */
.skeleton-card {
  display: flex;
  flex-direction: column;
  gap: 1rem;
}

.skeleton-line {
  background: rgba(255, 255, 255, 0.05);
  border-radius: 6px;
  animation: skeleton-pulse 1.5s ease-in-out infinite;
}

.skeleton-name {
  height: 1.5rem;
  width: 60%;
  margin: 0 auto;
}

.skeleton-price {
  height: 2.5rem;
  width: 45%;
  margin: 0 auto;
}

.skeleton-meta {
  height: 1rem;
  width: 80%;
}

.skeleton-features {
  display: flex;
  flex-direction: column;
  gap: 0.5rem;
  margin-top: 0.5rem;
}

.skeleton-feature {
  height: 0.9rem;
  width: 90%;
}

.skeleton-btn {
  height: 2.5rem;
  width: 100%;
  margin-top: auto;
}

@keyframes skeleton-pulse {
  0%, 100% { opacity: 0.4; }
  50% { opacity: 0.8; }
}

/* Responsive */
@media (max-width: 900px) {
  .pricing-grid {
    grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
  }
}

@media (max-width: 540px) {
  .pricing {
    padding: 4rem 1.25rem;
  }

  .pricing-grid {
    grid-template-columns: 1fr;
    gap: 1.25rem;
  }

  .pricing-card {
    padding: 1.5rem 1.25rem;
  }
}
</style>
