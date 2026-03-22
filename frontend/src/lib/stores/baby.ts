import { writable, derived, get } from 'svelte/store';
import { apiClient } from '$lib/api';

export interface Baby {
	id: string;
	name: string;
	date_of_birth: string;
	sex: 'male' | 'female';
	diagnosis_date: string | null;
	kasai_date: string | null;
	default_cal_per_feed?: number;
	notes?: string;
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

const ACTIVE_BABY_KEY = 'littleliver_active_baby_id';

function saveActiveBabyId(id: string): void {
	try {
		localStorage.setItem(ACTIVE_BABY_KEY, id);
	} catch {
		// localStorage unavailable (SSR, private browsing)
	}
}

function loadActiveBabyId(): string | null {
	try {
		return localStorage.getItem(ACTIVE_BABY_KEY);
	} catch {
		return null;
	}
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

	// Try to restore from localStorage
	const savedId = loadActiveBabyId();
	if (savedId) {
		const saved = data.babies.find((b) => b.id === savedId);
		if (saved) {
			activeBaby.set(saved);
			return;
		}
	}

	activeBaby.set(data.babies[0]);
	saveActiveBabyId(data.babies[0].id);
}

export function setActiveBaby(id: string): void {
	const currentBabies = get(babies);

	const found = currentBabies.find((b) => b.id === id);
	if (!found) {
		return;
	}

	activeBaby.set(found);
	saveActiveBabyId(id);
}

/** Reset stores to initial state. Exported for testing only. */
export function _resetBabyStores(): void {
	babies.set([]);
	activeBaby.set(null);
	try {
		localStorage.removeItem(ACTIVE_BABY_KEY);
	} catch {
		// localStorage unavailable
	}
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

export interface UpdateBabyInput {
	name?: string;
	date_of_birth?: string;
	sex?: 'male' | 'female';
	diagnosis_date?: string | null;
	kasai_date?: string | null;
	default_cal_per_feed?: number;
	notes?: string | null;
}

interface RecalculateResponse {
	baby: Baby;
	recalculated_count: number;
}

export async function updateBaby(
	id: string,
	input: UpdateBabyInput,
	recalculate = false
): Promise<Baby> {
	const path = recalculate
		? `/babies/${id}?recalculate_calories=true`
		: `/babies/${id}`;

	let baby: Baby;
	if (recalculate) {
		const envelope = await apiClient.put<RecalculateResponse>(path, input);
		baby = envelope.baby;
	} else {
		baby = await apiClient.put<Baby>(path, input);
	}

	babies.update((current) =>
		current.map((b) => (b.id === id ? baby : b))
	);
	const currentActive = get(activeBaby);
	if (currentActive?.id === id) {
		activeBaby.set(baby);
	}
	return baby;
}

export interface InviteResponse {
	code: string;
	expires_at: string;
}

export async function generateInvite(babyId: string): Promise<InviteResponse> {
	return apiClient.post<InviteResponse>(`/babies/${babyId}/invite`, {});
}

export async function unlinkFromBaby(babyId: string): Promise<void> {
	await apiClient.del(`/babies/${babyId}/parents/me`);
	await fetchBabies();
}

export async function deleteAccount(): Promise<void> {
	await apiClient.del('/users/me');
}
