import type { LegendItem, ChartDataset } from 'chart.js';

export const LINE_COLORS = [
	'#ef4444',
	'#3b82f6',
	'#22c55e',
	'#f59e0b',
	'#8b5cf6',
	'#ec4899',
	'#06b6d4',
	'#84cc16'
];

/** Maps an ordered list of test names to their chart line colors */
export function testColorMap(testNames: string[]): Map<string, string> {
	const map = new Map<string, string>();
	for (let i = 0; i < testNames.length; i++) {
		map.set(testNames[i], LINE_COLORS[i % LINE_COLORS.length]);
	}
	return map;
}

export const dateTickCallback = (value: string | number) =>
	new Date(value as number).toLocaleDateString();

/** Filters reference datasets (dashed lines, zero-width borders) out of the legend */
export function legendFilter(item: LegendItem, chartData: { datasets: ChartDataset[] }): boolean {
	const dataset = chartData.datasets[item.datasetIndex!];
	const bd = (dataset as Record<string, unknown>).borderDash;
	const hasDash = Array.isArray(bd) && bd.length > 0;
	const zeroBorder = (dataset as Record<string, unknown>).borderWidth === 0;
	return !hasDash && !zeroBorder;
}

/** Subtitle config for charts with WHO percentile curves */
export const percentileSubtitle = {
	display: true,
	text: 'Dashed lines: WHO percentiles (3rd, 15th, 50th, 85th, 97th)',
	font: { size: 11, style: 'italic' as const },
	padding: { bottom: 4 }
};
