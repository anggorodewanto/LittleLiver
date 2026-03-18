import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { apiClient, _resetCsrfToken } from '$lib/api';

describe('apiClient', () => {
	let fetchSpy: ReturnType<typeof vi.spyOn>;

	beforeEach(() => {
		fetchSpy = vi.spyOn(globalThis, 'fetch');
		_resetCsrfToken();
	});

	afterEach(() => {
		fetchSpy.mockRestore();
	});

	it('exports a healthCheck function', () => {
		expect(typeof apiClient.healthCheck).toBe('function');
	});

	it('healthCheck calls fetch with correct URL', async () => {
		const mockResponse = { ok: true, json: () => Promise.resolve({ status: 'ok' }) };
		fetchSpy.mockResolvedValue(mockResponse as Response);

		const result = await apiClient.healthCheck();

		expect(fetchSpy).toHaveBeenCalledWith(
			'/api/health',
			expect.objectContaining({ credentials: 'include' })
		);
		expect(result).toEqual({ status: 'ok' });
	});

	it('throws on non-ok response', async () => {
		const mockResponse = { ok: false, status: 500 };
		fetchSpy.mockResolvedValue(mockResponse as Response);

		await expect(apiClient.healthCheck()).rejects.toThrow('API error: 500');
	});

	it('attaches X-Timezone header with current timezone', async () => {
		const mockResponse = { ok: true, json: () => Promise.resolve({ status: 'ok' }) };
		fetchSpy.mockResolvedValue(mockResponse as Response);

		await apiClient.healthCheck();

		const callArgs = fetchSpy.mock.calls[0];
		const options = callArgs[1] as RequestInit;
		const headers = options.headers as Record<string, string>;
		expect(headers['X-Timezone']).toBe(Intl.DateTimeFormat().resolvedOptions().timeZone);
	});

	it('fetches CSRF token and attaches X-CSRF-Token on state-changing requests', async () => {
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'test-csrf-token' })
		};
		const postResponse = {
			ok: true,
			json: () => Promise.resolve({ id: '1' })
		};
		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(postResponse as Response);

		const result = await apiClient.post<{ id: string }>('/babies', { name: 'Baby' });

		expect(result).toEqual({ id: '1' });
		// First call should be CSRF token fetch
		expect(fetchSpy.mock.calls[0][0]).toBe('/api/csrf-token');
		// Second call should include X-CSRF-Token header
		const postCallOptions = fetchSpy.mock.calls[1][1] as RequestInit;
		const postHeaders = postCallOptions.headers as Record<string, string>;
		expect(postHeaders['X-CSRF-Token']).toBe('test-csrf-token');
	});

	it('caches CSRF token after first fetch', async () => {
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'cached-token' })
		};
		const postResponse1 = { ok: true, json: () => Promise.resolve({ id: '1' }) };
		const postResponse2 = { ok: true, json: () => Promise.resolve({ id: '2' }) };

		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(postResponse1 as Response);
		fetchSpy.mockResolvedValueOnce(postResponse2 as Response);

		await apiClient.post('/test1', {});
		await apiClient.post('/test2', {});

		// CSRF token fetch should only happen once
		const csrfCalls = fetchSpy.mock.calls.filter((c) => c[0] === '/api/csrf-token');
		expect(csrfCalls).toHaveLength(1);
		// Total calls: 1 CSRF + 2 POST = 3
		expect(fetchSpy.mock.calls).toHaveLength(3);
	});

	it('redirects to /login on 401 response', async () => {
		const mockResponse = { ok: false, status: 401 };
		fetchSpy.mockResolvedValue(mockResponse as Response);

		const originalLocation = window.location;
		Object.defineProperty(window, 'location', {
			writable: true,
			value: { ...originalLocation, href: 'http://localhost/' }
		});

		await expect(apiClient.healthCheck()).rejects.toThrow('API error: 401');

		expect(window.location.href).toBe('/login');

		Object.defineProperty(window, 'location', {
			writable: true,
			value: originalLocation
		});
	});

	it('does not attach CSRF token on GET requests', async () => {
		const mockResponse = { ok: true, json: () => Promise.resolve({ status: 'ok' }) };
		fetchSpy.mockResolvedValue(mockResponse as Response);

		await apiClient.get('/test');

		// Should not fetch CSRF token
		const csrfCalls = fetchSpy.mock.calls.filter((c) => c[0] === '/api/csrf-token');
		expect(csrfCalls).toHaveLength(0);
	});

	it('exports get, post, put, del methods', () => {
		expect(typeof apiClient.get).toBe('function');
		expect(typeof apiClient.post).toBe('function');
		expect(typeof apiClient.put).toBe('function');
		expect(typeof apiClient.del).toBe('function');
	});

	it('put sends PUT method with CSRF token', async () => {
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'put-csrf' })
		};
		const putResponse = { ok: true, json: () => Promise.resolve({ updated: true }) };
		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(putResponse as Response);

		await apiClient.put('/test', { data: 'value' });

		const putCall = fetchSpy.mock.calls[1];
		const options = putCall[1] as RequestInit;
		expect(options.method).toBe('PUT');
		const headers = options.headers as Record<string, string>;
		expect(headers['X-CSRF-Token']).toBe('put-csrf');
	});

	it('del sends DELETE method with CSRF token', async () => {
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'del-csrf' })
		};
		const delResponse = { ok: true, json: () => Promise.resolve({}) };
		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(delResponse as Response);

		await apiClient.del('/test');

		// calls[0] = CSRF fetch, calls[1] = DELETE request
		const delCall = fetchSpy.mock.calls[1];
		const options = delCall[1] as RequestInit;
		expect(options.method).toBe('DELETE');
		const headers = options.headers as Record<string, string>;
		expect(headers['X-CSRF-Token']).toBe('del-csrf');
	});

	it('throws when CSRF token fetch fails', async () => {
		const csrfError = { ok: false, status: 403 };
		fetchSpy.mockResolvedValueOnce(csrfError as Response);

		await expect(apiClient.post('/test', {})).rejects.toThrow(
			'Failed to fetch CSRF token: 403'
		);
	});

	it('returns undefined for 204 No Content responses', async () => {
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: '204-csrf' })
		};
		const mockResponse = { ok: true, status: 204 };
		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(mockResponse as Response);

		const result = await apiClient.del('/test');

		expect(result).toBeUndefined();
	});

	it('does not set Content-Type header on GET requests', async () => {
		const mockResponse = { ok: true, status: 200, json: () => Promise.resolve({ data: 'ok' }) };
		fetchSpy.mockResolvedValue(mockResponse as Response);

		await apiClient.get('/test');

		const callArgs = fetchSpy.mock.calls[0];
		const options = callArgs[1] as RequestInit;
		const headers = options.headers as Record<string, string>;
		expect(headers['Content-Type']).toBeUndefined();
	});

	it('sets Content-Type header on POST requests with body', async () => {
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'ct-csrf' })
		};
		const postResponse = { ok: true, status: 200, json: () => Promise.resolve({ id: '1' }) };
		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(postResponse as Response);

		await apiClient.post('/test', { data: 'value' });

		const postCall = fetchSpy.mock.calls[1];
		const options = postCall[1] as RequestInit;
		const headers = options.headers as Record<string, string>;
		expect(headers['Content-Type']).toBe('application/json');
	});

	it('clears CSRF token cache on 401 so it refetches next time', async () => {
		// First request succeeds with CSRF token
		const csrfResponse = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'first-token' })
		};
		const postResponse = { ok: true, json: () => Promise.resolve({}) };
		fetchSpy.mockResolvedValueOnce(csrfResponse as Response);
		fetchSpy.mockResolvedValueOnce(postResponse as Response);

		await apiClient.post('/test', {});

		// Now a GET returns 401 — should clear CSRF cache
		const response401 = { ok: false, status: 401 };
		fetchSpy.mockResolvedValueOnce(response401 as Response);

		const originalLocation = window.location;
		Object.defineProperty(window, 'location', {
			writable: true,
			value: { ...originalLocation, href: 'http://localhost/' }
		});

		await expect(apiClient.get('/me')).rejects.toThrow('API error: 401');

		Object.defineProperty(window, 'location', {
			writable: true,
			value: originalLocation
		});

		// Next POST should refetch CSRF token (cache was cleared on 401)
		const csrfResponse2 = {
			ok: true,
			json: () => Promise.resolve({ csrf_token: 'second-token' })
		};
		const postResponse2 = { ok: true, json: () => Promise.resolve({}) };
		fetchSpy.mockResolvedValueOnce(csrfResponse2 as Response);
		fetchSpy.mockResolvedValueOnce(postResponse2 as Response);

		await apiClient.post('/test2', {});

		const csrfCalls = fetchSpy.mock.calls.filter((c) => c[0] === '/api/csrf-token');
		expect(csrfCalls).toHaveLength(2);
	});
});
