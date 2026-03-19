import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import QuickLogButtons from '$lib/components/QuickLogButtons.svelte';

describe('QuickLogButtons', () => {
	let onselect: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onselect = vi.fn();
	});

	it('renders exactly 5 quick-log buttons per spec', () => {
		render(QuickLogButtons, { props: { onselect } });

		const buttons = screen.getAllByRole('button');
		expect(buttons).toHaveLength(5);
	});

	it('renders Feed, Wet Diaper, Stool, Temp, and Medication Given buttons', () => {
		render(QuickLogButtons, { props: { onselect } });

		expect(screen.getByRole('button', { name: /feed/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /wet diaper/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /stool/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /temp/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /medication given/i })).toBeInTheDocument();
	});

	it('calls onselect with "feeding" when Feed is clicked', async () => {
		render(QuickLogButtons, { props: { onselect } });

		await fireEvent.click(screen.getByRole('button', { name: /feed/i }));

		expect(onselect).toHaveBeenCalledWith('feeding');
	});

	it('calls onselect with "urine" when Wet Diaper is clicked', async () => {
		render(QuickLogButtons, { props: { onselect } });

		await fireEvent.click(screen.getByRole('button', { name: /wet diaper/i }));

		expect(onselect).toHaveBeenCalledWith('urine');
	});

	it('calls onselect with "stool" when Stool is clicked', async () => {
		render(QuickLogButtons, { props: { onselect } });

		await fireEvent.click(screen.getByRole('button', { name: /stool/i }));

		expect(onselect).toHaveBeenCalledWith('stool');
	});

	it('calls onselect with "temperature" when Temp is clicked', async () => {
		render(QuickLogButtons, { props: { onselect } });

		await fireEvent.click(screen.getByRole('button', { name: /temp/i }));

		expect(onselect).toHaveBeenCalledWith('temperature');
	});

	it('calls onselect with "med_given" when Medication Given is clicked', async () => {
		render(QuickLogButtons, { props: { onselect } });

		await fireEvent.click(screen.getByRole('button', { name: /medication given/i }));

		expect(onselect).toHaveBeenCalledWith('med_given');
	});
});
