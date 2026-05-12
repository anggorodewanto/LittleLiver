import { render, screen, act } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import LoadingSpinner from '$lib/components/LoadingSpinner.svelte';

describe('LoadingSpinner', () => {
	beforeEach(() => {
		vi.useFakeTimers();
	});

	afterEach(() => {
		vi.useRealTimers();
	});

	it('renders an accessible spinner with default message', () => {
		render(LoadingSpinner);

		const status = screen.getByRole('status');
		expect(status).toBeInTheDocument();
		expect(status.textContent).toMatch(/loading/i);
	});

	it('renders a custom message when provided', () => {
		render(LoadingSpinner, { props: { message: 'Fetching baby data…' } });

		expect(screen.getByRole('status').textContent).toMatch(/fetching baby data/i);
	});

	it('does not show slow message before slowAfterMs elapses', () => {
		render(LoadingSpinner, {
			props: { slowAfterMs: 3000, slowMessage: 'Waking server up…' }
		});

		expect(screen.queryByText(/waking server up/i)).toBeNull();
	});

	it('shows slow message once slowAfterMs elapses', async () => {
		render(LoadingSpinner, {
			props: { slowAfterMs: 3000, slowMessage: 'Waking server up…' }
		});

		await act(async () => {
			vi.advanceTimersByTime(3000);
		});

		expect(screen.getByText(/waking server up/i)).toBeInTheDocument();
	});

	it('renders a visual spinner element', () => {
		const { container } = render(LoadingSpinner);

		expect(container.querySelector('.spinner')).not.toBeNull();
	});
});
