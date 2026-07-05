/**
 * Unit tests for HealthDot component
 *
 * Tests color determination (green/yellow/red), aria-label accessibility,
 * tooltip text, and pulse animation class.
 */
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import HealthDot from './HealthDot.vue'

describe('HealthDot', () => {
  describe('color levels', () => {
    it('renders green dot for score >= 0.8', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.95 } })
      expect(wrapper.classes()).toContain('health-dot--green')
    })

    it('renders green dot at exactly 0.8', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.8 } })
      expect(wrapper.classes()).toContain('health-dot--green')
    })

    it('renders yellow dot for score >= 0.4 and < 0.8', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.6 } })
      expect(wrapper.classes()).toContain('health-dot--yellow')
    })

    it('renders yellow dot at exactly 0.4', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.4 } })
      expect(wrapper.classes()).toContain('health-dot--yellow')
    })

    it('renders red dot for score < 0.4', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.2 } })
      expect(wrapper.classes()).toContain('health-dot--red')
    })

    it('renders red dot for score 0', () => {
      const wrapper = mount(HealthDot, { props: { score: 0 } })
      expect(wrapper.classes()).toContain('health-dot--red')
    })

    it('renders green dot for score 1.0', () => {
      const wrapper = mount(HealthDot, { props: { score: 1.0 } })
      expect(wrapper.classes()).toContain('health-dot--green')
    })
  })

  describe('accessibility', () => {
    it('has role="img"', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.9 } })
      expect(wrapper.attributes('role')).toBe('img')
    })

    it('has aria-label with "good" for green state', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.9 } })
      expect(wrapper.attributes('aria-label')).toBe('Health good: 0.90')
    })

    it('has aria-label with "degraded" for yellow state', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.5 } })
      expect(wrapper.attributes('aria-label')).toBe('Health degraded: 0.50')
    })

    it('has aria-label with "critical" for red state', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.1 } })
      expect(wrapper.attributes('aria-label')).toBe('Health critical: 0.10')
    })
  })

  describe('tooltip', () => {
    it('shows exact score in title attribute', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.95 } })
      expect(wrapper.attributes('title')).toBe('Health: 0.95')
    })

    it('formats score to 2 decimal places', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.333 } })
      expect(wrapper.attributes('title')).toBe('Health: 0.33')
    })
  })

  describe('element structure', () => {
    it('renders a span element', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.5 } })
      expect(wrapper.element.tagName).toBe('SPAN')
    })

    it('always has the base health-dot class', () => {
      const wrapper = mount(HealthDot, { props: { score: 0.5 } })
      expect(wrapper.classes()).toContain('health-dot')
    })
  })
})
