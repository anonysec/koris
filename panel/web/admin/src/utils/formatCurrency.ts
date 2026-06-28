/**
 * Formats a numeric value as currency with exactly 2 decimal places
 * and a configurable symbol prefix.
 *
 * @param value - The numeric balance value (positive, negative, or zero)
 * @param symbol - The currency symbol prefix (default: '$')
 * @returns Formatted string matching "{symbol}X.XX" (e.g., "$10.50", "$-5.00")
 *
 * Validates: Requirement 11.1
 */
export function formatCurrency(value: number, symbol: string = '$'): string {
  if (!isFinite(value)) {
    return `${symbol}0.00`
  }

  return `${symbol}${value.toFixed(2)}`
}
