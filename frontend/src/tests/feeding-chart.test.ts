import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import FeedingChart from '$lib/components/FeedingChart.svelte';

const mockFeedingData = [
	{ date: '2026-03-13', total_volume_ml: 600, total_calories: 420, feed_count: 6, by_type: { breast_milk: 300, formula: 300, solid: 0, other: 0 } },
	{ date: '2026-03-14', total_volume_ml: 650, total_calories: 455, feed_count: 7, by_type: { breast_milk: 200, formula: 400, solid: 50, other: 0 } },
	{ date: '2026-03-15', total_volume_ml: 580, total_calories: 390, feed_count: 5, by_type: { breast_milk: 280, formula: 300, solid: 0, other: 0 } }
];

describe('FeedingChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(FeedingChart, {
			props: { data: mockFeedingData }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js bar chart', () => {
		render(FeedingChart, { props: { data: mockFeedingData } });

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('bar');
	});

	it('uses daily dates as labels', () => {
		render(FeedingChart, { props: { data: mockFeedingData } });

		const config = chartConstructorCalls[0][1] as {
			data: { labels: string[] };
		};
		expect(config.data.labels).toEqual(['2026-03-13', '2026-03-14', '2026-03-15']);
	});

	it('shows daily total calories as bar data', () => {
		render(FeedingChart, { props: { data: mockFeedingData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { data: number[]; label: string }[] };
		};
		const dataset = config.data.datasets.find((d) => d.label === 'Daily Calories');
		expect(dataset).toBeDefined();
		expect(dataset!.data).toEqual([420, 455, 390]);
	});

	it('configures y-axis with kcal unit label', () => {
		render(FeedingChart, { props: { data: mockFeedingData } });

		const config = chartConstructorCalls[0][1] as {
			options: { scales: { y: { title: { text: string } } } };
		};
		expect(config.options.scales.y.title.text).toBe('Calories (kcal)');
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(FeedingChart, {
			props: { data: mockFeedingData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data" message instead of chart when data is empty', () => {
		const { container } = render(FeedingChart, {
			props: { data: [] }
		});

		expect(container.textContent).toContain('No data');
		expect(chartConstructorCalls.length).toBe(0);
	});
});
