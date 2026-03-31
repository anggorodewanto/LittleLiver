import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn().mockResolvedValue({ data: [], next_cursor: null }),
		del: vi.fn()
	}
}));

import { activeBaby, _resetBabyStores } from '$lib/stores/baby';
import LogsPage from '../routes/logs/+page.svelte';

const mockBaby = {
	id: 'baby-1',
	name: 'Alice',
	date_of_birth: '2025-06-01',
	sex: 'female' as const,
	diagnosis_date: null,
	kasai_date: null
};

describe('Logs Page', () => {
	beforeEach(() => {
		_resetBabyStores();
	});

	afterEach(() => {
		_resetBabyStores();
	});

	it('shows "No baby selected" when no active baby', () => {
		render(LogsPage);

		expect(screen.getByText(/no baby selected/i)).toBeInTheDocument();
	});

	it('renders heading and log list when baby is active', async () => {
		activeBaby.set(mockBaby);

		render(LogsPage);

		expect(screen.getByRole('heading', { name: /logs/i })).toBeInTheDocument();
		// The RawLogList should render and show empty state since API returns empty data
		expect(await screen.findByText('No entries found.')).toBeInTheDocument();
	});
});
