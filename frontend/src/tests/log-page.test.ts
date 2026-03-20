import { render, screen } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Use vi.hoisted to create the store before vi.mock hoisting
const { pageStore } = vi.hoisted(() => {
	const { writable } = require('svelte/store') as typeof import('svelte/store');
	return {
		pageStore: writable({
			params: { metric: 'feeding' },
			url: new URL('http://localhost/log/feeding')
		})
	};
});

vi.mock('$app/stores', () => ({
	page: pageStore
}));

vi.mock('$app/navigation', () => ({
	goto: vi.fn()
}));

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn().mockResolvedValue({ medications: [] }),
		post: vi.fn()
	}
}));

import { activeBaby, _resetBabyStores } from '$lib/stores/baby';
import LogPage from '../routes/log/[metric]/+page.svelte';

const mockBaby = {
	id: 'baby-1',
	name: 'Alice',
	date_of_birth: '2025-06-01',
	sex: 'female' as const,
	diagnosis_date: null,
	kasai_date: null
};

describe('Log Page', () => {
	beforeEach(() => {
		_resetBabyStores();
	});

	afterEach(() => {
		_resetBabyStores();
	});

	it('shows "No baby selected" when no active baby', () => {
		pageStore.set({
			params: { metric: 'feeding' },
			url: new URL('http://localhost/log/feeding')
		});

		render(LogPage);

		expect(screen.getByText(/no baby selected/i)).toBeInTheDocument();
	});

	it('shows "Unknown metric type" for invalid metric', () => {
		activeBaby.set(mockBaby);

		pageStore.set({
			params: { metric: 'invalid' },
			url: new URL('http://localhost/log/invalid')
		});

		render(LogPage);

		expect(screen.getByText(/unknown metric type/i)).toBeInTheDocument();
	});

	it('renders FeedingForm for feeding metric', () => {
		activeBaby.set(mockBaby);

		pageStore.set({
			params: { metric: 'feeding' },
			url: new URL('http://localhost/log/feeding')
		});

		render(LogPage);

		expect(screen.getByRole('heading')).toHaveTextContent(/log feeding/i);
		expect(screen.getByLabelText(/feed type/i)).toBeInTheDocument();
	});

	it('renders StoolForm for stool metric', () => {
		activeBaby.set(mockBaby);

		pageStore.set({
			params: { metric: 'stool' },
			url: new URL('http://localhost/log/stool')
		});

		render(LogPage);

		expect(screen.getByRole('heading')).toHaveTextContent(/log stool/i);
		expect(screen.getByText(/stool color/i)).toBeInTheDocument();
	});

	it('renders DoseLogForm for med metric', () => {
		activeBaby.set(mockBaby);

		pageStore.set({
			params: { metric: 'med' },
			url: new URL('http://localhost/log/med?medication_id=med-1&scheduled_time=08:00')
		});

		render(LogPage);

		expect(screen.getByRole('heading')).toHaveTextContent(/log dose/i);
		expect(screen.getByLabelText(/medication/i)).toBeInTheDocument();
	});

	it('shows a back link to dashboard', () => {
		activeBaby.set(mockBaby);

		pageStore.set({
			params: { metric: 'feeding' },
			url: new URL('http://localhost/log/feeding')
		});

		render(LogPage);

		const backLink = screen.getByRole('link', { name: /back/i });
		expect(backLink).toBeInTheDocument();
		expect(backLink.getAttribute('href')).toBe('/');
	});

	it('renders heading with metric name', () => {
		activeBaby.set(mockBaby);

		pageStore.set({
			params: { metric: 'temperature' },
			url: new URL('http://localhost/log/temperature')
		});

		render(LogPage);

		expect(screen.getByRole('heading')).toHaveTextContent(/log temperature/i);
	});
});
