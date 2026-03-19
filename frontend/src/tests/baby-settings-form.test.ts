import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import BabySettingsForm from '$lib/components/BabySettingsForm.svelte';
import { mockBabies } from './fixtures';

describe('BabySettingsForm', () => {
	let onsave: ReturnType<typeof vi.fn>;

	const baby = {
		...mockBabies[0],
		default_cal_per_feed: 67
	};

	beforeEach(() => {
		onsave = vi.fn();
	});

	it('renders pre-filled name, DOB, sex, diagnosis date, kasai date fields', () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		expect((screen.getByLabelText(/name/i) as HTMLInputElement).value).toBe('Alice');
		expect((screen.getByLabelText(/date of birth/i) as HTMLInputElement).value).toBe(
			'2025-06-01'
		);
		expect((screen.getByLabelText(/sex/i) as HTMLSelectElement).value).toBe('female');
	});

	it('renders default cal per feed field', () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		const input = screen.getByLabelText(/default cal/i) as HTMLInputElement;
		expect(input.value).toBe('67');
	});

	it('shows recalculate checkbox when default cal per feed is changed', async () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		const calInput = screen.getByLabelText(/default cal/i);
		await fireEvent.input(calInput, { target: { value: '80' } });

		expect(screen.getByLabelText(/recalculate/i)).toBeInTheDocument();
	});

	it('does not show recalculate checkbox when default cal per feed is unchanged', () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		expect(screen.queryByLabelText(/recalculate/i)).not.toBeInTheDocument();
	});

	it('calls onsave with updated data and recalculate flag', async () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		await fireEvent.input(screen.getByLabelText(/name/i), { target: { value: 'Alice Updated' } });
		await fireEvent.input(screen.getByLabelText(/default cal/i), { target: { value: '80' } });
		await fireEvent.click(screen.getByLabelText(/recalculate/i));
		await fireEvent.click(screen.getByRole('button', { name: /save/i }));

		expect(onsave).toHaveBeenCalledWith(
			expect.objectContaining({
				name: 'Alice Updated',
				default_cal_per_feed: 80
			}),
			true
		);
	});

	it('calls onsave without recalculate when checkbox not checked', async () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		await fireEvent.input(screen.getByLabelText(/default cal/i), { target: { value: '80' } });
		await fireEvent.click(screen.getByRole('button', { name: /save/i }));

		expect(onsave).toHaveBeenCalledWith(expect.anything(), false);
	});

	it('validates that name is required', async () => {
		render(BabySettingsForm, { props: { baby, onsave } });

		await fireEvent.input(screen.getByLabelText(/name/i), { target: { value: '' } });
		await fireEvent.click(screen.getByRole('button', { name: /save/i }));

		expect(screen.getByText(/name is required/i)).toBeInTheDocument();
		expect(onsave).not.toHaveBeenCalled();
	});

	it('disables submit button when submitting', () => {
		render(BabySettingsForm, { props: { baby, onsave, submitting: true } });

		expect(screen.getByRole('button', { name: /saving/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(BabySettingsForm, { props: { baby, onsave, error: 'Update failed' } });

		expect(screen.getByText('Update failed')).toBeInTheDocument();
	});
});
