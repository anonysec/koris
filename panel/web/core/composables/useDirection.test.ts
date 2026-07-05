import { describe, it, expect, beforeEach } from 'vitest'
import { useDirection } from './useDirection'
import { useI18n } from './useI18n'
import { nextTick } from 'vue'

describe('useDirection', () => {
  beforeEach(() => {
    // Reset HTML element attributes before each test
    document.documentElement.dir = ''
    document.documentElement.lang = ''
    // Reset locale to English
    const { setLocale } = useI18n()
    setLocale('en')
  })

  it('returns direction "ltr" for English locale', () => {
    const { direction, isRTL } = useDirection()
    expect(direction.value).toBe('ltr')
    expect(isRTL.value).toBe(false)
  })

  it('returns direction "rtl" for Farsi locale', () => {
    const { setLocale } = useI18n()
    setLocale('fa')
    const { direction, isRTL } = useDirection()
    expect(direction.value).toBe('rtl')
    expect(isRTL.value).toBe(true)
  })

  it('sets dir attribute on html element', async () => {
    useDirection()
    await nextTick()
    expect(document.documentElement.dir).toBe('ltr')
  })

  it('sets lang attribute on html element', async () => {
    useDirection()
    await nextTick()
    expect(document.documentElement.lang).toBe('en')
  })

  it('updates DOM when locale changes to Farsi', async () => {
    const { setLocale } = useI18n()
    useDirection()
    await nextTick()

    setLocale('fa')
    await nextTick()

    expect(document.documentElement.dir).toBe('rtl')
    expect(document.documentElement.lang).toBe('fa')
  })

  it('updates DOM when locale changes back from Farsi to English', async () => {
    const { setLocale } = useI18n()
    setLocale('fa')
    useDirection()
    await nextTick()
    expect(document.documentElement.dir).toBe('rtl')

    setLocale('en')
    await nextTick()
    expect(document.documentElement.dir).toBe('ltr')
    expect(document.documentElement.lang).toBe('en')
  })

  it('returns LTR for Chinese locale', () => {
    const { setLocale } = useI18n()
    setLocale('zh')
    const { direction, isRTL } = useDirection()
    expect(direction.value).toBe('ltr')
    expect(isRTL.value).toBe(false)
  })

  it('returns LTR for Russian locale', () => {
    const { setLocale } = useI18n()
    setLocale('ru')
    const { direction, isRTL } = useDirection()
    expect(direction.value).toBe('ltr')
    expect(isRTL.value).toBe(false)
  })
})
