import { describe, it, expect, vi, beforeEach } from 'vitest'
import { mount, flushPromises } from '@vue/test-utils'
import { defineComponent, nextTick, type Ref, type ComputedRef } from 'vue'
import { createRouter, createMemoryHistory } from 'vue-router'
import { useDetailPanel } from './useDetailPanel'

/**
 * Unit tests for useDetailPanel composable.
 *
 * **Validates: Requirements 2.1, 2.8, 2.9**
 */

function createTestRouter() {
  return createRouter({
    history: createMemoryHistory(),
    routes: [
      { path: '/users', name: 'users', component: { template: '<div />' } },
    ],
  })
}

function mountWithRouter(router: ReturnType<typeof createTestRouter>) {
  let result: {
    selectedUserId: Ref<number | null>
    isOpen: ComputedRef<boolean>
    open: (userId: number) => void
    close: () => void
    switchUser: (userId: number) => void
  }

  const wrapper = mount(defineComponent({
    setup() {
      result = useDetailPanel()
      return { ...result }
    },
    template: '<div />',
  }), {
    global: {
      plugins: [router],
    },
  })

  return { wrapper, get result() { return result! } }
}

describe('useDetailPanel', () => {
  it('starts with panel closed when no query param', async () => {
    const router = createTestRouter()
    await router.push('/users')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    expect(result.selectedUserId.value).toBe(null)
    expect(result.isOpen.value).toBe(false)

    wrapper.unmount()
  })

  it('restores selected user from URL query param on mount', async () => {
    const router = createTestRouter()
    await router.push('/users?selected=42')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    expect(result.selectedUserId.value).toBe(42)
    expect(result.isOpen.value).toBe(true)

    wrapper.unmount()
  })

  it('open() sets selectedUserId and updates URL query param', async () => {
    const router = createTestRouter()
    await router.push('/users')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    result.open(7)
    await flushPromises()

    expect(result.selectedUserId.value).toBe(7)
    expect(result.isOpen.value).toBe(true)
    expect(router.currentRoute.value.query.selected).toBe('7')

    wrapper.unmount()
  })

  it('close() clears selectedUserId and removes URL query param', async () => {
    const router = createTestRouter()
    await router.push('/users?selected=5')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    expect(result.isOpen.value).toBe(true)

    result.close()
    await flushPromises()

    expect(result.selectedUserId.value).toBe(null)
    expect(result.isOpen.value).toBe(false)
    expect(router.currentRoute.value.query.selected).toBeUndefined()

    wrapper.unmount()
  })

  it('switchUser() changes selectedUserId without closing', async () => {
    const router = createTestRouter()
    await router.push('/users?selected=10')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    expect(result.selectedUserId.value).toBe(10)

    result.switchUser(20)
    await flushPromises()

    expect(result.selectedUserId.value).toBe(20)
    expect(result.isOpen.value).toBe(true)
    expect(router.currentRoute.value.query.selected).toBe('20')

    wrapper.unmount()
  })

  it('preserves other query params when opening/closing', async () => {
    const router = createTestRouter()
    await router.push('/users?filter=active&page=2')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    result.open(99)
    await flushPromises()

    expect(router.currentRoute.value.query.filter).toBe('active')
    expect(router.currentRoute.value.query.page).toBe('2')
    expect(router.currentRoute.value.query.selected).toBe('99')

    result.close()
    await flushPromises()

    expect(router.currentRoute.value.query.filter).toBe('active')
    expect(router.currentRoute.value.query.page).toBe('2')
    expect(router.currentRoute.value.query.selected).toBeUndefined()

    wrapper.unmount()
  })

  it('ignores invalid query param values on mount', async () => {
    const router = createTestRouter()
    await router.push('/users?selected=abc')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    expect(result.selectedUserId.value).toBe(null)
    expect(result.isOpen.value).toBe(false)

    wrapper.unmount()
  })

  it('ignores negative or zero query param values on mount', async () => {
    const router = createTestRouter()
    await router.push('/users?selected=-5')
    await router.isReady()

    const { result, wrapper } = mountWithRouter(router)

    expect(result.selectedUserId.value).toBe(null)
    expect(result.isOpen.value).toBe(false)

    wrapper.unmount()
  })
})
