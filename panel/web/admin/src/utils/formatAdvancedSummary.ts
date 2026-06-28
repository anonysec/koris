/**
 * Formats speed and connection limits as a collapsed summary string.
 *
 * - Speed: displays as "X Mbps" when > 0, or "unlimited" when 0
 * - Connection: displays as integer when > 0, or "unlimited" when 0
 * - Values are separated by a visible delimiter (" · ")
 *
 * @param speedLimit - Speed limit in Mbps (0 = unlimited)
 * @param connectionLimit - Maximum concurrent connections (0 = unlimited)
 * @returns Formatted summary string (e.g., "100 Mbps · 5", "unlimited · unlimited")
 *
 * Validates: Requirement 10.5
 */
export function formatAdvancedSummary(speedLimit: number, connectionLimit: number): string {
  const speed = speedLimit > 0 ? `${speedLimit} Mbps` : 'unlimited'
  const connections = connectionLimit > 0 ? `${connectionLimit}` : 'unlimited'

  return `${speed} · ${connections}`
}
