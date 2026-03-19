import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import UnlinkSection from '$lib/components/UnlinkSection.svelte';

describe('UnlinkSection', () => {
	let onunlink: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onunlink = vi.fn();
	});

	it('renders an unlink button', () => {
		render(UnlinkSection, { props: { babyName: 'Alice', onunlink } });

		expect(screen.getByRole('button', { name: /unlink/i })).toBeInTheDocument();
	});

	it('shows confirmation dialog when unlink is clicked', async () => {
		render(UnlinkSection, { props: { babyName: 'Alice', onunlink } });

		await fireEvent.click(screen.getByRole('button', { name: /unlink/i }));

		expect(screen.getByText(/are you sure/i)).toBeInTheDocument();
		expect(screen.getByText(/Alice/)).toBeInTheDocument();
	});

	it('calls onunlink when confirmation is accepted', async () => {
		render(UnlinkSection, { props: { babyName: 'Alice', onunlink } });

		await fireEvent.click(screen.getByRole('button', { name: /unlink/i }));
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		expect(onunlink).toHaveBeenCalled();
	});

	it('does not call onunlink when confirmation is cancelled', async () => {
		render(UnlinkSection, { props: { babyName: 'Alice', onunlink } });

		await fireEvent.click(screen.getByRole('button', { name: /unlink/i }));
		await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

		expect(onunlink).not.toHaveBeenCalled();
	});

	it('hides confirmation dialog after cancel', async () => {
		render(UnlinkSection, { props: { babyName: 'Alice', onunlink } });

		await fireEvent.click(screen.getByRole('button', { name: /unlink/i }));
		await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

		expect(screen.queryByText(/are you sure/i)).not.toBeInTheDocument();
	});

	it('shows error message when error prop is set', () => {
		render(UnlinkSection, {
			props: { babyName: 'Alice', onunlink, error: 'Unlink failed' }
		});

		expect(screen.getByText('Unlink failed')).toBeInTheDocument();
	});
});
