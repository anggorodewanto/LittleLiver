import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import UrineForm from '$lib/components/UrineForm.svelte';

describe('UrineForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders timestamp, color, volume, and notes fields', () => {
		render(UrineForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/color/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/volume/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('color selector has correct options', () => {
		render(UrineForm, { props: { onsubmit } });

		const select = screen.getByLabelText(/color/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('clear');
		expect(options).toContain('pale_yellow');
		expect(options).toContain('dark_yellow');
		expect(options).toContain('amber');
		expect(options).toContain('brown');
	});

	it('submits with timestamp even when optional fields empty', async () => {
		render(UrineForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /log urine/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.timestamp).toBeDefined();
		expect(payload.color).toBeUndefined();
		expect(payload.volume_ml).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('submits correct payload with all fields', async () => {
		render(UrineForm, { props: { onsubmit } });

		await fireEvent.change(screen.getByLabelText(/color/i), {
			target: { value: 'pale_yellow' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'normal' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log urine/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.color).toBe('pale_yellow');
		expect(payload.notes).toBe('normal');
	});

	it('submits volume_ml when provided', async () => {
		render(UrineForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/volume/i), {
			target: { value: '50.5' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log urine/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.volume_ml).toBe(50.5);
	});

	it('disables submit button when submitting', () => {
		render(UrineForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(UrineForm, { props: { onsubmit, error: 'Network error' } });

		expect(screen.getByText('Network error')).toBeInTheDocument();
	});

	it('renders a submit button', () => {
		render(UrineForm, { props: { onsubmit } });

		expect(screen.getByRole('button', { name: /log urine/i })).toBeInTheDocument();
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-06-15T14:30:00Z',
			color: 'pale_yellow',
			volume_ml: 50.5,
			notes: 'normal output'
		};

		render(UrineForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/color/i) as HTMLSelectElement).value).toBe('pale_yellow');
		expect((screen.getByLabelText(/volume/i) as HTMLInputElement).value).toBe('50.5');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('normal output');
		expect(screen.getByRole('button', { name: /update urine/i })).toBeInTheDocument();
	});
});
