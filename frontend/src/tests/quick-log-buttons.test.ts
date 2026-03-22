import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import QuickLogButtons from '$lib/components/QuickLogButtons.svelte';

describe('QuickLogButtons', () => {
	let onselect: ReturnType<typeof vi.fn>;
	let onnavigate: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onselect = vi.fn();
		onnavigate = vi.fn();
	});

	it('renders 5 quick-log buttons plus a More Entries toggle', () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		const buttons = screen.getAllByRole('button');
		// 5 quick-log + 1 "More Entries" toggle = 6
		expect(buttons).toHaveLength(6);
	});

	it('renders Feed, Wet Diaper, Stool, Temp, and Medication Given buttons', () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		expect(screen.getByRole('button', { name: /feed/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /wet diaper/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /stool/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /temp/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /medication given/i })).toBeInTheDocument();
	});

	it('calls onselect with "feeding" when Feed is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /feed/i }));

		expect(onselect).toHaveBeenCalledWith('feeding');
	});

	it('calls onselect with "urine" when Wet Diaper is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /wet diaper/i }));

		expect(onselect).toHaveBeenCalledWith('urine');
	});

	it('calls onselect with "stool" when Stool is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /stool/i }));

		expect(onselect).toHaveBeenCalledWith('stool');
	});

	it('calls onselect with "temperature" when Temp is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /temp/i }));

		expect(onselect).toHaveBeenCalledWith('temperature');
	});

	it('calls onselect with "med_given" when Medication Given is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /medication given/i }));

		expect(onselect).toHaveBeenCalledWith('med_given');
	});

	it('shows More Entries toggle button', () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		expect(screen.getByRole('button', { name: /more entries/i })).toBeInTheDocument();
	});

	it('expands to show extra buttons when More Entries is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /more entries/i }));

		expect(screen.getByRole('button', { name: /weight/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /abdomen/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /skin/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /bruising/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /lab/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /notes/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /manage medications/i })).toBeInTheDocument();
	});

	it('calls onselect with "weight" when Weight is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /more entries/i }));
		await fireEvent.click(screen.getByRole('button', { name: /weight/i }));

		expect(onselect).toHaveBeenCalledWith('weight');
	});

	it('calls onnavigate with "/medications" when Manage Medications is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /more entries/i }));
		await fireEvent.click(screen.getByRole('button', { name: /manage medications/i }));

		expect(onnavigate).toHaveBeenCalledWith('/medications');
	});

	it('collapses extra buttons when Less Entries is clicked', async () => {
		render(QuickLogButtons, { props: { onselect, onnavigate } });

		await fireEvent.click(screen.getByRole('button', { name: /more entries/i }));
		expect(screen.getByRole('button', { name: /less entries/i })).toBeInTheDocument();

		await fireEvent.click(screen.getByRole('button', { name: /less entries/i }));
		expect(screen.queryByRole('button', { name: /weight/i })).not.toBeInTheDocument();
	});
});
