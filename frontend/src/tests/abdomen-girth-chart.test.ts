import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import AbdomenGirthChart from '$lib/components/AbdomenGirthChart.svelte';

const mockAbdomenData = [
	{ timestamp: '2026-03-13T08:00:00Z', girth_cm: 38.5 },
	{ timestamp: '2026-03-14T09:00:00Z', girth_cm: 39.0 },
	{ timestamp: '2026-03-15T10:00:00Z', girth_cm: 39.2 },
	{ timestamp: '2026-03-16T08:30:00Z', girth_cm: 38.8 }
];

describe('AbdomenGirthChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(AbdomenGirthChart, {
			props: { data: mockAbdomenData }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js line chart with abdomen girth data', () => {
		render(AbdomenGirthChart, { props: { data: mockAbdomenData } });

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('line');
	});

	it('maps data points with x as timestamp and y as girth_cm', () => {
		render(AbdomenGirthChart, { props: { data: mockAbdomenData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { data: { x: number; y: number }[] }[] };
		};
		const points = config.data.datasets[0].data;
		expect(points).toHaveLength(4);
		expect(points[0].y).toBe(38.5);
		expect(points[1].y).toBe(39.0);
		expect(points[2].y).toBe(39.2);
		expect(points[3].y).toBe(38.8);
	});

	it('labels dataset as Abdomen Girth', () => {
		render(AbdomenGirthChart, { props: { data: mockAbdomenData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		expect(config.data.datasets[0].label).toBe('Abdomen Girth');
	});

	it('configures y-axis with cm unit label', () => {
		render(AbdomenGirthChart, { props: { data: mockAbdomenData } });

		const config = chartConstructorCalls[0][1] as {
			options: { scales: { y: { title: { text: string } } } };
		};
		expect(config.options.scales.y.title.text).toBe('Girth (cm)');
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(AbdomenGirthChart, {
			props: { data: mockAbdomenData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data" message instead of chart when data is empty', () => {
		const { container } = render(AbdomenGirthChart, {
			props: { data: [] }
		});

		expect(container.textContent).toContain('No data');
		expect(chartConstructorCalls.length).toBe(0);
	});
});
