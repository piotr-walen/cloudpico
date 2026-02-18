import { formatTimeLabels } from '../dateFormatter';

describe('formatTimeLabels', () => {
  // Helper to create timestamps
  const now = Date.now();
  const oneHour = 60 * 60 * 1000;
  const oneDay = 24 * oneHour;

  describe('7+ days range', () => {
    it('should format dates for long ranges', () => {
      const min = now - 10 * oneDay;
      const max = now;
      const splits = [min, min + 5 * oneDay, max];
      
      const result = formatTimeLabels(splits, min, max);
      
      expect(result).toHaveLength(3);
      // Should contain month abbreviations and day numbers
      result.forEach(label => {
        expect(label).toMatch(/^[A-Z][a-z]{2} \d{1,2}$/);
      });
    });
  });

  describe('1-7 days range', () => {
    it('should format date and time for medium ranges', () => {
      const min = now - 3 * oneDay;
      const max = now;
      const splits = [min, min + oneDay, max];
      
      const result = formatTimeLabels(splits, min, max);
      
      expect(result).toHaveLength(3);
      // Should contain month, day, hour, and minute
      result.forEach(label => {
        expect(label).toMatch(/^[A-Z][a-z]{2} \d{1,2}, \d{1,2}:\d{2}/);
      });
    });
  });

  describe('6-24 hours range', () => {
    it('should show time only when not crossing midnight', () => {
      const min = now - 12 * oneHour;
      const max = now;
      // All timestamps on the same day
      const sameDay = new Date(now);
      sameDay.setHours(10, 0, 0, 0);
      const splits = [
        sameDay.getTime(),
        sameDay.getTime() + 2 * oneHour,
        sameDay.getTime() + 4 * oneHour,
      ];
      
      const result = formatTimeLabels(splits, min, max);
      
      expect(result).toHaveLength(3);
      // Should be time-only format
      result.forEach(label => {
        expect(label).toMatch(/^\d{1,2}:\d{2}/);
      });
    });

    it('should show date and time when crossing midnight', () => {
      const min = now - 12 * oneHour;
      const max = now;
      // Create timestamps that cross midnight
      const beforeMidnight = new Date(now);
      beforeMidnight.setHours(23, 0, 0, 0);
      const afterMidnight = new Date(now);
      afterMidnight.setHours(1, 0, 0, 0);
      const splits = [
        beforeMidnight.getTime(),
        afterMidnight.getTime(),
      ];
      
      const result = formatTimeLabels(splits, min, max);
      
      expect(result).toHaveLength(2);
      // Second label should include date since it crossed midnight
      expect(result[1]).toMatch(/^[A-Z][a-z]{2} \d{1,2},/);
    });
  });

  describe('< 6 hours range', () => {
    it('should format time only for short ranges', () => {
      const min = now - 3 * oneHour;
      const max = now;
      const splits = [min, min + oneHour, max];
      
      const result = formatTimeLabels(splits, min, max);
      
      expect(result).toHaveLength(3);
      // Should be time-only format
      result.forEach(label => {
        expect(label).toMatch(/^\d{1,2}:\d{2}/);
      });
    });
  });

  describe('edge cases', () => {
    it('should handle empty splits array', () => {
      const result = formatTimeLabels([], now - oneDay, now);
      expect(result).toEqual([]);
    });

    it('should handle invalid timestamps gracefully', () => {
      const splits = [NaN, -1, 99999999999999, now];
      const result = formatTimeLabels(splits, now - oneDay, now);
      
      expect(result).toHaveLength(4);
      // Invalid timestamps should return empty strings
      expect(result[0]).toBe('');
      expect(result[1]).toBe('');
      expect(result[2]).toBe('');
      // Valid timestamp should be formatted
      expect(result[3]).toMatch(/^\d{1,2}:\d{2}/);
    });

    it('should handle first element correctly (no previous value)', () => {
      const min = now - 12 * oneHour;
      const max = now;
      const splits = [min];
      
      const result = formatTimeLabels(splits, min, max);
      
      expect(result).toHaveLength(1);
      expect(result[0]).toMatch(/^\d{1,2}:\d{2}/);
    });

    it('should not produce year 51213 bug', () => {
      // Test with realistic timestamps
      const baseTime = new Date('2024-01-15T12:00:00Z').getTime();
      const splits = [
        baseTime,
        baseTime + oneHour,
        baseTime + 2 * oneHour,
      ];
      
      const result = formatTimeLabels(splits, baseTime, baseTime + 2 * oneHour);
      
      result.forEach(label => {
        // Should not contain year 51213 or any weird years
        expect(label).not.toMatch(/51213/);
        expect(label).not.toMatch(/\d{5}/); // No 5-digit numbers
        // Should be a reasonable date/time format
        expect(label.length).toBeGreaterThan(0);
        expect(label.length).toBeLessThan(50);
      });
    });

    it('should handle string timestamps (conversion)', () => {
      const min = now - oneDay;
      const max = now;
      // Mix of number and string timestamps
      const splits = [String(min), min + oneHour, String(max)];
      
      const result = formatTimeLabels(splits as any, min, max);
      
      expect(result).toHaveLength(3);
      result.forEach(label => {
        expect(label.length).toBeGreaterThan(0);
      });
    });
  });
});
