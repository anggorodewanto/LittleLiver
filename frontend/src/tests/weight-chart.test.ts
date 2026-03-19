import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import WeightChart from '$lib/components/WeightChart.svelte';

const mockWeightData = [
	{ timestamp: '2026-03-01T10:00:00Z', weight_kg: 3.5, measurement_source: 'home_scale' },
	{ timestamp: '2026-03-08T10:00:00Z', weight_kg: 3.8, measurement_source: 'clinic' },
	{ timestamp: '2026-03-15T10:00:00Z', weight_kg: 4.1, measurement_source: 'home_scale' }
];

const mockPercentiles = {
	p3: [
		{ age_days: 0, weight_kg: 2.5 },
		{ age_days: 30, weight_kg: 3.2 },
		{ age_days: 60, weight_kg: 4.0 }
	],
	p15: [
		{ age_days: 0, weight_kg: 2.8 },
		{ age_days: 30, weight_kg: 3.6 },
		{ age_days: 60, weight_kg: 4.4 }
	],
	p50: [
		{ age_days: 0, weight_kg: 3.2 },
		{ age_days: 30, weight_kg: 4.2 },
		{ age_days: 60, weight_kg: 5.1 }
	],
	p85: [
		{ age_days: 0, weight_kg: 3.6 },
		{ age_days: 30, weight_kg: 4.8 },
		{ age_days: 60, weight_kg: 5.8 }
	],
	p97: [
		{ age_days: 0, weight_kg: 3.9 },
		{ age_days: 30, weight_kg: 5.2 },
		{ age_days: 60, weight_kg: 6.3 }
	]
};

describe('WeightChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(WeightChart, {
			props: { data: mockWeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js line chart with weight data', () => {
		render(WeightChart, {
			props: { data: mockWeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('line');
	});

	it('includes WHO percentile band datasets (p3, p15, p50, p85, p97)', () => {
		render(WeightChart, {
			props: { data: mockWeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		const datasets = config.data.datasets;

		expect(datasets.length).toBe(6);

		const labels = datasets.map((d) => d.label);
		expect(labels).toContain('Weight');
		expect(labels).toContain('3rd');
		expect(labels).toContain('15th');
		expect(labels).toContain('50th');
		expect(labels).toContain('85th');
		expect(labels).toContain('97th');
	});

	it('renders percentile datasets with dashed lines', () => {
		render(WeightChart, {
			props: { data: mockWeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; borderDash?: number[] }[] };
		};
		const datasets = config.data.datasets;

		const percentileDatasets = datasets.filter((d) => d.label !== 'Weight');
		expect(percentileDatasets.length).toBe(5);
		for (const ds of percentileDatasets) {
			expect(ds.borderDash).toBeDefined();
		}
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(WeightChart, {
			props: { data: mockWeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('renders a canvas without crashing when data is empty', () => {
		const { container } = render(WeightChart, {
			props: { data: [], percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
		expect(chartConstructorCalls.length).toBe(1);
	});
});
