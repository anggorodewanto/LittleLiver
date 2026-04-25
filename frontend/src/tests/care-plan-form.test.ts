import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import CarePlanForm, { generatePhases, validatePhases } from '$lib/components/CarePlanForm.svelte';

describe('generatePhases', () => {
	it('produces N rows with monthly spacing', () => {
		const phases = generatePhases(3, 'months', '2026-05-01', 'Phase {n}');
		expect(phases).toHaveLength(3);
		expect(phases[0]).toMatchObject({ seq: 1, label: 'Phase 1', start_date: '2026-05-01' });
		expect(phases[1]).toMatchObject({ seq: 2, label: 'Phase 2', start_date: '2026-06-01' });
		expect(phases[2]).toMatchObject({ seq: 3, label: 'Phase 3', start_date: '2026-07-01' });
	});

	it('produces weekly spacing', () => {
		const phases = generatePhases(3, 'weeks', '2026-05-01', 'P{n}');
		expect(phases.map((p) => p.start_date)).toEqual(['2026-05-01', '2026-05-08', '2026-05-15']);
	});

	it('returns empty when count < 1', () => {
		expect(generatePhases(0, 'days', '2026-05-01', 'X')).toEqual([]);
	});

	it('returns empty for invalid start_date', () => {
		expect(generatePhases(2, 'days', 'not-a-date', 'X')).toEqual([]);
	});
});

describe('validatePhases', () => {
	it('rejects empty list', () => {
		expect(validatePhases([])).toMatch(/at least one/i);
	});

	it('rejects empty label', () => {
		expect(
			validatePhases([{ seq: 1, label: '   ', start_date: '2026-05-01' }])
		).toMatch(/label/i);
	});

	it('rejects non-monotonic dates', () => {
		const msg = validatePhases([
			{ seq: 1, label: 'A', start_date: '2026-06-01' },
			{ seq: 2, label: 'B', start_date: '2026-05-01' }
		]);
		expect(msg).toMatch(/after phase 1/i);
	});

	it('passes for valid monotonic phases', () => {
		expect(
			validatePhases([
				{ seq: 1, label: 'A', start_date: '2026-05-01' },
				{ seq: 2, label: 'B', start_date: '2026-06-01' }
			])
		).toBe('');
	});
});

describe('CarePlanForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;
	beforeEach(() => {
		onsubmit = vi.fn();
	});

	it('renders name + generator fields', () => {
		render(CarePlanForm, { props: { onsubmit } });
		expect(screen.getByLabelText(/plan name/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/count/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/start date/i)).toBeInTheDocument();
	});

	it('shows alert when manual edit makes a date non-monotonic', async () => {
		render(CarePlanForm, {
			props: {
				onsubmit,
				initialData: {
					name: 'Plan',
					phases: [
						{ seq: 1, label: 'A', start_date: '2026-05-01' },
						{ seq: 2, label: 'B', start_date: '2026-06-01' }
					]
				}
			}
		});

		const start2 = screen.getByLabelText(/phase 2 start date/i) as HTMLInputElement;
		await fireEvent.input(start2, { target: { value: '2026-04-01' } });

		expect(screen.getByRole('alert').textContent).toMatch(/after phase 1/i);
	});

	it('calls onsubmit with phases when valid', async () => {
		render(CarePlanForm, {
			props: {
				onsubmit,
				initialData: {
					name: 'Plan',
					phases: [
						{ seq: 1, label: 'A', start_date: '2026-05-01' },
						{ seq: 2, label: 'B', start_date: '2026-06-01' }
					]
				}
			}
		});
		const form = screen.getByRole('button', { name: /save plan/i }).closest('form')!;
		await fireEvent.submit(form);

		expect(onsubmit).toHaveBeenCalledOnce();
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.name).toBe('Plan');
		expect(payload.phases).toHaveLength(2);
	});
});
