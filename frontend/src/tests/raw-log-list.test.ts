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
import { formatDateISO } from '$lib/datetime';
import RawLogList from '$lib/components/RawLogList.svelte';

function mockAllEndpoints(overrides: Record<string, unknown> = {}) {
	vi.mocked(apiClient.get).mockImplementation(async (url: string) => {
		if (url.includes('/medications')) {
			return overrides.medications ?? [];
		}
		const endpoint = Object.keys(overrides).find((k) => url.includes(`/${k}?`));
		if (endpoint) {
			return overrides[endpoint];
		}
		return { data: [], next_cursor: null };
	});
}

describe('RawLogList', () => {
	beforeEach(() => {
		vi.mocked(apiClient.get).mockReset();
		vi.mocked(apiClient.del).mockReset();
	});

	it('defaults date range to today', () => {
		mockAllEndpoints();

		render(RawLogList, { props: { babyId: 'baby-1' } });

		const today = formatDateISO(new Date());
		const fromInput = screen.getByLabelText('From date') as HTMLInputElement;
		const toInput = screen.getByLabelText('To date') as HTMLInputElement;
		expect(fromInput.value).toBe(today);
		expect(toInput.value).toBe(today);
	});

	it('shows loading state while fetching', () => {
		vi.mocked(apiClient.get).mockReturnValue(new Promise(() => {}));

		render(RawLogList, { props: { babyId: 'baby-1' } });

		expect(screen.getByText('Loading...')).toBeInTheDocument();
	});

	it('fetches all log types and renders entries', async () => {
		mockAllEndpoints({
			feedings: {
				data: [
					{ id: 'f1', timestamp: '2026-03-31T14:00:00Z', feed_type: 'formula', volume_ml: 120 }
				],
				next_cursor: null
			},
			stools: {
				data: [
					{ id: 's1', timestamp: '2026-03-31T10:00:00Z', color_rating: 4, consistency: 'soft' }
				],
				next_cursor: null
			}
		});

		render(RawLogList, { props: { babyId: 'baby-1' } });

		// 120 mL appears in both the row and the total, so use getAllByText
		const matches = await screen.findAllByText(/120\s*mL/);
		expect(matches.length).toBeGreaterThanOrEqual(1);
		expect(screen.getByText(/4\/7/)).toBeInTheDocument();
	});

	it('shows empty state when no entries from any type', async () => {
		mockAllEndpoints();

		render(RawLogList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText('No entries found.')).toBeInTheDocument();
	});

	it('shows error state on fetch failure', async () => {
		vi.mocked(apiClient.get).mockRejectedValue(new Error('API error: 500'));

		render(RawLogList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText('API error: 500')).toBeInTheDocument();
	});

	it('delete removes entry from list', async () => {
		mockAllEndpoints({
			feedings: {
				data: [
					{ id: 'f1', timestamp: '2026-03-31T14:00:00Z', feed_type: 'formula', volume_ml: 120 },
					{ id: 'f2', timestamp: '2026-03-31T10:00:00Z', feed_type: 'breast_milk', volume_ml: 90 }
				],
				next_cursor: null
			}
		});
		vi.mocked(apiClient.del).mockResolvedValue(undefined);

		render(RawLogList, { props: { babyId: 'baby-1' } });

		await screen.findByText(/120\s*mL/);

		const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
		await fireEvent.click(deleteButtons[0]);

		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(apiClient.del).toHaveBeenCalledWith('/babies/baby-1/feedings/f1');
		});

		await waitFor(() => {
			// After deleting f1 (120 mL), only the total "Feeding: 90 mL" should contain mL for 90
			expect(screen.queryByText(/120\s*mL/)).not.toBeInTheDocument();
		});

		// 90 mL appears in both the row and the feeding total
		const remaining = screen.getAllByText(/90\s*mL/);
		expect(remaining.length).toBeGreaterThanOrEqual(1);
	});

	it('shows feeding total volume', async () => {
		mockAllEndpoints({
			feedings: {
				data: [
					{ id: 'f1', timestamp: '2026-03-31T14:00:00Z', feed_type: 'formula', volume_ml: 120 },
					{ id: 'f2', timestamp: '2026-03-31T10:00:00Z', feed_type: 'formula', volume_ml: 80 }
				],
				next_cursor: null
			}
		});

		render(RawLogList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/Feeding:\s*200\s*mL/)).toBeInTheDocument();
	});

	it('shows fluid output total volume', async () => {
		mockAllEndpoints({
			'fluid-log': {
				data: [
					{ id: 'fl1', timestamp: '2026-03-31T14:00:00Z', direction: 'output', method: 'drain', volume_ml: 50 },
					{ id: 'fl2', timestamp: '2026-03-31T10:00:00Z', direction: 'output', method: 'drain', volume_ml: 30 }
				],
				next_cursor: null
			}
		});

		render(RawLogList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/Output:\s*80\s*mL/)).toBeInTheDocument();
	});

	it('does not count fluid intake in output total', async () => {
		mockAllEndpoints({
			'fluid-log': {
				data: [
					{ id: 'fl1', timestamp: '2026-03-31T14:00:00Z', direction: 'intake', method: 'oral', volume_ml: 100 },
					{ id: 'fl2', timestamp: '2026-03-31T10:00:00Z', direction: 'output', method: 'drain', volume_ml: 30 }
				],
				next_cursor: null
			}
		});

		render(RawLogList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/Output:\s*30\s*mL/)).toBeInTheDocument();
	});
});
