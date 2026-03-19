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
		keys: subJson.keys
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
