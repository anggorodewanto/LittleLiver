import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import TodayDashboard from '$lib/components/TodayDashboard.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

import { apiClient } from '$lib/api';
import { goto } from '$app/navigation';

const mockDashboardResponse = {
	summary_cards: {
		total_feeds: 6,
		total_calories: 480,
		total_wet_diapers: 4,
		total_stools: 2,
		worst_stool_color: null,
		last_temperature: null,
		last_weight: null
	},
	stool_color_trend: [],
	upcoming_meds: [],
	active_alerts: [],
	chart_data_series: {}
};

describe('Medication Given quick-log button on TodayDashboard', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockGet.mockResolvedValue(mockDashboardResponse);
		localStorage.clear();
	});

	afterEach(() => {
		localStorage.clear();
	});

	it('renders a Medication Given quick-log button', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText('6');
		expect(screen.getByRole('button', { name: /medication given/i })).toBeInTheDocument();
	});

	it('Medication Given button navigates to dose logging form', async () => {
		render(TodayDashboard, { props: { babyId: 'baby-1' } });

		await screen.findByText('6');
		await fireEvent.click(screen.getByRole('button', { name: /medication given/i }));

		expect(goto).toHaveBeenCalledWith('/log/med');
	});
});
