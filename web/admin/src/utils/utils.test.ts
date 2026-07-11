import { describe, it, expect } from 'vitest'
import { formatBytes } from '@koris/core'
import { formatCurrency } from './formatCurrency'
import { computeExpiryDate } from './computeExpiryDate'
import { formatAdvancedSummary } from './formatAdvancedSummary'

describe('formatBytes', () => {
  it('formats 0 bytes', () => {
    expect(formatBytes(0)).toBe('0.0 B')
  })

  it('formats bytes under 1 KB', () => {
    expect(formatBytes(512)).toBe('512.0 B')
  })

  it('formats kilobytes', () => {
    expect(formatBytes(1024)).toBe('1.0 KB')
    expect(formatBytes(1536)).toBe('1.5 KB')
  })

  it('formats megabytes', () => {
    expect(formatBytes(1048576)).toBe('1.0 MB')
  })

  it('formats gigabytes', () => {
    expect(formatBytes(2.4 * 1024 * 1024 * 1024)).toBe('2.4 GB')
  })

  it('formats terabytes', () => {
    expect(formatBytes(1024 * 1024 * 1024 * 1024)).toBe('1.0 TB')
  })

  it('handles negative values as 0', () => {
    expect(formatBytes(-100)).toBe('0.0 B')
  })
})

describe('formatCurrency', () => {
  it('formats positive values with default symbol', () => {
    expect(formatCurrency(10.5)).toBe('$10.50')
  })

  it('formats zero', () => {
    expect(formatCurrency(0)).toBe('$0.00')
  })

  it('formats negative values', () => {
    expect(formatCurrency(-5)).toBe('$-5.00')
  })

  it('uses configurable symbol', () => {
    expect(formatCurrency(99.99, '€')).toBe('€99.99')
  })

  it('handles NaN and Infinity', () => {
    expect(formatCurrency(NaN)).toBe('$0.00')
    expect(formatCurrency(Infinity)).toBe('$0.00')
  })

  it('produces exactly 2 decimal places', () => {
    expect(formatCurrency(1)).toBe('$1.00')
    expect(formatCurrency(1.1)).toBe('$1.10')
    expect(formatCurrency(1.999)).toBe('$2.00')
  })
})

describe('computeExpiryDate', () => {
  it('adds 1 day', () => {
    const base = new Date(2024, 0, 15) // Jan 15, 2024
    const result = computeExpiryDate(base, '+1d')
    expect(result.getFullYear()).toBe(2024)
    expect(result.getMonth()).toBe(0)
    expect(result.getDate()).toBe(16)
  })

  it('adds 7 days', () => {
    const base = new Date(2024, 0, 28) // Jan 28, 2024
    const result = computeExpiryDate(base, '+7d')
    expect(result.getFullYear()).toBe(2024)
    expect(result.getMonth()).toBe(1) // Feb
    expect(result.getDate()).toBe(4)
  })

  it('adds 1 month', () => {
    const base = new Date(2024, 0, 15) // Jan 15, 2024
    const result = computeExpiryDate(base, '+1m')
    expect(result.getFullYear()).toBe(2024)
    expect(result.getMonth()).toBe(1) // Feb
    expect(result.getDate()).toBe(15)
  })

  it('clamps month overflow (Jan 31 + 1m → Feb 29 in leap year)', () => {
    const base = new Date(2024, 0, 31) // Jan 31, 2024 (leap year)
    const result = computeExpiryDate(base, '+1m')
    expect(result.getFullYear()).toBe(2024)
    expect(result.getMonth()).toBe(1) // Feb
    expect(result.getDate()).toBe(29) // Leap year
  })

  it('adds 1 year', () => {
    const base = new Date(2024, 5, 15) // Jun 15, 2024
    const result = computeExpiryDate(base, '+1y')
    expect(result.getFullYear()).toBe(2025)
    expect(result.getMonth()).toBe(5) // Jun
    expect(result.getDate()).toBe(15)
  })

  it('does not mutate the original date', () => {
    const base = new Date(2024, 0, 15)
    const originalTime = base.getTime()
    computeExpiryDate(base, '+1m')
    expect(base.getTime()).toBe(originalTime)
  })
})

describe('formatAdvancedSummary', () => {
  it('formats both values when > 0', () => {
    expect(formatAdvancedSummary(100, 5)).toBe('100 Mbps · 5')
  })

  it('shows unlimited for speed when 0', () => {
    expect(formatAdvancedSummary(0, 3)).toBe('unlimited · 3')
  })

  it('shows unlimited for connections when 0', () => {
    expect(formatAdvancedSummary(50, 0)).toBe('50 Mbps · unlimited')
  })

  it('shows unlimited for both when 0', () => {
    expect(formatAdvancedSummary(0, 0)).toBe('unlimited · unlimited')
  })
})
