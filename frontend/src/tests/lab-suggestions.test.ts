import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

import { apiClient } from '$lib/api';
import {
	QUICK_PICKS,
	mergeWithQuickPicks,
	normalizeTestName,
	findSuggestionMatch,
	fetchLabSuggestions
} from '$lib/labSuggestions';

describe('labSuggestions', () => {
	beforeEach(() => {
		vi.mocked(apiClient.get).mockReset();
	});

	describe('mergeWithQuickPicks', () => {
		it('returns QUICK_PICKS when db is empty', () => {
			const merged = mergeWithQuickPicks([]);
			expect(merged.length).toBe(QUICK_PICKS.length);
			for (const pick of QUICK_PICKS) {
				expect(merged.some((m) => m.test_name === pick.testName)).toBe(true);
			}
		});

		it('db entry overrides quick pick with same test_name', () => {
			const merged = mergeWithQuickPicks([
				{ test_name: 'ALT', unit: 'µkat/L', normal_range: '0-0.7' }
			]);
			const alt = merged.find((m) => m.test_name === 'ALT');
			expect(alt?.unit).toBe('µkat/L');
			expect(alt?.normal_range).toBe('0-0.7');
		});

		it('includes db-only entries', () => {
			const merged = mergeWithQuickPicks([
				{ test_name: 'SGOT/AST', unit: 'U/L' }
			]);
			expect(merged.some((m) => m.test_name === 'SGOT/AST')).toBe(true);
		});
	});

	describe('normalizeTestName', () => {
		it('lowercases', () => {
			expect(normalizeTestName('AST')).toBe('ast');
		});

		it('strips slashes, spaces, dashes, underscores, dots', () => {
			expect(normalizeTestName('SGOT/AST')).toBe('sgotast');
			expect(normalizeTestName('Total Bilirubin')).toBe('totalbilirubin');
			expect(normalizeTestName('total_bilirubin')).toBe('totalbilirubin');
			expect(normalizeTestName('Gamma-GT')).toBe('gammagt');
			expect(normalizeTestName('Hgb.A1c')).toBe('hgba1c');
		});
	});

	describe('findSuggestionMatch', () => {
		const suggestions = [
			{ test_name: 'SGOT/AST', unit: 'U/L', normal_range: '0-40' },
			{ test_name: 'total_bilirubin', unit: 'mg/dL' },
			{ test_name: 'Hemoglobin A1c', unit: '%' }
		];

		it('exact match (case-insensitive)', () => {
			expect(findSuggestionMatch('sgot/ast', suggestions)?.test_name).toBe('SGOT/AST');
		});

		it('substring match: AST → SGOT/AST', () => {
			expect(findSuggestionMatch('AST', suggestions)?.test_name).toBe('SGOT/AST');
		});

		it('reverse substring: Total Bilirubin → total_bilirubin', () => {
			expect(findSuggestionMatch('Total Bilirubin', suggestions)?.test_name).toBe('total_bilirubin');
		});

		it('returns undefined for empty input', () => {
			expect(findSuggestionMatch('', suggestions)).toBeUndefined();
		});

		it('returns undefined for non-match', () => {
			expect(findSuggestionMatch('platelets', suggestions)).toBeUndefined();
		});
	});

	describe('fetchLabSuggestions', () => {
		it('returns API result on success', async () => {
			vi.mocked(apiClient.get).mockResolvedValue([{ test_name: 'foo' }]);
			const result = await fetchLabSuggestions('baby-1');
			expect(result).toEqual([{ test_name: 'foo' }]);
			expect(apiClient.get).toHaveBeenCalledWith('/babies/baby-1/labs/tests');
		});

		it('returns [] on API failure', async () => {
			vi.mocked(apiClient.get).mockRejectedValue(new Error('boom'));
			const result = await fetchLabSuggestions('baby-1');
			expect(result).toEqual([]);
		});
	});
});
