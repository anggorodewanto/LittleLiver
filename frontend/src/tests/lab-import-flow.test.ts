import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import LabImportFlow from '$lib/components/LabImportFlow.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		post: vi.fn(),
		postForm: vi.fn(),
		get: vi.fn()
	}
}));

import { apiClient } from '$lib/api';

function makeFile(name: string, content = 'x'): File {
	return new File([content], name, { type: 'image/jpeg' });
}

async function addFiles(files: File[]) {
	const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
	await fireEvent.change(input, { target: { files } });
}

async function clickSubmit() {
	await fireEvent.click(screen.getByRole('button', { name: /^submit$/i }));
}

describe('LabImportFlow', () => {
	let oncancel: ReturnType<typeof vi.fn>;
	let onsaved: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		oncancel = vi.fn();
		onsaved = vi.fn();
		vi.mocked(apiClient.post).mockReset();
		vi.mocked(apiClient.postForm).mockReset();
		vi.mocked(apiClient.get).mockResolvedValue([]);
	});

	it('shows import hint, file input, and disabled Submit button initially', () => {
		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		expect(screen.getByText(/select.*photo/i)).toBeInTheDocument();
		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input).toBeInTheDocument();
		expect(input.type).toBe('file');
		expect(input.multiple).toBe(true);

		const submit = screen.getByRole('button', { name: /^submit$/i });
		expect(submit).toBeDisabled();
	});

	it('accepts PDF files in addition to images', () => {
		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input.accept).toContain('image/*');
		expect(input.accept).toContain('.pdf');
		expect(input.accept).toContain('application/pdf');
	});

	it('does not upload until submit is clicked', async () => {
		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('a.jpg')]);

		expect(apiClient.postForm).not.toHaveBeenCalled();
		expect(apiClient.post).not.toHaveBeenCalled();
	});

	it('enables Submit once at least one file is queued', async () => {
		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		expect(screen.getByRole('button', { name: /^submit$/i })).toBeDisabled();

		await addFiles([makeFile('a.jpg')]);

		expect(screen.getByRole('button', { name: /^submit$/i })).toBeEnabled();
	});

	it('lists queued filenames', async () => {
		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report1.jpg'), makeFile('report2.jpg')]);

		expect(screen.getByText('report1.jpg')).toBeInTheDocument();
		expect(screen.getByText('report2.jpg')).toBeInTheDocument();
	});

	it('accumulates files across multiple selections', async () => {
		vi.mocked(apiClient.postForm)
			.mockResolvedValueOnce({ r2_key: 'k1' })
			.mockResolvedValueOnce({ r2_key: 'k2' });
		vi.mocked(apiClient.post).mockResolvedValueOnce({ extracted: [], notes: '' });

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('a.jpg')]);
		await addFiles([makeFile('b.jpg')]);

		expect(screen.getByText('a.jpg')).toBeInTheDocument();
		expect(screen.getByText('b.jpg')).toBeInTheDocument();

		await clickSubmit();

		await waitFor(() => {
			expect(apiClient.postForm).toHaveBeenCalledTimes(2);
		});
	});

	it('remove button drops a file from the queue', async () => {
		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('a.jpg'), makeFile('b.jpg')]);

		const removeButtons = screen.getAllByRole('button', { name: /remove/i });
		expect(removeButtons).toHaveLength(2);

		await fireEvent.click(removeButtons[0]);

		expect(screen.queryByText('a.jpg')).not.toBeInTheDocument();
		expect(screen.getByText('b.jpg')).toBeInTheDocument();
	});

	it('uploads multiple files and sends all R2 keys to extract endpoint on submit', async () => {
		vi.mocked(apiClient.postForm)
			.mockResolvedValueOnce({ r2_key: 'photos/key1.jpg' })
			.mockResolvedValueOnce({ r2_key: 'photos/key2.jpg' });

		vi.mocked(apiClient.post).mockResolvedValueOnce({
			extracted: [
				{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
			],
			notes: ''
		});

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report1.jpg', 'a'), makeFile('report2.jpg', 'b')]);
		await clickSubmit();

		await waitFor(() => {
			expect(apiClient.postForm).toHaveBeenCalledTimes(2);
		});

		await waitFor(() => {
			expect(apiClient.post).toHaveBeenCalledWith(
				'/babies/baby-1/labs/extract',
				{ photo_keys: ['photos/key1.jpg', 'photos/key2.jpg'] }
			);
		});
	});

	it('shows loading state during extraction', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post).mockImplementation(() => new Promise(() => {}));

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/extracting/i)).toBeInTheDocument();
		});
	});

	it('shows review screen after extraction completes', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post).mockResolvedValueOnce({
			extracted: [
				{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
			],
			notes: ''
		});

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
			expect(screen.getByDisplayValue('ALT')).toBeInTheDocument();
			expect(screen.getByDisplayValue('45')).toBeInTheDocument();
		});
	});

	it('calls batch endpoint on confirm with reviewed data', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post)
			.mockResolvedValueOnce({
				extracted: [
					{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
				],
				notes: ''
			})
			.mockResolvedValueOnce([]);

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(apiClient.post).toHaveBeenCalledWith(
				'/babies/baby-1/labs/batch',
				expect.objectContaining({
					items: expect.arrayContaining([
						expect.objectContaining({ test_name: 'ALT', value: '45' })
					])
				})
			);
		});
	});

	it('uses report_date as timestamp when available', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });

		vi.mocked(apiClient.post)
			.mockResolvedValueOnce({
				extracted: [
					{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
				],
				notes: '',
				report_date: '2026-03-15'
			})
			.mockResolvedValueOnce([]);

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			const batchCall = vi.mocked(apiClient.post).mock.calls.find(
				(call) => call[0] === '/babies/baby-1/labs/batch'
			);
			expect(batchCall).toBeDefined();
			const payload = batchCall![1] as { items: { timestamp: string }[] };
			expect(payload.items[0].timestamp).toContain('2026-03-15');
		});
	});

	it('filters out items with empty test_name on confirm', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });

		vi.mocked(apiClient.post)
			.mockResolvedValueOnce({
				extracted: [
					{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
				],
				notes: ''
			})
			.mockResolvedValueOnce([]);

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /add row/i }));

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			const batchCall = vi.mocked(apiClient.post).mock.calls.find(
				(call) => call[0] === '/babies/baby-1/labs/batch'
			);
			expect(batchCall).toBeDefined();
			const payload = batchCall![1] as { items: { test_name: string }[] };
			expect(payload.items).toHaveLength(1);
			expect(payload.items[0].test_name).toBe('ALT');
		});
	});

	it('calls oncancel when cancel is clicked during review', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post).mockResolvedValueOnce({
			extracted: [
				{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
			],
			notes: ''
		});

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /cancel/i }));

		expect(oncancel).toHaveBeenCalledTimes(1);
	});

	it('shows error message when extraction fails', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post).mockRejectedValueOnce(new Error('API error: 502'));

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/extraction failed/i)).toBeInTheDocument();
		});
	});

	it('shows error message when photo upload fails', async () => {
		vi.mocked(apiClient.postForm).mockRejectedValueOnce(new Error('Upload failed'));

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/upload failed/i)).toBeInTheDocument();
		});
	});

	it('calls onsaved after successful batch save', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post)
			.mockResolvedValueOnce({
				extracted: [
					{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
				],
				notes: ''
			})
			.mockResolvedValueOnce([]);

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(onsaved).toHaveBeenCalledTimes(1);
		});
	});

	it('keeps review screen visible when save fails and shows error', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post)
			.mockResolvedValueOnce({
				extracted: [
					{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
				],
				notes: ''
			})
			.mockRejectedValueOnce(new Error('Save failed'));

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(screen.getByText(/failed to save/i)).toBeInTheDocument();
		});

		expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		expect(screen.getByDisplayValue('ALT')).toBeInTheDocument();
	});

	it('allows retry after save failure', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post)
			.mockResolvedValueOnce({
				extracted: [
					{ test_name: 'ALT', value: '45', unit: 'U/L', normal_range: '7-56', confidence: 'high' }
				],
				notes: ''
			})
			.mockRejectedValueOnce(new Error('Save failed'))
			.mockResolvedValueOnce([]);

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/review extracted results/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(screen.getByText(/failed to save/i)).toBeInTheDocument();
		});

		await fireEvent.click(screen.getByRole('button', { name: /confirm/i }));

		await waitFor(() => {
			expect(onsaved).toHaveBeenCalledTimes(1);
		});
	});

	it('shows empty results message when extraction returns empty array', async () => {
		vi.mocked(apiClient.postForm).mockResolvedValue({ r2_key: 'photos/key1.jpg' });
		vi.mocked(apiClient.post).mockResolvedValueOnce({
			extracted: [],
			notes: ''
		});

		render(LabImportFlow, { props: { babyId: 'baby-1', oncancel, onsaved } });

		await addFiles([makeFile('report.jpg')]);
		await clickSubmit();

		await waitFor(() => {
			expect(screen.getByText(/no lab results found/i)).toBeInTheDocument();
		});
	});
});
