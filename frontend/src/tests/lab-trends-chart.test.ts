import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import LabTrendsChart from '$lib/components/LabTrendsChart.svelte';

const mockLabData: Record<string, { timestamp: string; test_name: string; value: string; unit: string }[]> = {
	total_bilirubin: [
		{ timestamp: '2026-03-01T10:00:00Z', test_name: 'total_bilirubin', value: '3.2', unit: 'mg/dL' },
		{ timestamp: '2026-03-08T10:00:00Z', test_name: 'total_bilirubin', value: '2.5', unit: 'mg/dL' },
		{ timestamp: '2026-03-15T10:00:00Z', test_name: 'total_bilirubin', value: '1.8', unit: 'mg/dL' }
	],
	ALT: [
		{ timestamp: '2026-03-01T10:00:00Z', test_name: 'ALT', value: '120', unit: 'U/L' },
		{ timestamp: '2026-03-08T10:00:00Z', test_name: 'ALT', value: '95', unit: 'U/L' }
	],
	GGT: [
		{ timestamp: '2026-03-01T10:00:00Z', test_name: 'GGT', value: '450', unit: 'U/L' },
		{ timestamp: '2026-03-15T10:00:00Z', test_name: 'GGT', value: '380', unit: 'U/L' }
	]
};

describe('LabTrendsChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(LabTrendsChart, {
			props: { data: mockLabData }
		});

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js line chart', () => {
		render(LabTrendsChart, { props: { data: mockLabData } });

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('line');
	});

	it('creates one dataset per test_name', () => {
		render(LabTrendsChart, { props: { data: mockLabData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		const labels = config.data.datasets
			.filter((d) => !d.label.includes('Normal'))
			.map((d) => d.label);
		expect(labels).toContain('total_bilirubin (mg/dL)');
		expect(labels).toContain('ALT (U/L)');
		expect(labels).toContain('GGT (U/L)');
		expect(labels).toHaveLength(3);
	});

	it('maps data points with x as timestamp and y as parsed numeric value for each test', () => {
		render(LabTrendsChart, { props: { data: mockLabData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { x: number; y: number }[] }[] };
		};
		const bilirubinDs = config.data.datasets.find((d) => d.label.includes('total_bilirubin'));
		expect(bilirubinDs).toBeDefined();
		expect(bilirubinDs!.data).toHaveLength(3);
		expect(bilirubinDs!.data[0].y).toBe(3.2);
		expect(bilirubinDs!.data[2].y).toBe(1.8);
	});

	it('includes normal range shading dataset for total_bilirubin', () => {
		render(LabTrendsChart, { props: { data: mockLabData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; fill: unknown }[] };
		};
		const normalDs = config.data.datasets.find((d) => d.label.includes('Normal') && d.label.includes('total_bilirubin'));
		expect(normalDs).toBeDefined();
		expect(normalDs!.fill).toBeTruthy();
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(LabTrendsChart, {
			props: { data: mockLabData }
		});

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data" message instead of chart when data is empty', () => {
		const { container } = render(LabTrendsChart, {
			props: { data: {} }
		});

		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});

	it('uses distinct colors for different test names', () => {
		render(LabTrendsChart, { props: { data: mockLabData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; borderColor: string }[] };
		};
		const dataDatasets = config.data.datasets.filter((d) => !d.label.includes('Normal'));
		const colors = dataDatasets.map((d) => d.borderColor);
		const uniqueColors = new Set(colors);
		expect(uniqueColors.size).toBe(colors.length);
	});

	it('filters out non-numeric lab values', () => {
		const dataWithNonNumeric: Record<string, { timestamp: string; test_name: string; value: string; unit: string }[]> = {
			total_bilirubin: [
				{ timestamp: '2026-03-01T10:00:00Z', test_name: 'total_bilirubin', value: '3.2', unit: 'mg/dL' },
				{ timestamp: '2026-03-08T10:00:00Z', test_name: 'total_bilirubin', value: 'pending', unit: 'mg/dL' },
				{ timestamp: '2026-03-15T10:00:00Z', test_name: 'total_bilirubin', value: '1.8', unit: 'mg/dL' }
			]
		};
		render(LabTrendsChart, { props: { data: dataWithNonNumeric } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { x: number; y: number }[] }[] };
		};
		const bilirubinDs = config.data.datasets.find((d) => d.label.includes('total_bilirubin') && !d.label.includes('Normal'));
		expect(bilirubinDs).toBeDefined();
		expect(bilirubinDs!.data).toHaveLength(2);
		expect(bilirubinDs!.data[0].y).toBe(3.2);
		expect(bilirubinDs!.data[1].y).toBe(1.8);
	});

	it('includes units in dataset labels', () => {
		render(LabTrendsChart, { props: { data: mockLabData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		const labels = config.data.datasets
			.filter((d) => !d.label.includes('Normal'))
			.map((d) => d.label);
		expect(labels).toContain('total_bilirubin (mg/dL)');
		expect(labels).toContain('ALT (U/L)');
		expect(labels).toContain('GGT (U/L)');
	});

});
