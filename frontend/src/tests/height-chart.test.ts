import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import HeightChart from '$lib/components/HeightChart.svelte';
import { dateTooltipTitle } from '$lib/chart-utils';

const mockHeightData = [
	{ timestamp: '2026-03-01T10:00:00Z', height_cm: 54.0, measurement_source: 'home_scale' },
	{ timestamp: '2026-03-08T10:00:00Z', height_cm: 54.8, measurement_source: 'clinic' },
	{ timestamp: '2026-03-15T10:00:00Z', height_cm: 55.5, measurement_source: 'home_scale' }
];

const mockPercentiles = {
	p3: [
		{ age_days: 0, value: 46.1 },
		{ age_days: 30, value: 50.8 },
		{ age_days: 60, value: 54.4 }
	],
	p15: [
		{ age_days: 0, value: 47.5 },
		{ age_days: 30, value: 52.2 },
		{ age_days: 60, value: 55.9 }
	],
	p50: [
		{ age_days: 0, value: 49.9 },
		{ age_days: 30, value: 54.7 },
		{ age_days: 60, value: 58.4 }
	],
	p85: [
		{ age_days: 0, value: 52.3 },
		{ age_days: 30, value: 57.2 },
		{ age_days: 60, value: 60.9 }
	],
	p97: [
		{ age_days: 0, value: 53.7 },
		{ age_days: 30, value: 58.6 },
		{ age_days: 60, value: 62.4 }
	]
};

describe('HeightChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		expect(container.querySelector('canvas')).not.toBeNull();
	});

	it('creates a Chart.js line chart with height data', () => {
		render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('line');
	});

	it('includes Height main dataset plus 5 WHO percentile bands', () => {
		render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		expect(config.data.datasets.map((d) => d.label)).toEqual([
			'Height',
			'3rd',
			'15th',
			'50th',
			'85th',
			'97th'
		]);
	});

	it('renders percentile datasets with dashed lines', () => {
		render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; borderDash?: number[] }[] };
		};
		const percentileDatasets = config.data.datasets.filter((d) => d.label !== 'Height');
		expect(percentileDatasets.length).toBe(5);
		for (const ds of percentileDatasets) {
			expect(ds.borderDash).toBeDefined();
		}
	});

	it('converts percentile age_days to date timestamps via dateOfBirth', () => {
		const dob = '2026-01-15';
		render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: dob }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { x: number; y: number }[] }[] };
		};
		const p50 = config.data.datasets.find((d) => d.label === '50th')!;
		const dobMs = new Date(dob).getTime();
		const dayMs = 24 * 60 * 60 * 1000;
		expect(p50.data[0].x).toBe(dobMs);
		expect(p50.data[1].x).toBe(dobMs + 30 * dayMs);
	});

	it('renders Height-only dataset when percentiles are null', () => {
		render(HeightChart, {
			props: { data: mockHeightData, percentiles: null, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		expect(config.data.datasets.map((d) => d.label)).toEqual(['Height']);
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		unmount();
		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data available" when data is empty', () => {
		const { container } = render(HeightChart, {
			props: { data: [], percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});

	it('configures tooltip title callback to format x value as date', () => {
		render(HeightChart, {
			props: { data: mockHeightData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			options: { plugins: { tooltip: { callbacks: { title: unknown } } } };
		};
		expect(config.options.plugins.tooltip.callbacks.title).toBe(dateTooltipTitle);
	});
});
