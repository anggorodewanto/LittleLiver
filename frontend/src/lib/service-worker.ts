// SYNC NOTE: This file must be kept in sync with static/service-worker.js.
// This .ts version is the testable source of truth; the .js file is the
// production deployment copy. When modifying logic, update both files.

const CACHE_NAME = 'littleliver-v1';
const APP_SHELL_URLS = ['/', '/index.html'];

interface PushPayload {
	title: string;
	body: string;
	data?: Record<string, unknown>;
}

export function initServiceWorker(sw: ServiceWorkerGlobalScope): void {
	sw.addEventListener('install', (event: Event) => {
		const installEvent = event as ExtendableEvent;
		installEvent.waitUntil(
			(async () => {
				const cache = await sw.caches.open(CACHE_NAME);
				await cache.addAll(APP_SHELL_URLS);
				await sw.skipWaiting();
			})()
		);
	});

	sw.addEventListener('activate', (event: Event) => {
		const activateEvent = event as ExtendableEvent;
		activateEvent.waitUntil(
			(async () => {
				const keys = await sw.caches.keys();
				await Promise.all(
					keys.filter((key) => key !== CACHE_NAME).map((key) => sw.caches.delete(key))
				);
				await sw.clients.claim();
			})()
		);
	});

	sw.addEventListener('fetch', (event: Event) => {
		const fetchEvent = event as FetchEvent;
		const url = new URL(fetchEvent.request.url);

		// Do not intercept API requests
		if (url.pathname.startsWith('/api')) {
			return;
		}

		fetchEvent.respondWith(
			(async () => {
				const cache = await sw.caches.open(CACHE_NAME);
				const cachedResponse = await cache.match(fetchEvent.request);
				if (cachedResponse) {
					return cachedResponse;
				}

				try {
					const networkResponse = await sw.fetch(fetchEvent.request);
					if (networkResponse.ok) {
						await cache.put(fetchEvent.request, networkResponse.clone());
					}
					return networkResponse;
				} catch {
					// Return a basic offline page for navigation requests
					const fallback = await cache.match('/');
					if (fallback) {
						return fallback;
					}
					return new Response('Offline', { status: 503 });
				}
			})()
		);
	});

	sw.addEventListener('push', (event: Event) => {
		const pushEvent = event as PushEvent;
		pushEvent.waitUntil(
			(async () => {
				if (!pushEvent.data) {
					return;
				}

				const payload: PushPayload = pushEvent.data.json();
				await sw.registration.showNotification(payload.title, {
					body: payload.body,
					data: payload.data || {}
				});
			})()
		);
	});

	sw.addEventListener('notificationclick', (event: Event) => {
		const clickEvent = event as NotificationEvent;
		clickEvent.waitUntil(
			(async () => {
				clickEvent.notification.close();

				const data = clickEvent.notification.data as Record<string, unknown> | undefined;
				const medicationId = data?.medication_id;
				const url = medicationId ? `/log/med?medication_id=${medicationId}` : '/';

				await sw.clients.openWindow(url);
			})()
		);
	});
}
