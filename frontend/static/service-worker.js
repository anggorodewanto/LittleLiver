// LittleLiver Service Worker
// App shell caching only — API calls are not cached
//
// SYNC NOTE: This file must be kept in sync with src/lib/service-worker.ts.
// The .ts version is the testable source of truth; this .js file is the
// production deployment copy. When modifying logic, update both files.

const CACHE_NAME = 'littleliver-v3';
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

	// Do not intercept API or auth requests
	if (url.pathname.startsWith('/api') || url.pathname.startsWith('/auth')) {
		return;
	}

	const isImmutableAsset = url.pathname.startsWith('/_app/immutable/');

	event.respondWith(
		(async () => {
			const cache = await caches.open(CACHE_NAME);

			// Immutable hashed assets: cache-first (safe because filenames change on rebuild)
			if (isImmutableAsset) {
				const cachedResponse = await cache.match(event.request);
				if (cachedResponse) {
					return cachedResponse;
				}
				const networkResponse = await fetch(event.request);
				if (networkResponse.ok) {
					await cache.put(event.request, networkResponse.clone());
				}
				return networkResponse;
			}

			// Navigation and other requests: network-first (prevents stale index.html)
			try {
				const networkResponse = await fetch(event.request);
				if (networkResponse.ok) {
					await cache.put(event.request, networkResponse.clone());
				}
				return networkResponse;
			} catch {
				const cachedResponse = await cache.match(event.request);
				if (cachedResponse) {
					return cachedResponse;
				}
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
			// Merge top-level `url` into `data.url` so the click handler
			// can route from a single field regardless of how the backend
			// shaped the payload.
			const notificationData = { ...(payload.data || {}) };
			if (payload.url && notificationData.url === undefined) {
				notificationData.url = payload.url;
			}
			await self.registration.showNotification(payload.title, {
				body: payload.body,
				data: notificationData
			});
		})()
	);
});

self.addEventListener('notificationclick', (event) => {
	event.waitUntil(
		(async () => {
			event.notification.close();

			const data = event.notification.data;
			let url;
			if (data?.url) {
				url = String(data.url);
			} else {
				const medicationId = data?.medication_id;
				const scheduledTime = data?.scheduled_time;
				url = medicationId ? `/log/med?medication_id=${medicationId}` : '/';
				if (scheduledTime) {
					const separator = url.includes('?') ? '&' : '?';
					url += `${separator}scheduled_time=${encodeURIComponent(scheduledTime)}`;
				}
			}

			const windowClients = await self.clients.matchAll({ type: 'window' });
			for (const client of windowClients) {
				if ('navigate' in client) {
					await client.navigate(url);
					await client.focus();
					return;
				}
			}
			await self.clients.openWindow(url);
		})()
	);
});
