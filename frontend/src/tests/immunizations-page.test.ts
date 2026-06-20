import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

import ImmunizationsPage from '../routes/immunizations/+page.svelte';
import { activeBaby, _resetBabyStores } from '$lib/stores/baby';
import { apiClient } from '$lib/api';

describe('/immunizations page', () => {
	beforeEach(() => {
		vi.clearAllMocks();
		_resetBabyStores();
		(apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ slots: [] });
	});

	afterEach(() => {
		_resetBabyStores();
	});

	it('shows "No baby selected" when no active baby', () => {
		render(ImmunizationsPage);

		expect(screen.getByText(/no baby selected/i)).toBeInTheDocument();
	});

	it('renders the heading and ImmunizationView when a baby is active', async () => {
		activeBaby.set({
			id: 'b1',
			name: 'Alice',
			date_of_birth: '2025-06-01',
			sex: 'female',
			diagnosis_date: null,
			kasai_date: null
		});

		render(ImmunizationsPage);

		expect(screen.getByRole('heading', { name: /immunizations/i })).toBeInTheDocument();
		// ImmunizationView shows a loading state initially, meaning it rendered.
		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	it('fetches the schedule for the active baby', async () => {
		activeBaby.set({
			id: 'b1',
			name: 'Alice',
			date_of_birth: '2025-06-01',
			sex: 'female',
			diagnosis_date: null,
			kasai_date: null
		});

		render(ImmunizationsPage);

		expect(apiClient.get).toHaveBeenCalledWith('/babies/b1/immunizations/schedule');
	});
});
