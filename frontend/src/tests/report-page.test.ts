import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest';

// Mock the API client
vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn(),
		getRaw: vi.fn()
	}
}));

import { apiClient } from '$lib/api';
import ReportPage from '$lib/components/ReportPage.svelte';

const mockGetRaw = apiClient.getRaw as ReturnType<typeof vi.fn>;

function createMockResponse(): Response {
	return {
		ok: true,
		status: 200,
		blob: vi.fn().mockResolvedValue(new Blob(['fake pdf'], { type: 'application/pdf' }))
	} as unknown as Response;
}

describe('ReportPage', () => {
	beforeEach(() => {
		vi.resetAllMocks();
		vi.stubGlobal('URL', {
			createObjectURL: vi.fn(() => 'blob:http://localhost/fake-blob-url'),
			revokeObjectURL: vi.fn()
		});
	});

	afterEach(() => {
		vi.unstubAllGlobals();
	});

	it('renders a date range picker with From and To inputs', () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		expect(screen.getByLabelText(/from/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/to/i)).toBeInTheDocument();
	});

	it('renders a Generate Report button', () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		expect(screen.getByRole('button', { name: /generate report/i })).toBeInTheDocument();
	});

	it('disables Generate Report button when dates are empty', () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const btn = screen.getByRole('button', { name: /generate report/i });
		expect(btn).toBeDisabled();
	});

	it('disables Generate Report button when From date is after To date', async () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-15' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-01' } });

		const btn = screen.getByRole('button', { name: /generate report/i });
		expect(btn).toBeDisabled();
	});

	it('enables Generate Report button when both dates are valid', async () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		const btn = screen.getByRole('button', { name: /generate report/i });
		expect(btn).not.toBeDisabled();
	});

	it('shows preview summary of what will be included', async () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		expect(screen.getByText(/report for alice/i)).toBeInTheDocument();
		expect(screen.getByText(/march 1, 2026/i)).toBeInTheDocument();
		expect(screen.getByText(/march 15, 2026/i)).toBeInTheDocument();
	});

	it('calls apiClient.getRaw with correct path when Generate Report is clicked', async () => {
		mockGetRaw.mockResolvedValue(createMockResponse());

		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		await fireEvent.click(screen.getByRole('button', { name: /generate report/i }));

		await waitFor(() => {
			expect(mockGetRaw).toHaveBeenCalledWith(
				'/babies/baby-1/report?from=2026-03-01&to=2026-03-15'
			);
		});
	});

	it('shows loading state during report generation', async () => {
		let resolveGetRaw!: (value: unknown) => void;
		const getRawPromise = new Promise((resolve) => {
			resolveGetRaw = resolve;
		});
		mockGetRaw.mockReturnValue(getRawPromise);

		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		await fireEvent.click(screen.getByRole('button', { name: /generate report/i }));

		await waitFor(() => {
			expect(screen.getByText(/generating/i)).toBeInTheDocument();
		});

		// Resolve to clean up
		resolveGetRaw(createMockResponse());
	});

	it('triggers PDF download on successful generation', async () => {
		mockGetRaw.mockResolvedValue(createMockResponse());

		// Mock createElement to capture the download link
		const mockLink = { href: '', download: '', click: vi.fn() };
		const originalCreateElement = document.createElement.bind(document);
		vi.spyOn(document, 'createElement').mockImplementation((tag: string) => {
			if (tag === 'a') {
				return mockLink as unknown as HTMLElement;
			}
			return originalCreateElement(tag);
		});

		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		await fireEvent.click(screen.getByRole('button', { name: /generate report/i }));

		await waitFor(() => {
			expect(mockLink.click).toHaveBeenCalled();
			expect(mockLink.download).toContain('report');
		});
	});

	it('shows error message with status code when report generation fails', async () => {
		mockGetRaw.mockRejectedValue(new Error('API error: 500'));

		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		await fireEvent.click(screen.getByRole('button', { name: /generate report/i }));

		await waitFor(() => {
			expect(screen.getByText(/failed to generate report \(500\)/i)).toBeInTheDocument();
		});
	});

	it('displays the report content sections that will be included', async () => {
		render(ReportPage, { props: { babyId: 'baby-1', babyName: 'Alice' } });

		const fromInput = screen.getByLabelText(/from/i);
		const toInput = screen.getByLabelText(/to/i);
		await fireEvent.input(fromInput, { target: { value: '2026-03-01' } });
		await fireEvent.input(toInput, { target: { value: '2026-03-15' } });

		// Preview summary should list what will be in the report
		expect(screen.getByText(/stool color/i)).toBeInTheDocument();
		expect(screen.getByText(/weight/i)).toBeInTheDocument();
		expect(screen.getByText(/lab trends/i)).toBeInTheDocument();
		expect(screen.getByText(/temperature/i)).toBeInTheDocument();
		expect(screen.getByText(/feeding/i)).toBeInTheDocument();
		expect(screen.getByText(/medication/i)).toBeInTheDocument();
	});
});
