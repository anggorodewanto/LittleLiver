import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ImmunizationView from '$lib/components/ImmunizationView.svelte';

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
import type { ImmunizationSlot } from '$lib/types/immunization';

const mandatoryDue: ImmunizationSlot = {
	code: 'HEPB',
	name: 'Hepatitis B',
	dose_number: 2,
	dose_label: 'Dose 2',
	age_months: 1,
	age_label: '1 month',
	mandatory: true,
	status: 'due',
	due_date: '2026-06-01',
	off_schedule: false
};

const mandatoryUpcoming: ImmunizationSlot = {
	code: 'DTAP',
	name: 'DTaP',
	dose_number: 1,
	dose_label: 'Dose 1',
	age_months: 2,
	age_label: '2 months',
	mandatory: true,
	status: 'upcoming',
	due_date: '2026-08-01',
	off_schedule: false
};

const mandatoryDone: ImmunizationSlot = {
	code: 'HEPB',
	name: 'Hepatitis B',
	dose_number: 1,
	dose_label: 'Dose 1',
	age_months: 0,
	age_label: 'Birth',
	mandatory: true,
	status: 'done',
	administered_date: '2025-12-15',
	record_id: 'rec-1',
	off_schedule: false
};

const optionalUpcoming: ImmunizationSlot = {
	code: 'ROTA',
	name: 'Rotavirus',
	dose_number: 1,
	dose_label: 'Dose 1',
	age_months: 2,
	age_label: '2 months',
	mandatory: false,
	status: 'upcoming',
	due_date: '2026-08-01',
	off_schedule: false
};

const offScheduleDone: ImmunizationSlot = {
	code: '',
	name: 'Travel Vaccine',
	dose_number: 1,
	dose_label: 'Dose 1',
	age_months: 0,
	age_label: '',
	mandatory: false,
	status: 'done',
	administered_date: '2026-05-01',
	record_id: 'rec-9',
	off_schedule: true
};

function mockSlots(slots: ImmunizationSlot[]): void {
	(apiClient.get as ReturnType<typeof vi.fn>).mockResolvedValue({ slots });
}

describe('ImmunizationView', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		vi.clearAllMocks();
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockSlots([mandatoryDue, mandatoryUpcoming, mandatoryDone, optionalUpcoming, offScheduleDone]);
	});

	it('fetches the schedule with the correct baby ID', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-42' } });

		await screen.findByText('DTaP');
		expect(mockGet).toHaveBeenCalledWith('/babies/baby-42/immunizations/schedule');
	});

	it('shows loading state while fetching', () => {
		mockGet.mockReturnValue(new Promise(() => {}));
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	it('shows error message when the API call fails', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/failed to load immunizations/i)).toBeInTheDocument();
	});

	it('shows empty state when there are no slots', async () => {
		mockSlots([]);
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText(/no immunizations/i)).toBeInTheDocument();
	});

	it('renders mandatory and optional group headings', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findByText('DTaP');
		expect(screen.getByRole('heading', { name: /mandatory/i })).toBeInTheDocument();
		expect(screen.getByRole('heading', { name: /optional/i })).toBeInTheDocument();
	});

	it('renders a completed (done) slot with its administered date', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findAllByText('Hepatitis B');
		const slots = screen.getAllByTestId('immunization-slot');
		const doneSlot = slots.find((s) => s.textContent?.includes('2025-12-15'));
		expect(doneSlot).toBeDefined();
	});

	it('renders an upcoming slot with its due date', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findByText('DTaP');
		expect(screen.getAllByText(/2026-08-01/).length).toBeGreaterThan(0);
	});

	it('marks a due slot distinctly from an upcoming slot', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findByText('DTaP');
		expect(screen.getByText('Due', { selector: '.badge' })).toBeInTheDocument();
		expect(screen.getAllByText('Upcoming', { selector: '.badge' }).length).toBeGreaterThan(0);
	});

	it('renders off-schedule done records in a separate section', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		expect(await screen.findByText('Travel Vaccine')).toBeInTheDocument();
		expect(screen.getAllByText(/off-schedule/i).length).toBeGreaterThan(0);
	});

	it('shows a mandatory done-count summary', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findByText('DTaP');
		// 1 of 3 mandatory slots are done (done, due, upcoming)
		expect(screen.getByText(/1 of 3 mandatory done/i)).toBeInTheDocument();
	});

	it('navigates to the log form when Log is clicked on a due slot', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findAllByText('Hepatitis B');
		const logButtons = screen.getAllByRole('button', { name: /^log$/i });
		await fireEvent.click(logButtons[0]);

		expect(goto).toHaveBeenCalledWith('/log/immunization?code=HEPB&name=Hepatitis+B&dose=2');
	});

	it('navigates to edit when a completed slot is edited', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findAllByText('Hepatitis B');
		const editButtons = screen.getAllByRole('button', { name: /edit/i });
		await fireEvent.click(editButtons[0]);

		expect(goto).toHaveBeenCalledWith('/log/immunization?edit=rec-1');
	});

	it('renders each slot with a data-testid', async () => {
		render(ImmunizationView, { props: { babyId: 'baby-1' } });

		await screen.findByText('DTaP');
		expect(screen.getAllByTestId('immunization-slot').length).toBeGreaterThan(0);
	});
});
