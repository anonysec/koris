<script setup lang="ts">
import { useI18n } from '../i18n'

const { t } = useI18n()
</script>

<template>
  <section class="hero">
    <!-- Animated gradient background -->
    <div class="hero-bg"></div>

    <!-- Floating geometric shapes for visual interest -->
    <div class="hero-shapes" aria-hidden="true">
      <div class="shape shape--1"></div>
      <div class="shape shape--2"></div>
      <div class="shape shape--3"></div>
      <div class="shape shape--4"></div>
      <div class="shape shape--5"></div>
    </div>

    <!-- Content -->
    <div class="hero-content">
      <h1 class="hero-headline">
        {{ t('hero.headline') }}
      </h1>
      <p class="hero-subheadline">
        {{ t('hero.subheadline') }}
      </p>
      <a href="/portal/" class="hero-cta">
        {{ t('hero.cta') }}
      </a>
    </div>
  </section>
</template>

<style scoped>
.hero {
  --hero-min-height: 100vh;
  --gradient-1: #0a0e1a;
  --gradient-2: #111833;
  --gradient-3: #1a1040;
  --gradient-4: #0d1a2e;
  --accent: #6366f1;
  --accent-glow: rgba(99, 102, 241, 0.3);

  position: relative;
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: var(--hero-min-height);
  overflow: hidden;
  padding: 5rem 2rem 2rem;
}

/* Animated gradient background */
.hero-bg {
  position: absolute;
  inset: 0;
  background: linear-gradient(
    135deg,
    var(--gradient-1) 0%,
    var(--gradient-2) 25%,
    var(--gradient-3) 50%,
    var(--gradient-4) 75%,
    var(--gradient-1) 100%
  );
  background-size: 400% 400%;
  animation: gradientShift 15s ease infinite;
  z-index: 0;
}

@keyframes gradientShift {
  0% {
    background-position: 0% 50%;
  }
  50% {
    background-position: 100% 50%;
  }
  100% {
    background-position: 0% 50%;
  }
}

/* Floating geometric shapes */
.hero-shapes {
  position: absolute;
  inset: 0;
  z-index: 1;
  pointer-events: none;
}

.shape {
  position: absolute;
  border-radius: 50%;
  opacity: 0.08;
  background: var(--accent);
  filter: blur(1px);
}

.shape--1 {
  width: 300px;
  height: 300px;
  top: 10%;
  left: -5%;
  animation: float 20s ease-in-out infinite;
}

.shape--2 {
  width: 200px;
  height: 200px;
  top: 60%;
  right: -3%;
  animation: float 25s ease-in-out infinite reverse;
}

.shape--3 {
  width: 150px;
  height: 150px;
  bottom: 15%;
  left: 20%;
  border-radius: 30% 70% 70% 30% / 30% 30% 70% 70%;
  animation: float 18s ease-in-out infinite;
  animation-delay: -5s;
}

.shape--4 {
  width: 100px;
  height: 100px;
  top: 20%;
  right: 15%;
  border-radius: 63% 37% 54% 46% / 55% 48% 52% 45%;
  animation: float 22s ease-in-out infinite;
  animation-delay: -8s;
}

.shape--5 {
  width: 80px;
  height: 80px;
  top: 50%;
  left: 10%;
  border-radius: 40% 60% 65% 35% / 40% 45% 55% 60%;
  animation: float 16s ease-in-out infinite reverse;
  animation-delay: -3s;
}

@keyframes float {
  0%, 100% {
    transform: translateY(0) rotate(0deg);
  }
  33% {
    transform: translateY(-20px) rotate(5deg);
  }
  66% {
    transform: translateY(10px) rotate(-3deg);
  }
}

/* Content */
.hero-content {
  position: relative;
  z-index: 2;
  text-align: center;
  max-width: 720px;
}

.hero-headline {
  font-size: clamp(2.2rem, 5vw, 4rem);
  font-weight: 800;
  line-height: 1.15;
  margin: 0 0 1.25rem;
  color: #f1f5f9;
  letter-spacing: -0.02em;
}

.hero-subheadline {
  font-size: clamp(1rem, 2vw, 1.25rem);
  line-height: 1.7;
  margin: 0 0 2.5rem;
  color: #94a3b8;
  max-width: 560px;
  margin-left: auto;
  margin-right: auto;
}

.hero-cta {
  display: inline-block;
  padding: 0.9rem 2.4rem;
  font-size: 1.05rem;
  font-weight: 600;
  color: #fff;
  background: var(--accent);
  border-radius: 8px;
  text-decoration: none;
  transition: transform 0.2s ease, box-shadow 0.2s ease, background 0.2s ease;
  box-shadow: 0 4px 20px var(--accent-glow);
}

.hero-cta:hover {
  transform: translateY(-2px);
  box-shadow: 0 8px 30px var(--accent-glow);
  background: #4f46e5;
}

.hero-cta:active {
  transform: translateY(0);
}

/* Responsive */
@media (max-width: 640px) {
  .hero {
    padding: 1.5rem;
    min-height: 85vh;
  }

  .hero-subheadline {
    margin-bottom: 2rem;
  }

  .shape--1 {
    width: 180px;
    height: 180px;
  }

  .shape--2 {
    width: 120px;
    height: 120px;
  }

  .shape--3,
  .shape--4,
  .shape--5 {
    display: none;
  }
}
</style>
