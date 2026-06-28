/**
 * Converts a byte value to a human-readable string.
 * Produces output matching "X.X UNIT" where UNIT is B, KB, MB, GB, or TB.
 *
 * @param bytes - Non-negative number of bytes
 * @returns Formatted string (e.g., "2.4 GB")
 *
 * Validates: Requirement 3.2
 */
export function formatBytes(bytes: number): string {
  if (bytes < 0) bytes = 0

  const units = ['B', 'KB', 'MB', 'GB', 'TB'] as const
  const base = 1024

  if (bytes === 0) {
    return '0.0 B'
  }

  let unitIndex = 0
  let value = bytes

  while (value >= base && unitIndex < units.length - 1) {
    value /= base
    unitIndex++
  }

  return `${value.toFixed(1)} ${units[unitIndex]}`
}
