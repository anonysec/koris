<script setup lang="ts">
import { useI18n } from '../i18n'

const { t } = useI18n()
</script>

<template>
  <section class="hero">
    <!-- Soft animated aurora — a popular, brand-neutral backdrop (no VPN metaphor) -->
    <div class="hero-aurora" aria-hidden="true">
      <span class="blob blob--1"></span>
      <span class="blob blob--2"></span>
      <span class="blob blob--3"></span>
    </div>

    <!-- Content -->
    <div class="hero-content">
      <span class="hero-eyebrow">{{ t('hero.eyebrow') }}</span>
      <h1 class="hero-headline">
        {{ t('hero.headline') }}
      </h1>
      <p class="hero-subheadline">
        {{ t('hero.subheadline') }}
      </p>
      <div class="hero-actions">
        <a href="/portal/" class="hero-cta">{{ t('hero.cta') }}</a>
        <a href="/admin/" class="hero-cta hero-cta--ghost">{{ t('hero.cta_secondary') }}</a>
      </div>
    </div>
  </section>
</template>

<style scoped>
.hero {
  --accent: #6366f1;
  --accent-2: #22d3ee;
  --accent-3: #a855f7;

  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  overflow: hidden;
  padding: 6rem 2rem 3rem;
  background: #070a12;
}

/* Aurora backdrop — calm, popular aesthetic */
.hero-aurora {
  position: absolute;
  inset: 0;
  z-index: 0;
  overflow: hidden;
  filter: blur(60px);
  opacity: 0.55;
}
.blob {
  position: absolute;
  width: 45vmax;
  height: 45vmax;
  border-radius: 50%;
  mix-blend-mode: screen;
  animation: drift 18s ease-in-out infinite;
}
.blob--1 {
  background: radial-gradient(circle at 30% 30%, var(--accent), transparent 60%);
  top: -10%;
  left: -5%;
}
.blob--2 {
  background: radial-gradient(circle at 70% 40%, var(--accent-2), transparent 60%);
  bottom: -15%;
  right: -10%;
  animation-delay: -6s;
}
.blob--3 {
  background: radial-gradient(circle at 50% 60%, var(--accent-3), transparent 60%);
  top: 20%;
  right: 25%;
  animation-delay: -12s;
}
@keyframes drift {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(4%, -3%) scale(1.08); }
  66% { transform: translate(-3%, 4%) scale(0.95); }
}
@media (prefers-reduced-motion: reduce) {
  .blob { animation: none; }
}

/* Content */
.hero-content {
  position: relative;
  z-index: 2;
  text-align: center;
  max-width: 760px;
}
.hero-eyebrow {
  display: inline-block;
  margin-bottom: 1.25rem;
  padding: 0.4rem 1rem;
  font-size: 0.8rem;
  font-weight: 600;
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: #c7d2fe;
  background: rgba(99, 102, 241, 0.12);
  border: 1px solid rgba(99, 102, 241, 0.25);
  border-radius: 999px;
}
.hero-headline {
  font-size: clamp(2.2rem, 5vw, 4rem);
  font-weight: 800;
  line-height: 1.12;
  margin: 0 0 1.25rem;
  color: #f1f5f9;
  letter-spacing: -0.02em;
}
.hero-subheadline {
  font-size: clamp(1rem, 2vw, 1.25rem);
  line-height: 1.7;
  margin: 0 auto 2.5rem;
  color: #94a3b8;
  max-width: 580px;
}
.hero-actions {
  display: flex;
  gap: 1rem;
  justify-content: center;
  flex-wrap: wrap;
}
.hero-cta {
  display: inline-block;
  padding: 0.9rem 2.4rem;
  font-size: 1.05rem;
  font-weight: 600;
  color: #fff;
  background: var(--accent);
  border-radius: 10px;
  text-decoration: none;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
  box-shadow: 0 6px 24px rgba(99, 102, 241, 0.35);
}
.hero-cta:hover {
  transform: translateY(-2px);
  background: #4f46e5;
}
.hero-cta--ghost {
  color: #cbd5e1;
  background: transparent;
  border: 1px solid rgba(148, 163, 184, 0.3);
  box-shadow: none;
}
.hero-cta--ghost:hover {
  color: #fff;
  background: rgba(148, 163, 184, 0.1);
}

/* Responsive */
@media (max-width: 640px) {
  .hero {
    padding: 2rem 1.25rem;
    min-height: 88vh;
  }
  .hero-actions {
    flex-direction: column;
    align-items: stretch;
  }
}
</style>
