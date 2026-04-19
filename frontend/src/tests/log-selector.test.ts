import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

import { goto } from '$app/navigation';
import LogSelector from '../routes/log/+page.svelte';

describe('Log Selector Page', () => {
	it('renders a heading', () => {
		render(LogSelector);

		expect(screen.getByRole('heading', { name: /what to log/i })).toBeInTheDocument();
	});

	it('shows quick log buttons (Feed, Wet Diaper, Stool, Temp, Medication Given)', () => {
		render(LogSelector);

		expect(screen.getByRole('button', { name: /feed/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /wet diaper/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /stool/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /temp/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /medication given/i })).toBeInTheDocument();
	});

	it('navigates to /log/feeding when Feed is clicked', async () => {
		render(LogSelector);

		await fireEvent.click(screen.getByRole('button', { name: /feed/i }));

		expect(goto).toHaveBeenCalledWith('/log/feeding');
	});

	it('navigates to /log/med when Medication Given is clicked', async () => {
		render(LogSelector);

		await fireEvent.click(screen.getByRole('button', { name: /medication given/i }));

		expect(goto).toHaveBeenCalledWith('/log/med');
	});

	it('navigates to /log/stool when Stool is clicked', async () => {
		render(LogSelector);

		await fireEvent.click(screen.getByRole('button', { name: /stool/i }));

		expect(goto).toHaveBeenCalledWith('/log/stool');
	});

	it('shows a back link to home', () => {
		render(LogSelector);

		const backLink = screen.getByRole('link', { name: /back/i });
		expect(backLink).toBeInTheDocument();
		expect(backLink.getAttribute('href')).toBe('/');
	});

	it('navigates to /medications when Manage Medications is clicked', async () => {
		render(LogSelector);

		await fireEvent.click(screen.getByRole('button', { name: /manage medications/i }));

		expect(goto).toHaveBeenCalledWith('/medications');
	});
});
