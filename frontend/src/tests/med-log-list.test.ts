import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import MedLogList from '$lib/components/MedLogList.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

import { apiClient } from '$lib/api';

const givenLog = {
	id: 'log-1',
	medication_id: 'med-1',
	baby_id: 'baby-1',
	scheduled_time: '2026-03-19T12:00:00Z',
	given_at: '2026-03-19T12:05:00Z',
	skipped: false,
	skip_reason: null,
	notes: 'Took full dose',
	created_at: '2026-03-19T12:05:00Z'
};

const skippedLog = {
	id: 'log-2',
	medication_id: 'med-1',
	baby_id: 'baby-1',
	scheduled_time: '2026-03-19T20:00:00Z',
	given_at: null,
	skipped: true,
	skip_reason: 'Vomiting',
	notes: null,
	created_at: '2026-03-19T20:01:00Z'
};

describe('MedLogList', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockGet.mockResolvedValue({ med_logs: [givenLog, skippedLog] });
	});

	it('shows given status for doses that were given', async () => {
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		expect(await screen.findByText(/given/i)).toBeInTheDocument();
	});

	it('shows skipped status for doses that were skipped', async () => {
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		await screen.findByText(/given/i);
		expect(screen.getByText(/skipped/i)).toBeInTheDocument();
	});

	it('shows skip reason when available', async () => {
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		expect(await screen.findByText(/vomiting/i)).toBeInTheDocument();
	});

	it('shows notes when available', async () => {
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		expect(await screen.findByText(/took full dose/i)).toBeInTheDocument();
	});

	it('fetches med-logs from the correct API endpoint', async () => {
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		await screen.findByText(/given/i);
		expect(mockGet).toHaveBeenCalledWith('/babies/baby-1/med-logs?medication_id=med-1');
	});

	it('shows loading state while fetching', () => {
		mockGet.mockReturnValue(new Promise(() => {}));
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	it('shows error message when API call fails', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		expect(await screen.findByText(/failed to load/i)).toBeInTheDocument();
	});

	it('shows empty state when no logs exist', async () => {
		mockGet.mockResolvedValue({ med_logs: [] });
		render(MedLogList, {
			props: { babyId: 'baby-1', medicationId: 'med-1' }
		});

		expect(await screen.findByText(/no dose logs/i)).toBeInTheDocument();
	});
});
