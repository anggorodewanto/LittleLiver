import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import DateRangeSelector from '$lib/components/DateRangeSelector.svelte';

describe('DateRangeSelector', () => {
	it('renders preset range buttons (7d, 14d, 30d, 90d, Custom)', () => {
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange: vi.fn() } });

		expect(screen.getByRole('button', { name: '7d' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '14d' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '30d' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: '90d' })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /custom/i })).toBeInTheDocument();
	});

	it('highlights the currently selected range', () => {
		render(DateRangeSelector, { props: { selectedRange: '14d', onchange: vi.fn() } });

		const btn = screen.getByRole('button', { name: '14d' });
		expect(btn.classList.contains('active')).toBe(true);
	});

	it('calls onchange with the new range when a preset button is clicked', async () => {
		const onchange = vi.fn();
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange } });

		await fireEvent.click(screen.getByRole('button', { name: '30d' }));

		expect(onchange).toHaveBeenCalledWith('30d');
	});

	it('shows custom date inputs when Custom is clicked', async () => {
		const onchange = vi.fn();
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange } });

		await fireEvent.click(screen.getByRole('button', { name: /custom/i }));

		expect(screen.getByLabelText(/from/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/to/i)).toBeInTheDocument();
	});

	it('calls onchange with custom dates when Apply is clicked', async () => {
		const onchange = vi.fn();
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange } });

		await fireEvent.click(screen.getByRole('button', { name: /custom/i }));

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		await fireEvent.click(screen.getByRole('button', { name: /apply/i }));

		expect(onchange).toHaveBeenCalledWith('custom', '2026-03-01', '2026-03-15');
	});

	it('disables Apply button when custom date inputs are empty', async () => {
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange: vi.fn() } });

		await fireEvent.click(screen.getByRole('button', { name: /custom/i }));

		const applyButton = screen.getByRole('button', { name: /apply/i });
		expect(applyButton).toBeDisabled();
	});

	it('disables Apply button when only From date is filled', async () => {
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange: vi.fn() } });

		await fireEvent.click(screen.getByRole('button', { name: /custom/i }));

		const fromInput = screen.getByLabelText(/from/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });

		const applyButton = screen.getByRole('button', { name: /apply/i });
		expect(applyButton).toBeDisabled();
	});

	it('disables Apply button when From date is after To date', async () => {
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange: vi.fn() } });

		await fireEvent.click(screen.getByRole('button', { name: /custom/i }));

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-15' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-01' } });

		const applyButton = screen.getByRole('button', { name: /apply/i });
		expect(applyButton).toBeDisabled();
	});

	it('enables Apply button when both dates are filled and From <= To', async () => {
		render(DateRangeSelector, { props: { selectedRange: '7d', onchange: vi.fn() } });

		await fireEvent.click(screen.getByRole('button', { name: /custom/i }));

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		const applyButton = screen.getByRole('button', { name: /apply/i });
		expect(applyButton).not.toBeDisabled();
	});
});
