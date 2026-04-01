import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
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

import LabsPage from '../routes/labs/+page.svelte';
import { activeBaby, _resetBabyStores } from '$lib/stores/baby';
import { apiClient } from '$lib/api';

describe('/labs page', () => {
	beforeEach(() => {
		resetChartMocks();
		vi.clearAllMocks();
		_resetBabyStores();
		(apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ data: [], next_cursor: null });
	});

	afterEach(() => {
		_resetBabyStores();
	});

	it('shows "No baby selected" when no active baby', () => {
		render(LabsPage);

		expect(screen.getByText(/no baby selected/i)).toBeInTheDocument();
	});

	it('renders LabResultsView when baby is active', async () => {
		activeBaby.set({
			id: 'b1',
			name: 'Alice',
			date_of_birth: '2025-06-01',
			sex: 'female',
			diagnosis_date: null,
			kasai_date: null
		});

		render(LabsPage);

		// LabResultsView shows loading initially, which means it rendered
		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});
});
