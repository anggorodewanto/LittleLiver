import { describe, it, expect } from 'vitest';
import { testColorMap, LINE_COLORS } from '$lib/chart-utils';

describe('testColorMap', () => {
	it('returns a map with correct color for each test in order', () => {
		const map = testColorMap(['ALT', 'AST', 'GGT']);

		expect(map.get('ALT')).toBe(LINE_COLORS[0]);
		expect(map.get('AST')).toBe(LINE_COLORS[1]);
		expect(map.get('GGT')).toBe(LINE_COLORS[2]);
	});

	it('wraps around when more tests than colors', () => {
		const tests = Array.from({ length: LINE_COLORS.length + 2 }, (_, i) => `test_${i}`);
		const map = testColorMap(tests);

		expect(map.get(`test_${LINE_COLORS.length}`)).toBe(LINE_COLORS[0]);
		expect(map.get(`test_${LINE_COLORS.length + 1}`)).toBe(LINE_COLORS[1]);
	});

	it('returns an empty map for empty input', () => {
		const map = testColorMap([]);
		expect(map.size).toBe(0);
	});
});
