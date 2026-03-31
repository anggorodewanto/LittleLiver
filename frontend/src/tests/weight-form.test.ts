import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import WeightForm from '$lib/components/WeightForm.svelte';

describe('WeightForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders timestamp, weight, measurement source, and notes fields', () => {
		render(WeightForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/weight/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/source/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('measurement source selector has correct options', () => {
		render(WeightForm, { props: { onsubmit } });

		const select = screen.getByLabelText(/source/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('home_scale');
		expect(options).toContain('clinic');
	});

	it('shows validation error when weight is empty', async () => {
		render(WeightForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /log weight/i }));

		expect(screen.getByText(/weight is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(WeightForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/weight/i), {
			target: { value: '4.5' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log weight/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.weight_kg).toBe(4.5);
		expect(payload.timestamp).toBeDefined();
		expect(payload.measurement_source).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('submits correct payload with all fields', async () => {
		render(WeightForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/weight/i), {
			target: { value: '5.2' }
		});
		await fireEvent.change(screen.getByLabelText(/source/i), {
			target: { value: 'clinic' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'post-feed' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log weight/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.weight_kg).toBe(5.2);
		expect(payload.measurement_source).toBe('clinic');
		expect(payload.notes).toBe('post-feed');
	});

	it('disables submit button when submitting', () => {
		render(WeightForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(WeightForm, { props: { onsubmit, error: 'Save failed' } });

		expect(screen.getByText('Save failed')).toBeInTheDocument();
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-06-15T14:30:00Z',
			weight_kg: 4.5,
			measurement_source: 'clinic',
			notes: 'post-feed'
		};

		render(WeightForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/weight/i) as HTMLInputElement).value).toBe('4.5');
		expect((screen.getByLabelText(/source/i) as HTMLSelectElement).value).toBe('clinic');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('post-feed');
		expect(screen.getByRole('button', { name: /update weight/i })).toBeInTheDocument();
	});
});
