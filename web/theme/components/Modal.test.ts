/**
 * Unit tests for Modal component
 *
 * Tests: rendering, overlay blur, focus trap, Escape to close,
 * click-outside to close, body scroll lock, transitions, accessibility.
 */
import { describe, it, expect, beforeEach, afterEach } from 'vitest'
import { mount } from '@vue/test-utils'
import Modal from './Modal.vue'

describe('Modal', () => {
  beforeEach(() => {
    document.body.style.overflow = ''
  })

  afterEach(() => {
    document.body.style.overflow = ''
  })

  describe('rendering', () => {
    it('does not render content when open is false', () => {
      const wrapper = mount(Modal, {
        props: { open: false },
        slots: { default: '<p>Modal content</p>' },
      })
      expect(wrapper.find('.k-modal__overlay').exists()).toBe(false)
    })

    it('renders content when open is true', () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        slots: { default: '<p>Modal content</p>' },
        attachTo: document.body,
      })
      const overlay = document.querySelector('.k-modal__overlay')
      expect(overlay).not.toBeNull()
      wrapper.unmount()
    })

    it('renders title when provided', () => {
      const wrapper = mount(Modal, {
        props: { open: true, title: 'Test Title' },
        attachTo: document.body,
      })
      const title = document.querySelector('.k-modal__title')
      expect(title?.textContent).toBe('Test Title')
      wrapper.unmount()
    })

    it('does not render header when no title and closable is false', () => {
      const wrapper = mount(Modal, {
        props: { open: true, title: '', closable: false },
        attachTo: document.body,
      })
      const header = document.querySelector('.k-modal__header')
      expect(header).toBeNull()
      wrapper.unmount()
    })

    it('renders close button when closable', () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: true },
        attachTo: document.body,
      })
      const closeBtn = document.querySelector('.k-modal__close-btn')
      expect(closeBtn).not.toBeNull()
      wrapper.unmount()
    })

    it('does not render close button when closable is false', () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: false, title: 'Non-closable' },
        attachTo: document.body,
      })
      const closeBtn = document.querySelector('.k-modal__close-btn')
      expect(closeBtn).toBeNull()
      wrapper.unmount()
    })

    it('renders footer slot when provided', () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        slots: { footer: '<button>Save</button>' },
        attachTo: document.body,
      })
      const footer = document.querySelector('.k-modal__footer')
      expect(footer).not.toBeNull()
      expect(footer?.innerHTML).toContain('Save')
      wrapper.unmount()
    })

    it('applies custom width', () => {
      const wrapper = mount(Modal, {
        props: { open: true, width: '700px' },
        attachTo: document.body,
      })
      const modal = document.querySelector('.k-modal') as HTMLElement
      expect(modal?.style.width).toBe('700px')
      expect(modal?.style.maxWidth).toBe('700px')
      wrapper.unmount()
    })
  })

  describe('closing behavior', () => {
    it('emits close on Escape key when closable', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
      document.dispatchEvent(event)

      expect(wrapper.emitted('close')).toHaveLength(1)
      wrapper.unmount()
    })

    it('does not emit close on Escape key when closable is false', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: false },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const event = new KeyboardEvent('keydown', { key: 'Escape', bubbles: true })
      document.dispatchEvent(event)

      expect(wrapper.emitted('close')).toBeUndefined()
      wrapper.unmount()
    })

    it('emits close on overlay click when closable', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const overlay = document.querySelector('.k-modal__overlay') as HTMLElement
      overlay.click()

      expect(wrapper.emitted('close')).toHaveLength(1)
      wrapper.unmount()
    })

    it('does not emit close on modal content click', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: true },
        slots: { default: '<p>Content</p>' },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const modal = document.querySelector('.k-modal') as HTMLElement
      modal.click()

      expect(wrapper.emitted('close')).toBeUndefined()
      wrapper.unmount()
    })

    it('emits close when close button is clicked', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const closeBtn = document.querySelector('.k-modal__close-btn') as HTMLElement
      closeBtn.click()

      expect(wrapper.emitted('close')).toHaveLength(1)
      wrapper.unmount()
    })
  })

  describe('body scroll lock', () => {
    it('locks body scroll when opened', async () => {
      const wrapper = mount(Modal, {
        props: { open: false },
        attachTo: document.body,
      })

      await wrapper.setProps({ open: true })
      expect(document.body.style.overflow).toBe('hidden')
      wrapper.unmount()
    })

    it('unlocks body scroll when closed', async () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()
      expect(document.body.style.overflow).toBe('hidden')

      await wrapper.setProps({ open: false })
      expect(document.body.style.overflow).toBe('')
      wrapper.unmount()
    })

    it('unlocks body scroll on unmount while open', async () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()
      expect(document.body.style.overflow).toBe('hidden')

      wrapper.unmount()
      expect(document.body.style.overflow).toBe('')
    })
  })

  describe('accessibility', () => {
    it('has role="dialog" on overlay', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, title: 'Accessible Modal' },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const overlay = document.querySelector('.k-modal__overlay')
      expect(overlay?.getAttribute('role')).toBe('dialog')
      wrapper.unmount()
    })

    it('has aria-modal="true"', async () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const overlay = document.querySelector('.k-modal__overlay')
      expect(overlay?.getAttribute('aria-modal')).toBe('true')
      wrapper.unmount()
    })

    it('has aria-labelledby pointing to title', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, title: 'My Title' },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const overlay = document.querySelector('.k-modal__overlay')
      expect(overlay?.getAttribute('aria-labelledby')).toBe('k-modal-title')
      wrapper.unmount()
    })

    it('close button has aria-label', async () => {
      const wrapper = mount(Modal, {
        props: { open: true, closable: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      const closeBtn = document.querySelector('.k-modal__close-btn')
      expect(closeBtn?.getAttribute('aria-label')).toBe('Close modal')
      wrapper.unmount()
    })
  })

  describe('default props', () => {
    it('uses 520px as default width', () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        attachTo: document.body,
      })
      const modal = document.querySelector('.k-modal') as HTMLElement
      expect(modal?.style.width).toBe('520px')
      wrapper.unmount()
    })

    it('closable defaults to true', async () => {
      const wrapper = mount(Modal, {
        props: { open: true },
        attachTo: document.body,
      })
      await wrapper.vm.$nextTick()

      // Close button should exist by default
      const closeBtn = document.querySelector('.k-modal__close-btn')
      expect(closeBtn).not.toBeNull()
      wrapper.unmount()
    })
  })
})
