import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import KUsageBar from './KUsageBar.vue'

describe('KUsageBar', () => {
  it('renders progress bar with correct fill percentage', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 500, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.attributes('style')).toContain('width: 50%')
  })

  it('caps fill at 100% when usage exceeds limit', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 1500, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.attributes('style')).toContain('width: 100%')
  })

  it('applies normal color class when usage ≤ 80%', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 400, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.classes()).toContain('k-usage-bar__fill--normal')
  })

  it('applies warning color class when usage > 80% and ≤ 100%', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 850, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.classes()).toContain('k-usage-bar__fill--warning')
  })

  it('applies error color class when usage > 100%', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 1100, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.classes()).toContain('k-usage-bar__fill--error')
  })

  it('shows "Unlimited" label when limit is 0', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 1024 * 1024 * 500, limit: 0 },
    })
    expect(wrapper.text()).toContain('Unlimited')
  })

  it('does not render track when limit is 0', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 1024 * 1024 * 500, limit: 0 },
    })
    expect(wrapper.find('.k-usage-bar__track').exists()).toBe(false)
  })

  it('hides label when showLabel is false', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 500, limit: 1000, showLabel: false },
    })
    expect(wrapper.find('.k-usage-bar__label').exists()).toBe(false)
  })

  it('renders with sm size class', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 500, limit: 1000, size: 'sm' },
    })
    expect(wrapper.find('.k-usage-bar--sm').exists()).toBe(true)
  })

  it('formats bytes in label correctly', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 2.4 * 1024 * 1024 * 1024, limit: 10 * 1024 * 1024 * 1024 },
    })
    const label = wrapper.find('.k-usage-bar__label')
    expect(label.text()).toContain('GB')
    expect(label.text()).toContain('/')
  })

  it('has correct aria attributes', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 500, limit: 1000 },
    })
    const root = wrapper.find('.k-usage-bar')
    expect(root.attributes('role')).toBe('progressbar')
    expect(root.attributes('aria-valuenow')).toBe('50')
    expect(root.attributes('aria-valuemin')).toBe('0')
    expect(root.attributes('aria-valuemax')).toBe('100')
  })

  it('boundary: exactly 80% usage is normal (not warning)', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 800, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.classes()).toContain('k-usage-bar__fill--normal')
  })

  it('boundary: exactly 100% usage is warning (not error)', () => {
    const wrapper = mount(KUsageBar, {
      props: { used: 1000, limit: 1000 },
    })
    const fill = wrapper.find('.k-usage-bar__fill')
    expect(fill.classes()).toContain('k-usage-bar__fill--warning')
  })
})
