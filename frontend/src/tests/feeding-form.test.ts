import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import FeedingForm from '$lib/components/FeedingForm.svelte';

describe('FeedingForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders feed type, volume, cal density, duration, and notes fields', () => {
		render(FeedingForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/feed type/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/volume/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/caloric density/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/duration/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('renders a timestamp field with default value', () => {
		render(FeedingForm, { props: { onsubmit } });

		const timestampInput = screen.getByLabelText(/timestamp/i) as HTMLInputElement;
		expect(timestampInput).toBeInTheDocument();
		expect(timestampInput.value).not.toBe('');
	});

	it('feed type selector has correct options', () => {
		render(FeedingForm, { props: { onsubmit } });

		const select = screen.getByLabelText(/feed type/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('breast_milk');
		expect(options).toContain('formula');
		expect(options).toContain('fortified_breast_milk');
		expect(options).toContain('solid');
		expect(options).toContain('other');
	});

	it('shows validation error when feed type is not selected', async () => {
		render(FeedingForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /log feeding/i }));

		expect(screen.getByText(/feed type is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with minimal fields', async () => {
		render(FeedingForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/feed type/i), {
			target: { value: 'breast_milk' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log feeding/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.feed_type).toBe('breast_milk');
		expect(payload.timestamp).toBeDefined();
	});

	it('submits correct payload with all fields', async () => {
		render(FeedingForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/feed type/i), {
			target: { value: 'formula' }
		});
		await fireEvent.input(screen.getByLabelText(/volume/i), { target: { value: '120' } });
		await fireEvent.input(screen.getByLabelText(/caloric density/i), {
			target: { value: '24' }
		});
		await fireEvent.input(screen.getByLabelText(/duration/i), { target: { value: '15' } });
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'tolerated well' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log feeding/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.feed_type).toBe('formula');
		expect(payload.volume_ml).toBe(120);
		expect(payload.cal_density).toBe(24);
		expect(payload.duration_min).toBe(15);
		expect(payload.notes).toBe('tolerated well');
	});

	it('omits optional fields when not provided', async () => {
		render(FeedingForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/feed type/i), {
			target: { value: 'breast_milk' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log feeding/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.volume_ml).toBeUndefined();
		expect(payload.cal_density).toBeUndefined();
		expect(payload.duration_min).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('disables submit button when submitting prop is true', () => {
		render(FeedingForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(FeedingForm, { props: { onsubmit, error: 'Server error' } });

		expect(screen.getByText('Server error')).toBeInTheDocument();
	});

	it('renders a submit button', () => {
		render(FeedingForm, { props: { onsubmit } });

		expect(screen.getByRole('button', { name: /log feeding/i })).toBeInTheDocument();
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-06-15T14:30:00Z',
			feed_type: 'formula',
			volume_ml: 120,
			cal_density: 24,
			duration_min: 15,
			notes: 'tolerated well'
		};

		render(FeedingForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/feed type/i) as HTMLSelectElement).value).toBe('formula');
		expect((screen.getByLabelText(/volume/i) as HTMLInputElement).value).toBe('120');
		expect((screen.getByLabelText(/caloric density/i) as HTMLInputElement).value).toBe('24');
		expect((screen.getByLabelText(/duration/i) as HTMLInputElement).value).toBe('15');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('tolerated well');
		expect(screen.getByRole('button', { name: /update feeding/i })).toBeInTheDocument();
	});
});
