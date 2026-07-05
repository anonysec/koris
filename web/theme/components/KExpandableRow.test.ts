/**
 * Unit tests for KExpandableRow component
 *
 * Tests expand/collapse behavior, aria attributes, chevron rotation,
 * transition duration prop, and reduced motion CSS support.
 */
import { describe, it, expect } from 'vitest'
import { mount } from '@vue/test-utils'
import KExpandableRow from './KExpandableRow.vue'

describe('KExpandableRow', () => {
  const defaultSlots = {
    row: '<span class="row-content">Row Content</span>',
    expanded: '<span class="expanded-content">Expanded Content</span>',
  }

  describe('collapsed state', () => {
    it('renders row slot content', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      expect(wrapper.find('.row-content').exists()).toBe(true)
    })

    it('does not visually reveal expanded content (panel has 0fr grid)', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const panel = wrapper.find('.k-expandable-row__panel')
      expect(panel.classes()).not.toContain('k-expandable-row__panel--open')
    })

    it('sets aria-hidden="true" on the panel when collapsed', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const panel = wrapper.find('.k-expandable-row__panel')
      expect(panel.attributes('aria-hidden')).toBe('true')
    })

    it('sets aria-expanded="false" on the chevron button', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const btn = wrapper.find('.k-expandable-row__chevron-btn')
      expect(btn.attributes('aria-expanded')).toBe('false')
    })

    it('chevron is not rotated', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const chevron = wrapper.find('.k-expandable-row__chevron')
      expect(chevron.classes()).not.toContain('k-expandable-row__chevron--rotated')
    })
  })

  describe('expanded state', () => {
    it('renders expanded slot content', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: true },
        slots: defaultSlots,
      })
      expect(wrapper.find('.expanded-content').exists()).toBe(true)
    })

    it('adds open class to panel when expanded', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: true },
        slots: defaultSlots,
      })
      const panel = wrapper.find('.k-expandable-row__panel')
      expect(panel.classes()).toContain('k-expandable-row__panel--open')
    })

    it('sets aria-hidden="false" on the panel when expanded', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: true },
        slots: defaultSlots,
      })
      const panel = wrapper.find('.k-expandable-row__panel')
      expect(panel.attributes('aria-hidden')).toBe('false')
    })

    it('sets aria-expanded="true" on the chevron button', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: true },
        slots: defaultSlots,
      })
      const btn = wrapper.find('.k-expandable-row__chevron-btn')
      expect(btn.attributes('aria-expanded')).toBe('true')
    })

    it('rotates the chevron icon', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: true },
        slots: defaultSlots,
      })
      const chevron = wrapper.find('.k-expandable-row__chevron')
      expect(chevron.classes()).toContain('k-expandable-row__chevron--rotated')
    })
  })

  describe('toggle emit', () => {
    it('emits toggle when chevron button is clicked', async () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      await wrapper.find('.k-expandable-row__chevron-btn').trigger('click')
      expect(wrapper.emitted('toggle')).toHaveLength(1)
    })

    it('does not emit toggle on row content click', async () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      await wrapper.find('.k-expandable-row__content').trigger('click')
      expect(wrapper.emitted('toggle')).toBeUndefined()
    })
  })

  describe('transition duration prop', () => {
    it('applies default 200ms duration as CSS variable', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const root = wrapper.find('.k-expandable-row')
      expect(root.attributes('style')).toContain('--k-expandable-duration: 200ms')
    })

    it('applies custom transition duration', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false, transitionDuration: 400 },
        slots: defaultSlots,
      })
      const root = wrapper.find('.k-expandable-row')
      expect(root.attributes('style')).toContain('--k-expandable-duration: 400ms')
    })
  })

  describe('accessibility', () => {
    it('links chevron button to panel via aria-controls', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const btn = wrapper.find('.k-expandable-row__chevron-btn')
      const panel = wrapper.find('.k-expandable-row__panel')
      const panelId = panel.attributes('id')
      expect(btn.attributes('aria-controls')).toBe(panelId)
    })

    it('panel has role="region"', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: true },
        slots: defaultSlots,
      })
      const panel = wrapper.find('.k-expandable-row__panel')
      expect(panel.attributes('role')).toBe('region')
    })

    it('chevron button has aria-label', () => {
      const wrapper = mount(KExpandableRow, {
        props: { expanded: false },
        slots: defaultSlots,
      })
      const btn = wrapper.find('.k-expandable-row__chevron-btn')
      expect(btn.attributes('aria-label')).toBe('Toggle row details')
    })
  })
})
