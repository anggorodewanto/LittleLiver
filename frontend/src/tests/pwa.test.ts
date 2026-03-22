import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

const mockFetch = vi.fn();
vi.stubGlobal('fetch', mockFetch);

describe('PWA registration', () => {
	beforeEach(() => {
		vi.resetModules();
		mockFetch.mockReset();
	});

	it('registerServiceWorker registers SW when serviceWorker is available', async () => {
		const mockRegistration = {
			pushManager: {
				getSubscription: vi.fn(() => Promise.resolve(null)),
				subscribe: vi.fn()
			}
		};
		const registerMock = vi.fn(() => Promise.resolve(mockRegistration));

		Object.defineProperty(navigator, 'serviceWorker', {
			value: { register: registerMock, ready: Promise.resolve(mockRegistration) },
			writable: true,
			configurable: true
		});

		const { registerServiceWorker } = await import('$lib/pwa');
		const result = await registerServiceWorker();

		expect(registerMock).toHaveBeenCalledWith('/service-worker.js');
		expect(result).toBe(mockRegistration);
	});

	it('registerServiceWorker returns null when serviceWorker is not available', async () => {
		const origDesc = Object.getOwnPropertyDescriptor(navigator, 'serviceWorker');
		// Delete the property entirely so 'in' check fails
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		delete (navigator as any).serviceWorker;

		const { registerServiceWorker } = await import('$lib/pwa');
		const result = await registerServiceWorker();

		expect(result).toBeNull();

		// Restore
		if (origDesc) {
			Object.defineProperty(navigator, 'serviceWorker', origDesc);
		}
	});
});

describe('push subscription', () => {
	beforeEach(() => {
		vi.resetModules();
		mockFetch.mockReset();
	});

	it('subscribeToPush calls POST /api/push/subscribe with subscription data', async () => {
		// Mock CSRF token fetch
		mockFetch.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ csrf_token: 'test-token' })
		});

		// Mock push subscribe response
		mockFetch.mockResolvedValueOnce({
			ok: true,
			status: 201,
			json: () => Promise.resolve({ id: '1' })
		});

		const mockSubscription = {
			endpoint: 'https://push.example.com/abc123',
			toJSON: () => ({
				endpoint: 'https://push.example.com/abc123',
				keys: {
					p256dh: 'test-p256dh-key',
					auth: 'test-auth-key'
				}
			})
		};

		const { subscribeToPush } = await import('$lib/pwa');
		await subscribeToPush(mockSubscription as unknown as PushSubscription);

		// Second call should be the POST to /api/push/subscribe
		expect(mockFetch).toHaveBeenCalledWith(
			'/api/push/subscribe',
			expect.objectContaining({
				method: 'POST',
				body: JSON.stringify({
					endpoint: 'https://push.example.com/abc123',
					p256dh: 'test-p256dh-key',
					auth: 'test-auth-key'
				})
			})
		);
	});
});

