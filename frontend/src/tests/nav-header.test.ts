import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';
import NavHeader from '$lib/components/NavHeader.svelte';
import { currentUser } from '$lib/stores/user';
import { babies, activeBaby, _resetBabyStores } from '$lib/stores/baby';

const { pageStore } = vi.hoisted(() => {
	const { writable } = require('svelte/store') as typeof import('svelte/store');
	return {
		pageStore: writable({
			url: new URL('http://localhost/')
		})
	};
});

vi.mock('$app/stores', () => ({
	page: pageStore
}));

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn(),
		post: vi.fn(),
		logout: vi.fn()
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

		expect(container.querySelector('nav')).toBeNull();
	});

	it('shows home link when user is logged in', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });

		render(NavHeader);

		const link = screen.getByRole('link', { name: /home/i });
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

	it('shows navigation tabs when user has babies', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });
		babies.set([
			{ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null }
		]);
		activeBaby.set({ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null });

		render(NavHeader);

		expect(screen.getByRole('link', { name: /trends/i })).toBeInTheDocument();
		expect(screen.getByRole('link', { name: /labs/i })).toBeInTheDocument();
		expect(screen.getByRole('link', { name: /meds/i })).toBeInTheDocument();
		expect(screen.getByRole('link', { name: /logs/i })).toBeInTheDocument();
	});

	it('shows Labs tab linking to /labs', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });
		babies.set([
			{ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null }
		]);
		activeBaby.set({ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null });

		render(NavHeader);

		const labsLink = screen.getByRole('link', { name: /labs/i });
		expect(labsLink).toBeInTheDocument();
		expect(labsLink.getAttribute('href')).toBe('/labs');
	});

	it('shows Logs tab linking to /logs', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });
		babies.set([
			{ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null }
		]);
		activeBaby.set({ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null });

		render(NavHeader);

		const logsLink = screen.getByRole('link', { name: /logs/i });
		expect(logsLink).toBeInTheDocument();
		expect(logsLink.getAttribute('href')).toBe('/logs');
	});

	it('Logs tab is active when on /logs path', () => {
		currentUser.set({ id: 'u1', email: 'test@example.com', name: 'Test' });
		babies.set([
			{ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null }
		]);
		activeBaby.set({ id: 'b1', name: 'Alice', date_of_birth: '2025-06-01', sex: 'female', diagnosis_date: null, kasai_date: null });

		pageStore.set({ url: new URL('http://localhost/logs') });

		render(NavHeader);

		const logsLink = screen.getByRole('link', { name: /logs/i });
		expect(logsLink.classList.contains('active')).toBe(true);
	});
});
