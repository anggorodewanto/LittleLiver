import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import MedicationList from '$lib/components/MedicationList.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn(),
		put: vi.fn()
	}
}));

import { apiClient } from '$lib/api';

const activeMed = {
	id: 'med-1',
	baby_id: 'baby-1',
	name: 'UDCA (ursodiol)',
	dose: '50mg',
	frequency: 'twice_daily',
	schedule: '["08:00","20:00"]',
	timezone: 'America/New_York',
	active: true,
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-01T00:00:00Z'
};

const inactiveMed = {
	id: 'med-2',
	baby_id: 'baby-1',
	name: 'Vitamin D',
	dose: '400IU',
	frequency: 'once_daily',
	schedule: '["09:00"]',
	timezone: 'America/New_York',
	active: false,
	created_at: '2026-03-01T00:00:00Z',
	updated_at: '2026-03-15T00:00:00Z'
};

describe('MedicationList', () => {
	let mockGet: ReturnType<typeof vi.fn>;
	let mockPut: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockPut = apiClient.put as ReturnType<typeof vi.fn>;
		mockGet.mockResolvedValue({ medications: [activeMed, inactiveMed] });
		mockPut.mockResolvedValue({ ...activeMed, active: false });
	});

	it('renders active medications highlighted', async () => {
		render(MedicationList, { props: { babyId: 'baby-1' } });

		const name = await screen.findByText('UDCA (ursodiol)');
		expect(name).toBeInTheDocument();
		const item = name.closest('[data-testid="medication-item"]');
		expect(item).not.toBeNull();
		expect(item!.classList.contains('inactive')).toBe(false);
	});

	it('renders inactive medications dimmed', async () => {
		render(MedicationList, { props: { babyId: 'baby-1' } });

		const name = await screen.findByText('Vitamin D');
		expect(name).toBeInTheDocument();
		const item = name.closest('[data-testid="medication-item"]');
		expect(item).not.toBeNull();
		expect(item!.classList.contains('inactive')).toBe(true);
	});

	it('shows dose and frequency for each medication', async () => {
		render(MedicationList, { props: { babyId: 'baby-1' } });

		await screen.findByText('UDCA (ursodiol)');
		expect(screen.getByText('50mg')).toBeInTheDocument();
		expect(screen.getByText(/twice daily/i)).toBeInTheDocument();
		expect(screen.getByText('400IU')).toBeInTheDocument();
		expect(screen.getByText(/once daily/i)).toBeInTheDocument();
	});

	it('fetches medications from the API with the correct baby ID', async () => {
		render(MedicationList, { props: { babyId: 'baby-42' } });

		await screen.findByText('UDCA (ursodiol)');
		expect(mockGet).toHaveBeenCalledWith('/babies/baby-42/medications');
	});

	it('shows loading state while fetching', () => {
		mockGet.mockReturnValue(new Promise(() => {}));
		render(MedicationList, { props: { babyId: 'baby-1' } });

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	it('shows error message when API call fails', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));
		render(MedicationList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/failed to load/i)).toBeInTheDocument();
	});

	it('deactivation toggle calls PUT with active=false', async () => {
		render(MedicationList, { props: { babyId: 'baby-1' } });

		await screen.findByText('UDCA (ursodiol)');
		const deactivateButtons = screen.getAllByRole('button', { name: /deactivate/i });
		await fireEvent.click(deactivateButtons[0]);

		expect(mockPut).toHaveBeenCalledWith('/babies/baby-1/medications/med-1', {
			name: 'UDCA (ursodiol)',
			dose: '50mg',
			frequency: 'twice_daily',
			schedule_times: [],
			active: false
		});
	});

	it('deactivation updates the display to show inactive', async () => {
		mockPut.mockResolvedValue({ ...activeMed, active: false });
		// After deactivation, refetch returns updated data
		mockGet
			.mockResolvedValueOnce({ medications: [activeMed, inactiveMed] })
			.mockResolvedValueOnce({
				medications: [{ ...activeMed, active: false }, inactiveMed]
			});

		render(MedicationList, { props: { babyId: 'baby-1' } });

		await screen.findByText('UDCA (ursodiol)');
		const deactivateButtons = screen.getAllByRole('button', { name: /deactivate/i });
		await fireEvent.click(deactivateButtons[0]);

		// Wait for refetch
		await vi.waitFor(() => {
			const item = screen.getByText('UDCA (ursodiol)').closest('[data-testid="medication-item"]');
			expect(item!.classList.contains('inactive')).toBe(true);
		});
	});

	it('calls oncreate when Add Medication button is clicked', async () => {
		const oncreate = vi.fn();
		render(MedicationList, { props: { babyId: 'baby-1', oncreate } });

		await screen.findByText('UDCA (ursodiol)');
		await fireEvent.click(screen.getByRole('button', { name: /add medication/i }));

		expect(oncreate).toHaveBeenCalled();
	});

	it('calls onedit when Edit button is clicked', async () => {
		const onedit = vi.fn();
		render(MedicationList, { props: { babyId: 'baby-1', onedit } });

		await screen.findByText('UDCA (ursodiol)');
		const editButtons = screen.getAllByRole('button', { name: /edit/i });
		await fireEvent.click(editButtons[0]);

		expect(onedit).toHaveBeenCalledWith('med-1');
	});

	it('calls onviewlogs when View Logs button is clicked', async () => {
		const onviewlogs = vi.fn();
		render(MedicationList, { props: { babyId: 'baby-1', onviewlogs } });

		await screen.findByText('UDCA (ursodiol)');
		const logButtons = screen.getAllByRole('button', { name: /view logs/i });
		await fireEvent.click(logButtons[0]);

		expect(onviewlogs).toHaveBeenCalledWith('med-1');
	});

	it('shows empty state when no medications exist', async () => {
		mockGet.mockResolvedValue({ medications: [] });
		render(MedicationList, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/no medications/i)).toBeInTheDocument();
	});

	it('shows Reactivate button for inactive medications', async () => {
		render(MedicationList, { props: { babyId: 'baby-1' } });

		await screen.findByText('Vitamin D');
		expect(screen.getByRole('button', { name: /reactivate/i })).toBeInTheDocument();
	});

	it('reactivation toggle calls PUT with active=true for inactive medication', async () => {
		mockPut.mockResolvedValue({ ...inactiveMed, active: true });
		mockGet
			.mockResolvedValueOnce({ medications: [activeMed, inactiveMed] })
			.mockResolvedValueOnce({
				medications: [activeMed, { ...inactiveMed, active: true }]
			});

		render(MedicationList, { props: { babyId: 'baby-1' } });

		await screen.findByText('Vitamin D');
		const reactivateButton = screen.getByRole('button', { name: /reactivate/i });
		await fireEvent.click(reactivateButton);

		expect(mockPut).toHaveBeenCalledWith('/babies/baby-1/medications/med-2', {
			name: 'Vitamin D',
			dose: '400IU',
			frequency: 'once_daily',
			schedule_times: [],
			active: true
		});
	});
});
