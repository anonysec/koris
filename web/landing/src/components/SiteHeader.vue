<script setup lang="ts">
import { ref, computed, onMounted, onUnmounted } from 'vue'
import { useI18n } from '../i18n'

const { t, locale, setLocale } = useI18n()

const isScrolled = ref(false)
const mobileMenuOpen = ref(false)
const langOpen = ref(false)

const currentLang = computed(() => locale.value.toUpperCase())

const languages = [
  { code: 'EN', label: 'English' },
  { code: 'FA', label: 'فارسی' },
  { code: 'ZH', label: '中文' },
  { code: 'RU', label: 'Русский' },
]

function handleScroll() {
  isScrolled.value = window.scrollY > 50
}

function toggleMobileMenu() {
  mobileMenuOpen.value = !mobileMenuOpen.value
}

function closeMobileMenu() {
  mobileMenuOpen.value = false
}

function selectLang(code: string) {
  setLocale(code)
  langOpen.value = false
}

function scrollToSection(e: Event, id: string) {
  e.preventDefault()
  closeMobileMenu()
  const el = document.querySelector(id)
  if (el) {
    el.scrollIntoView({ behavior: 'smooth' })
  }
}

onMounted(() => {
  window.addEventListener('scroll', handleScroll, { passive: true })
})

onUnmounted(() => {
  window.removeEventListener('scroll', handleScroll)
})
</script>

<template>
  <header class="site-header" :class="{ scrolled: isScrolled }">
    <div class="header-inner">
      <!-- Logo -->
      <a href="/" class="header-logo">Koris</a>

      <!-- Desktop Nav -->
      <nav class="header-nav desktop-nav" aria-label="Main navigation">
        <a href="#features" class="nav-link" @click="scrollToSection($event, '#features')">{{ t('header.features') }}</a>
        <a href="#pricing" class="nav-link" @click="scrollToSection($event, '#pricing')">{{ t('header.pricing') }}</a>
        <a href="#faq" class="nav-link" @click="scrollToSection($event, '#faq')">{{ t('header.faq') }}</a>
      </nav>

      <!-- Right side actions -->
      <div class="header-actions">
        <!-- Language selector -->
        <div class="lang-selector" @mouseenter="langOpen = true" @mouseleave="langOpen = false">
          <button class="lang-btn" @click="langOpen = !langOpen" aria-label="Select language">
            <span class="lang-icon">🌐</span>
            <span class="lang-code">{{ currentLang }}</span>
          </button>
          <div class="lang-dropdown" v-show="langOpen">
            <button
              v-for="lang in languages"
              :key="lang.code"
              class="lang-option"
              :class="{ active: currentLang === lang.code }"
              @click="selectLang(lang.code)"
            >
              {{ lang.label }}
            </button>
          </div>
        </div>

        <!-- Auth links (desktop) -->
        <a href="/dashboard/" class="header-link desktop-only">{{ t('header.dashboard') }}</a>
        <a href="/portal/" class="header-link header-link--primary desktop-only">{{ t('header.login') }}</a>

        <!-- Mobile hamburger -->
        <button
          class="hamburger"
          :class="{ open: mobileMenuOpen }"
          @click="toggleMobileMenu"
          aria-label="Toggle menu"
          :aria-expanded="mobileMenuOpen"
        >
          <span class="hamburger-line"></span>
          <span class="hamburger-line"></span>
          <span class="hamburger-line"></span>
        </button>
      </div>
    </div>

    <!-- Mobile menu overlay -->
    <Transition name="slide">
      <div class="mobile-menu" v-show="mobileMenuOpen">
        <nav class="mobile-nav" aria-label="Mobile navigation">
          <a href="#features" class="mobile-link" @click="scrollToSection($event, '#features')">{{ t('header.features') }}</a>
          <a href="#pricing" class="mobile-link" @click="scrollToSection($event, '#pricing')">{{ t('header.pricing') }}</a>
          <a href="#faq" class="mobile-link" @click="scrollToSection($event, '#faq')">{{ t('header.faq') }}</a>
          <hr class="mobile-divider" />
          <a href="/dashboard/" class="mobile-link">{{ t('header.dashboard') }}</a>
          <a href="/portal/" class="mobile-link mobile-link--primary">{{ t('header.login') }}</a>
        </nav>
      </div>
    </Transition>
  </header>
</template>

<style scoped>
.site-header {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  z-index: 1000;
  padding: 0 1.5rem;
  transition: background 0.3s ease, backdrop-filter 0.3s ease, box-shadow 0.3s ease;
  background: transparent;
}

.site-header.scrolled {
  background: rgba(7, 10, 18, 0.75);
  backdrop-filter: blur(12px);
  -webkit-backdrop-filter: blur(12px);
  box-shadow: 0 1px 0 rgba(255, 255, 255, 0.05);
}

.header-inner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  max-width: 1200px;
  margin: 0 auto;
  height: 64px;
}

/* Logo */
.header-logo {
  font-size: 1.5rem;
  font-weight: 800;
  color: #f1f5f9;
  text-decoration: none;
  letter-spacing: -0.02em;
}

