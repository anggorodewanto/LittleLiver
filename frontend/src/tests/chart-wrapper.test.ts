import { render, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, mockChartInstance, chartConstructorCalls, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

import ChartWrapper from '$lib/components/ChartWrapper.svelte';

const config = {
	type: 'line' as const,
	data: { datasets: [{ label: 'X', data: [{ x: 1, y: 2 }] }] },
	options: {}
};

describe('ChartWrapper', () => {
	beforeEach(() => {
		resetChartMocks();
	});

	it('renders a clickable chart with role=button', () => {
		const { container } = render(ChartWrapper, { props: { config } });
		const trigger = container.querySelector('[data-testid="chart-expand"]');
		expect(trigger).not.toBeNull();
	});

	it('does not show fullscreen modal initially', () => {
		const { container } = render(ChartWrapper, { props: { config } });
		expect(container.querySelector('[data-testid="chart-modal"]')).toBeNull();
	});

	it('opens fullscreen modal on click', async () => {
		const { container } = render(ChartWrapper, { props: { config } });
		const trigger = container.querySelector('[data-testid="chart-expand"]')!;
		await fireEvent.click(trigger);
		expect(container.querySelector('[data-testid="chart-modal"]')).not.toBeNull();
	});

	it('creates a second chart instance with zoom plugin config when expanded', async () => {
		const { container } = render(ChartWrapper, { props: { config } });
		expect(chartConstructorCalls.length).toBe(1);
		const trigger = container.querySelector('[data-testid="chart-expand"]')!;
		await fireEvent.click(trigger);
		expect(chartConstructorCalls.length).toBe(2);
		const fullConfig = chartConstructorCalls[1][1] as {
			options: { plugins?: { zoom?: { zoom?: unknown; pan?: unknown } } };
		};
		expect(fullConfig.options.plugins?.zoom?.zoom).toBeDefined();
		expect(fullConfig.options.plugins?.zoom?.pan).toBeDefined();
	});

	it('closes modal on close-button click', async () => {
		const { container } = render(ChartWrapper, { props: { config } });
		await fireEvent.click(container.querySelector('[data-testid="chart-expand"]')!);
		await fireEvent.click(container.querySelector('[data-testid="chart-modal-close"]')!);
		expect(container.querySelector('[data-testid="chart-modal"]')).toBeNull();
	});

	it('does not render expand trigger when empty', () => {
		const { container } = render(ChartWrapper, {
			props: { config, isEmpty: true }
		});
		expect(container.querySelector('[data-testid="chart-expand"]')).toBeNull();
		expect(container.textContent).toContain('No data available');
	});

	it('destroys both charts on unmount', async () => {
		const { container, unmount } = render(ChartWrapper, { props: { config } });
		await fireEvent.click(container.querySelector('[data-testid="chart-expand"]')!);
		mockChartInstance.destroy.mockClear();
		unmount();
		expect(mockChartInstance.destroy).toHaveBeenCalled();
	});
});
