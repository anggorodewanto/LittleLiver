import { render } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import StoolColorChart from '$lib/components/StoolColorChart.svelte';

const mockStoolData = [
	{ timestamp: '2026-03-13T08:00:00Z', color_score: 3 },
	{ timestamp: '2026-03-14T09:00:00Z', color_score: 5 },
	{ timestamp: '2026-03-15T10:00:00Z', color_score: 7 },
	{ timestamp: '2026-03-16T08:30:00Z', color_score: 2 },
	{ timestamp: '2026-03-17T11:00:00Z', color_score: 6 }
];

describe('StoolColorChart', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a canvas element', () => {
		const { container } = render(StoolColorChart, { props: { data: mockStoolData } });

		const canvas = container.querySelector('canvas');
		expect(canvas).not.toBeNull();
	});

	it('creates a Chart.js scatter chart with stool data', () => {
		render(StoolColorChart, { props: { data: mockStoolData } });

		expect(chartConstructorCalls.length).toBe(1);
		const config = chartConstructorCalls[0][1] as { type: string };
		expect(config.type).toBe('scatter');
	});

	it('uses correct color coding for each stool color score', () => {
		render(StoolColorChart, { props: { data: mockStoolData } });

		const config = chartConstructorCalls[0][1] as {
			data: { datasets: { pointBackgroundColor: string[] }[] };
		};
		const pointColors = config.data.datasets[0].pointBackgroundColor;

		expect(pointColors).toHaveLength(5);
		// color_score 3 = Pale Yellow (#FFFACD)
		expect(pointColors[0]).toBe('#FFFACD');
		// color_score 5 = Light Green (#90EE90)
		expect(pointColors[1]).toBe('#90EE90');
		// color_score 7 = Brown (#8B4513)
		expect(pointColors[2]).toBe('#8B4513');
		// color_score 2 = Clay (#D2B48C)
		expect(pointColors[3]).toBe('#D2B48C');
		// color_score 6 = Green (#228B22)
		expect(pointColors[4]).toBe('#228B22');
	});

	it('configures x-axis ticks to format dates and y-axis ticks to show color labels', () => {
		render(StoolColorChart, { props: { data: mockStoolData } });

		const config = chartConstructorCalls[0][1] as {
			options: {
				scales: {
					x: { ticks: { callback: (value: number) => string } };
					y: { ticks: { callback: (value: number) => string } };
				};
			};
		};

		// Test x-axis callback formats date
		const xCallback = config.options.scales.x.ticks.callback;
		const result = xCallback(new Date('2026-03-13').getTime());
		expect(result).toBeTruthy();

		// Test y-axis callback returns label for known rating
		const yCallback = config.options.scales.y.ticks.callback;
		expect(yCallback(1)).toBe('White');
		expect(yCallback(7)).toBe('Brown');
		// Unknown rating returns empty
		expect(yCallback(0)).toBe('');
	});

	it('destroys chart on component unmount', () => {
		const { unmount } = render(StoolColorChart, { props: { data: mockStoolData } });

		unmount();

		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});

	it('shows "No data available" when data is empty', () => {
		const { container } = render(StoolColorChart, { props: { data: [] } });

		expect(container.textContent).toContain('No data available');
		expect(chartConstructorCalls.length).toBe(0);
	});

	it('y-axis range is 0.5 to 7.5 for proper padding around valid scores 1-7', () => {
		render(StoolColorChart, { props: { data: mockStoolData } });

		const config = chartConstructorCalls[0][1] as {
			options: { scales: { y: { min: number; max: number } } };
		};
		expect(config.options.scales.y.min).toBe(0.5);
		expect(config.options.scales.y.max).toBe(7.5);
	});
});
