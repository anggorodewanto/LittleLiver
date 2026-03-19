import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import AccountDeletion from '$lib/components/AccountDeletion.svelte';

describe('AccountDeletion', () => {
	let ondelete: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		ondelete = vi.fn();
	});

	it('renders a delete account button', () => {
		render(AccountDeletion, { props: { ondelete } });

		expect(screen.getByRole('button', { name: /delete account/i })).toBeInTheDocument();
	});

	it('shows confirmation dialog when delete is clicked', async () => {
		render(AccountDeletion, { props: { ondelete } });

		await fireEvent.click(screen.getByRole('button', { name: /delete account/i }));

		expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
		expect(screen.getByText(/cannot be undone/i)).toBeInTheDocument();
	});

	it('calls ondelete when confirmation is accepted', async () => {
		render(AccountDeletion, { props: { ondelete } });

		await fireEvent.click(screen.getByRole('button', { name: /delete account/i }));
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		expect(ondelete).toHaveBeenCalled();
	});

	it('does not call ondelete when confirmation is cancelled', async () => {
		render(AccountDeletion, { props: { ondelete } });

		await fireEvent.click(screen.getByRole('button', { name: /delete account/i }));
		await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

		expect(ondelete).not.toHaveBeenCalled();
	});

	it('hides confirmation dialog after cancel', async () => {
		render(AccountDeletion, { props: { ondelete } });

		await fireEvent.click(screen.getByRole('button', { name: /delete account/i }));
		await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

		expect(screen.queryByText(/are you sure/i)).not.toBeInTheDocument();
	});

	it('shows error message when error prop is set', () => {
		render(AccountDeletion, { props: { ondelete, error: 'Deletion failed' } });

		expect(screen.getByText('Deletion failed')).toBeInTheDocument();
	});
});
