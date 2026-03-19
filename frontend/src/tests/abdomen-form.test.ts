import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import AbdomenForm from '$lib/components/AbdomenForm.svelte';

describe('AbdomenForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	let onphotoupload: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		onphotoupload = vi.fn();
	});

	it('renders timestamp, firmness, tenderness, girth, and notes fields', () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/firmness/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/tenderness/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/girth/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('firmness selector has correct options', () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/firmness/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('soft');
		expect(options).toContain('firm');
		expect(options).toContain('distended');
	});

	it('validates that firmness is required', async () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /log abdomen/i }));

		expect(screen.getByText(/firmness is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.change(screen.getByLabelText(/firmness/i), {
			target: { value: 'soft' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log abdomen/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.firmness).toBe('soft');
		expect(payload.tenderness).toBe(false);
		expect(payload.timestamp).toBeDefined();
	});

	it('submits correct payload with all fields', async () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.change(screen.getByLabelText(/firmness/i), {
			target: { value: 'distended' }
		});
		await fireEvent.click(screen.getByLabelText(/tenderness/i));
		await fireEvent.input(screen.getByLabelText(/girth/i), {
			target: { value: '35.5' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'slightly swollen' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log abdomen/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.firmness).toBe('distended');
		expect(payload.tenderness).toBe(true);
		expect(payload.girth_cm).toBe(35.5);
		expect(payload.notes).toBe('slightly swollen');
	});

	it('omits optional fields when not provided', async () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.change(screen.getByLabelText(/firmness/i), {
			target: { value: 'firm' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log abdomen/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.girth_cm).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('renders photo upload area', () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/photo/i)).toBeInTheDocument();
	});

	it('disables submit button when submitting', () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(AbdomenForm, { props: { onsubmit, onphotoupload, error: 'Failed' } });

		expect(screen.getByText('Failed')).toBeInTheDocument();
	});
});
