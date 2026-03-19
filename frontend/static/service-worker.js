// LittleLiver Service Worker
// App shell caching only — API calls are not cached
//
// SYNC NOTE: This file must be kept in sync with src/lib/service-worker.ts.
// The .ts version is the testable source of truth; this .js file is the
// production deployment copy. When modifying logic, update both files.

const CACHE_NAME = 'littleliver-v1';
const APP_SHELL_URLS = ['/', '/index.html'];

self.addEventListener('install', (event) => {
	event.waitUntil(
		(async () => {
			const cache = await caches.open(CACHE_NAME);
			await cache.addAll(APP_SHELL_URLS);
			await self.skipWaiting();
		})()
	);
});

self.addEventListener('activate', (event) => {
	event.waitUntil(
		(async () => {
			const keys = await caches.keys();
			await Promise.all(keys.filter((key) => key !== CACHE_NAME).map((key) => caches.delete(key)));
			await self.clients.claim();
		})()
	);
});

self.addEventListener('fetch', (event) => {
	const url = new URL(event.request.url);

	// Only cache same-origin requests
	if (url.origin !== self.location.origin) {
		return;
	}

	// Do not intercept API requests
	if (url.pathname.startsWith('/api')) {
		return;
	}

	event.respondWith(
		(async () => {
			const cache = await caches.open(CACHE_NAME);
			const cachedResponse = await cache.match(event.request);
			if (cachedResponse) {
				return cachedResponse;
			}

			try {
				const networkResponse = await fetch(event.request);
				if (networkResponse.ok) {
					await cache.put(event.request, networkResponse.clone());
				}
				return networkResponse;
			} catch {
				const fallback = await cache.match('/');
				if (fallback) {
					return fallback;
				}
				return new Response('Offline', { status: 503 });
			}
		})()
	);
});

self.addEventListener('push', (event) => {
	event.waitUntil(
		(async () => {
			if (!event.data) {
				return;
			}

			const payload = event.data.json();
			await self.registration.showNotification(payload.title, {
				body: payload.body,
				data: payload.data || {}
			});
		})()
	);
});

self.addEventListener('notificationclick', (event) => {
	event.waitUntil(
		(async () => {
			event.notification.close();

			const data = event.notification.data;
			const medicationId = data?.medication_id;
			const url = medicationId ? `/log/med?medication_id=${medicationId}` : '/';

			await self.clients.openWindow(url);
		})()
	);
});
