import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import JoinBabyForm from '$lib/components/JoinBabyForm.svelte';

describe('JoinBabyForm', () => {
	let onjoin: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onjoin = vi.fn();
	});

	it('renders an invite code input', () => {
		render(JoinBabyForm, { props: { onjoin } });

		expect(screen.getByLabelText(/invite code/i)).toBeInTheDocument();
	});

	it('renders a submit button', () => {
		render(JoinBabyForm, { props: { onjoin } });

		expect(screen.getByRole('button', { name: /join/i })).toBeInTheDocument();
	});

	it('shows validation error when submitting empty code', async () => {
		render(JoinBabyForm, { props: { onjoin } });

		await fireEvent.click(screen.getByRole('button', { name: /join/i }));

		expect(screen.getByText(/invite code is required/i)).toBeInTheDocument();
		expect(onjoin).not.toHaveBeenCalled();
	});

	it('calls onjoin with code when valid', async () => {
		render(JoinBabyForm, { props: { onjoin } });

		await fireEvent.input(screen.getByLabelText(/invite code/i), {
			target: { value: 'ABC123' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /join/i }));

		expect(onjoin).toHaveBeenCalledWith('ABC123');
	});

	it('trims whitespace from code', async () => {
		render(JoinBabyForm, { props: { onjoin } });

		await fireEvent.input(screen.getByLabelText(/invite code/i), {
			target: { value: '  ABC123  ' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /join/i }));

		expect(onjoin).toHaveBeenCalledWith('ABC123');
	});

	it('disables submit button when submitting prop is true', () => {
		render(JoinBabyForm, { props: { onjoin, submitting: true } });

		expect(screen.getByRole('button', { name: /joining/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(JoinBabyForm, { props: { onjoin, error: 'Invalid code' } });

		expect(screen.getByText('Invalid code')).toBeInTheDocument();
	});

	it('calls preventDefault on form submission', async () => {
		render(JoinBabyForm, { props: { onjoin } });

		await fireEvent.input(screen.getByLabelText(/invite code/i), {
			target: { value: 'ABC123' }
		});

		const form = screen.getByRole('button', { name: /join/i }).closest('form')!;
		const submitEvent = new Event('submit', { bubbles: true, cancelable: true });
		const prevented = !form.dispatchEvent(submitEvent);

		expect(prevented).toBe(true);
	});
});
