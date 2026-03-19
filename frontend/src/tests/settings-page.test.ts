import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import SettingsPage from '$lib/components/SettingsPage.svelte';
import { apiClient } from '$lib/api';
import { babies, activeBaby, _resetBabyStores, fetchBabies } from '$lib/stores/baby';
import { mockBabies } from './fixtures';

describe('SettingsPage', () => {
	beforeEach(async () => {
		vi.restoreAllMocks();
		_resetBabyStores();
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
		await fetchBabies();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('renders settings heading', () => {
		render(SettingsPage);

		expect(screen.getByRole('heading', { level: 1, name: /settings/i })).toBeInTheDocument();
	});

	it('renders baby settings form for the active baby', () => {
		render(SettingsPage);

		expect(screen.getByLabelText(/name/i)).toBeInTheDocument();
		expect((screen.getByLabelText(/name/i) as HTMLInputElement).value).toBe('Alice');
	});

	it('renders invite section', () => {
		render(SettingsPage);

		expect(screen.getByRole('button', { name: /generate invite/i })).toBeInTheDocument();
	});

	it('renders unlink section', () => {
		render(SettingsPage);

		expect(screen.getByRole('button', { name: /unlink/i })).toBeInTheDocument();
	});

	it('renders account deletion section', () => {
		render(SettingsPage);

		expect(screen.getByRole('button', { name: /delete account/i })).toBeInTheDocument();
	});

	it('generates invite and displays code', async () => {
		const inviteResp = { code: 'XYZ789', expires_at: '2026-03-21T12:00:00Z' };
		vi.spyOn(apiClient, 'post').mockResolvedValue(inviteResp);

		render(SettingsPage);

		await fireEvent.click(screen.getByRole('button', { name: /generate invite/i }));

		await waitFor(() => {
			expect(screen.getByText('XYZ789')).toBeInTheDocument();
		});
	});

	it('calls PUT on baby settings save', async () => {
		const updatedBaby = { ...mockBabies[0], name: 'Updated Alice' };
		const putSpy = vi.spyOn(apiClient, 'put').mockResolvedValue(updatedBaby);

		render(SettingsPage);

		await fireEvent.input(screen.getByLabelText(/name/i), { target: { value: 'Updated Alice' } });
		await fireEvent.click(screen.getByRole('button', { name: /save/i }));

		await waitFor(() => {
			expect(putSpy).toHaveBeenCalled();
		});
	});

	it('calls DELETE on unlink confirmation', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: [mockBabies[1]] });
		const delSpy = vi.spyOn(apiClient, 'del').mockResolvedValue(undefined);

		render(SettingsPage);

		await fireEvent.click(screen.getByRole('button', { name: /unlink/i }));
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(delSpy).toHaveBeenCalledWith('/babies/baby-1/parents/me');
		});
	});

	it('calls DELETE on account deletion confirmation', async () => {
		const delSpy = vi.spyOn(apiClient, 'del').mockResolvedValue(undefined);

		render(SettingsPage);

		await fireEvent.click(screen.getByRole('button', { name: /delete account/i }));
		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(delSpy).toHaveBeenCalledWith('/users/me');
		});
	});

	it('shows message when no active baby', async () => {
		_resetBabyStores();

		render(SettingsPage);

		expect(screen.getByText(/no baby selected/i)).toBeInTheDocument();
	});
});
