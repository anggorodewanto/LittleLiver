import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import LabExtractionReview from '$lib/components/LabExtractionReview.svelte';

describe('LabExtractionReview', () => {
	let onconfirm: ReturnType<typeof vi.fn>;
	let oncancel: ReturnType<typeof vi.fn>;

	const mockExtracted = [
		{
			test_name: 'total_bilirubin',
			value: '1.5',
			unit: 'mg/dL',
			normal_range: '0.1-1.2',
			confidence: 'high'
		},
		{
			test_name: 'ALT',
			value: '45',
			unit: 'U/L',
			normal_range: '7-56',
			confidence: 'medium'
		},
		{
			test_name: 'AST',
			value: '38',
			unit: 'U/L',
			normal_range: '10-40',
			confidence: 'low'
		}
	];

	beforeEach(() => {
		onconfirm = vi.fn();
		oncancel = vi.fn();
	});

	it('renders all extracted results with editable fields', () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		// All test names should be visible
		expect(screen.getByDisplayValue('total_bilirubin')).toBeInTheDocument();
		expect(screen.getByDisplayValue('ALT')).toBeInTheDocument();
		expect(screen.getByDisplayValue('AST')).toBeInTheDocument();

		// Values should be displayed
		expect(screen.getByDisplayValue('1.5')).toBeInTheDocument();
		expect(screen.getByDisplayValue('45')).toBeInTheDocument();
		expect(screen.getByDisplayValue('38')).toBeInTheDocument();

		// Units should be displayed
		const unitInputs = screen.getAllByDisplayValue('U/L');
		expect(unitInputs.length).toBe(2);
		expect(screen.getByDisplayValue('mg/dL')).toBeInTheDocument();

		// Normal ranges
		expect(screen.getByDisplayValue('0.1-1.2')).toBeInTheDocument();
		expect(screen.getByDisplayValue('7-56')).toBeInTheDocument();
		expect(screen.getByDisplayValue('10-40')).toBeInTheDocument();
	});

	it('highlights medium confidence in yellow', () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		const mediumIndicator = screen.getByTestId('confidence-1');
		expect(mediumIndicator.textContent).toContain('medium');
		expect(mediumIndicator.classList.contains('confidence-medium')).toBe(true);
	});

	it('highlights low confidence in red', () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		const lowIndicator = screen.getByTestId('confidence-2');
		expect(lowIndicator.textContent).toContain('low');
		expect(lowIndicator.classList.contains('confidence-low')).toBe(true);
	});

	it('removes a row when delete button is clicked', async () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		// Should have 3 rows initially
		expect(screen.getByDisplayValue('total_bilirubin')).toBeInTheDocument();
		expect(screen.getByDisplayValue('ALT')).toBeInTheDocument();
		expect(screen.getByDisplayValue('AST')).toBeInTheDocument();

		// Remove the first row
		const removeButtons = screen.getAllByRole('button', { name: /remove/i });
		await fireEvent.click(removeButtons[0]);

		// total_bilirubin row should be gone
		expect(screen.queryByDisplayValue('total_bilirubin')).not.toBeInTheDocument();
		// Other rows remain
		expect(screen.getByDisplayValue('ALT')).toBeInTheDocument();
		expect(screen.getByDisplayValue('AST')).toBeInTheDocument();
	});

	it('reflects edited field values in confirm payload', async () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		// Edit the value of the first row
		const valueInput = screen.getByDisplayValue('1.5');
		await fireEvent.input(valueInput, { target: { value: '2.0' } });

		// Confirm
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		expect(onconfirm).toHaveBeenCalledTimes(1);
		const payload = onconfirm.mock.calls[0][0];
		expect(payload[0].value).toBe('2.0');
	});

	it('calls onconfirm with all checked rows on confirm', async () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		expect(onconfirm).toHaveBeenCalledTimes(1);
		const payload = onconfirm.mock.calls[0][0];
		expect(payload).toHaveLength(3);
		expect(payload[0].test_name).toBe('total_bilirubin');
		expect(payload[0].value).toBe('1.5');
		expect(payload[0].unit).toBe('mg/dL');
		expect(payload[0].normal_range).toBe('0.1-1.2');
		expect(payload[1].test_name).toBe('ALT');
		expect(payload[2].test_name).toBe('AST');
	});

	it('calls oncancel when cancel button is clicked', async () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

		expect(oncancel).toHaveBeenCalledTimes(1);
	});

	it('shows helpful message when extracted array is empty', () => {
		render(LabExtractionReview, {
			props: { extracted: [], notes: '', onconfirm, oncancel }
		});

		expect(screen.getByText(/no lab results found/i)).toBeInTheDocument();
	});

	it('shows "Already logged" badge for items with existing_match', () => {
		const extractedWithDuplicate = [
			{
				test_name: 'total_bilirubin',
				value: '1.5',
				unit: 'mg/dL',
				normal_range: '0.1-1.2',
				confidence: 'high',
				existing_match: {
					id: 'existing-1',
					timestamp: '2025-01-15T10:30:00Z',
					value: '1.5',
					unit: 'mg/dL'
				}
			}
		];

		render(LabExtractionReview, {
			props: { extracted: extractedWithDuplicate, notes: '', onconfirm, oncancel }
		});

		expect(screen.getByText(/already logged/i)).toBeInTheDocument();
	});

	it('unchecks duplicate rows by default', async () => {
		const extractedWithDuplicate = [
			{
				test_name: 'total_bilirubin',
				value: '1.5',
				unit: 'mg/dL',
				normal_range: '0.1-1.2',
				confidence: 'high',
				existing_match: {
					id: 'existing-1',
					timestamp: '2025-01-15T10:30:00Z',
					value: '1.5',
					unit: 'mg/dL'
				}
			},
			{
				test_name: 'ALT',
				value: '45',
				unit: 'U/L',
				normal_range: '7-56',
				confidence: 'high'
			}
		];

		render(LabExtractionReview, {
			props: { extracted: extractedWithDuplicate, notes: '', onconfirm, oncancel }
		});

		// Confirm without changing anything - duplicate should be excluded
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		const payload = onconfirm.mock.calls[0][0];
		expect(payload).toHaveLength(1);
		expect(payload[0].test_name).toBe('ALT');
	});

	it('includes duplicate row when user checks it (override)', async () => {
		const extractedWithDuplicate = [
			{
				test_name: 'total_bilirubin',
				value: '1.5',
				unit: 'mg/dL',
				normal_range: '0.1-1.2',
				confidence: 'high',
				existing_match: {
					id: 'existing-1',
					timestamp: '2025-01-15T10:30:00Z',
					value: '1.5',
					unit: 'mg/dL'
				}
			}
		];

		render(LabExtractionReview, {
			props: { extracted: extractedWithDuplicate, notes: '', onconfirm, oncancel }
		});

		// Check the duplicate row
		const checkbox = screen.getByRole('checkbox');
		await fireEvent.click(checkbox);

		// Confirm
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		const payload = onconfirm.mock.calls[0][0];
		expect(payload).toHaveLength(1);
		expect(payload[0].test_name).toBe('total_bilirubin');
	});

	it('handles mixed results: 1 duplicate unchecked, 2 new checked by default', async () => {
		const mixed = [
			{
				test_name: 'total_bilirubin',
				value: '1.5',
				unit: 'mg/dL',
				normal_range: '0.1-1.2',
				confidence: 'high',
				existing_match: {
					id: 'existing-1',
					timestamp: '2025-01-15T10:30:00Z',
					value: '1.5',
					unit: 'mg/dL'
				}
			},
			{
				test_name: 'ALT',
				value: '45',
				unit: 'U/L',
				normal_range: '7-56',
				confidence: 'high'
			},
			{
				test_name: 'AST',
				value: '38',
				unit: 'U/L',
				normal_range: '10-40',
				confidence: 'medium'
			}
		];

		render(LabExtractionReview, {
			props: { extracted: mixed, notes: '', onconfirm, oncancel }
		});

		// Confirm without changes
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		const payload = onconfirm.mock.calls[0][0];
		expect(payload).toHaveLength(2);
		expect(payload[0].test_name).toBe('ALT');
		expect(payload[1].test_name).toBe('AST');
	});

	it('shows extraction notes when provided', () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: 'Some values were hard to read', onconfirm, oncancel }
		});

		expect(screen.getByText('Some values were hard to read')).toBeInTheDocument();
	});

	it('adds a blank row when "Add row" button is clicked', async () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		// Should have 3 rows initially
		const initialRemoveButtons = screen.getAllByRole('button', { name: /remove/i });
		expect(initialRemoveButtons).toHaveLength(3);

		// Click "Add row"
		await fireEvent.click(screen.getByRole('button', { name: /add row/i }));

		// Should now have 4 rows
		const removeButtons = screen.getAllByRole('button', { name: /remove/i });
		expect(removeButtons).toHaveLength(4);
	});

	it('new blank row has empty fields, manual confidence, and is checked', async () => {
		render(LabExtractionReview, {
			props: { extracted: [mockExtracted[0]], notes: '', onconfirm, oncancel }
		});

		await fireEvent.click(screen.getByRole('button', { name: /add row/i }));

		// Check the confidence badge on the new row (index 1)
		const badge = screen.getByTestId('confidence-1');
		expect(badge.textContent).toContain('manual');

		// Confirm to check the payload includes the blank row (checked by default)
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		const payload = onconfirm.mock.calls[0][0];
		// Original row + blank row
		expect(payload).toHaveLength(2);
		expect(payload[1].test_name).toBe('');
		expect(payload[1].value).toBe('');
	});

	it('excludes deleted rows from confirm payload', async () => {
		render(LabExtractionReview, {
			props: { extracted: mockExtracted, notes: '', onconfirm, oncancel }
		});

		// Remove the first row
		const removeButtons = screen.getAllByRole('button', { name: /remove/i });
		await fireEvent.click(removeButtons[0]);

		// Confirm
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		const payload = onconfirm.mock.calls[0][0];
		expect(payload).toHaveLength(2);
		expect(payload[0].test_name).toBe('ALT');
		expect(payload[1].test_name).toBe('AST');
	});
});
