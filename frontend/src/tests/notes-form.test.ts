import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import NotesForm from '$lib/components/NotesForm.svelte';

describe('NotesForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	let onphotoupload: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		onphotoupload = vi.fn();
	});

	it('renders timestamp, content, category, and notes fields', () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/content/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/category/i)).toBeInTheDocument();
	});

	it('category selector has correct options', () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/category/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('behavior');
		expect(options).toContain('sleep');
		expect(options).toContain('vomiting');
		expect(options).toContain('irritability');
		expect(options).toContain('skin');
		expect(options).toContain('other');
	});

	it('validates that content is required', async () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /log note/i }));

		expect(screen.getByText(/content is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/content/i), {
			target: { value: 'Baby seemed fussy today' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log note/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.content).toBe('Baby seemed fussy today');
		expect(payload.timestamp).toBeDefined();
	});

	it('submits correct payload with all fields', async () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/content/i), {
			target: { value: 'Vomited after feeding' }
		});
		await fireEvent.change(screen.getByLabelText(/category/i), {
			target: { value: 'vomiting' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log note/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.content).toBe('Vomited after feeding');
		expect(payload.category).toBe('vomiting');
	});

	it('omits category when not selected', async () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.input(screen.getByLabelText(/content/i), {
			target: { value: 'General observation' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log note/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.category).toBeUndefined();
	});

	it('renders multi-photo upload area', () => {
		render(NotesForm, { props: { onsubmit, onphotoupload } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input).toBeInTheDocument();
		expect(input.multiple).toBe(true);
	});

	it('shows photo count indicator', () => {
		render(NotesForm, { props: { onsubmit, onphotoupload, photoKeys: ['a.jpg', 'b.jpg'] } });

		expect(screen.getByText('2 / 4 photos')).toBeInTheDocument();
	});

	it('disables photo upload when 4 photos are attached', () => {
		render(NotesForm, {
			props: { onsubmit, onphotoupload, photoKeys: ['a.jpg', 'b.jpg', 'c.jpg', 'd.jpg'] }
		});

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input.disabled).toBe(true);
		expect(screen.getByText('4 / 4 photos')).toBeInTheDocument();
	});

	it('allows photo upload when fewer than 4 photos are attached', () => {
		render(NotesForm, {
			props: { onsubmit, onphotoupload, photoKeys: ['a.jpg', 'b.jpg', 'c.jpg'] }
		});

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input.disabled).toBe(false);
		expect(screen.getByText('3 / 4 photos')).toBeInTheDocument();
	});

	it('disables submit button when submitting', () => {
		render(NotesForm, { props: { onsubmit, onphotoupload, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(NotesForm, { props: { onsubmit, onphotoupload, error: 'Failed' } });

		expect(screen.getByText('Failed')).toBeInTheDocument();
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-01-15T10:30:00Z',
			content: 'Baby seemed fussy today',
			category: 'behavior'
		};

		render(NotesForm, { props: { onsubmit, onphotoupload, initialData } });

		expect((screen.getByLabelText(/content/i) as HTMLTextAreaElement).value).toBe('Baby seemed fussy today');
		expect((screen.getByLabelText(/category/i) as HTMLSelectElement).value).toBe('behavior');
		expect(screen.getByRole('button', { name: /update note/i })).toBeInTheDocument();
	});
});
