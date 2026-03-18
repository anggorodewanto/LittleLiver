import { writable, derived, get } from 'svelte/store';
import { apiClient } from '$lib/api';

export interface Baby {
	id: string;
	name: string;
	date_of_birth: string;
	sex: 'male' | 'female';
	diagnosis_date: string | null;
	kasai_date: string | null;
}

export interface CreateBabyInput {
	name: string;
	date_of_birth: string;
	sex: 'male' | 'female';
	diagnosis_date?: string;
	kasai_date?: string;
}

interface BabiesResponse {
	babies: Baby[];
}

interface JoinResponse {
	baby_id: string;
	message: string;
}

export const babies = writable<Baby[]>([]);
export const activeBaby = writable<Baby | null>(null);

export const hasBabies = derived(babies, ($babies) => $babies.length > 0);

export async function fetchBabies(): Promise<void> {
	const data = await apiClient.get<BabiesResponse>('/babies');
	babies.set(data.babies);

	const currentActive = get(activeBaby);

	if (data.babies.length === 0) {
		activeBaby.set(null);
		return;
	}

	if (currentActive) {
		const stillExists = data.babies.find((b) => b.id === currentActive!.id);
		if (stillExists) {
			activeBaby.set(stillExists);
			return;
		}
	}

	activeBaby.set(data.babies[0]);
}

export function setActiveBaby(id: string): void {
	const currentBabies = get(babies);

	const found = currentBabies.find((b) => b.id === id);
	if (!found) {
		return;
	}

	activeBaby.set(found);
}

/** Reset stores to initial state. Exported for testing only. */
export function _resetBabyStores(): void {
	babies.set([]);
	activeBaby.set(null);
}

export async function createBaby(input: CreateBabyInput): Promise<Baby> {
	const result = await apiClient.post<Baby>('/babies', input);
	babies.update((current) => [...current, result]);
	activeBaby.set(result);
	return result;
}

export async function joinBaby(code: string): Promise<JoinResponse> {
	const result = await apiClient.post<JoinResponse>('/babies/join', { code });
	// Fetch full baby list since join response only contains baby_id, not full baby data
	await fetchBabies();
	return result;
}
