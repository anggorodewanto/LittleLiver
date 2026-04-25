import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import TodayDashboard from '$lib/components/TodayDashboard.svelte';
import type { Baby } from '$lib/stores/baby';

// Mock the API client
vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

// Mock $app/navigation
vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

import { apiClient } from '$lib/api';
import { goto } from '$app/navigation';

const mockBaby: Baby = {
	id: 'baby-1',
	name: 'Lily',
	sex: 'female',
	date_of_birth: '2026-01-10',
	diagnosis_date: '2026-01-15',
	kasai_date: '2026-01-20'
};

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
		{ date: '2026-03-18', color: 'green', color_rating: 6 },
		{ date: '2026-03-19', color: 'brown', color_rating: 7 }
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
			value: '08:00',
			timestamp: '2026-03-19T08:00:00Z',
			medication_id: 'med-1',
			medication_name: 'Ursodiol'
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
	},
	current_care_plan_phases: []
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
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		const feeds = await screen.findByText('6');
		expect(feeds).toBeInTheDocument();
		expect(screen.getByText(/feeds/i)).toBeInTheDocument();
	});

	it('displays total calories from dashboard data', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText('480')).toBeInTheDocument();
		expect(screen.getByText(/calories/i)).toBeInTheDocument();
	});

	it('displays total wet diapers from dashboard data', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText('4')).toBeInTheDocument();
		expect(screen.getByText(/wet diapers/i)).toBeInTheDocument();
	});

	it('displays total stools with worst color indicator', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText('2')).toBeInTheDocument();
		expect(screen.getByText(/stools/i)).toBeInTheDocument();
	});

	it('shows a stool color indicator dot on the stools card using worst_stool_color', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

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

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('2');

		const dot = document.querySelector('.stool-color-indicator');
		expect(dot).toBeNull();
	});

	it('displays last temperature', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText(/37\.2/)).toBeInTheDocument();
		expect(screen.getByText(/last temp/i)).toBeInTheDocument();
	});

	it('displays last weight', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText(/4\.5/)).toBeInTheDocument();
		expect(screen.getByText(/last weight/i)).toBeInTheDocument();
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

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		const dashes = await screen.findAllByText('—');
		expect(dashes.length).toBeGreaterThanOrEqual(2);
	});

	// --- Stool Color Trend ---

	it('renders stool color trend dots for each of the last 7 days', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		// Wait for loading to complete
		await screen.findByText('6');

		const dots = document.querySelectorAll('.stool-trend-dot');
		expect(dots.length).toBe(7);
	});

	it('colors stool trend dots according to color rating', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');

		const dots = document.querySelectorAll('.stool-trend-dot');
		// Rating 3 (pale) should have a warning/red-ish color
		// Rating 9 (brown) should have a green/good color
		expect(dots.length).toBe(7);
		// Each dot should have a data-rating attribute for testing
		expect(dots[0].getAttribute('data-rating')).toBe('3');
		expect(dots[6].getAttribute('data-rating')).toBe('7');
	});

	// --- Upcoming Medications ---

	it('shows upcoming medication names and doses', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		// Wait for dashboard to load
		await screen.findByText('Upcoming Medications');

		// Ursodiol appears in both alert and upcoming meds, so use getAllByText
		const ursodiolElements = screen.getAllByText(/ursodiol/i);
		expect(ursodiolElements.length).toBeGreaterThanOrEqual(1);
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

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

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
					next_dose_at: '2026-03-19T10:00:00Z' // 2 hours overdue (outside due-now window)
				}
			]
		};
		mockGet.mockResolvedValue(pastMedResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

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

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText(/vitamin d/i)).toBeInTheDocument();
		expect(screen.getByText(/no schedule/i)).toBeInTheDocument();
	});

	// --- Alert Banners ---

	it('renders alert banners for each active alert type', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		// Use exact text matching on the alert-label elements to avoid matching both label and message
		const labels = await screen.findAllByText(
			(content, element) => element?.classList.contains('alert-label') && /acholic stool/i.test(content)
		);
		expect(labels.length).toBeGreaterThanOrEqual(1);

		expect(screen.getAllByText((content, element) =>
			element?.classList.contains('alert-label') && /fever/i.test(content)
		).length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText((content, element) =>
			element?.classList.contains('alert-label') && /jaundice/i.test(content)
		).length).toBeGreaterThanOrEqual(1);
		expect(screen.getAllByText((content, element) =>
			element?.classList.contains('alert-label') && /missed medication/i.test(content)
		).length).toBeGreaterThanOrEqual(1);
	});

	it('renders dismiss buttons on alert banners', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Acholic Stool');

		const dismissButtons = screen.getAllByRole('button', { name: /dismiss/i });
		expect(dismissButtons.length).toBe(4);
	});

	// --- Alert Dismissal ---

	it('dismissing an alert hides it and persists to localStorage', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Acholic Stool');

		const dismissButtons = screen.getAllByRole('button', { name: /dismiss/i });
		await fireEvent.click(dismissButtons[0]);

		expect(screen.queryByText('Acholic Stool')).not.toBeInTheDocument();

		const dismissed = JSON.parse(localStorage.getItem('dismissed_alerts') ?? '[]');
		expect(dismissed).toContain('alert-1');
	});

	it('previously dismissed alerts are hidden on mount', async () => {
		localStorage.setItem('dismissed_alerts', JSON.stringify(['alert-1', 'alert-3']));

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		// Wait for data to load
		await screen.findByText('Fever');

		expect(screen.queryByText('Acholic Stool')).not.toBeInTheDocument();
		expect(screen.queryByText('Jaundice Worsening')).not.toBeInTheDocument();
		// fever and missed medication should still show
		expect(screen.getByText('Fever')).toBeInTheDocument();
		expect(screen.getByText('Missed Medication')).toBeInTheDocument();
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

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Fever');

		// After recovery, dismissed IDs for resolved alerts should be cleaned up
		// Only IDs that match current active alerts should remain
		const dismissed = JSON.parse(localStorage.getItem('dismissed_alerts') ?? '[]');
		expect(dismissed).not.toContain('alert-1');
		expect(dismissed).not.toContain('stale-id');
	});

	// --- Missed medication alert tap-to-log ---

	it('navigates to med log page with medicationId when tapping a missed_medication alert banner', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Missed Medication');

		// Find the missed_medication alert banner and click it
		const missedMedBanner = screen.getByText('Missed Medication').closest('.alert-banner');
		expect(missedMedBanner).not.toBeNull();
		await fireEvent.click(missedMedBanner!);

		expect(goto).toHaveBeenCalledWith('/log/med?medicationId=med-1&scheduled_time=2026-03-19T08%3A00%3A00Z');
	});

	it('shows medication name in missed_medication alert message', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Missed Medication');

		expect(screen.getByText(/Ursodiol dose was missed/)).toBeInTheDocument();
	});

	// --- API call ---

	it('calls the dashboard API with the correct baby ID', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-42', baby: { ...mockBaby, id: 'baby-42' } } });

		await screen.findByText('6');

		expect(mockGet).toHaveBeenCalledWith('/babies/baby-42/dashboard');
	});

	// --- Loading state ---

	it('shows a loading indicator while fetching', () => {
		mockGet.mockReturnValue(new Promise(() => {})); // never resolves
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	// --- Error state ---

	it('shows error message when API call fails', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText(/failed to load/i)).toBeInTheDocument();
	});

	// --- Due Now Banner ---

	it('shows due-now banner for medication due within 30 minutes', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const dueNowResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T12:10:00Z' // 10 min from now
				}
			]
		};
		mockGet.mockResolvedValue(dueNowResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		const banner = await screen.findByTestId('due-now-banner');
		expect(banner).toBeInTheDocument();
		expect(banner.textContent).toContain('Due Now');
		expect(banner.textContent).toContain('Ursodiol');
		expect(banner.textContent).toContain('50mg');

		vi.useRealTimers();
	});

	it('shows due-now banner for overdue medication (within 60 minutes)', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const overdueResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T11:30:00Z' // 30 min overdue
				}
			]
		};
		mockGet.mockResolvedValue(overdueResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText(/due now/i)).toBeInTheDocument();

		vi.useRealTimers();
	});

	it('does not show due-now banner for medication more than 30 min in the future', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const futureResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T14:00:00Z' // 2 hours away
				}
			]
		};
		mockGet.mockResolvedValue(futureResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		// Wait for dashboard to render (use a stable element from the response)
		await screen.findByText('Upcoming Medications');
		expect(screen.queryByText(/due now/i)).not.toBeInTheDocument();

		vi.useRealTimers();
	});

	it('navigates to dose log form when tapping due-now banner', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const dueNowResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T12:10:00Z'
				}
			]
		};
		mockGet.mockResolvedValue(dueNowResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		const banner = await screen.findByTestId('due-now-banner');
		await fireEvent.click(banner);

		expect(goto).toHaveBeenCalledWith('/log/med?medicationId=med-1');

		vi.useRealTimers();
	});

	it('does not show due-now banner for medication more than 60 min overdue', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const longOverdueResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-1',
					name: 'Ursodiol',
					dose: '50mg',
					frequency: 'twice_daily',
					schedule_times: ['08:00', '20:00'],
					timezone: 'America/New_York',
					next_dose_at: '2026-03-19T10:30:00Z' // 90 min overdue
				}
			]
		};
		mockGet.mockResolvedValue(longOverdueResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		// Wait for dashboard to render (use a stable element from the response)
		await screen.findByText('Upcoming Medications');
		expect(screen.queryByText(/due now/i)).not.toBeInTheDocument();

		vi.useRealTimers();
	});

	// --- Every X Days medications ---

	it('shows due-now banner for every_x_days medication due today', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const everyXDaysResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-interval',
					name: 'Vitamin A',
					dose: '5000IU',
					frequency: 'every_x_days',
					schedule_times: [],
					timezone: 'UTC',
					interval_days: 3,
					next_dose_at: '2026-03-19T00:00:00Z' // Today at midnight = due today
				}
			]
		};
		mockGet.mockResolvedValue(everyXDaysResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		const banner = await screen.findByTestId('due-now-banner');
		expect(banner).toBeInTheDocument();
		expect(banner.textContent).toContain('Vitamin A');
		expect(banner.textContent).toContain('Due today');

		vi.useRealTimers();
	});

	it('shows "Due in X days" for every_x_days medication not yet due', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const futureResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-interval',
					name: 'Vitamin A',
					dose: '5000IU',
					frequency: 'every_x_days',
					schedule_times: [],
					timezone: 'UTC',
					interval_days: 3,
					next_dose_at: '2026-03-22T00:00:00Z' // 3 days from now
				}
			]
		};
		mockGet.mockResolvedValue(futureResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText(/vitamin a/i);
		expect(screen.getByText(/due in 3 days/i)).toBeInTheDocument();
		// Should NOT show due-now banner for future
		expect(screen.queryByTestId('due-now-banner')).not.toBeInTheDocument();

		vi.useRealTimers();
	});

	it('shows "Overdue by X days" for overdue every_x_days medication', async () => {
		const now = new Date('2026-03-19T12:00:00Z');
		vi.useFakeTimers({ now });

		const overdueResponse = {
			...mockDashboardResponse,
			upcoming_meds: [
				{
					id: 'med-interval',
					name: 'Vitamin A',
					dose: '5000IU',
					frequency: 'every_x_days',
					schedule_times: [],
					timezone: 'UTC',
					interval_days: 3,
					next_dose_at: '2026-03-17T00:00:00Z' // 2 days ago
				}
			]
		};
		mockGet.mockResolvedValue(overdueResponse);

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		const banner = await screen.findByTestId('due-now-banner');
		expect(banner).toBeInTheDocument();
		expect(banner.textContent).toContain('Overdue by 2 days');

		vi.useRealTimers();
	});

	// --- Quick log buttons ---

	it('renders quick log buttons section', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');

		expect(screen.getByRole('button', { name: /feed/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /wet diaper/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /stool/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /temp/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /medication given/i })).toBeInTheDocument();
	});

	// --- Auto-refresh on visibility change ---

	it('re-fetches dashboard when page becomes visible again', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Missed Medication');

		const callsBefore = mockGet.mock.calls.length;

		// Simulate returning to the page (visibilitychange to visible)
		Object.defineProperty(document, 'visibilityState', {
			value: 'visible',
			writable: true,
			configurable: true
		});
		document.dispatchEvent(new Event('visibilitychange'));

		// Should trigger an additional fetch
		await vi.waitFor(() => {
			expect(mockGet.mock.calls.length).toBeGreaterThan(callsBefore);
		});
	});

	it('removes missed alert after re-fetch when dose has been logged', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Missed Medication');

		// Simulate other parent logging the dose — API no longer returns missed alert
		mockGet.mockResolvedValue({
			...mockDashboardResponse,
			active_alerts: mockDashboardResponse.active_alerts.filter(
				(a) => a.alert_type !== 'missed_medication'
			)
		});

		// Simulate returning to page
		Object.defineProperty(document, 'visibilityState', {
			value: 'visible',
			writable: true,
			configurable: true
		});
		document.dispatchEvent(new Event('visibilitychange'));

		// Missed medication alert should disappear
		await vi.waitFor(() => {
			expect(screen.queryByText('Missed Medication')).not.toBeInTheDocument();
		});
	});

	// --- Quick Glance ---

	it('displays baby sex in the quick glance section', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6'); // wait for dashboard load
		expect(screen.getByText('Female')).toBeInTheDocument();
	});

	it('displays baby birth date formatted', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');
		expect(screen.getByText('10 Jan 2026')).toBeInTheDocument();
	});

	it('displays baby age computed from date of birth', async () => {
		vi.useFakeTimers({ now: new Date('2026-04-02T12:00:00Z') });

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');
		// Born 2026-01-10, now 2026-04-02 = 2 months 23 days
		expect(screen.getByText('2 mo 23 d')).toBeInTheDocument();

		vi.useRealTimers();
	});

	it('displays days since Kasai when kasai_date is set', async () => {
		vi.useFakeTimers({ now: new Date('2026-04-02T12:00:00Z') });

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');
		// Kasai 2026-01-20, now 2026-04-02 = 72 days
		expect(screen.getByText('72 days')).toBeInTheDocument();
		expect(screen.getByText('Since Kasai')).toBeInTheDocument();

		vi.useRealTimers();
	});

	it('hides days since Kasai when kasai_date is null', async () => {
		const babyNoKasai = { ...mockBaby, kasai_date: null };
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: babyNoKasai } });

		await screen.findByText('6');
		expect(screen.queryByText('Since Kasai')).not.toBeInTheDocument();
	});

	it('displays diagnosis date when set', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');
		expect(screen.getByText('15 Jan 2026')).toBeInTheDocument();
		expect(screen.getByText('Diagnosed')).toBeInTheDocument();
	});

	it('hides diagnosis date when null', async () => {
		const babyNoDiagnosis = { ...mockBaby, diagnosis_date: null };
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: babyNoDiagnosis } });

		await screen.findByText('6');
		expect(screen.queryByText('Diagnosed')).not.toBeInTheDocument();
	});
});

