import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import TodayDashboard from '$lib/components/TodayDashboard.svelte';

// Mock the API client
vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

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
	stool_color_trend: [
		{ date: '2026-03-13', color: 'pale', color_rating: 3 },
		{ date: '2026-03-14', color: 'pale_yellow', color_rating: 4 },
		{ date: '2026-03-15', color: 'yellow', color_rating: 5 },
		{ date: '2026-03-16', color: 'yellow_green', color_rating: 6 },
		{ date: '2026-03-17', color: 'green', color_rating: 7 },
		{ date: '2026-03-18', color: 'brown_green', color_rating: 8 },
		{ date: '2026-03-19', color: 'brown', color_rating: 9 }
	],
	upcoming_meds: [
		{
			id: 'med-1',
			name: 'Ursodiol',
			dose: '50mg',
			frequency: 'twice_daily',
			schedule_times: ['08:00', '20:00'],
			timezone: 'America/New_York',
			next_dose_at: new Date(Date.now() + 2 * 60 * 60 * 1000 + 15 * 60 * 1000).toISOString()
		},
		{
			id: 'med-2',
			name: 'Vitamin D',
			dose: '400IU',
			frequency: 'once_daily',
			schedule_times: ['09:00'],
			timezone: 'America/New_York',
			next_dose_at: null
		}
	],
	active_alerts: [
		{
			entry_id: 'alert-1',
			alert_type: 'acholic_stool',
			value: 2,
			timestamp: '2026-03-19T10:00:00Z'
		},
		{
			entry_id: 'alert-2',
			alert_type: 'fever',
			method: 'rectal',
			value: 38.5,
			timestamp: '2026-03-19T11:00:00Z'
		},
		{
			entry_id: 'alert-3',
			alert_type: 'jaundice_worsening',
			value: 'severe_limbs_and_trunk',
			timestamp: '2026-03-19T09:00:00Z'
		},
		{
			entry_id: 'alert-4',
			alert_type: 'missed_medication',
			value: 'Ursodiol 50mg',
			timestamp: '2026-03-19T08:00:00Z'
		}
	],
	chart_data_series: {
		feeding_daily: [],
		diaper_daily: [],
		temperature: [],
		weight: [],
		abdomen_girth: [],
		stool_color: [],
		lab_trends: {}
	}
};

