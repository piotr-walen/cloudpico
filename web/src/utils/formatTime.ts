/**
 * Formats a datetime string to a readable time format
 */
export function formatReadingTime(timeString: string): string {
  const date = new Date(timeString)
  return date.toLocaleTimeString('en-US', {
    hour: 'numeric',
    minute: '2-digit',
    hour12: true,
  })
}
