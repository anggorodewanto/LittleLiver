import { apiClient } from '$lib/api';

let deferredPrompt: (Event & { prompt?: () => Promise<{ outcome: string }> }) | null = null;

export async function registerServiceWorker(): Promise<ServiceWorkerRegistration | null> {
	if (!('serviceWorker' in navigator)) {
		return null;
	}

	return navigator.serviceWorker.register('/service-worker.js');
}

export async function subscribeToPush(subscription: PushSubscription): Promise<void> {
	const subJson = subscription.toJSON();
	await apiClient.post('/push/subscribe', {
		endpoint: subJson.endpoint,
		p256dh: subJson.keys?.p256dh ?? '',
		auth: subJson.keys?.auth ?? ''
	});
}

export async function requestPushSubscription(
	registration: ServiceWorkerRegistration,
	vapidPublicKey: string
): Promise<PushSubscription | null> {
	const permission = await Notification.requestPermission();
	if (permission !== 'granted') {
		return null;
	}

	const subscription = await registration.pushManager.subscribe({
		userVisibleOnly: true,
		applicationServerKey: vapidPublicKey
	});

	await subscribeToPush(subscription);
	return subscription;
}

/**
 * Initialize push notifications: request permission, subscribe, and register with backend.
 * Should be called after the user is authenticated and the service worker is registered.
 * Silently no-ops if push is not supported, permission is denied, or VAPID key is unavailable.
 */
export async function initPushNotifications(): Promise<void> {
	if (!('serviceWorker' in navigator) || !('PushManager' in window)) {
		return;
	}

	// Don't prompt if already denied
	if (Notification.permission === 'denied') {
		return;
	}

	try {
		const registration = await navigator.serviceWorker.ready;

		// Check if already subscribed
		const existing = await registration.pushManager.getSubscription();
		if (existing) {
			// Re-register with backend in case the subscription was lost server-side
			await subscribeToPush(existing);
			return;
		}

		// Fetch VAPID public key from server (use apiClient for X-Timezone/auth headers)
		let vapid_public_key: string;
		try {
			const data = await apiClient.get<{ vapid_public_key: string }>('/push/vapid-key');
			vapid_public_key = data.vapid_public_key;
		} catch {
			return; // VAPID not configured or auth error
		}
		if (!vapid_public_key) {
			return;
		}

		await requestPushSubscription(registration, vapid_public_key);
	} catch {
		// Push notification setup failed — non-fatal
	}
}

export function setupInstallPrompt(): void {
	window.addEventListener('beforeinstallprompt', (event: Event) => {
		event.preventDefault();
		deferredPrompt = event as Event & { prompt: () => Promise<{ outcome: string }> };
	});
}

export function getDeferredPrompt(): Event | null {
	return deferredPrompt;
}

export async function promptInstall(): Promise<string | null> {
	if (!deferredPrompt?.prompt) {
		return null;
	}

	const result = await deferredPrompt.prompt();
	deferredPrompt = null;
	return result.outcome;
}
