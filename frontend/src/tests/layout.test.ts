import { render, screen, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { createRawSnippet } from 'svelte';
import Layout from '../routes/+layout.svelte';
import { currentUser } from '$lib/stores/user';
import { _resetBabyStores } from '$lib/stores/baby';

const childrenSnippet = createRawSnippet(() => ({
	render: () => '<div data-testid="child">child content</div>'
}));

vi.mock('$lib/pwa', () => ({
	registerServiceWorker: vi.fn(),
	setupInstallPrompt: vi.fn(),
	initPushNotifications: vi.fn()
}));

const apiGet = vi.fn();

vi.mock('$lib/api', () => ({
	apiClient: {
		get: (...args: unknown[]) => apiGet(...args),
		post: vi.fn(),
		logout: vi.fn()
	}
}));

const { pageStore } = vi.hoisted(() => {
	// eslint-disable-next-line @typescript-eslint/no-require-imports
	const { writable } = require('svelte/store') as typeof import('svelte/store');
	return { pageStore: writable({ url: new URL('http://localhost/') }) };
});

vi.mock('$app/stores', () => ({ page: pageStore }));

describe('+layout.svelte', () => {
	beforeEach(() => {
		_resetBabyStores();
		currentUser.set(null);
		apiGet.mockReset();
	});

	afterEach(() => {
		_resetBabyStores();
		currentUser.set(null);
	});

	it('shows a loading spinner while initialization is pending', async () => {
		// Never resolve — simulates slow Fly.io cold start.
		apiGet.mockImplementation(() => new Promise(() => {}));

		render(Layout, { props: { children: childrenSnippet } });

		await waitFor(() => {
			expect(screen.getByRole('status')).toBeInTheDocument();
		});
	});

	it('hides the loading spinner once initialization finishes', async () => {
		apiGet.mockResolvedValue({ user: null });

		render(Layout, { props: { children: childrenSnippet } });

		await waitFor(() => {
			expect(screen.queryByRole('status')).toBeNull();
		});
	});
});
