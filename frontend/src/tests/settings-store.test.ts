import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	babies,
	activeBaby,
	fetchBabies,
	_resetBabyStores,
	updateBaby,
	generateInvite,
	unlinkFromBaby,
	deleteAccount
} from '$lib/stores/baby';
import { apiClient } from '$lib/api';
import { mockBabies } from './fixtures';

describe('settings store functions', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		_resetBabyStores();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	describe('updateBaby', () => {
		it('calls PUT /babies/:id with baby data', async () => {
			const updatedBaby = { ...mockBabies[0], name: 'Updated Alice' };
			const putSpy = vi.spyOn(apiClient, 'put').mockResolvedValue(updatedBaby);

			vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
			await fetchBabies();

			await updateBaby('baby-1', { name: 'Updated Alice' });

			expect(putSpy).toHaveBeenCalledWith('/babies/baby-1', { name: 'Updated Alice' });
		});

		it('updates baby in store after successful update', async () => {
			const updatedBaby = { ...mockBabies[0], name: 'Updated Alice' };
			vi.spyOn(apiClient, 'put').mockResolvedValue(updatedBaby);
			vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
			await fetchBabies();

			await updateBaby('baby-1', { name: 'Updated Alice' });

			const currentBabies = get(babies);
			expect(currentBabies[0].name).toBe('Updated Alice');
		});

		it('updates activeBaby if the updated baby is active', async () => {
			const updatedBaby = { ...mockBabies[0], name: 'Updated Alice' };
			vi.spyOn(apiClient, 'put').mockResolvedValue(updatedBaby);
			vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
			await fetchBabies();

			await updateBaby('baby-1', { name: 'Updated Alice' });

			expect(get(activeBaby)?.name).toBe('Updated Alice');
		});

		it('sends recalculate query param when recalculate is true', async () => {
			const updatedBaby = { ...mockBabies[0], default_cal_per_feed: 80 };
			const putSpy = vi.spyOn(apiClient, 'put').mockResolvedValue(updatedBaby);
			vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
			await fetchBabies();

			await updateBaby('baby-1', { default_cal_per_feed: 80 }, true);

			expect(putSpy).toHaveBeenCalledWith('/babies/baby-1?recalculate_calories=true', {
				default_cal_per_feed: 80
			});
		});

		it('unwraps envelope response when recalculate is true', async () => {
			const updatedBaby = { ...mockBabies[0], default_cal_per_feed: 80 };
			const envelopeResponse = { baby: updatedBaby, recalculated_count: 5 };
			vi.spyOn(apiClient, 'put').mockResolvedValue(envelopeResponse);
			vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
			await fetchBabies();

			const result = await updateBaby('baby-1', { default_cal_per_feed: 80 }, true);

			expect(result).toEqual(updatedBaby);
			expect(result).not.toHaveProperty('recalculated_count');
			const currentBabies = get(babies);
			expect(currentBabies[0].default_cal_per_feed).toBe(80);
			expect(get(activeBaby)?.default_cal_per_feed).toBe(80);
		});
	});

	describe('generateInvite', () => {
		it('calls POST /babies/:id/invite and returns code and expiry', async () => {
			const inviteResp = { code: 'ABC123', expires_at: '2026-03-21T00:00:00Z' };
			const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(inviteResp);

			const result = await generateInvite('baby-1');

			expect(postSpy).toHaveBeenCalledWith('/babies/baby-1/invite', {});
			expect(result.code).toBe('ABC123');
			expect(result.expires_at).toBe('2026-03-21T00:00:00Z');
		});
	});

	describe('unlinkFromBaby', () => {
		it('calls DELETE /babies/:id/parents/me', async () => {
			vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: [] });
			const delSpy = vi.spyOn(apiClient, 'del').mockResolvedValue(undefined);

			await unlinkFromBaby('baby-1');

			expect(delSpy).toHaveBeenCalledWith('/babies/baby-1/parents/me');
		});

		it('refreshes babies list after unlinking', async () => {
			const getSpy = vi
				.spyOn(apiClient, 'get')
				.mockResolvedValue({ babies: [mockBabies[1]] });
			vi.spyOn(apiClient, 'del').mockResolvedValue(undefined);

			await unlinkFromBaby('baby-1');

			expect(getSpy).toHaveBeenCalledWith('/babies');
		});
	});

	describe('deleteAccount', () => {
		it('calls DELETE /users/me', async () => {
			const delSpy = vi.spyOn(apiClient, 'del').mockResolvedValue(undefined);

			await deleteAccount();

			expect(delSpy).toHaveBeenCalledWith('/users/me');
		});
	});
});
