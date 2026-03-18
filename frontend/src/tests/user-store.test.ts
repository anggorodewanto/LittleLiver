import { describe, it, expect, vi, beforeEach } from 'vitest';
import { get } from 'svelte/store';
import { _resetCsrfToken } from '$lib/api';

// We need to mock fetch before importing the store
const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('user store', () => {
	beforeEach(() => {
		vi.resetModules();
		mockFetch.mockReset();
		_resetCsrfToken();
	});

	it('exports a currentUser writable store', async () => {
		const { currentUser } = await import('$lib/stores/user');
		expect(currentUser).toBeDefined();
		expect(get(currentUser)).toBeNull();
	});

	it('exports an isAuthenticated derived store', async () => {
		const { currentUser, isAuthenticated } = await import('$lib/stores/user');
		expect(get(isAuthenticated)).toBe(false);
		currentUser.set({
			id: '1',
			email: 'test@example.com',
			name: 'Test User'
		});
		expect(get(isAuthenticated)).toBe(true);
	});

	it('fetchCurrentUser updates store on successful /api/me via apiClient', async () => {
		const meResponse = {
			user: { id: '1', email: 'test@example.com', name: 'Test User' },
			babies: []
		};
		mockFetch.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () => Promise.resolve(meResponse)
		});

		const { currentUser, fetchCurrentUser } = await import('$lib/stores/user');
		await fetchCurrentUser();

		// Verify it called /api/me (via apiClient.get which prepends /api)
		expect(mockFetch).toHaveBeenCalledWith(
			'/api/me',
			expect.objectContaining({ credentials: 'include' })
		);

		const user = get(currentUser);
		expect(user).toEqual({ id: '1', email: 'test@example.com', name: 'Test User' });
	});

	it('fetchCurrentUser sets user to null on 401', async () => {
		// apiClient.get will throw on non-ok response, so we simulate that
		mockFetch.mockResolvedValueOnce({
			ok: false,
			status: 401
		});

		const originalLocation = window.location;
		Object.defineProperty(window, 'location', {
			writable: true,
			value: { ...originalLocation, href: 'http://localhost/' }
		});

		const { currentUser, fetchCurrentUser } = await import('$lib/stores/user');
		currentUser.set({ id: '1', email: 'test@example.com', name: 'Test User' });
		await fetchCurrentUser();

		expect(get(currentUser)).toBeNull();

		Object.defineProperty(window, 'location', {
			writable: true,
			value: originalLocation
		});
	});

	it('fetchCurrentUser sets user to null on network error', async () => {
		mockFetch.mockRejectedValueOnce(new Error('Network error'));

		const { currentUser, fetchCurrentUser } = await import('$lib/stores/user');
		currentUser.set({ id: '1', email: 'test@example.com', name: 'Test User' });
		await fetchCurrentUser();

		expect(get(currentUser)).toBeNull();
	});
});
