import { describe, it, expect, vi, beforeEach } from 'vitest';

// Mock ServiceWorkerGlobalScope
function createMockSwScope() {
	const listeners: Record<string, Function[]> = {};
	return {
		addEventListener: vi.fn((event: string, handler: Function) => {
			if (!listeners[event]) listeners[event] = [];
			listeners[event].push(handler);
		}),
		_trigger: (event: string, data: unknown) => {
			for (const handler of listeners[event] || []) {
				handler(data);
			}
		},
		_listeners: listeners,
		skipWaiting: vi.fn(() => Promise.resolve()),
		caches: {
			open: vi.fn(() =>
				Promise.resolve({
					addAll: vi.fn(() => Promise.resolve()),
					match: vi.fn(() => Promise.resolve(undefined)),
					put: vi.fn(() => Promise.resolve()),
					keys: vi.fn(() => Promise.resolve([]))
				})
			),
			delete: vi.fn(() => Promise.resolve(true)),
			keys: vi.fn(() => Promise.resolve([]))
		},
		clients: {
			claim: vi.fn(() => Promise.resolve()),
			openWindow: vi.fn(() => Promise.resolve(null)),
			matchAll: vi.fn(() => Promise.resolve([]))
		},
		registration: {
			showNotification: vi.fn(() => Promise.resolve())
		},
		fetch: vi.fn(),
		location: { origin: 'http://localhost' }
	};
}