describe('TodayDashboard', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockGet.mockResolvedValue(mockDashboardResponse);
		localStorage.clear();
	});

	afterEach(() => {
		localStorage.clear();
	});

	// --- Summary Cards ---

	it('displays total feeds from dashboard data', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		const feeds = await screen.findByText('6');
		expect(feeds).toBeInTheDocument();
		expect(screen.getByText(/feeds/i)).toBeInTheDocument();
	});

	it('displays total calories from dashboard data', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText('480')).toBeInTheDocument();
		expect(screen.getByText(/calories/i)).toBeInTheDocument();
	});

	it('displays total wet diapers from dashboard data', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText('4')).toBeInTheDocument();
		expect(screen.getByText(/wet diapers/i)).toBeInTheDocument();
	});

	it('displays total stools with worst color indicator', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText('2')).toBeInTheDocument();
		expect(screen.getByText(/stools/i)).toBeInTheDocument();
	});

	it('shows a stool color indicator dot on the stools card using worst_stool_color', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText('2');

		const dot = document.querySelector('.stool-color-indicator');
		expect(dot).not.toBeNull();
		// worst_stool_color is 3, which maps to red (#dc2626 / rgb(220, 38, 38))
		expect(dot!.getAttribute('style')).toContain('rgb(220, 38, 38)');
	});

	it('does not show stool color indicator when worst_stool_color is null', async () => {
		mockGet.mockResolvedValue({
			...mockDashboardResponse,
			summary_cards: {
				...mockDashboardResponse.summary_cards,
				worst_stool_color: null
			}
		});

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText('2');

		const dot = document.querySelector('.stool-color-indicator');
		expect(dot).toBeNull();
	});

	it('displays last temperature', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/37\.2/)).toBeInTheDocument();
		expect(screen.getByText(/last temp/i)).toBeInTheDocument();
	});

	it('displays last weight', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/4\.5/)).toBeInTheDocument();
		expect(screen.getByText(/weight/i)).toBeInTheDocument();
	});

	it('displays dashes for null summary values', async () => {
		mockGet.mockResolvedValue({
			...mockDashboardResponse,
			summary_cards: {
				total_feeds: 0,
				total_calories: 0,
				total_wet_diapers: 0,
				total_stools: 0,
				worst_stool_color: null,
				last_temperature: null,
				last_weight: null
			}
		});

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		const dashes = await screen.findAllByText('—');
		expect(dashes.length).toBeGreaterThanOrEqual(2);
	});

	// --- Stool Color Trend ---

	it('renders stool color trend dots for each of the last 7 days', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		// Wait for loading to complete
		await screen.findByText('6');

		const dots = document.querySelectorAll('.stool-trend-dot');
		expect(dots.length).toBe(7);
	});

	it('colors stool trend dots according to color rating', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText('6');

		const dots = document.querySelectorAll('.stool-trend-dot');
		// Rating 3 (pale) should have a warning/red-ish color
		// Rating 9 (brown) should have a green/good color
		expect(dots.length).toBe(7);
		// Each dot should have a data-rating attribute for testing
		expect(dots[0].getAttribute('data-rating')).toBe('3');
		expect(dots[6].getAttribute('data-rating')).toBe('9');
	});

	// --- Upcoming Medications ---

	it('shows upcoming medication names and doses', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/ursodiol/i)).toBeInTheDocument();
		expect(screen.getByText(/50mg/)).toBeInTheDocument();
		expect(screen.getByText(/vitamin d/i)).toBeInTheDocument();
		expect(screen.getByText(/400IU/)).toBeInTheDocument();
	});

	it('shows countdown for medications with next_dose_at', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const medResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T14:15:00Z'
				}
			]
		};
		mockGet.mockResolvedValue(medResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		// 2h15m from now
		expect(await screen.findByText(/in 2 h 15 min/i)).toBeInTheDocument();

		vi.useRealTimers();
	});

	it('shows overdue countdown for past medications', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const pastMedResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T11:15:00Z'
				}
			]
		};
		mockGet.mockResolvedValue(pastMedResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/overdue/i)).toBeInTheDocument();

		vi.useRealTimers();
	});

	it('shows no countdown when next_dose_at is null', async () => {
		const noScheduleResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-2',
					name: 'Vitamin D',
					dose: '400IU',
					frequency: 'once_daily',
					schedule_times: [],
					timezone: 'America/New_York',
					next_dose_at: null
				}
			]
		};
		mockGet.mockResolvedValue(noScheduleResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/vitamin d/i)).toBeInTheDocument();
		expect(screen.getByText(/no schedule/i)).toBeInTheDocument();
	});

	// --- Alert Banners ---

	it('renders alert banners for each active alert type', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/acholic stool/i)).toBeInTheDocument();
		expect(screen.getByText(/fever/i)).toBeInTheDocument();
		expect(screen.getByText(/jaundice/i)).toBeInTheDocument();
		expect(screen.getByText(/missed medication/i)).toBeInTheDocument();
	});

	it('renders dismiss buttons on alert banners', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText(/acholic stool/i);

		const dismissButtons = screen.getAllByRole('button', { name: /dismiss/i });
		expect(dismissButtons.length).toBe(4);
	});

	// --- Alert Dismissal ---

	it('dismissing an alert hides it and persists to localStorage', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText(/acholic stool/i);

		const dismissButtons = screen.getAllByRole('button', { name: /dismiss/i });
		await fireEvent.click(dismissButtons[0]);

		expect(screen.queryByText(/acholic stool/i)).not.toBeInTheDocument();

		const dismissed = JSON.parse(localStorage.getItem('dismissed_alerts') ?? '[]');
		expect(dismissed).toContain('alert-1');
	});

	it('previously dismissed alerts are hidden on mount', async () => {
		localStorage.setItem('dismissed_alerts', JSON.stringify(['alert-1', 'alert-3']));

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		// Wait for data to load
		await screen.findByText(/fever/i);

		expect(screen.queryByText(/acholic stool/i)).not.toBeInTheDocument();
		expect(screen.queryByText(/jaundice/i)).not.toBeInTheDocument();
		// fever and missed medication should still show
		expect(screen.getByText(/fever/i)).toBeInTheDocument();
		expect(screen.getByText(/missed medication/i)).toBeInTheDocument();
	});

	// --- Recovery clears dismissed IDs ---

	it('clears dismissed IDs for alert types no longer present in API response', async () => {
		// Pre-dismiss alert-1 (acholic_stool) and some stale ID
		localStorage.setItem('dismissed_alerts', JSON.stringify(['alert-1', 'stale-id']));

		// API response has no acholic_stool alert anymore (recovered)
		const recoveredResponse = {
			...mockDashboardResponse,
			active_alerts: [
				{
					entry_id: 'alert-2',
					alert_type: 'fever',
					method: 'rectal',
					value: 38.5,
					timestamp: '2026-03-19T11:00:00Z'
				}
			]
		};
		mockGet.mockResolvedValue(recoveredResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText(/fever/i);

		// After recovery, dismissed IDs for resolved alerts should be cleaned up
		// Only IDs that match current active alerts should remain
		const dismissed = JSON.parse(localStorage.getItem('dismissed_alerts') ?? '[]');
		expect(dismissed).not.toContain('alert-1');
		expect(dismissed).not.toContain('stale-id');
	});

	// --- API call ---

	it('calls the dashboard API with the correct baby ID', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-42' } });

		await screen.findByText('6');

		expect(mockGet).toHaveBeenCalledWith('/babies/baby-42/dashboard');
	});

	// --- Loading state ---

	it('shows a loading indicator while fetching', () => {
		mockGet.mockReturnValue(new Promise(() => {})); // never resolves
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	// --- Error state ---

	it('shows error message when API call fails', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));

		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/failed to load/i)).toBeInTheDocument();
	});

	// --- Quick log buttons ---

	it('renders quick log buttons section', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText('6');

		expect(screen.getByRole('button', { name: /feed/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /wet diaper/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /stool/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /temp/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /medication/i })).toBeInTheDocument();
	});
});
