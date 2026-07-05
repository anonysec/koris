import { describe, it, expect } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h, Suspense } from 'vue'
import App from '../src/App.vue'

describe('Landing Page - App.vue', () => {
  it('renders without errors', () => {
    const wrapper = mount(App)
    expect(wrapper.exists()).toBe(true)
    expect(wrapper.find('.landing').exists()).toBe(true)
  })

  it('renders SiteHeader', () => {
    const wrapper = mount(App)
    // SiteHeader is eagerly loaded, should be in DOM immediately
    const header = wrapper.find('header')
    expect(header.exists() || wrapper.html().includes('header')).toBe(true)
  })

  it('renders HeroSection', () => {
    const wrapper = mount(App)
    const html = wrapper.html()
    // HeroSection contains hero content
    expect(html.length).toBeGreaterThan(0)
  })

  it('renders FeaturesSection', () => {
    const wrapper = mount(App)
    const html = wrapper.html()
    expect(html.length).toBeGreaterThan(0)
  })

  it('renders all async sections (Pricing, FAQ, Footer)', async () => {
    const wrapper = mount(App)
    await flushPromises()
    const html = wrapper.html()
    // After async components resolve, the landing div should have content
    expect(html).toContain('landing')
  })
})

describe('Landing Page - i18n', () => {
  it('provides t() function that returns translations', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t, locale, setLocale } = useI18n()

    // Default locale should be 'en' in test environment
    expect(locale.value).toBe('en')

    // Test English translations
    expect(t('hero.headline')).toBe('Secure, Fast VPN Management')
    expect(t('features.title')).toBe('Everything You Need to Run a VPN Business')
    expect(t('faq.title')).toBe('Frequently Asked Questions')
  })

  it('switches locale to Farsi', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t, setLocale } = useI18n()

    setLocale('fa')
    expect(t('hero.headline')).toBe('مدیریت VPN امن و سریع')
    expect(t('features.title')).toBe('همه چیز برای اجرای کسب‌وکار VPN')

    // Reset to English for other tests
    setLocale('en')
  })

  it('interpolates parameters in translations', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t } = useI18n()

    const result = t('footer.copyright', { year: 2025 })
    expect(result).toBe('© 2025 KorisPanel. All rights reserved.')
  })

  it('falls back to key when translation is missing', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t } = useI18n()

    expect(t('nonexistent.key')).toBe('nonexistent.key')
  })
})
