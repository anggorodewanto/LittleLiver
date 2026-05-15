import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import HeadCircumferenceChart from '$lib/components/HeadCircumferenceChart.svelte';
import { dateTooltipTitle } from '$lib/chart-utils';

const mockHcData = [
	{ timestamp: '2026-03-01T10:00:00Z', circumference_cm: 35.0 },
	{ timestamp: '2026-03-15T10:00:00Z', circumference_cm: 36.4 }
];

const mockPercentiles = {
	p3: [
		{ age_days: 0, value: 31.5 },
		{ age_days: 30, value: 34.0 },
		{ age_days: 60, value: 36.0 }
	],
	p15: [
		{ age_days: 0, value: 32.4 },
		{ age_days: 30, value: 35.0 },
		{ age_days: 60, value: 37.0 }
	],
	p50: [
		{ age_days: 0, value: 34.5 },
		{ age_days: 30, value: 37.3 },
		{ age_days: 60, value: 39.2 }
	],
	p85: [
		{ age_days: 0, value: 36.0 },
		{ age_days: 30, value: 38.8 },
		{ age_days: 60, value: 40.6 }
	],
	p97: [
		{ age_days: 0, value: 37.0 },
		{ age_days: 30, value: 39.8 },
		{ age_days: 60, value: 41.6 }
	]
};

describe('HeadCircumferenceChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(HeadCircumferenceChart, {
			props: { data: mockHcData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		expect(container.querySelector('canvas')).not.toBeNull();
	});

	it('uses date timestamps for the main data x-axis', () => {
		render(HeadCircumferenceChart, {
			props: { data: mockHcData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { x: number; y: number }[] }[] };
		};
		const main = config.data.datasets.find((d) => d.label === 'Head Circumference')!;
		expect(main.data[0].x).toBe(new Date('2026-03-01T10:00:00Z').getTime());
		expect(main.data[1].x).toBe(new Date('2026-03-15T10:00:00Z').getTime());
	});

	it('converts percentile age_days to date timestamps via dateOfBirth', () => {
		const dob = '2026-01-15';
		render(HeadCircumferenceChart, {
			props: { data: mockHcData, percentiles: mockPercentiles, dateOfBirth: dob }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string; data: { x: number; y: number }[] }[] };
		};
		const p50 = config.data.datasets.find((d) => d.label === '50th')!;
		const dobMs = new Date(dob).getTime();
		const dayMs = 24 * 60 * 60 * 1000;
		expect(p50.data[0].x).toBe(dobMs);
		expect(p50.data[1].x).toBe(dobMs + 30 * dayMs);
		expect(p50.data[2].x).toBe(dobMs + 60 * dayMs);
	});

	it('includes 5 percentile band datasets', () => {
		render(HeadCircumferenceChart, {
			props: { data: mockHcData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { label: string }[] };
		};
		const labels = config.data.datasets.map((d) => d.label);
		expect(labels).toEqual(['Head Circumference', '3rd', '15th', '50th', '85th', '97th']);
	});

	it('destroys chart on unmount', () => {
		const { unmount } = render(HeadCircumferenceChart, {
			props: { data: mockHcData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		unmount();
		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data available" when data is empty', () => {
		const { container } = render(HeadCircumferenceChart, {
			props: { data: [], percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		expect(container.textContent).toContain('No data available');
	});

	it('configures tooltip title callback to format x value as date', () => {
		render(HeadCircumferenceChart, {
			props: { data: mockHcData, percentiles: mockPercentiles, dateOfBirth: '2026-01-15' }
		});
		const config = chartConstructorCalls[0][1] as {
			options: { plugins: { tooltip: { callbacks: { title: unknown } } } };
		};
		expect(config.options.plugins.tooltip.callbacks.title).toBe(dateTooltipTitle);
	});
});
