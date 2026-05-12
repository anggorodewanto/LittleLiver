import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import HeightForm from '$lib/components/HeightForm.svelte';

describe('HeightForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders timestamp, height, measurement source, and notes fields', () => {
		render(HeightForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/height/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/source/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('measurement source selector has correct options', () => {
		render(HeightForm, { props: { onsubmit } });

		const select = screen.getByLabelText(/source/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('home_scale');
		expect(options).toContain('clinic');
	});

	it('shows validation error when height is empty', async () => {
		render(HeightForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /log height/i }));

		expect(screen.getByText(/height is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(HeightForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/height/i), {
			target: { value: '54.2' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log height/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.height_cm).toBe(54.2);
		expect(payload.timestamp).toBeDefined();
		expect(payload.measurement_source).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('submits correct payload with all fields', async () => {
		render(HeightForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/height/i), {
			target: { value: '56.0' }
		});
		await fireEvent.change(screen.getByLabelText(/source/i), {
			target: { value: 'clinic' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'lying flat' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log height/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.height_cm).toBe(56.0);
		expect(payload.measurement_source).toBe('clinic');
		expect(payload.notes).toBe('lying flat');
	});

	it('disables submit button when submitting', () => {
		render(HeightForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(HeightForm, { props: { onsubmit, error: 'Save failed' } });

		expect(screen.getByText('Save failed')).toBeInTheDocument();
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-06-15T14:30:00Z',
			height_cm: 56.5,
			measurement_source: 'clinic',
			notes: 'lying flat'
		};

		render(HeightForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/height/i) as HTMLInputElement).value).toBe('56.5');
		expect((screen.getByLabelText(/source/i) as HTMLSelectElement).value).toBe('clinic');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('lying flat');
		expect(screen.getByRole('button', { name: /update height/i })).toBeInTheDocument();
	});
});
