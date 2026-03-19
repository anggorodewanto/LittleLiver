import { vi } from 'vitest';

export const mockChartInstance = {
	destroy: vi.fn(),
	update: vi.fn()
};

export const chartConstructorCalls: unknown[][] = [];

export class MockChart {
	static register = vi.fn();

	constructor(...args: unknown[]) {
		chartConstructorCalls.push(args);
		Object.assign(this, mockChartInstance);
	}

	destroy = mockChartInstance.destroy;
	update = mockChartInstance.update;
}

export function resetChartMocks(): void {
	mockChartInstance.destroy.mockClear();
	mockChartInstance.update.mockClear();
	MockChart.register.mockClear();
	chartConstructorCalls.length = 0;
}
