import { describe, it, expect, vi } from 'vitest';
import { apiClient } from '$lib/api';

describe('apiClient', () => {
	it('exports a healthCheck function', () => {
		expect(typeof apiClient.healthCheck).toBe('function');
	});

	it('healthCheck calls fetch with correct URL', async () => {
		const mockResponse = { ok: true, json: () => Promise.resolve({ status: 'ok' }) };
		const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse as Response);

		const result = await apiClient.healthCheck();

		expect(fetchSpy).toHaveBeenCalledWith(
			'/api/health',
			expect.objectContaining({ credentials: 'include' })
		);
		expect(result).toEqual({ status: 'ok' });

		fetchSpy.mockRestore();
	});

	it('throws on non-ok response', async () => {
		const mockResponse = { ok: false, status: 500 };
		const fetchSpy = vi.spyOn(globalThis, 'fetch').mockResolvedValue(mockResponse as Response);

		await expect(apiClient.healthCheck()).rejects.toThrow('API error: 500');

		fetchSpy.mockRestore();
	});
});
