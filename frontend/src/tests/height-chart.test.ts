import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import HeightChart from '$lib/components/HeightChart.svelte';

const mockHeightData = [
	{ timestamp: '2026-03-01T10:00:00Z', height_cm: 54.0, measurement_source: 'home_scale' },
	{ timestamp: '2026-03-08T10:00:00Z', height_cm: 54.8, measurement_source: 'clinic' },
	{ timestamp: '2026-03-15T10:00:00Z', height_cm: 55.5, measurement_source: 'home_scale' }
];

describe('HeightChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(HeightChart, {
			props: { data: mockHeightData }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js line chart with height data', () => {
		render(HeightChart, {
			props: { data: mockHeightData }
		});

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('line');
	});

	it('renders Height dataset only', () => {
		render(HeightChart, {
			props: { data: mockHeightData }
		});

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		const labels = config.data.datasets.map((d) => d.label);
		expect(labels).toEqual(['Height']);
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(HeightChart, {
			props: { data: mockHeightData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data available" when data is empty', () => {
		const { container } = render(HeightChart, {
			props: { data: [] }
		});

		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});
});