describe('service worker push handler', () => {
	let sw: ReturnType<typeof createMockSwScope>;

	beforeEach(() => {
		vi.resetModules();
		sw = createMockSwScope();
	});

	it('displays notification with correct title and body from push payload', async () => {
		// Load service worker code in mock scope
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const pushData = {
			title: 'Medication Reminder',
			body: 'Time to give Ursodiol',
			data: { medication_id: '123' }
		};

		const pushEvent = {
			data: {
				json: () => pushData
			},
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('push', pushEvent);

		// waitUntil should have been called
		expect(pushEvent.waitUntil).toHaveBeenCalled();

		// Wait for the promise passed to waitUntil
		await pushEvent.waitUntil.mock.calls[0][0];

		expect(sw.registration.showNotification).toHaveBeenCalledWith('Medication Reminder', {
			body: 'Time to give Ursodiol',
			data: { medication_id: '123' }
		});
	});

	it('promotes top-level payload.url into notification data.url', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const pushData = {
			title: 'Care plan: Antibiotic Rotation',
			body: "'Amoxicillin' starts 2026-06-01",
			url: '/care-plans/plan-1',
			data: { care_plan_id: 'plan-1', phase_id: 'phase-2', kind: 'switch' }
		};

		const pushEvent = {
			data: { json: () => pushData },
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('push', pushEvent);
		await pushEvent.waitUntil.mock.calls[0][0];

		expect(sw.registration.showNotification).toHaveBeenCalledWith(
			'Care plan: Antibiotic Rotation',
			{
				body: "'Amoxicillin' starts 2026-06-01",
				data: {
					care_plan_id: 'plan-1',
					phase_id: 'phase-2',
					kind: 'switch',
					url: '/care-plans/plan-1'
				}
			}
		);
	});

	it('handles push event with no data gracefully', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const pushEvent = {
			data: null,
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('push', pushEvent);

		expect(pushEvent.waitUntil).toHaveBeenCalled();
		await pushEvent.waitUntil.mock.calls[0][0];

		expect(sw.registration.showNotification).not.toHaveBeenCalled();
	});
});

describe('service worker notificationclick handler', () => {
	let sw: ReturnType<typeof createMockSwScope>;

	beforeEach(() => {
		vi.resetModules();
		sw = createMockSwScope();
	});

	it('opens /log/med?medication_id=X on notification click', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const notification = {
			data: { medication_id: '42' },
			close: vi.fn()
		};

		const clickEvent = {
			notification,
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('notificationclick', clickEvent);

		expect(clickEvent.waitUntil).toHaveBeenCalled();
		await clickEvent.waitUntil.mock.calls[0][0];

		expect(notification.close).toHaveBeenCalled();
		expect(sw.clients.openWindow).toHaveBeenCalledWith('/log/med?medication_id=42');
	});

	it('opens data.url when present (e.g. /care-plans/{id})', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const notification = {
			data: { url: '/care-plans/abc', care_plan_id: 'abc', kind: 'switch' },
			close: vi.fn()
		};

		const clickEvent = {
			notification,
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('notificationclick', clickEvent);

		expect(clickEvent.waitUntil).toHaveBeenCalled();
		await clickEvent.waitUntil.mock.calls[0][0];

		expect(sw.clients.openWindow).toHaveBeenCalledWith('/care-plans/abc');
	});

	it('opens root URL when notification has no medication_id', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const notification = {
			data: {},
			close: vi.fn()
		};

		const clickEvent = {
			notification,
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('notificationclick', clickEvent);

		expect(clickEvent.waitUntil).toHaveBeenCalled();
		await clickEvent.waitUntil.mock.calls[0][0];

		expect(notification.close).toHaveBeenCalled();
		expect(sw.clients.openWindow).toHaveBeenCalledWith('/');
	});
});

describe('service worker install and activate', () => {
	let sw: ReturnType<typeof createMockSwScope>;

	beforeEach(() => {
		vi.resetModules();
		sw = createMockSwScope();
	});

	it('caches app shell on install', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const installEvent = {
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('install', installEvent);

		expect(installEvent.waitUntil).toHaveBeenCalled();
		await installEvent.waitUntil.mock.calls[0][0];

		expect(sw.caches.open).toHaveBeenCalled();
		expect(sw.skipWaiting).toHaveBeenCalled();
	});

	it('cleans up old caches on activate', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		sw.caches.keys = vi.fn(() => Promise.resolve(['littleliver-v3', 'old-cache']));

		const activateEvent = {
			waitUntil: vi.fn((p: Promise<unknown>) => p)
		};

		sw._trigger('activate', activateEvent);

		expect(activateEvent.waitUntil).toHaveBeenCalled();
		await activateEvent.waitUntil.mock.calls[0][0];

		expect(sw.clients.claim).toHaveBeenCalled();
		expect(sw.caches.delete).toHaveBeenCalledWith('old-cache');
		expect(sw.caches.delete).not.toHaveBeenCalledWith('littleliver-v3');
	});

	it('serves cached responses for app shell requests via fetch', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const cachedResponse = new Response('cached');
		const mockCache = {
			addAll: vi.fn(() => Promise.resolve()),
			match: vi.fn(() => Promise.resolve(cachedResponse)),
			put: vi.fn(() => Promise.resolve()),
			keys: vi.fn(() => Promise.resolve([]))
		};
		sw.caches.open = vi.fn(() => Promise.resolve(mockCache));

		// Create a fetch event for a non-API request
		const request = { url: 'http://localhost/app.js', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		// For navigation/app-shell requests, respondWith should be called
		// API requests should NOT be intercepted
		expect(fetchEvent.respondWith).toHaveBeenCalled();
	});

	it('does not intercept API requests', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const request = { url: 'http://localhost/api/health', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		// API requests should NOT be intercepted by the service worker
		expect(fetchEvent.respondWith).not.toHaveBeenCalled();
	});
});

