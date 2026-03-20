import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import NavHeader from '$lib/components/NavHeader.svelte';
import { currentUser } from '$lib/stores/user';
import { babies, activeBaby, _resetBabyStores } from '$lib/stores/baby';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn(),
		post: vi.fn()
	}
}));

describe('NavHeader', () => {
	beforeEach(() => {
		vi.restoreAllMocks();
		_resetBabyStores();
		currentUser.set(null);
	});

	afterEach(() => {
		_resetBabyStores();
		currentUser.set(null);
	});

	it('does not render when no user is logged in', () => {
		const { container } = render(NavHeader);

		expect(container.querySelector('header')).toBeNull();
	});

	it('shows app name link when user is logged in', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });

		render(NavHeader);

		const link = screen.getByRole('link', { name: /littleliver/i });
		expect(link).toBeInTheDocument();
		expect(link.getAttribute('href')).toBe('/');
	});

	it('shows settings link when user is logged in', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });

		render(NavHeader);

		const link = screen.getByRole('link', { name: /settings/i });
		expect(link).toBeInTheDocument();
		expect(link.getAttribute('href')).toBe('/settings');
	});

	it('shows baby selector when user has multiple babies', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });
		babies.set([
			{ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null },
			{ id: 'b2', name: 'Bob', date_of_birth: '2025-09-01', sex: 'male', diagnosis_date: null, kasai_date: null }
		]);
		activeBaby.set({ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null });

		render(NavHeader);

		expect(screen.getByRole('combobox')).toBeInTheDocument();
	});

	it('shows baby name when user has one baby', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });
		babies.set([
			{ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null }
		]);
		activeBaby.set({ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null });

		render(NavHeader);

		expect(screen.getByText('Alice')).toBeInTheDocument();
	});

	it('shows logout button when user is logged in', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });

		render(NavHeader);

		expect(screen.getByRole('button', { name: /logout/i })).toBeInTheDocument();
	});
});
