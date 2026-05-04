import { describe, it, expect } from 'vitest';
import {
	prematurityDays,
	chronologicalAge,
	correctedAge,
	formatAge,
	isPreterm
} from '$lib/age';

describe('prematurityDays', () => {
	it('returns 0 when gestational age is null', () => {
		expect(prematurityDays(null, null)).toBe(0);
	});

	it('returns 0 for full-term (40w 0d)', () => {
		expect(prematurityDays(40, 0)).toBe(0);
	});

	it('returns 0 for post-term (no negative correction)', () => {
		expect(prematurityDays(41, 3)).toBe(0);
	});

	it('returns weeks-difference in days for preterm', () => {
		// 32w 0d → 8 weeks early = 56 days
		expect(prematurityDays(32, 0)).toBe(56);
	});

	it('handles weeks + days', () => {
		// 32w 4d → 7w 3d early = 7*7 + 3 = 52 days
		expect(prematurityDays(32, 4)).toBe(52);
	});

	it('treats null days as 0', () => {
		expect(prematurityDays(36, null)).toBe(28);
	});
});

describe('isPreterm', () => {
	it('is false for full-term', () => {
		expect(isPreterm(40, 0)).toBe(false);
	});

	it('is true under 37 weeks', () => {
		expect(isPreterm(36, 6)).toBe(true);
		expect(isPreterm(32, 4)).toBe(true);
	});

	it('is false at 37+ weeks (term)', () => {
		expect(isPreterm(37, 0)).toBe(false);
		expect(isPreterm(38, 0)).toBe(false);
	});

	it('is false when gestational age unknown', () => {
		expect(isPreterm(null, null)).toBe(false);
	});
});

describe('chronologicalAge', () => {
	it('returns whole months and remaining days', () => {
		// DOB 2026-01-10, today 2026-04-02 → 2 months 23 days
		const age = chronologicalAge('2026-01-10', new Date('2026-04-02T12:00:00Z').getTime());
		expect(age.months).toBe(2);
		expect(age.days).toBe(23);
	});

	it('handles same-day birthday', () => {
		const age = chronologicalAge('2026-01-10', new Date('2026-01-10T12:00:00Z').getTime());
		expect(age.months).toBe(0);
		expect(age.days).toBe(0);
	});

	it('rolls back month when day is negative', () => {
		// DOB 2026-01-31, today 2026-02-28 → 0 months, 28 days
		const age = chronologicalAge('2026-01-31', new Date('2026-02-28T12:00:00Z').getTime());
		expect(age.months).toBe(0);
		expect(age.days).toBe(28);
	});
});

describe('correctedAge', () => {
	it('returns null when not preterm', () => {
		expect(
			correctedAge('2026-01-10', 40, 0, new Date('2026-06-01T12:00:00Z').getTime())
		).toBeNull();
	});

	it('returns null when gestational age missing', () => {
		expect(
			correctedAge('2026-01-10', null, null, new Date('2026-06-01T12:00:00Z').getTime())
		).toBeNull();
	});

	it('subtracts prematurity from chronological age', () => {
		// Born 2026-01-10 at 32w 0d → 8 weeks (56 days) early.
		// Corrected DOB = 2026-01-10 + 56 days = 2026-03-07.
		// Today = 2026-05-09 → 2 months 2 days corrected.
		const age = correctedAge(
			'2026-01-10',
			32,
			0,
			new Date('2026-05-09T12:00:00Z').getTime()
		);
		expect(age).not.toBeNull();
		expect(age!.months).toBe(2);
		expect(age!.days).toBe(2);
	});

	it('returns 0/0 when today equals corrected DOB', () => {
		// Born 2026-01-10 at 32w 0d → corrected DOB 2026-03-07.
		const age = correctedAge(
			'2026-01-10',
			32,
			0,
			new Date('2026-03-07T12:00:00Z').getTime()
		);
		expect(age).toEqual({ months: 0, days: 0, beforeTerm: false });
	});

	it('flags beforeTerm when today is before corrected DOB', () => {
		// Born 2026-01-10 at 32w 0d → corrected DOB 2026-03-07.
		// Today = 2026-02-15 → before term.
		const age = correctedAge(
			'2026-01-10',
			32,
			0,
			new Date('2026-02-15T12:00:00Z').getTime()
		);
		expect(age).not.toBeNull();
		expect(age!.beforeTerm).toBe(true);
	});
});

describe('formatAge', () => {
	it('formats months and days', () => {
		expect(formatAge({ months: 2, days: 23, beforeTerm: false })).toBe('2 mo 23 d');
	});

	it('formats zero', () => {
		expect(formatAge({ months: 0, days: 0, beforeTerm: false })).toBe('0 mo 0 d');
	});

	it('renders "before term" when beforeTerm is true', () => {
		expect(formatAge({ months: 0, days: 0, beforeTerm: true })).toBe('not yet at term');
	});
});