describe('service worker fetch caching strategy', () => {
	let sw: ReturnType<typeof createMockSwScope>;

	beforeEach(() => {
		vi.resetModules();
		sw = createMockSwScope();
	});

	it('uses cache-first for immutable hashed assets', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const cachedResponse = new Response('cached-asset');
		const mockCache = {
			addAll: vi.fn(() => Promise.resolve()),
			match: vi.fn(() => Promise.resolve(cachedResponse)),
			put: vi.fn(() => Promise.resolve()),
			keys: vi.fn(() => Promise.resolve([]))
		};
		sw.caches.open = vi.fn(() => Promise.resolve(mockCache));

		const request = { url: 'http://localhost/_app/immutable/entry/start.abc123.js', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		expect(fetchEvent.respondWith).toHaveBeenCalled();
		const response = await fetchEvent.respondWith.mock.calls[0][0];
		expect(response).toBe(cachedResponse);
		// Network should NOT be called since cache had a hit
		expect(sw.fetch).not.toHaveBeenCalled();
	});

	it('falls back to network for immutable assets not in cache', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const networkResponse = new Response('network-asset', { status: 200 });
		Object.defineProperty(networkResponse, 'ok', { value: true });
		const mockCache = {
			addAll: vi.fn(() => Promise.resolve()),
			match: vi.fn(() => Promise.resolve(undefined)),
			put: vi.fn(() => Promise.resolve()),
			keys: vi.fn(() => Promise.resolve([]))
		};
		sw.caches.open = vi.fn(() => Promise.resolve(mockCache));
		sw.fetch = vi.fn(() => Promise.resolve(networkResponse));

		const request = { url: 'http://localhost/_app/immutable/entry/start.xyz789.js', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		const response = await fetchEvent.respondWith.mock.calls[0][0];
		expect(sw.fetch).toHaveBeenCalledWith(request);
		expect(mockCache.put).toHaveBeenCalled();
		expect(response).toBe(networkResponse);
	});

	it('uses network-first for navigation requests (prevents stale index.html)', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const cachedResponse = new Response('stale-html');
		const networkResponse = new Response('fresh-html', { status: 200 });
		Object.defineProperty(networkResponse, 'ok', { value: true });
		const mockCache = {
			addAll: vi.fn(() => Promise.resolve()),
			match: vi.fn(() => Promise.resolve(cachedResponse)),
			put: vi.fn(() => Promise.resolve()),
			keys: vi.fn(() => Promise.resolve([]))
		};
		sw.caches.open = vi.fn(() => Promise.resolve(mockCache));
		sw.fetch = vi.fn(() => Promise.resolve(networkResponse));

		const request = { url: 'http://localhost/log/feeding', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		const response = await fetchEvent.respondWith.mock.calls[0][0];
		// Should return network response, NOT the cached one
		expect(response).toBe(networkResponse);
		expect(sw.fetch).toHaveBeenCalledWith(request);
	});

	it('falls back to cache when network fails for navigation requests', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const cachedResponse = new Response('cached-html');
		const mockCache = {
			addAll: vi.fn(() => Promise.resolve()),
			match: vi.fn(() => Promise.resolve(cachedResponse)),
			put: vi.fn(() => Promise.resolve()),
			keys: vi.fn(() => Promise.resolve([]))
		};
		sw.caches.open = vi.fn(() => Promise.resolve(mockCache));
		sw.fetch = vi.fn(() => Promise.reject(new Error('offline')));

		const request = { url: 'http://localhost/trends', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		const response = await fetchEvent.respondWith.mock.calls[0][0];
		// Offline: should fall back to cached response
		expect(response).toBe(cachedResponse);
	});

	it('returns offline response when both network and cache fail', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const mockCache = {
			addAll: vi.fn(() => Promise.resolve()),
			match: vi.fn(() => Promise.resolve(undefined)),
			put: vi.fn(() => Promise.resolve()),
			keys: vi.fn(() => Promise.resolve([]))
		};
		sw.caches.open = vi.fn(() => Promise.resolve(mockCache));
		sw.fetch = vi.fn(() => Promise.reject(new Error('offline')));

		const request = { url: 'http://localhost/settings', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);

		const response = await fetchEvent.respondWith.mock.calls[0][0];
		expect(response.status).toBe(503);
	});

	it('does not intercept auth requests', async () => {
		const { initServiceWorker } = await import('$lib/service-worker');
		initServiceWorker(sw as unknown as ServiceWorkerGlobalScope);

		const request = { url: 'http://localhost/auth/google/login', method: 'GET' };
		const fetchEvent = {
			request,
			respondWith: vi.fn(),
			waitUntil: vi.fn()
		};

		sw._trigger('fetch', fetchEvent);
		expect(fetchEvent.respondWith).not.toHaveBeenCalled();
	});
});
