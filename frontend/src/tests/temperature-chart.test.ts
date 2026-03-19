import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import TemperatureChart from '$lib/components/TemperatureChart.svelte';

const mockTemperatureData = [
	{ timestamp: '2026-03-13T08:00:00Z', value: 36.8, method: 'rectal' },
	{ timestamp: '2026-03-14T09:00:00Z', value: 37.2, method: 'axillary' },
	{ timestamp: '2026-03-15T10:00:00Z', value: 38.5, method: 'rectal' },
	{ timestamp: '2026-03-16T08:30:00Z', value: 37.0, method: 'ear' }
];

describe('TemperatureChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(TemperatureChart, {
			props: { data: mockTemperatureData }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js line chart with temperature data', () => {
		render(TemperatureChart, { props: { data: mockTemperatureData } });

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('line');
	});

	it('includes a fever threshold annotation line at 38.0 degrees C', () => {
		render(TemperatureChart, { props: { data: mockTemperatureData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { y: number }[] }[] };
		};
		const datasets = config.data.datasets;
		const thresholdDs = datasets.find((d) => d.label === 'Fever Threshold');
		expect(thresholdDs).toBeDefined();

		for (const point of thresholdDs!.data) {
			expect(point.y).toBe(38.0);
		}
	});

	it('fever threshold line uses a distinct dashed red style', () => {
		render(TemperatureChart, { props: { data: mockTemperatureData } });

		const config = chartConstructorCalls[0][1] as {
			data: {
				datasets: {
					label: string;
					borderColor: string;
					borderDash: number[];
				}[];
			};
		};
		const datasets = config.data.datasets;
		const thresholdDs = datasets.find((d) => d.label === 'Fever Threshold');

		expect(thresholdDs!.borderColor).toBe('red');
		expect(thresholdDs!.borderDash).toBeDefined();
		expect(thresholdDs!.borderDash.length).toBeGreaterThan(0);
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(TemperatureChart, {
			props: { data: mockTemperatureData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('renders a canvas without crashing when data is empty', () => {
		const { container } = render(TemperatureChart, {
			props: { data: [] }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
		expect(chartConstructorCalls.length).toBe(1);

		// Threshold dataset should have empty data when no temperature points
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { y: number }[] }[] };
		};
		const thresholdDs = config.data.datasets.find((d) => d.label === 'Fever Threshold');
		expect(thresholdDs).toBeDefined();
		expect(thresholdDs!.data).toHaveLength(0);
	});
});
