import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import KExpiryChips from './KExpiryChips.vue'

describe('KExpiryChips', () => {
  let now: Date

  beforeEach(() => {
    now = new Date('2025-06-15T12:00:00.000Z')
    vi.useFakeTimers()
    vi.setSystemTime(now)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('renders default chips', () => {
    const wrapper = mount(KExpiryChips)
    const chips = wrapper.findAll('.k-expiry-chips__chip')
    expect(chips).toHaveLength(7)
    expect(chips[0].text()).toBe('+1d')
    expect(chips[1].text()).toBe('+7d')
    expect(chips[2].text()).toBe('+1m')
    expect(chips[3].text()).toBe('+2m')
    expect(chips[4].text()).toBe('+3m')
    expect(chips[5].text()).toBe('+6m')
    expect(chips[6].text()).toBe('+1y')
  })

  it('renders custom chips', () => {
    const wrapper = mount(KExpiryChips, {
      props: { chips: ['+1d', '+1m'] },
    })
    const chips = wrapper.findAll('.k-expiry-chips__chip')
    expect(chips).toHaveLength(2)
    expect(chips[0].text()).toBe('+1d')
    expect(chips[1].text()).toBe('+1m')
  })

  it('emits update:modelValue with ISO date on chip click (+1d)', async () => {
    const wrapper = mount(KExpiryChips)
    await wrapper.findAll('.k-expiry-chips__chip')[0].trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toHaveLength(1)

    const emittedDate = new Date(emitted![0][0] as string)
    const expected = new Date('2025-06-16T12:00:00.000Z')
    expect(emittedDate.toISOString()).toBe(expected.toISOString())
  })

  it('emits update:modelValue with ISO date on chip click (+7d)', async () => {
    const wrapper = mount(KExpiryChips)
    await wrapper.findAll('.k-expiry-chips__chip')[1].trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toHaveLength(1)

    const emittedDate = new Date(emitted![0][0] as string)
    const expected = new Date('2025-06-22T12:00:00.000Z')
    expect(emittedDate.toISOString()).toBe(expected.toISOString())
  })

  it('emits update:modelValue with ISO date on chip click (+1m)', async () => {
    const wrapper = mount(KExpiryChips)
    await wrapper.findAll('.k-expiry-chips__chip')[2].trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toHaveLength(1)

    const emittedDate = new Date(emitted![0][0] as string)
    const expected = new Date('2025-07-15T12:00:00.000Z')
    expect(emittedDate.toISOString()).toBe(expected.toISOString())
  })

  it('emits update:modelValue with ISO date on chip click (+1y)', async () => {
    const wrapper = mount(KExpiryChips)
    await wrapper.findAll('.k-expiry-chips__chip')[6].trigger('click')

    const emitted = wrapper.emitted('update:modelValue')
    expect(emitted).toHaveLength(1)

    const emittedDate = new Date(emitted![0][0] as string)
    const expected = new Date('2026-06-15T12:00:00.000Z')
    expect(emittedDate.toISOString()).toBe(expected.toISOString())
  })

  it('marks active chip when modelValue matches computed date', async () => {
    const target = new Date('2025-06-16T12:00:00.000Z')
    const wrapper = mount(KExpiryChips, {
      props: { modelValue: target.toISOString() },
    })

    const firstChip = wrapper.findAll('.k-expiry-chips__chip')[0]
    expect(firstChip.classes()).toContain('k-expiry-chips__chip--active')
  })

  it('has correct aria attributes', () => {
    const wrapper = mount(KExpiryChips)
    const group = wrapper.find('.k-expiry-chips')
    expect(group.attributes('role')).toBe('group')
    expect(group.attributes('aria-label')).toBe('Expiry shortcut chips')

    const chips = wrapper.findAll('.k-expiry-chips__chip')
    chips.forEach((chip) => {
      expect(chip.attributes('aria-pressed')).toBeDefined()
    })
  })

  it('handles month-end clamping (+1m from Jan 31)', async () => {
    vi.setSystemTime(new Date('2025-01-31T12:00:00.000Z'))
    const wrapper = mount(KExpiryChips)
    await wrapper.findAll('.k-expiry-chips__chip')[2].trigger('click') // +1m

    const emitted = wrapper.emitted('update:modelValue')
    const emittedDate = new Date(emitted![0][0] as string)
    // Jan 31 + 1 month = Feb 28 (2025 is not a leap year)
    expect(emittedDate.getMonth()).toBe(1) // February
    expect(emittedDate.getDate()).toBe(28)
  })
})
