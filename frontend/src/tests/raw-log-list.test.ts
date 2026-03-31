import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn(),
		del: vi.fn()
	}
}));

import { apiClient } from '$lib/api';
import RawLogList from '$lib/components/RawLogList.svelte';
import type { LogTypeConfig } from '$lib/types/logs';

const feedingType: LogTypeConfig = {
	key: 'feeding',
	label: 'Feedings',
	endpoint: 'feedings',
	metricParam: 'feeding'
};

const mockResponse = {
	data: [
		{
			id: 'entry-1',
			timestamp: '2026-03-19T14:00:00Z',
			feed_type: 'formula',
			volume_ml: 120
		},
		{
			id: 'entry-2',
			timestamp: '2026-03-19T10:00:00Z',
			feed_type: 'breast_milk',
			volume_ml: 90
		}
	],
	next_cursor: null
};

describe('RawLogList', () => {
	beforeEach(() => {
		vi.mocked(apiClient.get).mockReset();
		vi.mocked(apiClient.del).mockReset();
	});

	it('shows loading state while fetching', () => {
		// Never resolve so it stays loading
		vi.mocked(apiClient.get).mockReturnValue(new Promise(() => {}));

		render(RawLogList, { props: { babyId: 'baby-1', logType: feedingType } });

		expect(screen.getByText('Loading...')).toBeInTheDocument();
	});

	it('renders entries after fetch', async () => {
		vi.mocked(apiClient.get).mockResolvedValue(mockResponse);

		render(RawLogList, { props: { babyId: 'baby-1', logType: feedingType } });

		expect(await screen.findByText(/120\s*mL/)).toBeInTheDocument();
		expect(screen.getByText(/90\s*mL/)).toBeInTheDocument();
	});

	it('shows empty state when no entries', async () => {
		vi.mocked(apiClient.get).mockResolvedValue({ data: [], next_cursor: null });

		render(RawLogList, { props: { babyId: 'baby-1', logType: feedingType } });

		expect(await screen.findByText('No entries found.')).toBeInTheDocument();
	});

	it('shows error state on fetch failure', async () => {
		vi.mocked(apiClient.get).mockRejectedValue(new Error('API error: 500'));

		render(RawLogList, { props: { babyId: 'baby-1', logType: feedingType } });

		expect(await screen.findByText('API error: 500')).toBeInTheDocument();
	});

	it('Load More button appears when next_cursor is present', async () => {
		vi.mocked(apiClient.get).mockResolvedValue({
			data: [
				{
					id: 'entry-1',
					timestamp: '2026-03-19T14:00:00Z',
					feed_type: 'formula',
					volume_ml: 120
				}
			],
			next_cursor: 'cursor-abc'
		});

		render(RawLogList, { props: { babyId: 'baby-1', logType: feedingType } });

		expect(await screen.findByText('Load More')).toBeInTheDocument();
	});

	it('delete removes entry from list', async () => {
		vi.mocked(apiClient.get).mockResolvedValue(mockResponse);
		vi.mocked(apiClient.del).mockResolvedValue(undefined);

		render(RawLogList, { props: { babyId: 'baby-1', logType: feedingType } });

		// Wait for entries to render
		await screen.findByText(/120\s*mL/);

		// Click delete on first entry
		const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
		await fireEvent.click(deleteButtons[0]);

		// Confirm deletion
		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(apiClient.del).toHaveBeenCalledWith('/babies/baby-1/feedings/entry-1');
		});

		await waitFor(() => {
			expect(screen.queryByText(/120\s*mL/)).not.toBeInTheDocument();
		});

		// Second entry should still be there
		expect(screen.getByText(/90\s*mL/)).toBeInTheDocument();
	});
});
