import { describe, it, expect } from 'vitest';
import { toISO8601 } from '$lib/datetime';

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
