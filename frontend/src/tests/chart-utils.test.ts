import { describe, it, expect } from 'vitest';
import { testColorMap, LINE_COLORS, dateTooltipTitle } from '$lib/chart-utils';
import type { ChartType, TooltipItem } from 'chart.js';

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

describe('dateTooltipTitle', () => {
	it('formats epoch ms x value as locale date+time string', () => {
		const ts = new Date('2026-03-15T14:30:00Z').getTime();
		const items = [{ parsed: { x: ts } }] as unknown as TooltipItem<ChartType>[];
		expect(dateTooltipTitle(items)).toBe(new Date(ts).toLocaleString());
	});

	it('returns empty string for empty items', () => {
		expect(dateTooltipTitle([] as TooltipItem<ChartType>[])).toBe('');
	});
});
