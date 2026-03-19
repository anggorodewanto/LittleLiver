import { render, screen, fireEvent, waitFor, cleanup } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import InviteSection from '$lib/components/InviteSection.svelte';

describe('InviteSection', () => {
	let ongenerate: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		ongenerate = vi.fn();
	});

	it('renders a generate invite button', () => {
		render(InviteSection, { props: { ongenerate } });

		expect(screen.getByRole('button', { name: /generate invite/i })).toBeInTheDocument();
	});

	it('calls ongenerate when generate button is clicked', async () => {
		render(InviteSection, { props: { ongenerate } });

		await fireEvent.click(screen.getByRole('button', { name: /generate invite/i }));

		expect(ongenerate).toHaveBeenCalled();
	});

	it('displays invite code when provided', () => {
		render(InviteSection, {
			props: {
				ongenerate,
				inviteCode: 'ABC123',
				expiresAt: '2026-03-21T00:00:00Z'
			}
		});

		expect(screen.getByText('ABC123')).toBeInTheDocument();
	});

	it('displays expiry information when invite code is shown', () => {
		render(InviteSection, {
			props: {
				ongenerate,
				inviteCode: 'ABC123',
				expiresAt: '2026-03-21T00:00:00Z'
			}
		});

		expect(screen.getByText(/expires/i)).toBeInTheDocument();
	});

	it('shows a copy button when invite code is displayed', () => {
		render(InviteSection, {
			props: {
				ongenerate,
				inviteCode: 'ABC123',
				expiresAt: '2026-03-21T00:00:00Z'
			}
		});

		expect(screen.getByRole('button', { name: /copy/i })).toBeInTheDocument();
	});

	it('copies code to clipboard when copy button is clicked', async () => {
		const writeText = vi.fn().mockResolvedValue(undefined);
		Object.assign(navigator, { clipboard: { writeText } });

		render(InviteSection, {
			props: {
				ongenerate,
				inviteCode: 'ABC123',
				expiresAt: '2026-03-21T00:00:00Z'
			}
		});

		await fireEvent.click(screen.getByRole('button', { name: /copy/i }));

		expect(writeText).toHaveBeenCalledWith('ABC123');
	});

	it('shows loading state when generating', () => {
		render(InviteSection, { props: { ongenerate, generating: true } });

		expect(screen.getByRole('button', { name: /generating/i })).toBeDisabled();
	});

	it('shows error message when error prop is set', () => {
		render(InviteSection, { props: { ongenerate, error: 'Failed to generate' } });

		expect(screen.getByText('Failed to generate')).toBeInTheDocument();
	});

	it('displays expiry countdown with remaining time', () => {
		vi.useFakeTimers();
		const now = new Date('2026-03-20T10:00:00Z');
		vi.setSystemTime(now);

		const expiresAt = '2026-03-21T09:45:00Z';

		render(InviteSection, {
			props: {
				ongenerate,
				inviteCode: 'ABC123',
				expiresAt
			}
		});

		expect(screen.getByText(/23h 45m/)).toBeInTheDocument();

		vi.useRealTimers();
	});

	it('clears countdown interval on component destroy', () => {
		vi.useFakeTimers();
		vi.setSystemTime(new Date('2026-03-20T10:00:00Z'));

		const clearIntervalSpy = vi.spyOn(globalThis, 'clearInterval');

		const { unmount } = render(InviteSection, {
			props: {
				ongenerate,
				inviteCode: 'ABC123',
				expiresAt: '2026-03-21T09:45:00Z'
			}
		});

		unmount();

		expect(clearIntervalSpy).toHaveBeenCalled();

		clearIntervalSpy.mockRestore();
		vi.useRealTimers();
	});
});
