import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import MedicationContainerForm from '$lib/components/MedicationContainerForm.svelte';

describe('MedicationContainerForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders kind, unit, quantity, opened-at, max-days, expiration fields', () => {
		render(MedicationContainerForm, { props: { onsubmit } });
		expect(screen.getByLabelText(/container kind/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/^unit$/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/quantity \(initial\)/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/opened at/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/max days after opening/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/manufacturer expiration date/i)).toBeInTheDocument();
	});

	it('does not show quantity_remaining input on create (no initial)', () => {
		render(MedicationContainerForm, { props: { onsubmit } });
		expect(screen.queryByLabelText(/quantity remaining/i)).not.toBeInTheDocument();
	});

	it('shows quantity_remaining and depleted in edit mode', () => {
		const initial = {
			id: 'c1',
			medication_id: 'm1',
			baby_id: 'b1',
			kind: 'bottle' as const,
			unit: 'ml' as const,
			quantity_initial: 100,
			quantity_remaining: 50,
			opened_at: null,
			max_days_after_opening: null,
			expiration_date: null,
			effective_expiry: null,
			depleted: false,
			notes: null,
			created_at: '2026-01-01T00:00:00Z',
			updated_at: '2026-01-01T00:00:00Z'
		};
		render(MedicationContainerForm, { props: { onsubmit, initial } });
		expect(screen.getByLabelText(/quantity remaining/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/mark as depleted/i)).toBeInTheDocument();
	});

	it('submits a valid payload', async () => {
		render(MedicationContainerForm, { props: { onsubmit } });
		await fireEvent.input(screen.getByLabelText(/quantity \(initial\)/i), {
			target: { value: '100' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /add container/i }));

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.kind).toBe('bottle');
		expect(payload.unit).toBe('ml');
		expect(payload.quantity_initial).toBe(100);
	});

	it('rejects empty quantity_initial', async () => {
		render(MedicationContainerForm, { props: { onsubmit } });
		await fireEvent.click(screen.getByRole('button', { name: /add container/i }));

		expect(onsubmit).not.toHaveBeenCalled();
		expect(screen.getByRole('alert')).toBeInTheDocument();
	});
});
