/**
 * Formats timestamp values for chart axis labels based on the time range.
 * 
 * @param splits - Array of timestamp values (milliseconds since epoch)
 * @param minTimestamp - Minimum timestamp in the data range
 * @param maxTimestamp - Maximum timestamp in the data range
 * @returns Array of formatted date/time strings
 */
export function formatTimeLabels(
  splits: number[],
  minTimestamp: number,
  maxTimestamp: number
): string[] {
  const range = maxTimestamp - minTimestamp;
  const hours = range / (1000 * 60 * 60);
  const days = hours / 24;

  return splits.map((val, index) => {
    // Ensure val is a valid number (timestamp in milliseconds)
    const timestamp = typeof val === 'number' && !isNaN(val) ? val : Number(val);
    
    // Validate timestamp is reasonable (between 1970 and 2100)
    if (isNaN(timestamp) || timestamp < 0 || timestamp > 4102444800000) {
      console.warn(`Invalid timestamp: ${val}`);
      return '';
    }

    const d = new Date(timestamp);
    
    // Validate date object
    if (isNaN(d.getTime())) {
      console.warn(`Invalid date created from timestamp: ${timestamp}`);
      return '';
    }

    if (days >= 7) {
      // 7+ days: show date
      return d.toLocaleDateString('en-US', { month: 'short', day: 'numeric' });
    } else if (hours >= 24) {
      // 1-7 days: show date and time
      return d.toLocaleString('en-US', { 
        month: 'short', 
        day: 'numeric', 
        hour: 'numeric', 
        minute: '2-digit' 
      });
    } else if (hours >= 6) {
      // 6-24 hours: show time with date if crossing midnight
      // FIXED: Use index parameter instead of indexOf to avoid bugs
      const prevVal = index > 0 ? splits[index - 1] : null;
      if (prevVal !== null) {
        const prevDate = new Date(prevVal);
        // Check if we crossed midnight (different day)
        if (!isNaN(prevDate.getTime()) && d.getDate() !== prevDate.getDate()) {
          return d.toLocaleString('en-US', { 
            month: 'short', 
            day: 'numeric', 
            hour: 'numeric', 
            minute: '2-digit' 
          });
        }
      }
      return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
    } else {
      // < 6 hours: show time
      return d.toLocaleTimeString('en-US', { hour: 'numeric', minute: '2-digit' });
    }
  });
}
