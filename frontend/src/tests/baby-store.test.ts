import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import { get } from 'svelte/store';
import {
	babies,
	activeBaby,
	hasBabies,
	fetchBabies,
	setActiveBaby,
	createBaby,
	joinBaby,
	_resetBabyStores
} from '$lib/stores/baby';
import { apiClient } from '$lib/api';
import { mockBabies } from './fixtures';

describe('baby store', () => {

	beforeEach(() => {
		vi.restoreAllMocks();
		_resetBabyStores();
	});

	afterEach(() => {
		vi.restoreAllMocks();
	});

	it('babies store starts as empty array', () => {
		expect(get(babies)).toEqual([]);
	});

	it('activeBaby store starts as null', () => {
		expect(get(activeBaby)).toBeNull();
	});

	it('fetchBabies calls GET /babies and populates store', async () => {
		const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		await fetchBabies();

		expect(getSpy).toHaveBeenCalledWith('/babies');
		expect(get(babies)).toEqual(mockBabies);
	});

	it('fetchBabies sets activeBaby to first baby if none selected', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		await fetchBabies();

		expect(get(activeBaby)).toEqual(mockBabies[0]);
	});

	it('fetchBabies preserves activeBaby if already set and still in list', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		await fetchBabies();
		setActiveBaby('baby-2');
		expect(get(activeBaby)?.id).toBe('baby-2');

		await fetchBabies();
		expect(get(activeBaby)?.id).toBe('baby-2');
	});

	it('fetchBabies resets activeBaby if current one no longer in list', async () => {
		vi.spyOn(apiClient, 'get')
			.mockResolvedValueOnce({ babies: mockBabies })
			.mockResolvedValueOnce({ babies: [mockBabies[0]] });

		await fetchBabies();
		setActiveBaby('baby-2');

		await fetchBabies();
		expect(get(activeBaby)?.id).toBe('baby-1');
	});

	it('fetchBabies sets activeBaby to null if empty list', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: [] });

		await fetchBabies();

		expect(get(activeBaby)).toBeNull();
	});

	it('setActiveBaby switches to the baby with given id', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		await fetchBabies();
		setActiveBaby('baby-2');

		expect(get(activeBaby)).toEqual(mockBabies[1]);
	});

	it('setActiveBaby does nothing if id not found', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		await fetchBabies();
		setActiveBaby('nonexistent');

		expect(get(activeBaby)).toEqual(mockBabies[0]);
	});

	it('createBaby calls POST /babies and updates stores directly', async () => {
		const newBaby = {
			id: 'baby-3',
			name: 'Charlie',
			date_of_birth: '2025-12-01',
			sex: 'male' as const,
			diagnosis_date: null,
			kasai_date: null
		};
		// Seed initial babies
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });
		await fetchBabies();

		const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(newBaby);

		const result = await createBaby({
			name: 'Charlie',
			date_of_birth: '2025-12-01',
			sex: 'male'
		});

		expect(postSpy).toHaveBeenCalledWith('/babies', {
			name: 'Charlie',
			date_of_birth: '2025-12-01',
			sex: 'male'
		});
		expect(result).toEqual(newBaby);
		expect(get(babies)).toEqual([...mockBabies, newBaby]);
		expect(get(activeBaby)).toEqual(newBaby);
	});

	it('joinBaby calls POST /babies/join and refreshes list', async () => {
		const joinResponse = { baby_id: 'baby-1', message: 'Joined baby profile' };
		const postSpy = vi.spyOn(apiClient, 'post').mockResolvedValue(joinResponse);
		const getSpy = vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		const result = await joinBaby('ABC123');

		expect(postSpy).toHaveBeenCalledWith('/babies/join', { code: 'ABC123' });
		expect(result).toEqual(joinResponse);
		expect(getSpy).toHaveBeenCalledWith('/babies');
	});

	it('hasBabies derived store is false when no babies', () => {
		expect(get(hasBabies)).toBe(false);
	});

	it('hasBabies derived store is true when babies exist', async () => {
		vi.spyOn(apiClient, 'get').mockResolvedValue({ babies: mockBabies });

		await fetchBabies();

		expect(get(hasBabies)).toBe(true);
	});
});
