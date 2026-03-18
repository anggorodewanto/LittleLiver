import { writable, derived } from 'svelte/store';
import { apiClient } from '$lib/api';

export interface User {
	id: string;
	email: string;
	name: string;
	timezone?: string;
}

interface MeResponse {
	user: User;
}

export const currentUser = writable<User | null>(null);

export const isAuthenticated = derived(currentUser, ($user) => $user !== null);

export async function fetchCurrentUser(): Promise<void> {
	try {
		const data = await apiClient.get<MeResponse>('/me');
		currentUser.set(data.user);
	} catch {
		currentUser.set(null);
	}
}
