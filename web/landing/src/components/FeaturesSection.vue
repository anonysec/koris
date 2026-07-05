<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useI18n } from '../i18n'

const { t } = useI18n()

interface Feature {
  icon: string
  titleKey: string
  descKey: string
}

const featureKeys: Feature[] = [
  { icon: '🔐', titleKey: 'features.0.title', descKey: 'features.0.desc' },
  { icon: '📡', titleKey: 'features.1.title', descKey: 'features.1.desc' },
  { icon: '💳', titleKey: 'features.2.title', descKey: 'features.2.desc' },
  { icon: '👤', titleKey: 'features.3.title', descKey: 'features.3.desc' },
  { icon: '🤖', titleKey: 'features.4.title', descKey: 'features.4.desc' },
  { icon: '🌐', titleKey: 'features.5.title', descKey: 'features.5.desc' },
  { icon: '🔄', titleKey: 'features.6.title', descKey: 'features.6.desc' },
  { icon: '🌍', titleKey: 'features.7.title', descKey: 'features.7.desc' },
]

const cardRefs = ref<HTMLElement[]>([])
let observer: IntersectionObserver | null = null

function setCardRef(el: any, index: number) {
  if (el) cardRefs.value[index] = el
}

onMounted(() => {
  observer = new IntersectionObserver(
    (entries) => {
      entries.forEach((entry) => {
        if (entry.isIntersecting) {
          entry.target.classList.add('visible')
          observer?.unobserve(entry.target)
        }
      })
    },
    { threshold: 0.15, rootMargin: '0px 0px -40px 0px' }
  )

  cardRefs.value.forEach((card) => {
    if (card) observer?.observe(card)
  })
})

onUnmounted(() => {
  observer?.disconnect()
})
</script>

<template>
  <section class="features" id="features">
    <div class="features-container">
      <h2 class="features-title">{{ t('features.title') }}</h2>
      <p class="features-subtitle">
        {{ t('features.subtitle') }}
      </p>

      <div class="features-grid">
        <div
          v-for="(feature, index) in featureKeys"
          :key="index"
          :ref="(el) => setCardRef(el, index)"
          class="feature-card"
          :style="{ transitionDelay: `${index * 80}ms` }"
        >
          <span class="feature-icon" aria-hidden="true">{{ feature.icon }}</span>
          <h3 class="feature-card-title">{{ t(feature.titleKey) }}</h3>
          <p class="feature-card-desc">{{ t(feature.descKey) }}</p>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.features {
  padding: 6rem 2rem;
  background: #070a12;
}

.features-container {
  max-width: 1200px;
  margin: 0 auto;
}

.features-title {
  text-align: center;
  font-size: clamp(1.75rem, 3.5vw, 2.5rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
  letter-spacing: -0.01em;
}

.features-subtitle {
  text-align: center;
  font-size: clamp(0.95rem, 1.5vw, 1.1rem);
  color: #94a3b8;
  margin: 0 0 3.5rem;
  max-width: 560px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.6;
}

.features-grid {
  display: grid;
  grid-template-columns: repeat(3, 1fr);
  gap: 1.5rem;
}

/* Card styles */
.feature-card {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  padding: 2rem 1.5rem;
  transition: transform 0.3s ease, box-shadow 0.3s ease, border-color 0.3s ease,
    opacity 0.6s ease, translate 0.6s ease;
  opacity: 0;
  translate: 0 24px;
}

.feature-card.visible {
  opacity: 1;
  translate: 0 0;
}

.feature-card:hover {
  transform: translateY(-4px);
  border-color: rgba(99, 102, 241, 0.25);
  box-shadow: 0 8px 32px rgba(99, 102, 241, 0.08);
}

.feature-icon {
  display: block;
  font-size: 2rem;
  margin-bottom: 1rem;
}

.feature-card-title {
  font-size: 1.1rem;
  font-weight: 600;
  color: #e2e8f0;
  margin: 0 0 0.5rem;
}

.feature-card-desc {
  font-size: 0.9rem;
  color: #94a3b8;
  line-height: 1.6;
  margin: 0;
}

/* Responsive: tablet — 2 columns */
@media (max-width: 900px) {
  .features-grid {
    grid-template-columns: repeat(2, 1fr);
  }
}

/* Responsive: mobile — 1 column */
@media (max-width: 540px) {
  .features {
    padding: 4rem 1.25rem;
  }

  .features-grid {
    grid-template-columns: 1fr;
    gap: 1.25rem;
  }

  .feature-card {
    padding: 1.5rem 1.25rem;
  }
}

/* Respect reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  .feature-card {
    opacity: 1;
    translate: 0 0;
    transition: transform 0.3s ease, box-shadow 0.3s ease, border-color 0.3s ease;
  }
}
</style>
