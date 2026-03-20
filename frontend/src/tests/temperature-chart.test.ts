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

	it('includes method-specific fever threshold lines', () => {
		render(TemperatureChart, { props: { data: mockTemperatureData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { y: number }[] }[] };
		};
		const datasets = config.data.datasets;

		// Should have threshold line for axillary/forehead (37.5) and rectal/ear (38.0)
		const lowerThreshold = datasets.find((d) => d.label === 'Threshold (axillary/forehead)');
		expect(lowerThreshold).toBeDefined();
		for (const point of lowerThreshold!.data) {
			expect(point.y).toBe(37.5);
		}

		const upperThreshold = datasets.find((d) => d.label === 'Threshold (rectal/ear)');
		expect(upperThreshold).toBeDefined();
		for (const point of upperThreshold!.data) {
			expect(point.y).toBe(38.0);
		}
	});

	it('fever threshold lines use distinct dashed styles', () => {
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

		const upperThreshold = datasets.find((d) => d.label === 'Threshold (rectal/ear)');
		expect(upperThreshold!.borderColor).toBe('red');
		expect(upperThreshold!.borderDash).toBeDefined();
		expect(upperThreshold!.borderDash.length).toBeGreaterThan(0);
	});

	it('separates normal and fever data points', () => {
		render(TemperatureChart, { props: { data: mockTemperatureData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { y: number }[] }[] };
		};
		const datasets = config.data.datasets;

		const normalDs = datasets.find((d) => d.label === 'Normal');
		const feverDs = datasets.find((d) => d.label === 'Fever');
		expect(normalDs).toBeDefined();
		expect(feverDs).toBeDefined();

		// 38.5 rectal (>= 38.0) is fever; others are normal
		expect(feverDs!.data.length).toBe(1);
		expect(feverDs!.data[0].y).toBe(38.5);
		// 36.8, 37.2, 37.0 are normal (37.2 axillary is < 37.5 threshold)
		expect(normalDs!.data.length).toBe(3);
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(TemperatureChart, {
			props: { data: mockTemperatureData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data available" when data is empty', () => {
		const { container } = render(TemperatureChart, {
			props: { data: [] }
		});

		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});
});
