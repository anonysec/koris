/**
 * Unit tests for useFormatDate utility functions
 *
 * Tests formatDate and formatDateTime covering null/undefined/empty inputs,
 * valid ISO strings, custom fallback values, and the formatDateTime variant.
 */
import { describe, it, expect } from 'vitest'
import { formatDate, formatDateTime } from './useFormatDate'

describe('formatDate', () => {
  it('returns fallback for null input', () => {
    expect(formatDate(null)).toBe('--')
  })

  it('returns fallback for undefined input', () => {
    expect(formatDate(undefined)).toBe('--')
  })

  it('returns fallback for empty string input', () => {
    expect(formatDate('')).toBe('--')
  })

  it('returns custom fallback when provided', () => {
    expect(formatDate(null, 'N/A')).toBe('N/A')
  })

  it('formats a valid ISO date string', () => {
    const result = formatDate('2025-01-15T10:30:00.000Z')
    // Intl.DateTimeFormat with en locale, year: numeric, month: short, day: 2-digit
    expect(result).toContain('Jan')
    expect(result).toContain('15')
    expect(result).toContain('2025')
  })

  it('formats a date-only ISO string', () => {
    const result = formatDate('2024-06-01')
    expect(result).toContain('Jun')
    expect(result).toContain('01')
    expect(result).toContain('2024')
  })

  it('formats dates from different months correctly', () => {
    expect(formatDate('2024-12-25')).toContain('Dec')
    expect(formatDate('2024-03-01')).toContain('Mar')
    expect(formatDate('2024-09-15')).toContain('Sep')
  })
})

describe('formatDateTime', () => {
  it('returns fallback for null input', () => {
    expect(formatDateTime(null)).toBe('--')
  })

  it('returns fallback for undefined input', () => {
    expect(formatDateTime(undefined)).toBe('--')
  })

  it('returns fallback for empty string input', () => {
    expect(formatDateTime('')).toBe('--')
  })

  it('returns custom fallback when provided', () => {
    expect(formatDateTime(null, 'N/A')).toBe('N/A')
  })

  it('formats a valid ISO datetime string with time component', () => {
    const result = formatDateTime('2025-01-15T14:30:00.000Z')
    // Should contain month and day
    expect(result).toContain('Jan')
    expect(result).toContain('15')
    // Should contain time digits (hour and minute will be locale-dependent due to timezone)
    // Verify that time is present by checking for colon in time format or AM/PM
    expect(result).toMatch(/\d{1,2}:\d{2}/)
  })

  it('formats midnight correctly', () => {
    const result = formatDateTime('2025-06-01T00:00:00.000Z')
    expect(result).toContain('Jun')
    expect(result).toContain('01')
    expect(result).toMatch(/\d{1,2}:\d{2}/)
  })
})
