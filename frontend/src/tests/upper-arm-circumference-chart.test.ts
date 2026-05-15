import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import UpperArmCircumferenceChart from '$lib/components/UpperArmCircumferenceChart.svelte';
import { dateTooltipTitle } from '$lib/chart-utils';

const mockData = [
	{ timestamp: '2026-03-01T10:00:00Z', circumference_cm: 12.5 },
	{ timestamp: '2026-03-15T10:00:00Z', circumference_cm: 13.0 }
];

describe('UpperArmCircumferenceChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('configures tooltip title callback to format x value as date', () => {
		render(UpperArmCircumferenceChart, { props: { data: mockData } });
		const config = chartConstructorCalls[0][1] as {
			options: { plugins: { tooltip: { callbacks: { title: unknown } } } };
		};
		expect(config.options.plugins.tooltip.callbacks.title).toBe(dateTooltipTitle);
	});
});
