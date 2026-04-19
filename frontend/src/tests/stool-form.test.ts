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

	// Helper: find swatch button by label text (accessible name includes clinical meaning).
	// Uses word boundary to avoid "Yellow" matching "Pale Yellow".
	function getSwatchButton(label: string): HTMLElement {
		return screen.getByRole('button', { name: new RegExp(`^${label} `, 'i') });
	}

	it('renders timestamp, consistency, volume estimate, volume (mL), and notes fields', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/consistency/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/volume estimate/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/volume \(mL\)/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('renders 7 tappable CSS color swatches with labels and clinical meanings', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		// Each button's accessible name includes both the label and clinical meaning
		const labels = ['White', 'Clay', 'Pale Yellow', 'Yellow', 'Light Green', 'Green', 'Brown'];
		for (const label of labels) {
			expect(getSwatchButton(label)).toBeInTheDocument();
		}
	});

	it('color swatches have correct labels and clinical meanings', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		expect(getSwatchButton('White')).toBeInTheDocument();
		expect(getSwatchButton('Clay')).toBeInTheDocument();
		expect(getSwatchButton('Pale Yellow')).toBeInTheDocument();
		expect(getSwatchButton('Yellow')).toBeInTheDocument();
		expect(getSwatchButton('Light Green')).toBeInTheDocument();
		expect(getSwatchButton('Green')).toBeInTheDocument();
		expect(getSwatchButton('Brown')).toBeInTheDocument();

		// Verify clinical meanings are displayed
		expect(screen.getByText(/NO bile flow/i)).toBeInTheDocument();
		expect(screen.getByText(/Normal bile flow/i)).toBeInTheDocument();
	});

	it('selecting a color swatch sets color_rating and color_label', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(getSwatchButton('Yellow'));
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
		await fireEvent.click(getSwatchButton('White'));
		await fireEvent.click(screen.getByRole('button', { name: /log stool/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.color_rating).toBeGreaterThanOrEqual(1);
		expect(payload.color_rating).toBeLessThanOrEqual(7);
	});

	it('submits correct payload with all fields', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		await fireEvent.click(getSwatchButton('Green'));
		await fireEvent.change(screen.getByLabelText(/consistency/i), {
			target: { value: 'soft' }
		});
		await fireEvent.change(screen.getByLabelText(/volume estimate/i), {
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

	it('volume estimate selector has correct options', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const select = screen.getByLabelText(/volume estimate/i) as HTMLSelectElement;
		const options = Array.from(select.options).map((o) => o.value);
		expect(options).toContain('small');
		expect(options).toContain('medium');
		expect(options).toContain('large');
	});

	it('renders photo upload area with camera and gallery inputs', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		expect(screen.getByLabelText(/take photo/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/choose photo/i)).toBeInTheDocument();
	});

	it('highlights selected color swatch', async () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const brownButton = getSwatchButton('Brown');
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

		await fireEvent.click(getSwatchButton('White'));

		// The warning banner text (separate from the swatch meaning text)
		expect(screen.getByRole('alert')).toHaveTextContent(/acholic/i);
	});

	it('uses white text on dark-background color swatches for readability', () => {
		render(StoolForm, { props: { onsubmit, onphotoupload } });

		const greenButton = getSwatchButton('Green');
		const brownButton = getSwatchButton('Brown');

		expect(greenButton.style.color).toBe('white');
		expect(brownButton.style.color).toBe('white');

		// Light swatches should not force white text
		const whiteButton = getSwatchButton('White');
		const yellowButton = getSwatchButton('Yellow');
		expect(whiteButton.style.color).not.toBe('white');
		expect(yellowButton.style.color).not.toBe('white');
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-06-15T14:30:00Z',
			color_rating: 6,
			consistency: 'soft',
			volume_estimate: 'medium',
			volume_ml: 25,
			notes: 'normal color'
		};

		render(StoolForm, { props: { onsubmit, onphotoupload, initialData } });

		expect((screen.getByLabelText(/consistency/i) as HTMLSelectElement).value).toBe('soft');
		expect((screen.getByLabelText(/volume estimate/i) as HTMLSelectElement).value).toBe('medium');
		expect((screen.getByLabelText(/volume \(mL\)/i) as HTMLInputElement).value).toBe('25');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('normal color');
		expect(getSwatchButton('Green').getAttribute('aria-pressed')).toBe('true');
		expect(screen.getByRole('button', { name: /update stool/i })).toBeInTheDocument();
	});
});
