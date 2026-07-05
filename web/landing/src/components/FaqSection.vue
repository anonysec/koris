<script setup lang="ts">
import { ref, computed } from 'vue'
import { useI18n } from '../i18n'

const { t } = useI18n()

const faqCount = 7

const faqs = computed(() =>
  Array.from({ length: faqCount }, (_, i) => ({
    question: t(`faq.${i}.q`),
    answer: t(`faq.${i}.a`),
  }))
)

const openIndex = ref<number | null>(null)

function toggle(index: number) {
  openIndex.value = openIndex.value === index ? null : index
}
</script>

<template>
  <section class="faq" id="faq">
    <div class="faq-container">
      <h2 class="faq-title">{{ t('faq.title') }}</h2>
      <p class="faq-subtitle">
        {{ t('faq.subtitle') }}
      </p>

      <div class="faq-list">
        <div
          v-for="(item, index) in faqs"
          :key="index"
          class="faq-item"
          :class="{ active: openIndex === index }"
        >
          <button
            class="faq-trigger"
            :aria-expanded="openIndex === index"
            :aria-controls="`faq-panel-${index}`"
            @click="toggle(index)"
          >
            <span class="faq-question">{{ item.question }}</span>
            <svg
              class="faq-chevron"
              :class="{ rotated: openIndex === index }"
              width="20"
              height="20"
              viewBox="0 0 20 20"
              fill="none"
              aria-hidden="true"
            >
              <path
                d="M5 7.5L10 12.5L15 7.5"
                stroke="currentColor"
                stroke-width="1.5"
                stroke-linecap="round"
                stroke-linejoin="round"
              />
            </svg>
          </button>
          <div
            :id="`faq-panel-${index}`"
            class="faq-panel"
            :class="{ open: openIndex === index }"
            role="region"
            :aria-labelledby="`faq-trigger-${index}`"
          >
            <div class="faq-answer">
              {{ item.answer }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.faq {
  padding: 6rem 2rem;
  background: #070a12;
}

.faq-container {
  max-width: 760px;
  margin: 0 auto;
}

.faq-title {
  text-align: center;
  font-size: clamp(1.75rem, 3.5vw, 2.5rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 0.75rem;
  letter-spacing: -0.01em;
}

.faq-subtitle {
  text-align: center;
  font-size: clamp(0.95rem, 1.5vw, 1.1rem);
  color: #94a3b8;
  margin: 0 0 3rem;
  line-height: 1.6;
}

.faq-list {
  display: flex;
  flex-direction: column;
  gap: 0.75rem;
}

.faq-item {
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 12px;
  overflow: hidden;
  transition: border-color 0.3s ease, box-shadow 0.3s ease;
}

.faq-item.active {
  border-color: rgba(99, 102, 241, 0.35);
  box-shadow: 0 0 0 1px rgba(99, 102, 241, 0.1);
  border-left: 3px solid #6366f1;
}

.faq-trigger {
  display: flex;
  align-items: center;
  justify-content: space-between;
  width: 100%;
  padding: 1.25rem 1.5rem;
  background: none;
  border: none;
  cursor: pointer;
  text-align: left;
  color: #e2e8f0;
  font-size: 1rem;
  font-weight: 500;
  font-family: inherit;
  line-height: 1.5;
  gap: 1rem;
  transition: color 0.2s ease;
}

.faq-trigger:hover {
  color: #f8fafc;
}

.faq-trigger:focus-visible {
  outline: 2px solid #6366f1;
  outline-offset: -2px;
  border-radius: 10px;
}

.faq-question {
  flex: 1;
}

.faq-chevron {
  flex-shrink: 0;
  color: #64748b;
  transition: transform 0.3s ease, color 0.3s ease;
}

.faq-chevron.rotated {
  transform: rotate(180deg);
  color: #6366f1;
}

.faq-panel {
  display: grid;
  grid-template-rows: 0fr;
  transition: grid-template-rows 0.35s ease;
}

.faq-panel.open {
  grid-template-rows: 1fr;
}

.faq-answer {
  overflow: hidden;
  padding: 0 1.5rem;
  color: #94a3b8;
  font-size: 0.925rem;
  line-height: 1.7;
  transition: padding 0.35s ease;
}

.faq-panel.open .faq-answer {
  padding: 0 1.5rem 1.25rem;
}

/* Responsive: mobile */
@media (max-width: 540px) {
  .faq {
    padding: 4rem 1.25rem;
  }

  .faq-trigger {
    padding: 1rem 1.25rem;
    font-size: 0.95rem;
  }

  .faq-answer {
    padding: 0 1.25rem;
    font-size: 0.875rem;
  }

  .faq-panel.open .faq-answer {
    padding: 0 1.25rem 1rem;
  }
}

/* Respect reduced motion preference */
@media (prefers-reduced-motion: reduce) {
  .faq-panel {
    transition: none;
  }

  .faq-chevron {
    transition: none;
  }

  .faq-answer {
    transition: none;
  }
}
</style>
