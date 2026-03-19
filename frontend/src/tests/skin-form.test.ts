import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import SkinForm from '$lib/components/SkinForm.svelte';

describe('SkinForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	let onphotoupload: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		onphotoupload = vi.fn();
	});

	it('renders timestamp, jaundice level, scleral icterus, rashes, bruising, and notes fields', () => {
		render(SkinForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/jaundice/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/scleral icterus/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/rashes/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/bruising/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('jaundice level selector has correct options', () => {
		render(SkinForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/jaundice/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('none');
		expect(options).toContain('mild_face');
		expect(options).toContain('moderate_trunk');
		expect(options).toContain('severe_limbs_and_trunk');
	});

	it('submits correct payload with all fields', async () => {
		render(SkinForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.change(screen.getByLabelText(/jaundice/i), {
			target: { value: 'mild_face' }
		});
		await fireEvent.click(screen.getByLabelText(/scleral icterus/i));
		await fireEvent.input(screen.getByLabelText(/rashes/i), {
			target: { value: 'mild rash on cheeks' }
		});
		await fireEvent.input(screen.getByLabelText(/bruising/i), {
			target: { value: 'small bruise on arm' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'looks better today' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log skin/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.jaundice_level).toBe('mild_face');
		expect(payload.scleral_icterus).toBe(true);
		expect(payload.rashes).toBe('mild rash on cheeks');
		expect(payload.bruising).toBe('small bruise on arm');
		expect(payload.notes).toBe('looks better today');
	});

	it('submits with defaults when optional fields empty', async () => {
		render(SkinForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /log skin/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.scleral_icterus).toBe(false);
		expect(payload.timestamp).toBeDefined();
	});

	it('displays consistent lighting hint near photo upload', () => {
		render(SkinForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByText(/consistent lighting recommended/i)).toBeInTheDocument();
	});

	it('renders photo upload area', () => {
		render(SkinForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/photo/i)).toBeInTheDocument();
	});

	it('disables submit button when submitting', () => {
		render(SkinForm, { props: { onsubmit, onphotoupload, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(SkinForm, { props: { onsubmit, onphotoupload, error: 'Failed' } });

		expect(screen.getByText('Failed')).toBeInTheDocument();
	});
});
