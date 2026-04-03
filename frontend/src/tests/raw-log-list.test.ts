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

	it('fetches all log types and renders grouped by type with headings', async () => {
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

		// Should show type headings
		expect(await screen.findByRole('heading', { name: 'Feedings' })).toBeInTheDocument();
		expect(screen.getByRole('heading', { name: 'Stools' })).toBeInTheDocument();

		// Should show entry data
		const matches = screen.getAllByText(/120\s*mL/);
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
					{ id: 'f1', timestamp: '2026-03-31T10:00:00Z', feed_type: 'formula', volume_ml: 90 },
					{ id: 'f2', timestamp: '2026-03-31T14:00:00Z', feed_type: 'breast_milk', volume_ml: 120 }
				],
				next_cursor: null
			}
		});
		vi.mocked(apiClient.del).mockResolvedValue(undefined);

		render(RawLogList, { props: { babyId: 'baby-1' } });

		// Wait for entries (ascending sort: f1 at 10:00 first, f2 at 14:00 second)
		await screen.findAllByText(/mL/);

		// Delete the first entry (f1, 90 mL)
		const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
		await fireEvent.click(deleteButtons[0]);

		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(apiClient.del).toHaveBeenCalledWith('/babies/baby-1/feedings/f1');
		});

		await waitFor(() => {
			expect(screen.queryByText(/90\s*mL/)).not.toBeInTheDocument();
		});

		const remaining = screen.getAllByText(/120\s*mL/);
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

	it('deleting urine entry also removes linked fluid_log entry', async () => {
		mockAllEndpoints({
			urine: {
				data: [
					{ id: 'u1', timestamp: '2026-03-31T10:00:00Z', color: 'pale_yellow', volume_ml: 50 }
				],
				next_cursor: null
			},
			'fluid-log': {
				data: [
					{
						id: 'fl1',
						timestamp: '2026-03-31T10:00:00Z',
						direction: 'output',
						method: 'urine',
						volume_ml: 50,
						source_type: 'urine',
						source_id: 'u1'
					}
				],
				next_cursor: null
			}
		});
		vi.mocked(apiClient.del).mockResolvedValue(undefined);

		render(RawLogList, { props: { babyId: 'baby-1' } });

		// Wait for both entries to render (2 delete buttons = 2 entries)
		await waitFor(() => {
			expect(screen.getAllByRole('button', { name: /delete/i })).toHaveLength(2);
		});

		// Delete the urine entry (urine appears before fluid in LOG_TYPES)
		const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
		await fireEvent.click(deleteButtons[0]);

		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(apiClient.del).toHaveBeenCalledWith('/babies/baby-1/urine/u1');
		});

		// Both entries should be removed - no delete buttons remain
		await waitFor(() => {
			expect(screen.queryAllByRole('button', { name: /delete/i })).toHaveLength(0);
		});
	});

	it('deleting linked fluid_log entry deletes source instead', async () => {
		mockAllEndpoints({
			urine: {
				data: [
					{ id: 'u1', timestamp: '2026-03-31T10:00:00Z', color: 'pale_yellow', volume_ml: 50 }
				],
				next_cursor: null
			},
			'fluid-log': {
				data: [
					{
						id: 'fl1',
						timestamp: '2026-03-31T10:00:00Z',
						direction: 'output',
						method: 'urine',
						volume_ml: 50,
						source_type: 'urine',
						source_id: 'u1'
					}
				],
				next_cursor: null
			}
		});
		vi.mocked(apiClient.del).mockResolvedValue(undefined);

		render(RawLogList, { props: { babyId: 'baby-1' } });

		// Wait for both entries to render
		await waitFor(() => {
			expect(screen.getAllByRole('button', { name: /delete/i })).toHaveLength(2);
		});

		// Delete the fluid_log entry (second delete button - fluid group comes after urine)
		const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
		await fireEvent.click(deleteButtons[1]);

		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		// Should call the SOURCE endpoint, not fluid-log
		await waitFor(() => {
			expect(apiClient.del).toHaveBeenCalledWith('/babies/baby-1/urine/u1');
		});

		// Both entries should be removed
		await waitFor(() => {
			expect(screen.queryAllByRole('button', { name: /delete/i })).toHaveLength(0);
		});
	});

	it('deleting feeding entry also removes linked fluid_log entry', async () => {
		mockAllEndpoints({
			feedings: {
				data: [
					{ id: 'f1', timestamp: '2026-03-31T10:00:00Z', feed_type: 'formula', volume_ml: 120 }
				],
				next_cursor: null
			},
			'fluid-log': {
				data: [
					{
						id: 'fl2',
						timestamp: '2026-03-31T10:00:00Z',
						direction: 'intake',
						method: 'formula',
						volume_ml: 120,
						source_type: 'feeding',
						source_id: 'f1'
					}
				],
				next_cursor: null
			}
		});
		vi.mocked(apiClient.del).mockResolvedValue(undefined);

		render(RawLogList, { props: { babyId: 'baby-1' } });

		// Wait for both entries to render (2 delete buttons)
		await waitFor(() => {
			expect(screen.getAllByRole('button', { name: /delete/i })).toHaveLength(2);
		});

		// Delete the feeding entry (first group)
		const deleteButtons = screen.getAllByRole('button', { name: /delete/i });
		await fireEvent.click(deleteButtons[0]);

		const confirmBtn = screen.getByRole('button', { name: /confirm/i });
		await fireEvent.click(confirmBtn);

		await waitFor(() => {
			expect(apiClient.del).toHaveBeenCalledWith('/babies/baby-1/feedings/f1');
		});

		// Both entries should be removed
		await waitFor(() => {
			expect(screen.queryAllByRole('button', { name: /delete/i })).toHaveLength(0);
		});

		// Feeding total should update (no more feeding entries)
		expect(screen.queryByText(/Feeding:/)).not.toBeInTheDocument();
	});

	describe('quick date range presets', () => {
		it('renders Today, Yesterday, and Past 7 Days buttons', () => {
			mockAllEndpoints();
			render(RawLogList, { props: { babyId: 'baby-1' } });

			expect(screen.getByRole('button', { name: 'Today' })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'Yesterday' })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'Past 7 Days' })).toBeInTheDocument();
		});

		it('Today is active by default', () => {
			mockAllEndpoints();
			render(RawLogList, { props: { babyId: 'baby-1' } });

			const todayBtn = screen.getByRole('button', { name: 'Today' });
			expect(todayBtn.className).toContain('active');
		});

		it('clicking Yesterday sets dates to yesterday', async () => {
			mockAllEndpoints();
			render(RawLogList, { props: { babyId: 'baby-1' } });

			const yesterdayBtn = screen.getByRole('button', { name: 'Yesterday' });
			await fireEvent.click(yesterdayBtn);

			const yesterday = new Date();
			yesterday.setDate(yesterday.getDate() - 1);
			const expectedDate = formatDateISO(yesterday);

			const fromInput = screen.getByLabelText('From date') as HTMLInputElement;
			const toInput = screen.getByLabelText('To date') as HTMLInputElement;
			expect(fromInput.value).toBe(expectedDate);
			expect(toInput.value).toBe(expectedDate);
		});

		it('clicking Past 7 Days sets from to 7 days ago and to to today', async () => {
			mockAllEndpoints();
			render(RawLogList, { props: { babyId: 'baby-1' } });

			const pastWeekBtn = screen.getByRole('button', { name: 'Past 7 Days' });
			await fireEvent.click(pastWeekBtn);

			const today = new Date();
			const weekAgo = new Date();
			weekAgo.setDate(weekAgo.getDate() - 6);

			const fromInput = screen.getByLabelText('From date') as HTMLInputElement;
			const toInput = screen.getByLabelText('To date') as HTMLInputElement;
			expect(fromInput.value).toBe(formatDateISO(weekAgo));
			expect(toInput.value).toBe(formatDateISO(today));
		});
	});

	describe('log type filter', () => {
		it('renders All button and type filter chips', () => {
			mockAllEndpoints();
			render(RawLogList, { props: { babyId: 'baby-1' } });

			expect(screen.getByRole('button', { name: 'All' })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'Feedings' })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'Stools' })).toBeInTheDocument();
		});

		it('All is active by default and shows all types', async () => {
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

			const allBtn = screen.getByRole('button', { name: 'All' });
			expect(allBtn.className).toContain('active');

			expect(await screen.findByRole('heading', { name: 'Feedings' })).toBeInTheDocument();
			expect(screen.getByRole('heading', { name: 'Stools' })).toBeInTheDocument();
		});

		it('clicking a type chip filters to only that type', async () => {
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
			await screen.findByRole('heading', { name: 'Feedings' });

			// Click Feedings chip
			const feedingsChip = screen.getByRole('button', { name: 'Feedings' });
			await fireEvent.click(feedingsChip);

			// Feedings heading should remain, Stools heading should be hidden
			await waitFor(() => {
				expect(screen.getByRole('heading', { name: 'Feedings' })).toBeInTheDocument();
				expect(screen.queryByRole('heading', { name: 'Stools' })).not.toBeInTheDocument();
			});
		});

		it('clicking multiple type chips shows multiple types', async () => {
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
				},
				urine: {
					data: [
						{ id: 'u1', timestamp: '2026-03-31T12:00:00Z', color: 'pale_yellow', volume_ml: 30 }
					],
					next_cursor: null
				}
			});

			render(RawLogList, { props: { babyId: 'baby-1' } });
			await screen.findByRole('heading', { name: 'Feedings' });

			// Select Feedings and Stools
			await fireEvent.click(screen.getByRole('button', { name: 'Feedings' }));
			await fireEvent.click(screen.getByRole('button', { name: 'Stools' }));

			await waitFor(() => {
				expect(screen.getByRole('heading', { name: 'Feedings' })).toBeInTheDocument();
				expect(screen.getByRole('heading', { name: 'Stools' })).toBeInTheDocument();
				expect(screen.queryByRole('heading', { name: 'Urine' })).not.toBeInTheDocument();
			});
		});

		it('clicking All resets the filter', async () => {
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
			await screen.findByRole('heading', { name: 'Feedings' });

			// Select only Feedings
			await fireEvent.click(screen.getByRole('button', { name: 'Feedings' }));
			await waitFor(() => {
				expect(screen.queryByRole('heading', { name: 'Stools' })).not.toBeInTheDocument();
			});

			// Click All to reset
			await fireEvent.click(screen.getByRole('button', { name: 'All' }));
			await waitFor(() => {
				expect(screen.getByRole('heading', { name: 'Feedings' })).toBeInTheDocument();
				expect(screen.getByRole('heading', { name: 'Stools' })).toBeInTheDocument();
			});
		});
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
