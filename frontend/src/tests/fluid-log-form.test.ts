import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FluidLogForm from '$lib/components/FluidLogForm.svelte';

describe('FluidLogForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders timestamp, method, volume, and notes fields', () => {
		render(FluidLogForm, { props: { direction: 'intake', onsubmit } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/method/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/volume/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('submits with direction=intake when configured as intake', async () => {
		render(FluidLogForm, { props: { direction: 'intake', onsubmit } });

		await fireEvent.input(screen.getByLabelText(/method/i), {
			target: { value: 'IV' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log intake/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.direction).toBe('intake');
		expect(payload.method).toBe('IV');
		expect(payload.timestamp).toBeDefined();
	});

	it('submits with direction=output when configured as output', async () => {
		render(FluidLogForm, { props: { direction: 'output', onsubmit } });

		await fireEvent.input(screen.getByLabelText(/method/i), {
			target: { value: 'Stoma' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log output/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.direction).toBe('output');
		expect(payload.method).toBe('Stoma');
	});

	it('shows validation error when method is empty', async () => {
		render(FluidLogForm, { props: { direction: 'intake', onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /log intake/i }));

		expect(screen.getByText('Method is required')).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('includes volume_ml when provided', async () => {
		render(FluidLogForm, { props: { direction: 'intake', onsubmit } });

		await fireEvent.input(screen.getByLabelText(/method/i), {
			target: { value: 'IV' }
		});
		await fireEvent.input(screen.getByLabelText(/volume/i), {
			target: { value: '100' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log intake/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.volume_ml).toBe(100);
	});

	it('omits volume_ml when empty', async () => {
		render(FluidLogForm, { props: { direction: 'output', onsubmit } });

		await fireEvent.input(screen.getByLabelText(/method/i), {
			target: { value: 'Drain' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log output/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.volume_ml).toBeUndefined();
	});

	it('disables submit button when submitting', () => {
		render(FluidLogForm, { props: { direction: 'intake', onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(FluidLogForm, { props: { direction: 'intake', onsubmit, error: 'Network error' } });

		expect(screen.getByText('Network error')).toBeInTheDocument();
	});
});
