/**
 * Computes a target date from a base date plus an offset string.
 * Supported offsets: +1d, +7d, +1m, +2m, +3m, +6m, +1y
 *
 * - Days are added directly
 * - Months are added via calendar month addition (same day, clamped to month end)
 * - Years are added via calendar year addition
 *
 * @param baseDate - The starting date
 * @param offset - The offset string (e.g., '+1d', '+7d', '+1m', '+2m', '+3m', '+6m', '+1y')
 * @returns The computed target date
 *
 * Validates: Requirement 2.15
 */
export type ExpiryOffset = '+1d' | '+7d' | '+1m' | '+2m' | '+3m' | '+6m' | '+1y'

export function computeExpiryDate(baseDate: Date, offset: ExpiryOffset): Date {
  const result = new Date(baseDate.getTime())

  switch (offset) {
    case '+1d':
      result.setDate(result.getDate() + 1)
      break
    case '+7d':
      result.setDate(result.getDate() + 7)
      break
    case '+1m':
      addMonths(result, 1)
      break
    case '+2m':
      addMonths(result, 2)
      break
    case '+3m':
      addMonths(result, 3)
      break
    case '+6m':
      addMonths(result, 6)
      break
    case '+1y':
      addMonths(result, 12)
      break
  }

  return result
}

/**
 * Adds months to a date, clamping to the end of the target month
 * if the original day exceeds the target month's length.
 */
function addMonths(date: Date, months: number): void {
  const originalDay = date.getDate()
  date.setMonth(date.getMonth() + months)

  // If the day changed (e.g., Jan 31 + 1 month → Mar 3), clamp to end of target month
  if (date.getDate() !== originalDay) {
    date.setDate(0) // Sets to last day of previous month (i.e., the intended target month)
  }
}
