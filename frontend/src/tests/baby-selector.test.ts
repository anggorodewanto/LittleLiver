import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import BabySelector from '$lib/components/BabySelector.svelte';
import type { Baby } from '$lib/stores/baby';

describe('BabySelector', () => {
	const mockBabies: Baby[] = [
		{
			id: 'baby-1',
			name: 'Alice',
			date_of_birth: '2025-06-01',
			sex: 'female',
			diagnosis_date: '2025-06-15',
			kasai_date: '2025-06-20'
		},
		{
			id: 'baby-2',
			name: 'Bob',
			date_of_birth: '2025-09-01',
			sex: 'male',
			diagnosis_date: null,
			kasai_date: null
		}
	];

	let onselect: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onselect = vi.fn();
	});

	it('renders a select element with baby names', () => {
		render(BabySelector, {
			props: { babies: mockBabies, activeBabyId: 'baby-1', onselect }
		});

		const select = screen.getByRole('combobox') as HTMLSelectElement;
		expect(select).toBeInTheDocument();

		const options = Array.from(select.options);
		expect(options).toHaveLength(2);
		expect(options[0].textContent).toBe('Alice');
		expect(options[1].textContent).toBe('Bob');
	});

	it('selects the active baby', () => {
		render(BabySelector, {
			props: { babies: mockBabies, activeBabyId: 'baby-2', onselect }
		});

		const select = screen.getByRole('combobox') as HTMLSelectElement;
		expect(select.value).toBe('baby-2');
	});

	it('calls onselect when baby is changed', async () => {
		render(BabySelector, {
			props: { babies: mockBabies, activeBabyId: 'baby-1', onselect }
		});

		const select = screen.getByRole('combobox');
		await fireEvent.change(select, { target: { value: 'baby-2' } });

		expect(onselect).toHaveBeenCalledWith('baby-2');
	});

	it('renders nothing when babies list is empty', () => {
		const { container } = render(BabySelector, {
			props: { babies: [], activeBabyId: null, onselect }
		});

		expect(container.querySelector('select')).toBeNull();
	});

	it('does not render when only one baby', () => {
		const { container } = render(BabySelector, {
			props: { babies: [mockBabies[0]], activeBabyId: 'baby-1', onselect }
		});

		// With only one baby, show name but no dropdown needed
		expect(container.querySelector('select')).toBeNull();
		expect(screen.getByText('Alice')).toBeInTheDocument();
	});
});
