import type { LegendItem, ChartDataset } from 'chart.js';

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
