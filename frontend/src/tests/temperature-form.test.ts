import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import TemperatureForm from '$lib/components/TemperatureForm.svelte';

describe('TemperatureForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders timestamp, value, method, and notes fields', () => {
		render(TemperatureForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/temperature/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/method/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('method selector has correct options', () => {
		render(TemperatureForm, { props: { onsubmit } });

		const select = screen.getByLabelText(/method/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('rectal');
		expect(options).toContain('axillary');
		expect(options).toContain('ear');
		expect(options).toContain('forehead');
	});

	it('shows validation error when temperature value is empty', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /log temperature/i }));

		expect(screen.getByText(/temperature is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('shows validation error when method is not selected', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/temperature/i), {
			target: { value: '37.5' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log temperature/i }));

		expect(screen.getByText(/method is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/temperature/i), {
			target: { value: '37.2' }
		});
		await fireEvent.change(screen.getByLabelText(/method/i), {
			target: { value: 'rectal' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log temperature/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.value).toBe(37.2);
		expect(payload.method).toBe('rectal');
		expect(payload.timestamp).toBeDefined();
		expect(payload.notes).toBeUndefined();
	});

	it('submits correct payload with all fields', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/temperature/i), {
			target: { value: '38.5' }
		});
		await fireEvent.change(screen.getByLabelText(/method/i), {
			target: { value: 'axillary' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'after bath' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log temperature/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.value).toBe(38.5);
		expect(payload.method).toBe('axillary');
		expect(payload.notes).toBe('after bath');
	});

	it('shows cholangitis warning when rectal temperature >= 38.0', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/temperature/i), {
			target: { value: '38.0' }
		});
		await fireEvent.change(screen.getByLabelText(/method/i), {
			target: { value: 'rectal' }
		});

		expect(screen.getByText(/cholangitis/i)).toBeInTheDocument();
	});

	it('shows cholangitis warning when axillary temperature >= 37.5', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/temperature/i), {
			target: { value: '37.5' }
		});
		await fireEvent.change(screen.getByLabelText(/method/i), {
			target: { value: 'axillary' }
		});

		expect(screen.getByText(/cholangitis/i)).toBeInTheDocument();
	});

	it('does not show fever warning when temperature is below threshold', async () => {
		render(TemperatureForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/temperature/i), {
			target: { value: '37.0' }
		});
		await fireEvent.change(screen.getByLabelText(/method/i), {
			target: { value: 'rectal' }
		});

		expect(screen.queryByText(/cholangitis/i)).not.toBeInTheDocument();
	});

	it('disables submit button when submitting', () => {
		render(TemperatureForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(TemperatureForm, { props: { onsubmit, error: 'Failed' } });

		expect(screen.getByText('Failed')).toBeInTheDocument();
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-01-15T10:30:00Z',
			value: 37.8,
			method: 'rectal',
			notes: 'after nap'
		};

		render(TemperatureForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/temperature/i) as HTMLInputElement).value).toBe('37.8');
		expect((screen.getByLabelText(/method/i) as HTMLSelectElement).value).toBe('rectal');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('after nap');
		expect(screen.getByRole('button', { name: /update temperature/i })).toBeInTheDocument();
	});
});
