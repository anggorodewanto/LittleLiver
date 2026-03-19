import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import StoolForm from '$lib/components/StoolForm.svelte';

describe('StoolForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	let onphotoupload: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		onphotoupload = vi.fn();
	});

	it('renders timestamp, consistency, volume, notes fields', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/consistency/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/volume/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('renders 7 tappable CSS color swatches with labels', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const swatches = screen.getAllByRole('button', { name: /^(White|Clay|Pale Yellow|Yellow|Light Green|Green|Brown)$/i });
		expect(swatches).toHaveLength(7);
	});

	it('color swatches have correct labels', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByRole('button', { name: /^White$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Clay$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Pale Yellow$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Yellow$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Light Green$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Green$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Brown$/i })).toBeInTheDocument();
	});

	it('selecting a color swatch sets color_rating and color_label', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /^Yellow$/i }));
		await fireEvent.click(screen.getByRole('button', { name: /log stool/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.color_rating).toBe(4);
		expect(payload.color_label).toBe('yellow');
	});

	it('validates that color rating is required', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /log stool/i }));

		expect(screen.getByText(/stool color is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates color_rating is between 1-7', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		// Selecting swatch 1 (White) should produce rating 1
		await fireEvent.click(screen.getByRole('button', { name: /^White$/i }));
		await fireEvent.click(screen.getByRole('button', { name: /log stool/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.color_rating).toBeGreaterThanOrEqual(1);
		expect(payload.color_rating).toBeLessThanOrEqual(7);
	});

	it('submits correct payload with all fields', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /^Green$/i }));
		await fireEvent.change(screen.getByLabelText(/consistency/i), {
			target: { value: 'soft' }
		});
		await fireEvent.change(screen.getByLabelText(/volume/i), {
			target: { value: 'medium' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'normal color' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log stool/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.color_rating).toBe(6);
		expect(payload.color_label).toBe('green');
		expect(payload.consistency).toBe('soft');
		expect(payload.volume_estimate).toBe('medium');
		expect(payload.notes).toBe('normal color');
	});

	it('consistency selector has correct options', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/consistency/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('watery');
		expect(options).toContain('loose');
		expect(options).toContain('soft');
		expect(options).toContain('formed');
		expect(options).toContain('hard');
	});

	it('volume selector has correct options', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/volume/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('small');
		expect(options).toContain('medium');
		expect(options).toContain('large');
	});

	it('renders photo upload area', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/photo/i)).toBeInTheDocument();
	});

	it('highlights selected color swatch', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const brownButton = screen.getByRole('button', { name: /^Brown$/i });
		await fireEvent.click(brownButton);

		expect(brownButton.getAttribute('aria-pressed')).toBe('true');
	});

	it('disables submit button when submitting', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload, error: 'Server error' } });

		expect(screen.getByText('Server error')).toBeInTheDocument();
	});

	it('displays acholic warning when color 1-3 is selected', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(screen.getByRole('button', { name: /^White$/i }));

		expect(screen.getByText(/acholic/i)).toBeInTheDocument();
	});
});
