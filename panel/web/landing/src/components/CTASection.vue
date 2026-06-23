<script setup lang="ts">
import { useI18n } from '../i18n'

const { t } = useI18n()
</script>

<template>
  <section class="cta-section">
    <div class="cta-container">
      <div class="cta-card">
        <h2 class="cta-title">{{ t('cta.title') }}</h2>
        <p class="cta-description">{{ t('cta.description') }}</p>
        <div class="cta-actions">
          <a href="/portal/" class="cta-btn cta-btn--primary">
            {{ t('cta.register') }}
          </a>
          <a href="/portal/" class="cta-btn cta-btn--secondary">
            {{ t('cta.login') }}
          </a>
        </div>
      </div>
    </div>
  </section>
</template>

<style scoped>
.cta-section {
  padding: 6rem 2rem;
  background: #070a12;
}

.cta-container {
  max-width: 800px;
  margin: 0 auto;
}

.cta-card {
  position: relative;
  text-align: center;
  background: linear-gradient(135deg, rgba(99, 102, 241, 0.08), rgba(129, 140, 248, 0.04));
  border: 1px solid rgba(99, 102, 241, 0.2);
  border-radius: 20px;
  padding: 4rem 2.5rem;
  overflow: hidden;
}

.cta-card::before {
  content: '';
  position: absolute;
  inset: 0;
  background: radial-gradient(
    ellipse 80% 60% at 50% 0%,
    rgba(99, 102, 241, 0.12),
    transparent
  );
  pointer-events: none;
}

.cta-title {
  position: relative;
  font-size: clamp(1.5rem, 3vw, 2.25rem);
  font-weight: 700;
  color: #f1f5f9;
  margin: 0 0 1rem;
  letter-spacing: -0.01em;
}

.cta-description {
  position: relative;
  font-size: clamp(0.95rem, 1.5vw, 1.1rem);
  color: #94a3b8;
  margin: 0 0 2.5rem;
  max-width: 520px;
  margin-left: auto;
  margin-right: auto;
  line-height: 1.7;
}

.cta-actions {
  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 1rem;
  flex-wrap: wrap;
}

.cta-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 0.85rem 2rem;
  font-size: 1rem;
  font-weight: 600;
  border-radius: 8px;
  text-decoration: none;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease,
    border-color 0.2s ease;
}

.cta-btn--primary {
  background: #6366f1;
  color: #fff;
  box-shadow: 0 4px 20px rgba(99, 102, 241, 0.3);
}

.cta-btn--primary:hover {
  background: #4f46e5;
  transform: translateY(-2px);
  box-shadow: 0 8px 30px rgba(99, 102, 241, 0.35);
}

.cta-btn--secondary {
  background: transparent;
  color: #e2e8f0;
  border: 1px solid rgba(255, 255, 255, 0.15);
}

.cta-btn--secondary:hover {
  border-color: rgba(99, 102, 241, 0.4);
  background: rgba(99, 102, 241, 0.08);
  transform: translateY(-1px);
}

.cta-btn:active {
  transform: translateY(0);
}

/* Responsive */
@media (max-width: 540px) {
  .cta-section {
    padding: 4rem 1.25rem;
  }

  .cta-card {
    padding: 3rem 1.5rem;
  }

  .cta-actions {
    flex-direction: column;
    gap: 0.75rem;
  }

  .cta-btn {
    width: 100%;
  }
}
</style>