describe('TodayDashboard - Care Plans card', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		localStorage.clear();
	});

	afterEach(() => {
		localStorage.clear();
	});

	it('renders care plans card when current_care_plan_phases has entries', async () => {
		mockGet.mockResolvedValue({
			...mockDashboardResponse,
			current_care_plan_phases: [
				{
					plan_id: 'plan-1',
					plan_name: 'Antibiotic Rotation',
					phase_id: 'phase-2',
					label: 'Cefixime',
					ends_on: '2026-06-01',
					days_remaining: 12
				}
			]
		});

		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		expect(await screen.findByText('Care Plans')).toBeInTheDocument();
		expect(screen.getByText('Antibiotic Rotation')).toBeInTheDocument();
		expect(screen.getByText('Cefixime')).toBeInTheDocument();
		expect(screen.getByText('12d left')).toBeInTheDocument();
	});

	it('renders empty-state CTA when phases array is empty', async () => {
		mockGet.mockResolvedValue({ ...mockDashboardResponse, current_care_plan_phases: [] });
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('6');
		expect(screen.getByText('Care Plans')).toBeInTheDocument();
		const cta = screen.getByRole('link', { name: /create your first plan/i });
		expect(cta).toHaveAttribute('href', '/care-plans/new');
	});

	it('omits days-left chip when days_remaining is null', async () => {
		mockGet.mockResolvedValue({
			...mockDashboardResponse,
			current_care_plan_phases: [
				{
					plan_id: 'plan-1',
					plan_name: 'Open Plan',
					phase_id: 'phase-1',
					label: 'Phase A',
					ends_on: null,
					days_remaining: null
				}
			]
		});
		render(TodayDashboard, { props: { babyId: 'baby-1', baby: mockBaby } });

		await screen.findByText('Open Plan');
		expect(screen.queryByText(/d left/)).toBeNull();
	});
});
