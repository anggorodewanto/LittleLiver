import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import DoseLogForm from '$lib/components/DoseLogForm.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

import { apiClient } from '$lib/api';

const medications = [
	{
		id: 'med-1',
		name: 'UDCA (ursodiol)',
		dose: '50mg',
		frequency: 'twice_daily',
		active: true
	},
	{
		id: 'med-2',
		name: 'Vitamin D',
		dose: '400IU',
		frequency: 'once_daily',
		active: true
	}
];

describe('DoseLogForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockGet.mockResolvedValue({ medications });
	});

	it('renders medication selector, status, and notes fields', async () => {
		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		expect(screen.getByLabelText(/medication/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('renders Given and Skipped status buttons', async () => {
		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		expect(screen.getByRole('button', { name: /^given$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^skipped$/i })).toBeInTheDocument();
	});

	it('loads medication list from API', async () => {
		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		expect(mockGet).toHaveBeenCalledWith('/babies/baby-1/medications');
	});

	it('pre-fills medication from medicationId prop', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', medicationId: 'med-1' }
		});

		await screen.findByLabelText(/medication/i);
		const select = screen.getByLabelText(/medication/i) as HTMLSelectElement;
		expect(select.value).toBe('med-1');
	});

	it('shows skip reason field when Skipped is selected', async () => {
		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /^skipped$/i }));

		expect(screen.getByLabelText(/skip reason/i)).toBeInTheDocument();
	});

	it('hides skip reason field when Given is selected', async () => {
		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /^skipped$/i }));
		await fireEvent.click(screen.getByRole('button', { name: /^given$/i }));

		expect(screen.queryByLabelText(/skip reason/i)).not.toBeInTheDocument();
	});

	it('submits correct payload for a given dose', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', medicationId: 'med-1' }
		});

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /^given$/i }));
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'Full dose taken' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log dose/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.medication_id).toBe('med-1');
		expect(payload.skipped).toBe(false);
		expect(payload.notes).toBe('Full dose taken');
	});

	it('submits correct payload for a skipped dose', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', medicationId: 'med-2' }
		});

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /^skipped$/i }));
		await fireEvent.input(screen.getByLabelText(/skip reason/i), {
			target: { value: 'Vomiting' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log dose/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.medication_id).toBe('med-2');
		expect(payload.skipped).toBe(true);
		expect(payload.skip_reason).toBe('Vomiting');
	});

	it('validates that a medication is selected', async () => {
		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /^given$/i }));
		await fireEvent.click(screen.getByRole('button', { name: /log dose/i }));

		expect(screen.getByText(/medication is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates that a status (given/skipped) is selected', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', medicationId: 'med-1' }
		});

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /log dose/i }));

		expect(screen.getByText(/select given or skipped/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('disables submit button when submitting prop is true', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', submitting: true }
		});

		await screen.findByLabelText(/medication/i);
		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', error: 'Server error' }
		});

		await screen.findByLabelText(/medication/i);
		expect(screen.getByText('Server error')).toBeInTheDocument();
	});

	it('includes scheduledTime in the submitted payload', async () => {
		render(DoseLogForm, {
			props: { onsubmit, babyId: 'baby-1', medicationId: 'med-1', scheduledTime: '08:00' }
		});

		await screen.findByLabelText(/medication/i);
		await fireEvent.click(screen.getByRole('button', { name: /^given$/i }));
		await fireEvent.click(screen.getByRole('button', { name: /log dose/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.scheduled_time).toBe('08:00');
	});

	it('shows error message when medication list fails to load', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));

		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		expect(await screen.findByText(/failed to load medications/i)).toBeInTheDocument();
	});

	it('populates medication dropdown with active medications only', async () => {
		mockGet.mockResolvedValue({
			medications: [
				...medications,
				{ id: 'med-3', name: 'Inactive Med', dose: '10mg', frequency: 'once_daily', active: false }
			]
		});

		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1' } });

		await screen.findByLabelText(/medication/i);
		const select = screen.getByLabelText(/medication/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.textContent);
		expect(options.some((o) => o?.includes('UDCA'))).toBe(true);
		expect(options.some((o) => o?.includes('Vitamin D'))).toBe(true);
		expect(options.some((o) => o?.includes('Inactive Med'))).toBe(false);
	});

	it('pre-populates fields when initialData is provided', async () => {
		const initialData = {
			medication_id: 'med-2',
			skipped: true,
			skip_reason: 'Vomiting',
			notes: 'Will retry later'
		};

		render(DoseLogForm, { props: { onsubmit, babyId: 'baby-1', initialData } });

		await screen.findByLabelText(/medication/i);
		expect((screen.getByLabelText(/medication/i) as HTMLSelectElement).value).toBe('med-2');
		expect((screen.getByLabelText(/skip reason/i) as HTMLInputElement).value).toBe('Vomiting');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('Will retry later');
		expect(screen.getByRole('button', { name: /update dose/i })).toBeInTheDocument();
	});
});
