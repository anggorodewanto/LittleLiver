import { describe, it, expect } from 'vitest';
import { toISO8601, fromISO8601, formatTime } from '$lib/datetime';

describe('toISO8601', () => {
	it('converts local datetime-local value to correct UTC ISO 8601', () => {
		// datetime-local input gives local time like "2026-03-31T20:00"
		// This should be converted to UTC using the browser's timezone offset,
		// NOT just appending "Z" (which falsely claims the local time is UTC).
		const localInput = '2026-03-31T20:00';
		const result = toISO8601(localInput);

		// The result should be a valid ISO 8601 UTC string
		expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}Z$/);

		// Parse the result back — it should represent the same moment in time
		// as interpreting the input in the local timezone.
		const expectedDate = new Date('2026-03-31T20:00');
		const resultDate = new Date(result);
		expect(resultDate.getTime()).toBe(expectedDate.getTime());
	});

	it('already-valid ISO 8601 with Z suffix is returned as-is', () => {
		expect(toISO8601('2026-03-31T15:30:00Z')).toBe('2026-03-31T15:30:00Z');
	});
});

describe('fromISO8601', () => {
	it('converts UTC ISO 8601 to datetime-local format in local timezone', () => {
		const utcInput = '2026-03-31T15:30:00Z';
		const result = fromISO8601(utcInput);

		// Result should be in datetime-local format: YYYY-MM-DDTHH:MM
		expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);

		// Round-trip: fromISO8601 then toISO8601 should return the same UTC time
		const roundTrip = toISO8601(result);
		expect(new Date(roundTrip).getTime()).toBe(new Date(utcInput).getTime());
	});

	it('handles ISO 8601 strings with milliseconds', () => {
		const result = fromISO8601('2026-06-15T08:45:30.123Z');
		expect(result).toMatch(/^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}$/);
	});
});

describe('formatTime', () => {
	it('returns time-only string from ISO timestamp', () => {
		const result = formatTime('2026-03-20T14:30:00Z');
		// Should contain hour and minute but not the date
		expect(result).not.toContain('2026');
		expect(result).toMatch(/\d{1,2}:\d{2}/);
	});

	it('handles timestamps with milliseconds', () => {
		const result = formatTime('2026-06-15T08:45:30.123Z');
		expect(result).toMatch(/\d{1,2}:\d{2}/);
	});
});
