import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import FeedingChart from '$lib/components/FeedingChart.svelte';

const mockFeedingData = [
	{ date: '2026-03-13', total_volume_ml: 600, liquid_volume_ml: 600, solid_volume_ml: 0, solid_amount_g: 0, total_calories: 420, feed_count: 6, by_type: { breast_milk: 300, formula: 300, solid: 0, other: 0 } },
	{ date: '2026-03-14', total_volume_ml: 650, liquid_volume_ml: 600, solid_volume_ml: 50, solid_amount_g: 30, total_calories: 455, feed_count: 7, by_type: { breast_milk: 200, formula: 400, solid: 50, other: 0 } },
	{ date: '2026-03-15', total_volume_ml: 580, liquid_volume_ml: 580, solid_volume_ml: 0, solid_amount_g: 0, total_calories: 390, feed_count: 5, by_type: { breast_milk: 280, formula: 300, solid: 0, other: 0 } }
];

// total_volume_ml deliberately differs from liquid_volume_ml so a dataset that
// still reads total_volume_ml cannot pass by coincidence.
const mockSplitFeedingData = [
	{ date: '2026-03-13', total_volume_ml: 180, liquid_volume_ml: 120, solid_volume_ml: 60, solid_amount_g: 25, total_calories: 130, feed_count: 3, by_type: { breast_milk: 120, formula: 0, solid: 60, other: 0 } },
	{ date: '2026-03-14', total_volume_ml: 240, liquid_volume_ml: 200, solid_volume_ml: 40, solid_amount_g: 75, total_calories: 170, feed_count: 4, by_type: { breast_milk: 200, formula: 0, solid: 40, other: 0 } }
];

// Older payloads that predate the split fields.
const mockLegacyFeedingData = [
	{ date: '2026-03-13', total_volume_ml: 600, total_calories: 420, feed_count: 6, by_type: { breast_milk: 300, formula: 300, solid: 0, other: 0 } },
	{ date: '2026-03-14', total_volume_ml: 650, total_calories: 455, feed_count: 7, by_type: { breast_milk: 200, formula: 400, solid: 50, other: 0 } }
];

type DatasetConfig = {
	label: string;
	data: number[];
	backgroundColor: string;
	borderColor: string;
	borderWidth: number;
	yAxisID: string;
};

function datasetsOf(): DatasetConfig[] {
	const config = chartConstructorCalls[0][1] as { data: { datasets: DatasetConfig[] } };
	return config.data.datasets;
}

function datasetNamed(label: string): DatasetConfig {
	const dataset = datasetsOf().find((d) => d.label === label);
	expect(dataset).toBeDefined();
	return dataset!;
}

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
		const dataset = config.data.datasets.find((d) => d.label === 'Daily Calories (kcal)');
		expect(dataset).toBeDefined();
		expect(dataset!.data).toEqual([420, 455, 390]);
	});

	it('splits intake into calories, liquid volume, solid volume and solid weight datasets', () => {
		render(FeedingChart, { props: { data: mockFeedingData } });

		expect(datasetsOf().map((d) => d.label)).toEqual([
			'Daily Calories (kcal)',
			'Liquid Volume (mL)',
			'Solid Volume (mL)',
			'Solid Food (g)'
		]);
	});

	it('plots liquid volume from liquid_volume_ml, not total_volume_ml', () => {
		render(FeedingChart, { props: { data: mockSplitFeedingData } });

		expect(datasetNamed('Liquid Volume (mL)').data).toEqual([120, 200]);
	});

	it('plots solid volume from solid_volume_ml', () => {
		render(FeedingChart, { props: { data: mockSplitFeedingData } });

		expect(datasetNamed('Solid Volume (mL)').data).toEqual([60, 40]);
	});

	it('plots solid food weight from solid_amount_g', () => {
		render(FeedingChart, { props: { data: mockSplitFeedingData } });

		expect(datasetNamed('Solid Food (g)').data).toEqual([25, 75]);
	});

	it('defaults missing split fields to 0 instead of undefined or NaN', () => {
		render(FeedingChart, { props: { data: mockLegacyFeedingData } });

		expect(datasetNamed('Liquid Volume (mL)').data).toEqual([0, 0]);
		expect(datasetNamed('Solid Volume (mL)').data).toEqual([0, 0]);
		expect(datasetNamed('Solid Food (g)').data).toEqual([0, 0]);
	});

	it('puts every volume and weight dataset on the right-hand y1 axis', () => {
		render(FeedingChart, { props: { data: mockSplitFeedingData } });

		expect(datasetNamed('Daily Calories (kcal)').yAxisID).toBe('y');
		expect(datasetNamed('Liquid Volume (mL)').yAxisID).toBe('y1');
		expect(datasetNamed('Solid Volume (mL)').yAxisID).toBe('y1');
		expect(datasetNamed('Solid Food (g)').yAxisID).toBe('y1');
	});

	it('gives each intake series a distinct colour', () => {
		render(FeedingChart, { props: { data: mockSplitFeedingData } });

		const liquid = datasetNamed('Liquid Volume (mL)');
		expect(liquid.backgroundColor).toBe('#3b82f6');
		expect(liquid.borderColor).toBe('#2563eb');

		const solidVolume = datasetNamed('Solid Volume (mL)');
		expect(solidVolume.backgroundColor).toBe('#14b8a6');
		expect(solidVolume.borderColor).toBe('#0d9488');

		const solidWeight = datasetNamed('Solid Food (g)');
		expect(solidWeight.backgroundColor).toBe('#a855f7');
		expect(solidWeight.borderColor).toBe('#9333ea');

		expect(datasetsOf().every((d) => d.borderWidth === 1)).toBe(true);
	});

	it('labels the y1 axis for both volume and weight', () => {
		render(FeedingChart, { props: { data: mockSplitFeedingData } });

		const config = chartConstructorCalls[0][1] as {
			options: { scales: { y1: { title: { text: string } } } };
		};
		expect(config.options.scales.y1.title.text).toBe('Volume (mL) / Weight (g)');
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

		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});
});
