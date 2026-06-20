import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Use vi.hoisted to create the store before vi.mock hoisting.
const { pageStore } = vi.hoisted(() => {
	const { writable } = require('svelte/store') as typeof import('svelte/store');
	return {
		pageStore: writable({
			url: new URL('http://localhost/log/immunization')
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
		get: vi.fn(),
		post: vi.fn(),
		put: vi.fn()
	}
}));

import { activeBaby, _resetBabyStores } from '$lib/stores/baby';
import { apiClient } from '$lib/api';
import { goto } from '$app/navigation';
import LogImmunizationPage from '../routes/log/immunization/+page.svelte';

const mockBaby = {
	id: 'baby-1',
	name: 'Alice',
	date_of_birth: '2025-06-01',
	sex: 'female' as const,
	diagnosis_date: null,
	kasai_date: null
};

const reference = {
	schedule: [
		{
			code: 'HEPB',
			name: 'Hepatitis B',
			dose_number: 1,
			dose_label: 'Dose 1',
			age_months: 0,
			age_label: 'Birth',
			mandatory: true
		},
		{
			code: 'DTAP',
			name: 'DTaP',
			dose_number: 1,
			dose_label: 'Dose 1',
			age_months: 2,
			age_label: '2 months',
			mandatory: true
		}
	]
};

function setUrl(url: string): void {
	pageStore.set({ url: new URL(url) });
}

describe('/log/immunization page', () => {
	let mockGet: ReturnType<typeof vi.fn>;
	let mockPost: ReturnType<typeof vi.fn>;
	let mockPut: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		vi.clearAllMocks();
		_resetBabyStores();
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockPost = apiClient.post as ReturnType<typeof vi.fn>;
		mockPut = apiClient.put as ReturnType<typeof vi.fn>;
		mockGet.mockImplementation((path: string) => {
			if (path === '/immunizations/reference') return Promise.resolve(reference);
			return Promise.resolve({});
		});
		mockPost.mockResolvedValue({});
		mockPut.mockResolvedValue({});
		setUrl('http://localhost/log/immunization');
		activeBaby.set(mockBaby);
	});

	afterEach(() => {
		_resetBabyStores();
	});

	it('shows "No baby selected" when no active baby', () => {
		_resetBabyStores();
		render(LogImmunizationPage);

		expect(screen.getByText(/no baby selected/i)).toBeInTheDocument();
	});

	it('loads the vaccine reference list into the picker', async () => {
		render(LogImmunizationPage);

		await waitFor(() => {
			expect(mockGet).toHaveBeenCalledWith('/immunizations/reference');
		});

		const select = (await screen.findByLabelText(/vaccine/i)) as HTMLSelectElement;
		const optionText = Array.from(select.options).map((o) => o.textContent);
		expect(optionText.some((t) => t?.includes('Hepatitis B'))).toBe(true);
		expect(optionText.some((t) => t?.includes('DTaP'))).toBe(true);
	});

	it('renders date, provider, lot number, and notes fields', async () => {
		render(LogImmunizationPage);

		expect(await screen.findByLabelText(/date administered/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/provider/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/lot number/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/notes/i)).toBeInTheDocument();
	});

	it('defaults the administered date to today', async () => {
		render(LogImmunizationPage);

		const dateInput = (await screen.findByLabelText(/date administered/i)) as HTMLInputElement;
		const today = new Date();
		const expected = `${today.getFullYear()}-${String(today.getMonth() + 1).padStart(2, '0')}-${String(today.getDate()).padStart(2, '0')}`;
		expect(dateInput.value).toBe(expected);
	});

	it('pre-fills the vaccine from code/name/dose query params', async () => {
		setUrl('http://localhost/log/immunization?code=DTAP&name=DTaP&dose=1');

		render(LogImmunizationPage);

		const select = (await screen.findByLabelText(/vaccine/i)) as HTMLSelectElement;
		await waitFor(() => {
			expect(select.value).toBe('DTAP::1');
		});
	});

	it('posts a new immunization record on submit', async () => {
		render(LogImmunizationPage);

		const select = (await screen.findByLabelText(/vaccine/i)) as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'HEPB::1' } });
		await fireEvent.change(screen.getByLabelText(/date administered/i), {
			target: { value: '2026-06-20' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /save|log/i }));

		await waitFor(() => {
			expect(mockPost).toHaveBeenCalledWith(
				'/babies/baby-1/immunizations',
				expect.objectContaining({
					vaccine_code: 'HEPB',
					vaccine_name: 'Hepatitis B',
					dose_number: 1,
					administered_date: '2026-06-20'
				})
			);
		});
		expect(goto).toHaveBeenCalledWith('/immunizations');
	});

	it('allows logging a custom (other) vaccine with free-text name', async () => {
		render(LogImmunizationPage);

		const select = (await screen.findByLabelText(/vaccine/i)) as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: '__custom__' } });

		const nameInput = await screen.findByLabelText(/vaccine name/i);
		await fireEvent.input(nameInput, { target: { value: 'Travel Vaccine' } });
		await fireEvent.change(screen.getByLabelText(/date administered/i), {
			target: { value: '2026-06-20' }
		});
		await fireEvent.click(screen.getByRole('button', { name: /save|log/i }));

		await waitFor(() => {
			expect(mockPost).toHaveBeenCalledWith(
				'/babies/baby-1/immunizations',
				expect.objectContaining({
					vaccine_name: 'Travel Vaccine',
					administered_date: '2026-06-20'
				})
			);
		});
		const body = mockPost.mock.calls[0][1];
		expect(body.vaccine_code).toBeUndefined();
	});

	it('loads an existing record and PUTs on edit', async () => {
		mockGet.mockImplementation((path: string) => {
			if (path === '/immunizations/reference') return Promise.resolve(reference);
			if (path === '/babies/baby-1/immunizations/rec-7') {
				return Promise.resolve({
					id: 'rec-7',
					baby_id: 'baby-1',
					logged_by: 'u1',
					vaccine_code: 'HEPB',
					vaccine_name: 'Hepatitis B',
					dose_number: 1,
					administered_date: '2025-12-15',
					provider: 'Dr. Smith',
					lot_number: 'LOT123',
					notes: 'all good',
					created_at: '2025-12-15T00:00:00Z',
					updated_at: '2025-12-15T00:00:00Z'
				});
			}
			return Promise.resolve({});
		});
		setUrl('http://localhost/log/immunization?edit=rec-7');

		render(LogImmunizationPage);

		const dateInput = (await screen.findByLabelText(/date administered/i)) as HTMLInputElement;
		await waitFor(() => {
			expect(dateInput.value).toBe('2025-12-15');
		});
		expect((screen.getByLabelText(/provider/i) as HTMLInputElement).value).toBe('Dr. Smith');

		await fireEvent.click(screen.getByRole('button', { name: /save|update/i }));

		await waitFor(() => {
			expect(mockPut).toHaveBeenCalledWith(
				'/babies/baby-1/immunizations/rec-7',
				expect.objectContaining({
					vaccine_code: 'HEPB',
					administered_date: '2025-12-15',
					provider: 'Dr. Smith'
				})
			);
		});
		expect(goto).toHaveBeenCalledWith('/immunizations');
	});

	it('shows an error message when the submit fails', async () => {
		mockPost.mockRejectedValue(new Error('boom'));
		render(LogImmunizationPage);

		const select = (await screen.findByLabelText(/vaccine/i)) as HTMLSelectElement;
		await fireEvent.change(select, { target: { value: 'HEPB::1' } });
		await fireEvent.click(screen.getByRole('button', { name: /save|log/i }));

		expect(await screen.findByText(/boom/i)).toBeInTheDocument();
	});
});
