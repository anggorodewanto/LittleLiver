import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import DiaperChart from '$lib/components/DiaperChart.svelte';

const mockDiaperData = [
	{ date: '2026-03-13', wet_count: 5, stool_count: 2 },
	{ date: '2026-03-14', wet_count: 6, stool_count: 3 },
	{ date: '2026-03-15', wet_count: 4, stool_count: 1 },
	{ date: '2026-03-16', wet_count: 7, stool_count: 2 }
];

describe('DiaperChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(DiaperChart, {
			props: { data: mockDiaperData }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js bar chart', () => {
		render(DiaperChart, { props: { data: mockDiaperData } });

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('bar');
	});

	it('uses daily dates as labels', () => {
		render(DiaperChart, { props: { data: mockDiaperData } });

		const config = chartConstructorCalls[0][1] as {
			data: { labels: string[] };
		};
		expect(config.data.labels).toEqual(['2026-03-13', '2026-03-14', '2026-03-15', '2026-03-16']);
	});

	it('has separate datasets for wet and stool counts', () => {
		render(DiaperChart, { props: { data: mockDiaperData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: number[] }[] };
		};
		const datasets = config.data.datasets;

		const wetDs = datasets.find((d) => d.label === 'Wet');
		expect(wetDs).toBeDefined();
		expect(wetDs!.data).toEqual([5, 6, 4, 7]);

		const stoolDs = datasets.find((d) => d.label === 'Stool');
		expect(stoolDs).toBeDefined();
		expect(stoolDs!.data).toEqual([2, 3, 1, 2]);
	});

	it('uses stacked bar configuration', () => {
		render(DiaperChart, { props: { data: mockDiaperData } });

		const config = chartConstructorCalls[0][1] as {
			options: { scales: { x: { stacked: boolean }; y: { stacked: boolean } } };
		};
		expect(config.options.scales.x.stacked).toBe(true);
		expect(config.options.scales.y.stacked).toBe(true);
	});

	it('configures y-axis with count label', () => {
		render(DiaperChart, { props: { data: mockDiaperData } });

		const config = chartConstructorCalls[0][1] as {
			options: { scales: { y: { title: { text: string } } } };
		};
		expect(config.options.scales.y.title.text).toBe('Count');
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(DiaperChart, {
			props: { data: mockDiaperData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data" message instead of chart when data is empty', () => {
		const { container } = render(DiaperChart, {
			props: { data: [] }
		});

		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});
});
