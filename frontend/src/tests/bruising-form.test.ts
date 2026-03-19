import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import BruisingForm from '$lib/components/BruisingForm.svelte';

describe('BruisingForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	let onphotoupload: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		onphotoupload = vi.fn();
	});

	it('renders timestamp, location, size estimate, size cm, color, and notes fields', () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/location/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/size estimate/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/size.*cm/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/color/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('size estimate selector has correct options', () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/size estimate/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('small_<1cm');
		expect(options).toContain('medium_1-3cm');
		expect(options).toContain('large_>3cm');
	});

	it('validates that location is required', async () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.change(screen.getByLabelText(/size estimate/i), {
			target: { value: 'small_<1cm' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log bruising/i }));

		expect(screen.getByText(/location is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates that size estimate is required', async () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/location/i), {
			target: { value: 'left arm' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log bruising/i }));

		expect(screen.getByText(/size estimate is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/location/i), {
			target: { value: 'left arm' }
		});
		await fireEvent.change(screen.getByLabelText(/size estimate/i), {
			target: { value: 'small_<1cm' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log bruising/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.location).toBe('left arm');
		expect(payload.size_estimate).toBe('small_<1cm');
		expect(payload.timestamp).toBeDefined();
	});

	it('submits correct payload with all fields', async () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/location/i), {
			target: { value: 'torso' }
		});
		await fireEvent.change(screen.getByLabelText(/size estimate/i), {
			target: { value: 'medium_1-3cm' }
		});
		await fireEvent.input(screen.getByLabelText(/size.*cm/i), {
			target: { value: '2.5' }
		});
		await fireEvent.input(screen.getByLabelText(/color/i), {
			target: { value: 'purple' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'appeared overnight' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log bruising/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.location).toBe('torso');
		expect(payload.size_estimate).toBe('medium_1-3cm');
		expect(payload.size_cm).toBe(2.5);
		expect(payload.color).toBe('purple');
		expect(payload.notes).toBe('appeared overnight');
	});

	it('omits optional fields when not provided', async () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/location/i), {
			target: { value: 'leg' }
		});
		await fireEvent.change(screen.getByLabelText(/size estimate/i), {
			target: { value: 'large_>3cm' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log bruising/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.size_cm).toBeUndefined();
		expect(payload.color).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('renders photo upload area', () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/photo/i)).toBeInTheDocument();
	});

	it('disables submit button when submitting', () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(BruisingForm, { props: { onsubmit, onphotoupload, error: 'Failed' } });

		expect(screen.getByText('Failed')).toBeInTheDocument();
	});
});