.header-logo:hover {
  color: #6366f1;
}

/* Desktop nav */
.desktop-nav {
  display: flex;
  gap: 2rem;
}

.nav-link {
  color: #94a3b8;
  text-decoration: none;
  font-size: 0.9rem;
  font-weight: 500;
  transition: color 0.2s ease;
}

.nav-link:hover {
  color: #f1f5f9;
}

/* Header actions */
.header-actions {
  display: flex;
  align-items: center;
  gap: 1rem;
}

/* Language selector */
.lang-selector {
  position: relative;
}

.lang-btn {
  display: flex;
  align-items: center;
  gap: 0.35rem;
  background: none;
  border: 1px solid rgba(148, 163, 184, 0.2);
  border-radius: 6px;
  padding: 0.35rem 0.6rem;
  color: #94a3b8;
  font-size: 0.8rem;
  cursor: pointer;
  transition: border-color 0.2s ease, color 0.2s ease;
}

.lang-btn:hover {
  border-color: rgba(148, 163, 184, 0.4);
  color: #f1f5f9;
}

.lang-icon {
  font-size: 0.9rem;
}

.lang-dropdown {
  position: absolute;
  top: calc(100% + 6px);
  right: 0;
  background: rgba(15, 20, 35, 0.95);
  backdrop-filter: blur(12px);
  border: 1px solid rgba(148, 163, 184, 0.15);
  border-radius: 8px;
  padding: 0.35rem;
  min-width: 120px;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}

.lang-option {
  display: block;
  width: 100%;
  text-align: left;
  background: none;
  border: none;
  color: #94a3b8;
  padding: 0.5rem 0.75rem;
  border-radius: 5px;
  font-size: 0.85rem;
  cursor: pointer;
  transition: background 0.15s ease, color 0.15s ease;
}

.lang-option:hover {
  background: rgba(99, 102, 241, 0.1);
  color: #f1f5f9;
}

.lang-option.active {
  color: #6366f1;
  font-weight: 600;
}

/* Auth links */
.header-link {
  color: #94a3b8;
  text-decoration: none;
  font-size: 0.9rem;
  font-weight: 500;
  transition: color 0.2s ease;
}

.header-link:hover {
  color: #f1f5f9;
}

.header-link--primary {
  background: #6366f1;
  color: #fff;
  padding: 0.45rem 1.1rem;
  border-radius: 6px;
  font-weight: 600;
  transition: background 0.2s ease, transform 0.15s ease;
}

.header-link--primary:hover {
  background: #4f46e5;
  color: #fff;
  transform: translateY(-1px);
}

/* Hamburger */
.hamburger {
  display: none;
  flex-direction: column;
  justify-content: center;
  gap: 5px;
  width: 32px;
  height: 32px;
  background: none;
  border: none;
  cursor: pointer;
  padding: 4px;
}

.hamburger-line {
  display: block;
  width: 100%;
  height: 2px;
  background: #f1f5f9;
  border-radius: 2px;
  transition: transform 0.3s ease, opacity 0.3s ease;
}

.hamburger.open .hamburger-line:nth-child(1) {
  transform: translateY(7px) rotate(45deg);
}

.hamburger.open .hamburger-line:nth-child(2) {
  opacity: 0;
}

.hamburger.open .hamburger-line:nth-child(3) {
  transform: translateY(-7px) rotate(-45deg);
}

/* Desktop only */
.desktop-only {
  display: inline-flex;
}

/* Mobile menu */
.mobile-menu {
  display: none;
  position: absolute;
  top: 64px;
  left: 0;
  right: 0;
  background: rgba(7, 10, 18, 0.95);
  backdrop-filter: blur(16px);
  -webkit-backdrop-filter: blur(16px);
  border-bottom: 1px solid rgba(148, 163, 184, 0.1);
  padding: 1rem 1.5rem 1.5rem;
}

.mobile-nav {
  display: flex;
  flex-direction: column;
  gap: 0.25rem;
}

.mobile-link {
  display: block;
  color: #cbd5e1;
  text-decoration: none;
  font-size: 1rem;
  font-weight: 500;
  padding: 0.75rem 0.5rem;
  border-radius: 6px;
  transition: background 0.15s ease, color 0.15s ease;
}

.mobile-link:hover {
  background: rgba(99, 102, 241, 0.08);
  color: #f1f5f9;
}

.mobile-link--primary {
  color: #6366f1;
  font-weight: 600;
}

.mobile-divider {
  border: none;
  border-top: 1px solid rgba(148, 163, 184, 0.1);
  margin: 0.5rem 0;
}

/* Transition */
.slide-enter-active,
.slide-leave-active {
  transition: opacity 0.25s ease, transform 0.25s ease;
}

.slide-enter-from,
.slide-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}

/* Responsive */
@media (max-width: 768px) {
  .desktop-nav {
    display: none;
  }

  .desktop-only {
    display: none;
  }

  .hamburger {
    display: flex;
  }

  .mobile-menu {
    display: block;
  }
}
</style>
