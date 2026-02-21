/**
 * Calculates battery percentage from voltage.
 * 3.3V = 0%, 4.2V = 100%
 */
export function calculateBatteryPercentage(voltage: number): number {
  if (voltage <= 0.0) {
    return 0.0
  }
  if (voltage <= 3.3) {
    return 0.0
  }
  if (voltage >= 4.2) {
    return 100.0
  }
  // Linear interpolation: ((voltage - 3.3) / (4.2 - 3.3)) * 100
  const percentage = ((voltage - 3.3) / 0.9) * 100.0
  return Math.round(percentage * 10) / 10 // Round to 1 decimal place
}
