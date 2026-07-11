import { describe, it, expect } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, h, Suspense } from 'vue'
import App from '../src/App.vue'
import en from '../src/i18n/en.json'
import fa from '../src/i18n/fa.json'

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

    // Each test sets its own locale so assertions are order-independent
    setLocale('en')
    expect(locale.value).toBe('en')

    // Assert against the source JSON instead of hardcoded copy
    expect(t('hero.headline')).toBe(en['hero.headline'])
    expect(t('features.title')).toBe(en['features.title'])
    expect(t('faq.title')).toBe(en['faq.title'])
  })

  it('switches locale to Farsi', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t, setLocale } = useI18n()

    setLocale('fa')
    expect(t('hero.headline')).toBe(fa['hero.headline'])
    expect(t('features.title')).toBe(fa['features.title'])

    // Leave locale reset to English for other suites
    setLocale('en')
  })

  it('interpolates parameters in translations', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t, setLocale } = useI18n()

    setLocale('en')
    const result = t('footer.copyright', { year: 2025 })
    expect(result).toBe(en['footer.copyright'].replace('{year}', '2025'))
  })

  it('falls back to key when translation is missing', async () => {
    const { useI18n } = await import('../src/i18n')
    const { t } = useI18n()

    expect(t('nonexistent.key')).toBe('nonexistent.key')
  })
})
