import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import MedicationForm from '$lib/components/MedicationForm.svelte';

describe('MedicationForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders name, dose, frequency, and notes fields', () => {
		render(MedicationForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/medication name/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/dose/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/frequency/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('shows pre-populated medication name suggestions', () => {
		render(MedicationForm, { props: { onsubmit } });

		const datalist = document.getElementById('medication-suggestions');
		expect(datalist).not.toBeNull();
		const options = datalist!.querySelectorAll('option');
		const values = Array.from(options).map((o) => o.value);
		expect(values).toContain('UDCA (ursodiol)');
		expect(values).toContain('Vitamin D');
		expect(values).toContain('Vitamin A');
		expect(values).toContain('Vitamin E (TPGS)');
		expect(values).toContain('Vitamin K');
		expect(values).toContain('Iron');
		expect(values).toContain('Sulfamethoxazole-Trimethoprim (Bactrim)');
		expect(values).toContain('Other');
	});

	it('frequency selector has correct options', () => {
		render(MedicationForm, { props: { onsubmit } });

		const select = screen.getByLabelText(/frequency/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('once_daily');
		expect(options).toContain('twice_daily');
		expect(options).toContain('three_times_daily');
		expect(options).toContain('as_needed');
		expect(options).toContain('custom');
	});

	it('shows schedule time pickers when frequency is not as_needed', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'once_daily' }
		});

		expect(screen.getByLabelText(/schedule time 1/i)).toBeInTheDocument();
	});

	it('shows correct number of time pickers for twice_daily', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'twice_daily' }
		});

		expect(screen.getByLabelText(/schedule time 1/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/schedule time 2/i)).toBeInTheDocument();
	});

	it('shows three time pickers for three_times_daily', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'three_times_daily' }
		});

		expect(screen.getByLabelText(/schedule time 1/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/schedule time 2/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/schedule time 3/i)).toBeInTheDocument();
	});

	it('hides schedule time pickers for as_needed', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'as_needed' }
		});

		expect(screen.queryByLabelText(/schedule time/i)).not.toBeInTheDocument();
	});

	it('allows adding custom time pickers for custom frequency', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'custom' }
		});

		expect(screen.getByLabelText(/schedule time 1/i)).toBeInTheDocument();
		await fireEvent.click(screen.getByRole('button', { name: /add time/i }));
		expect(screen.getByLabelText(/schedule time 2/i)).toBeInTheDocument();
	});

	it('validates that medication name is required', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'once_daily' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /save medication/i }));

		expect(screen.getByText(/medication name is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates that dose is required', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/medication name/i), {
			target: { value: 'UDCA' }
		});
		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'once_daily' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /save medication/i }));

		expect(screen.getByText(/dose is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates that frequency is required', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/medication name/i), {
			target: { value: 'UDCA' }
		});
		await fireEvent.input(screen.getByLabelText(/dose/i), { target: { value: '50mg' } });
		await fireEvent.click(screen.getByRole('button', { name: /save medication/i }));

		expect(screen.getByText(/frequency is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with schedule_times JSON', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/medication name/i), {
			target: { value: 'UDCA (ursodiol)' }
		});
		await fireEvent.input(screen.getByLabelText(/dose/i), { target: { value: '50mg' } });
		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'twice_daily' }
		});
		await fireEvent.input(screen.getByLabelText(/schedule time 1/i), {
			target: { value: '08:00' }
		});
		await fireEvent.input(screen.getByLabelText(/schedule time 2/i), {
			target: { value: '20:00' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /save medication/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.name).toBe('UDCA (ursodiol)');
		expect(payload.dose).toBe('50mg');
		expect(payload.frequency).toBe('twice_daily');
		expect(payload.schedule_times).toEqual(['08:00', '20:00']);
	});

	it('submits empty schedule_times for as_needed', async () => {
		render(MedicationForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/medication name/i), {
			target: { value: 'Vitamin D' }
		});
		await fireEvent.input(screen.getByLabelText(/dose/i), { target: { value: '400IU' } });
		await fireEvent.change(screen.getByLabelText(/frequency/i), {
			target: { value: 'as_needed' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /save medication/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.schedule_times).toEqual([]);
	});

	it('pre-fills form fields when initialData is provided (edit mode)', () => {
		const initialData = {
			name: 'UDCA (ursodiol)',
			dose: '50mg',
			frequency: 'twice_daily',
			schedule_times: ['08:00', '20:00'],
			active: true
		};

		render(MedicationForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/medication name/i) as HTMLInputElement).value).toBe(
			'UDCA (ursodiol)'
		);
		expect((screen.getByLabelText(/dose/i) as HTMLInputElement).value).toBe('50mg');
		expect((screen.getByLabelText(/frequency/i) as HTMLSelectElement).value).toBe('twice_daily');
	});

	it('disables submit button when submitting prop is true', () => {
		render(MedicationForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(MedicationForm, { props: { onsubmit, error: 'Server error' } });

		expect(screen.getByText('Server error')).toBeInTheDocument();
	});

	it('pre-fills notes field from initialData in edit mode', () => {
		const initialData = {
			name: 'UDCA (ursodiol)',
			dose: '50mg',
			frequency: 'twice_daily',
			schedule_times: ['08:00', '20:00'],
			active: true,
			notes: 'Take with food'
		};

		render(MedicationForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('Take with food');
	});
});
