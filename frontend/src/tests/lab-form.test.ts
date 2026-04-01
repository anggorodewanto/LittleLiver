import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import LabForm from '$lib/components/LabForm.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

import { apiClient } from '$lib/api';

describe('LabForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		vi.mocked(apiClient.get).mockResolvedValue([]);
	});

	it('renders timestamp, test name, value, unit, normal range, and notes fields', () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		expect(screen.getByLabelText(/timestamp/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/test name/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/^value$/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/unit/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/normal range/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('renders quick-pick buttons for common lab tests', () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

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
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('total_bilirubin');
		expect(unitInput.value).toBe('mg/dL');
	});

	it('clicking ALT quick-pick pre-fills test_name and unit', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.click(screen.getByRole('button', { name: /^ALT$/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('ALT');
		expect(unitInput.value).toBe('U/L');
	});

	it('clicking INR quick-pick pre-fills test_name with empty unit', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.click(screen.getByRole('button', { name: /^INR$/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('INR');
		expect(unitInput.value).toBe('');
	});

	it('clicking platelets quick-pick pre-fills correctly', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.click(screen.getByRole('button', { name: /platelets/i }));

		const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
		const unitInput = screen.getByLabelText(/unit/i) as HTMLInputElement;
		expect(testNameInput.value).toBe('platelets');
		expect(unitInput.value).toContain('10');
	});

	it('validates that test name is required', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.input(screen.getByLabelText(/^value$/i), {
			target: { value: '1.5' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		expect(screen.getByText(/test name is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('validates that value is required', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.input(screen.getByLabelText(/test name/i), {
			target: { value: 'total_bilirubin' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /log lab/i }));

		expect(screen.getByText(/value is required/i)).toBeInTheDocument();
		expect(onsubmit).not.toHaveBeenCalled();
	});

	it('submits correct payload with required fields', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

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
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

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
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

		await fireEvent.input(screen.getByLabelText(/test name/i), {
			target: { value: 'some_custom_test' }
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
		render(LabForm, { props: { onsubmit, babyId: 'baby-1', submitting: true } });

		expect(screen.getByRole('button', { name: /logging/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1', error: 'Server error' } });

		expect(screen.getByText('Server error')).toBeInTheDocument();
	});

	it('highlights selected quick-pick button', async () => {
		render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

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

		render(LabForm, { props: { onsubmit, babyId: 'baby-1', initialData } });

		expect((screen.getByLabelText(/test name/i) as HTMLInputElement).value).toBe('total_bilirubin');
		expect((screen.getByLabelText(/^value$/i) as HTMLInputElement).value).toBe('1.5');
		expect((screen.getByLabelText(/unit/i) as HTMLInputElement).value).toBe('mg/dL');
		expect((screen.getByLabelText(/normal range/i) as HTMLInputElement).value).toBe('0.1-1.2');
		expect((screen.getByLabelText(/notes/i) as HTMLTextAreaElement).value).toBe('slightly elevated');
		expect(screen.getByRole('button', { name: /update lab/i })).toBeInTheDocument();
	});

	describe('batch entry (Add More)', () => {
		it('shows "Add More" button in create mode', () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			expect(screen.getByRole('button', { name: /add more/i })).toBeInTheDocument();
		});

		it('does not show "Add More" button in edit mode', () => {
			const initialData = {
				timestamp: '2025-01-15T10:30:00Z',
				test_name: 'total_bilirubin',
				value: '1.5'
			};
			render(LabForm, { props: { onsubmit, babyId: 'baby-1', initialData } });

			expect(screen.queryByRole('button', { name: /add more/i })).not.toBeInTheDocument();
		});

		it('validates current entry before adding', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Click add more with empty form
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			expect(screen.getByText(/test name is required/i)).toBeInTheDocument();
		});

		it('adds entry to saved list and resets form fields', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Fill in a lab entry
			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '2.5' }
			});

			// Click Add More
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			// Entry should appear in saved list with value
			expect(screen.getByText(/2\.5/)).toBeInTheDocument();

			// Form fields should be reset
			expect((screen.getByLabelText(/test name/i) as HTMLInputElement).value).toBe('');
			expect((screen.getByLabelText(/^value$/i) as HTMLInputElement).value).toBe('');
			expect((screen.getByLabelText(/unit/i) as HTMLInputElement).value).toBe('');
		});

		it('preserves timestamp after adding entry', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			const timestampInput = screen.getByLabelText(/timestamp/i) as HTMLInputElement;
			const originalTimestamp = timestampInput.value;

			await fireEvent.click(screen.getByRole('button', { name: /^ALT$/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '30' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			expect((screen.getByLabelText(/timestamp/i) as HTMLInputElement).value).toBe(originalTimestamp);
		});

		it('can remove a saved entry', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Add an entry
			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '2.5' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			// Verify it's there (value displayed in saved entries)
			expect(screen.getByText(/2\.5/)).toBeInTheDocument();

			// Remove it
			await fireEvent.click(screen.getByRole('button', { name: /remove/i }));

			// Should be gone (only the quick-pick button remains, not the saved entry summary)
			expect(screen.queryByText(/2\.5/)).not.toBeInTheDocument();
		});

		it('submits all saved entries plus current form as array', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Add first entry
			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '2.5' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			// Fill second entry in form
			await fireEvent.click(screen.getByRole('button', { name: /^ALT$/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '45' }
			});

			// Submit
			await fireEvent.click(screen.getByRole('button', { name: /log labs/i }));

			expect(onsubmit).toHaveBeenCalledTimes(1);
			const payload = onsubmit.mock.calls[0][0];
			expect(Array.isArray(payload)).toBe(true);
			expect(payload).toHaveLength(2);
			expect(payload[0].test_name).toBe('total_bilirubin');
			expect(payload[0].value).toBe('2.5');
			expect(payload[1].test_name).toBe('ALT');
			expect(payload[1].value).toBe('45');
		});

		it('submits only saved entries when current form is empty', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Add an entry
			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '2.5' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			// Leave form empty and submit
			await fireEvent.click(screen.getByRole('button', { name: /log labs/i }));

			expect(onsubmit).toHaveBeenCalledTimes(1);
			const payload = onsubmit.mock.calls[0][0];
			expect(Array.isArray(payload)).toBe(true);
			expect(payload).toHaveLength(1);
			expect(payload[0].test_name).toBe('total_bilirubin');
		});

		it('shows entry count on submit button when entries are saved', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Add an entry
			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '2.5' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			// Submit button should indicate batch
			expect(screen.getByRole('button', { name: /log labs/i })).toBeInTheDocument();
		});

		it('shares the same timestamp across all batch entries', async () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Add first entry
			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '2.5' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /add more/i }));

			// Add second entry via form and submit
			await fireEvent.click(screen.getByRole('button', { name: /^ALT$/i }));
			await fireEvent.input(screen.getByLabelText(/^value$/i), {
				target: { value: '45' }
			});
			await fireEvent.click(screen.getByRole('button', { name: /log labs/i }));

			const payload = onsubmit.mock.calls[0][0];
			expect(payload[0].timestamp).toBe(payload[1].timestamp);
		});
	});

	describe('test suggestions', () => {
		it('renders a datalist for test name suggestions', () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			const datalist = document.getElementById('lab-test-suggestions');
			expect(datalist).toBeInTheDocument();
			expect(datalist!.tagName).toBe('DATALIST');

			const input = screen.getByLabelText(/test name/i) as HTMLInputElement;
			expect(input.getAttribute('list')).toBe('lab-test-suggestions');
		});

		it('includes QUICK_PICKS in datalist when API returns empty', async () => {
			vi.mocked(apiClient.get).mockResolvedValue([]);
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			await waitFor(() => {
				const datalist = document.getElementById('lab-test-suggestions');
				const options = datalist!.querySelectorAll('option');
				expect(options.length).toBeGreaterThanOrEqual(8); // 8 QUICK_PICKS
			});
		});

		it('auto-fills unit and normal_range when selecting a known test', async () => {
			vi.mocked(apiClient.get).mockResolvedValue([
				{ test_name: 'custom_test', unit: 'mmol/L', normal_range: '3.5-5.0' }
			]);

			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			await waitFor(() => {
				const datalist = document.getElementById('lab-test-suggestions');
				const options = Array.from(datalist!.querySelectorAll('option'));
				expect(options.some(o => o.value === 'custom_test')).toBe(true);
			});

			const testNameInput = screen.getByLabelText(/test name/i) as HTMLInputElement;
			await fireEvent.input(testNameInput, { target: { value: 'custom_test' } });

			expect((screen.getByLabelText(/unit/i) as HTMLInputElement).value).toBe('mmol/L');
			expect((screen.getByLabelText(/normal range/i) as HTMLInputElement).value).toBe('3.5-5.0');
		});

		it('DB suggestions override QUICK_PICKS unit/range on quick-pick click', async () => {
			vi.mocked(apiClient.get).mockResolvedValue([
				{ test_name: 'total_bilirubin', unit: 'µmol/L', normal_range: '1.7-20.5' }
			]);

			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			await waitFor(() => {
				expect(vi.mocked(apiClient.get)).toHaveBeenCalled();
			});

			await fireEvent.click(screen.getByRole('button', { name: /total.?bilirubin/i }));

			expect((screen.getByLabelText(/unit/i) as HTMLInputElement).value).toBe('µmol/L');
			expect((screen.getByLabelText(/normal range/i) as HTMLInputElement).value).toBe('1.7-20.5');
		});

		it('handles API failure gracefully', async () => {
			vi.mocked(apiClient.get).mockRejectedValue(new Error('Network error'));

			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			// Form should still render and work
			await fireEvent.click(screen.getByRole('button', { name: /^ALT$/i }));
			expect((screen.getByLabelText(/test name/i) as HTMLInputElement).value).toBe('ALT');
		});

		it('does not auto-fill for freeform test names', async () => {
			vi.mocked(apiClient.get).mockResolvedValue([]);
			render(LabForm, { props: { onsubmit, babyId: 'baby-1' } });

			await fireEvent.input(screen.getByLabelText(/test name/i), {
				target: { value: 'some_new_test' }
			});

			expect((screen.getByLabelText(/unit/i) as HTMLInputElement).value).toBe('');
			expect((screen.getByLabelText(/normal range/i) as HTMLInputElement).value).toBe('');
		});

		it('fetches suggestions for the given babyId', () => {
			render(LabForm, { props: { onsubmit, babyId: 'baby-42' } });

			expect(vi.mocked(apiClient.get)).toHaveBeenCalledWith('/babies/baby-42/labs/tests');
		});
	});
});
