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
import { LOG_TYPES } from '$lib/types/logs';
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

	it('renders type selector with all log types', () => {
		activeBaby.set(mockBaby);

		render(LogsPage);

		const select = screen.getByLabelText(/log type/i);
		expect(select).toBeInTheDocument();

		for (const lt of LOG_TYPES) {
			expect(screen.getByRole('option', { name: lt.label })).toBeInTheDocument();
		}
	});

	it('renders RawLogList for default type (feeding)', async () => {
		activeBaby.set(mockBaby);

		render(LogsPage);

		// The RawLogList should render and show empty state since API returns empty data
		expect(await screen.findByText('No entries found.')).toBeInTheDocument();
	});
});
