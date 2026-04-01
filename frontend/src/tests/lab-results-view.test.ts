import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import { MockChart, resetChartMocks } from './chart-mock';

vi.mock('chart.js', () => ({
	Chart: MockChart,
	registerables: []
}));

vi.mock('$lib/chart-setup', () => ({}));

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn()
	}
}));

import LabResultsView from '$lib/components/LabResultsView.svelte';
import { apiClient } from '$lib/api';

const mockResults = [
	{
		id: 'lr1',
		baby_id: 'b1',
		logged_by: 'u1',
		timestamp: '2026-03-15T10:00:00Z',
		test_name: 'ALT',
		value: '120',
		unit: 'U/L',
		created_at: '2026-03-15T10:00:00Z',
		updated_at: '2026-03-15T10:00:00Z'
	},
	{
		id: 'lr2',
		baby_id: 'b1',
		logged_by: 'u1',
		timestamp: '2026-03-15T10:00:00Z',
		test_name: 'total_bilirubin',
		value: '3.2',
		unit: 'mg/dL',
		created_at: '2026-03-15T10:00:00Z',
		updated_at: '2026-03-15T10:00:00Z'
	},
	{
		id: 'lr3',
		baby_id: 'b1',
		logged_by: 'u1',
		timestamp: '2026-03-08T10:00:00Z',
		test_name: 'ALT',
		value: '95',
		unit: 'U/L',
		created_at: '2026-03-08T10:00:00Z',
		updated_at: '2026-03-08T10:00:00Z'
	}
];

describe('LabResultsView', () => {
	let mockGet: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		resetChartMocks();
		vi.clearAllMocks();
		mockGet = apiClient.get as ReturnType<typeof vi.fn>;
		mockGet.mockResolvedValue({ data: mockResults, next_cursor: null });
	});

	it('renders the date range selector', async () => {
		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(screen.getByRole('button', { name: '30d' })).toBeInTheDocument();
		});
	});

	it('fetches lab data on mount', async () => {
		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(mockGet).toHaveBeenCalled();
			const call = mockGet.mock.calls[0][0] as string;
			expect(call).toContain('/babies/b1/labs');
		});
	});

	it('renders test filter with available tests', async () => {
		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(screen.getByRole('button', { name: /all/i })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'ALT' })).toBeInTheDocument();
			expect(screen.getByRole('button', { name: 'Total Bilirubin' })).toBeInTheDocument();
		});
	});

	it('groups results by date', async () => {
		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			const headings = screen.getAllByRole('heading', { level: 3 });
			const headingTexts = headings.map((h) => h.textContent);
			expect(headingTexts.length).toBeGreaterThanOrEqual(2);
		});
	});

	it('auto-paginates when next_cursor is present', async () => {
		mockGet
			.mockResolvedValueOnce({ data: [mockResults[0]], next_cursor: 'cursor1' })
			.mockResolvedValueOnce({ data: [mockResults[1], mockResults[2]], next_cursor: null });

		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(mockGet).toHaveBeenCalledTimes(2);
		});
	});

	it('shows loading state', () => {
		mockGet.mockReturnValue(new Promise(() => {})); // never resolves

		render(LabResultsView, { props: { babyId: 'b1' } });

		expect(screen.getByText(/loading/i)).toBeInTheDocument();
	});

	it('shows error state on API failure', async () => {
		mockGet.mockRejectedValue(new Error('Network error'));

		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(screen.getByText(/failed/i)).toBeInTheDocument();
		});
	});

	it('shows empty state when no lab data', async () => {
		mockGet.mockResolvedValue({ data: [], next_cursor: null });

		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(screen.getByText(/no lab results/i)).toBeInTheDocument();
		});
	});

	it('changing date range triggers new fetch', async () => {
		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(mockGet).toHaveBeenCalled();
		});

		mockGet.mockClear();
		mockGet.mockResolvedValue({ data: mockResults, next_cursor: null });

		await fireEvent.click(screen.getByRole('button', { name: '7d' }));

		await waitFor(() => {
			expect(mockGet).toHaveBeenCalled();
		});
	});

	it('filtering by test updates displayed results', async () => {
		render(LabResultsView, { props: { babyId: 'b1' } });

		await waitFor(() => {
			expect(screen.getByRole('button', { name: 'ALT' })).toBeInTheDocument();
		});

		// Before filtering: ALT should appear in tables
		let tables = document.querySelectorAll('table');
		let tableText = Array.from(tables).map((t) => t.textContent).join('');
		expect(tableText).toContain('ALT');

		// Click Total Bilirubin to filter to only that test
		await fireEvent.click(screen.getByRole('button', { name: 'Total Bilirubin' }));

		await waitFor(() => {
			tables = document.querySelectorAll('table');
			tableText = Array.from(tables).map((t) => t.textContent).join('');
			expect(tableText).toContain('Total Bilirubin');
			expect(tableText).not.toContain('ALT');
		});
	});
});