describe('requestPushSubscription', () => {
	beforeEach(() => {
		vi.resetModules();
		mockFetch.mockReset();
	});

	afterEach(() => {
		// Clean up Notification stub
		// eslint-disable-next-line @typescript-eslint/no-explicit-any
		delete (globalThis as any).Notification;
	});

	it('requests permission, subscribes, and sends to backend when granted', async () => {
		const mockSubscription = {
			endpoint: 'https://push.example.com/sub1',
			toJSON: () => ({
				endpoint: 'https://push.example.com/sub1',
				keys: { p256dh: 'key1', auth: 'auth1' }
			})
		};

		const mockRegistration = {
			pushManager: {
				subscribe: vi.fn(() => Promise.resolve(mockSubscription))
			}
		};

		vi.stubGlobal('Notification', {
			requestPermission: vi.fn(() => Promise.resolve('granted'))
		});

		// Mock CSRF token fetch
		mockFetch.mockResolvedValueOnce({
			ok: true,
			status: 200,
			json: () => Promise.resolve({ csrf_token: 'test-token' })
		});
		// Mock push subscribe POST
		mockFetch.mockResolvedValueOnce({
			ok: true,
			status: 201,
			json: () => Promise.resolve({ id: '1' })
		});

		const { requestPushSubscription } = await import('$lib/pwa');
		const result = await requestPushSubscription(
			mockRegistration as unknown as ServiceWorkerRegistration,
			'test-vapid-key'
		);

		expect(Notification.requestPermission).toHaveBeenCalled();
		expect(mockRegistration.pushManager.subscribe).toHaveBeenCalledWith({
			userVisibleOnly: true,
			applicationServerKey: 'test-vapid-key'
		});
		expect(result).toBe(mockSubscription);
		// Verify backend was called
		expect(mockFetch).toHaveBeenCalledWith(
			'/api/push/subscribe',
			expect.objectContaining({
				method: 'POST',
				body: JSON.stringify({
					endpoint: 'https://push.example.com/sub1',
					p256dh: 'key1',
					auth: 'auth1'
				})
			})
		);
	});

	it('returns null when permission is denied', async () => {
		const mockRegistration = {
			pushManager: {
				subscribe: vi.fn()
			}
		};

		vi.stubGlobal('Notification', {
			requestPermission: vi.fn(() => Promise.resolve('denied'))
		});

		const { requestPushSubscription } = await import('$lib/pwa');
		const result = await requestPushSubscription(
			mockRegistration as unknown as ServiceWorkerRegistration,
			'test-vapid-key'
		);

		expect(Notification.requestPermission).toHaveBeenCalled();
		expect(mockRegistration.pushManager.subscribe).not.toHaveBeenCalled();
		expect(result).toBeNull();
	});

	it('returns null when permission is dismissed (default)', async () => {
		const mockRegistration = {
			pushManager: {
				subscribe: vi.fn()
			}
		};

		vi.stubGlobal('Notification', {
			requestPermission: vi.fn(() => Promise.resolve('default'))
		});

		const { requestPushSubscription } = await import('$lib/pwa');
		const result = await requestPushSubscription(
			mockRegistration as unknown as ServiceWorkerRegistration,
			'test-vapid-key'
		);

		expect(Notification.requestPermission).toHaveBeenCalled();
		expect(mockRegistration.pushManager.subscribe).not.toHaveBeenCalled();
		expect(result).toBeNull();
	});
});

describe('install prompt', () => {
	beforeEach(() => {
		vi.resetModules();
		mockFetch.mockReset();
	});

	it('captures beforeinstallprompt event and exposes deferredPrompt', async () => {
		const { setupInstallPrompt, getDeferredPrompt } = await import('$lib/pwa');
		setupInstallPrompt();

		expect(getDeferredPrompt()).toBeNull();

		// Simulate the beforeinstallprompt event
		const mockEvent = new Event('beforeinstallprompt');
		Object.defineProperty(mockEvent, 'preventDefault', { value: vi.fn() });
		window.dispatchEvent(mockEvent);

		expect(getDeferredPrompt()).toBe(mockEvent);
	});

	it('promptInstall calls prompt() on deferred event', async () => {
		const { setupInstallPrompt, promptInstall } = await import('$lib/pwa');
		setupInstallPrompt();

		const mockPrompt = vi.fn(() => Promise.resolve({ outcome: 'accepted' }));
		const mockEvent = new Event('beforeinstallprompt') as Event & {
			prompt: () => Promise<{ outcome: string }>;
		};
		Object.defineProperty(mockEvent, 'prompt', { value: mockPrompt });
		Object.defineProperty(mockEvent, 'preventDefault', { value: vi.fn() });
		window.dispatchEvent(mockEvent);

		const result = await promptInstall();
		expect(mockPrompt).toHaveBeenCalled();
		expect(result).toBe('accepted');
	});

	it('promptInstall returns null when no deferred prompt', async () => {
		const { promptInstall } = await import('$lib/pwa');
		const result = await promptInstall();
		expect(result).toBeNull();
	});
});
