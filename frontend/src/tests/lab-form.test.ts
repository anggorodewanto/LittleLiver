import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import LabForm from '$lib/components/LabForm.svelte';

describe('LabForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders timestamp, test name, value, unit, normal range, and notes fields', () => {
		render(LabForm, { props: { onsubmit } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/test name/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/^value$/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/unit/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/normal range/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('renders quick-pick buttons for common lab tests', () => {
		render(LabForm, { props: { onsubmit } });

		expect(screen.getByRole('button', { name: /total.?bilirubin/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /direct.?bilirubin/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^ALT$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^AST$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^GGT$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /albumin/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^INR$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /platelets/i })).toBeInTheDocument();
	});

	it('clicking total bilirubin quick-pick pre-fills test_name and unit', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('total_bilirubin');
		expect(unitInput.value).toBe('mg/dL');
	});

	it('clicking ALT quick-pick pre-fills test_name and unit', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /^ALT$/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('ALT');
		expect(unitInput.value).toBe('U/L');
	});

	it('clicking INR quick-pick pre-fills test_name with empty unit', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /^INR$/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('INR');
		expect(unitInput.value).toBe('');
	});

	it('clicking platelets quick-pick pre-fills correctly', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.click(screen.getByRole('button', { name: /platelets/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('platelets');
		expect(unitInput.value).toContain('10');
	});

	it('validates that test name is required', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/^value$/i), {
			target: { value: '1.5' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		expect(screen.getByText(/test name is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates that value is required', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/test name/i), {
			target: { value: 'total_bilirubin' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		expect(screen.getByText(/value is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/test name/i), {
			target: { value: 'total_bilirubin' }
		});
		await fireEvent.input(screen.getByLabelText(/^value$/i), {
			target: { value: '1.5' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.test_name).toBe('total_bilirubin');
		expect(payload.value).toBe('1.5');
		expect(payload.timestamp).toBeDefined();
	});

	it('submits correct payload with all fields', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/test name/i), {
			target: { value: 'ALT' }
		});
		await fireEvent.input(screen.getByLabelText(/^value$/i), {
			target: { value: '45' }
		});
		await fireEvent.input(screen.getByLabelText(/unit/i), {
			target: { value: 'U/L' }
		});
		await fireEvent.input(screen.getByLabelText(/normal range/i), {
			target: { value: '7-56' }
		});
		await fireEvent.input(screen.getByLabelText(/notes/i), {
			target: { value: 'slightly elevated' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.test_name).toBe('ALT');
		expect(payload.value).toBe('45');
		expect(payload.unit).toBe('U/L');
		expect(payload.normal_range).toBe('7-56');
		expect(payload.notes).toBe('slightly elevated');
	});

	it('omits optional fields when not provided', async () => {
		render(LabForm, { props: { onsubmit } });

		await fireEvent.input(screen.getByLabelText(/test name/i), {
			target: { value: 'GGT' }
		});
		await fireEvent.input(screen.getByLabelText(/^value$/i), {
			target: { value: '100' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		const payload = onsubmit.mock.calls[0][0];
		expect(payload.unit).toBeUndefined();
		expect(payload.normal_range).toBeUndefined();
		expect(payload.notes).toBeUndefined();
	});

	it('disables submit button when submitting', () => {
		render(LabForm, { props: { onsubmit, submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(LabForm, { props: { onsubmit, error: 'Server error' } });

		expect(screen.getByText('Server error')).toBeInTheDocument();
	});

	it('highlights selected quick-pick button', async () => {
		render(LabForm, { props: { onsubmit } });

		const altButton = screen.getByRole('button', { name: /^ALT$/i });
		await fireEvent.click(altButton);

		expect(altButton.getAttribute('aria-pressed')).toBe('true');
	});

	it('pre-populates fields when initialData is provided', () => {
		const initialData = {
			timestamp: '2025-01-15T10:30:00Z',
			test_name: 'total_bilirubin',
			value: '1.5',
			unit: 'mg/dL',
			normal_range: '0.1-1.2',
			notes: 'slightly elevated'
		};

		render(LabForm, { props: { onsubmit, initialData } });

		expect((screen.getByLabelText(/test name/i) as HTMLInputElement).value).toBe('total_bilirubin');
		expect((screen.getByLabelText(/^value$/i) as HTMLInputElement).value).toBe('1.5');
		expect((screen.getByLabelText(/unit/i) as HTMLInputElement).value).toBe('mg/dL');
		expect((screen.getByLabelText(/normal range/i) as HTMLInputElement).value).toBe('0.1-1.2');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('slightly elevated');
		expect(screen.getByRole('button', { name: /update lab/i })).toBeInTheDocument();
	});
});
