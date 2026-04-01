import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

vi.mock('$lib/chart-setup', () => ({}));

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

import TrendsView from '$lib/components/TrendsView.svelte';
import { apiClient } from '$lib/api';

const mockDashboardResponse = {
	summary_cards: {
		total_feeds: 6,
		total_calories: 480,
		total_wet_diapers: 4,
		total_stools: 2,
		worst_stool_color: 3,
		last_temperature: 37.2,
		last_weight: 4.5
	},
	stool_color_trend: [],
	upcoming_meds: [],
	active_alerts: [],
	chart_data_series: {
		feeding_daily: [],
		diaper_daily: [],
		temperature: [
			{ timestamp: '2026-03-13T08:00:00Z', value: 36.8, method: 'rectal' },
			{ timestamp: '2026-03-14T09:00:00Z', value: 37.2, method: 'axillary' }
		],
		weight: [
			{ timestamp: '2026-03-13T10:00:00Z', weight_kg: 4.2, measurement_source: 'home_scale' }
		],
		abdomen_girth: [],
		stool_color: [
			{ timestamp: '2026-03-13T08:00:00Z', color_score: 5 },
			{ timestamp: '2026-03-14T09:00:00Z', color_score: 3 }
		],
		head_circumference: [],
		upper_arm_circumference: [],
		lab_trends: {}
	}
};

const mockPercentileResponse = {
	percentiles: {
		p3: [{ age_days: 0, weight_kg: 2.5 }],
		p15: [{ age_days: 0, weight_kg: 2.8 }],
		p50: [{ age_days: 0, weight_kg: 3.2 }],
		p85: [{ age_days: 0, weight_kg: 3.6 }],
		p97: [{ age_days: 0, weight_kg: 3.9 }]
	}
};

describe('TrendsView', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		resetChartMocks();
		vi.clearAllMocks();
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockGet.mockImplementation((path: string) => {
			if (path.includes('/who/percentiles')) {
				return Promise.resolve(mockPercentileResponse);
			}
			return Promise.resolve(mockDashboardResponse);
		});
	});

	it('renders the date range selector', async () => {
		render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		await screen.findByRole('button', { name: '7d' });

		expect(screen.getByRole('button', { name: '7d' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '30d' })).toBeInTheDocument();
	});

	it('date range selector updates API call params', async () => {
		render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		// Wait for initial data load
		await screen.findByText(/stool color/i);

		// Initial call should use 7d range
		expect(mockGet).toHaveBeenCalledWith(
			expect.stringContaining('/babies/baby-1/dashboard?')
		);

		// Click 30d
		await fireEvent.click(screen.getByRole('button', { name: '30d' }));

		// Wait for second fetch
		await screen.findByText(/stool color/i);

		// Should call API again with 30d range
		const calls = mockGet.mock.calls.filter((c: string[]) =>
			c[0].includes('/babies/baby-1/dashboard')
		);
		expect(calls.length).toBeGreaterThanOrEqual(2);

		// The second dashboard call should have from/to params
		const lastDashboardCall = calls[calls.length - 1][0] as string;
		expect(lastDashboardCall).toContain('from=');
		expect(lastDashboardCall).toContain('to=');
	});

	it('renders canvas elements only for charts with data', async () => {
		const { container } = render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		// Wait for data to load
		await screen.findByText(/stool color/i);

		// Only temperature, weight, and stool_color have data; others show "No data"
		const canvases = container.querySelectorAll('canvas');
		expect(canvases.length).toBe(3);
	});

	it('fetches WHO percentile data alongside dashboard data', async () => {
		render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		await screen.findByText(/stool color/i);

		expect(mockGet).toHaveBeenCalledWith(
			expect.stringContaining('/who/percentiles?')
		);
	});

	it('shows chart section headings for all nine charts', async () => {
		render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		expect(await screen.findByText(/stool color/i)).toBeInTheDocument();
		expect(screen.getByText(/weight/i)).toBeInTheDocument();
		expect(screen.getByText(/temperature/i)).toBeInTheDocument();
		expect(screen.getByText(/abdomen girth/i)).toBeInTheDocument();
		expect(screen.getByText(/head circumference/i)).toBeInTheDocument();
		expect(screen.getByText(/upper arm circumference/i)).toBeInTheDocument();
		expect(screen.getByText(/feeding/i)).toBeInTheDocument();
		expect(screen.getByText(/diaper/i)).toBeInTheDocument();
		expect(screen.getByText(/lab trends/i)).toBeInTheDocument();
	});

	it('shows loading state while data is being fetched', () => {
		mockGet.mockReturnValue(new Promise(() => {})); // never resolves
		render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	it('shows error state when API call fails', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));

		render(TrendsView, {
			props: { babyId: 'baby-1', sex: 'female', dateOfBirth: '2026-01-15' }
		});

		expect(await screen.findByText(/failed to load/i)).toBeInTheDocument();
	});
});
