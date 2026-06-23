<script setup lang="ts">
import { defineAsyncComponent } from 'vue'

// Above-the-fold: loaded eagerly in main chunk
import SiteHeader from './components/SiteHeader.vue'
import HeroSection from './components/HeroSection.vue'
import FeaturesSection from './components/FeaturesSection.vue'

// Below-the-fold: lazy-loaded (code-split into separate chunks)
const PricingSection = defineAsyncComponent(() => import('./components/PricingSection.vue'))
const CTASection = defineAsyncComponent(() => import('./components/CTASection.vue'))
const FaqSection = defineAsyncComponent(() => import('./components/FaqSection.vue'))
const SiteFooter = defineAsyncComponent(() => import('./components/SiteFooter.vue'))
</script>

<template>
  <div class="landing">
    <!-- Site Header -->
    <SiteHeader />

    <!-- Hero Section -->
    <HeroSection />

    <!-- Features Section -->
    <FeaturesSection />

    <!-- Pricing Section -->
    <PricingSection />

    <!-- CTA Section -->
    <CTASection />

    <!-- FAQ Section -->
    <FaqSection />

    <!-- Footer -->
    <SiteFooter />
  </div>
</template>

<style scoped>
.landing {
  min-height: 100vh;
  font-family: system-ui, -apple-system, sans-serif;
  color: #f8fafc;
  background: #070a12;
}

.landing-section {
  padding: 4rem 2rem;
  max-width: 1200px;
  margin: 0 auto;
}
</style>
