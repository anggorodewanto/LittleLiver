import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import LoadMoreButton from '$lib/components/LoadMoreButton.svelte';

describe('LoadMoreButton', () => {
	it('renders nothing when hasMore is false', () => {
		const { container } = render(LoadMoreButton, {
			props: { hasMore: false, onloadmore: vi.fn() }
		});

		expect(container.textContent).toBe('');
	});

	it('renders "Load more" button when hasMore is true', () => {
		render(LoadMoreButton, {
			props: { hasMore: true, onloadmore: vi.fn() }
		});

		expect(screen.getByRole('button', { name: /load more/i })).toBeInTheDocument();
	});

	it('shows "Loading..." when loading is true', () => {
		render(LoadMoreButton, {
			props: { hasMore: true, loading: true, onloadmore: vi.fn() }
		});

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
		expect(screen.queryByRole('button')).not.toBeInTheDocument();
	});

	it('calls onloadmore when button is clicked', async () => {
		const onloadmore = vi.fn();
		render(LoadMoreButton, {
			props: { hasMore: true, onloadmore }
		});

		await fireEvent.click(screen.getByRole('button', { name: /load more/i }));
		expect(onloadmore).toHaveBeenCalledTimes(1);
	});
});
