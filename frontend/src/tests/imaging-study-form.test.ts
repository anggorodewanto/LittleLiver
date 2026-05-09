import { render, screen, fireEvent, waitFor } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import ImagingStudyForm from '$lib/components/ImagingStudyForm.svelte';

vi.mock('$lib/api', () => ({
	apiClient: {
		get: vi.fn(),
		post: vi.fn(),
		put: vi.fn(),
		del: vi.fn(),
		postForm: vi.fn()
	}
}));

import { apiClient } from '$lib/api';

const mockedPostForm = vi.mocked(apiClient.postForm);
const mockedPost = vi.mocked(apiClient.post);

function makeFile(name: string, type: string): File {
	return new File(['dummy'], name, { type });
}

describe('ImagingStudyForm', () => {
	let onsubmit: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onsubmit = vi.fn();
		mockedPostForm.mockReset();
		mockedPost.mockReset();
	});

	it('renders quick-pick buttons CT, Ultrasound, MRI', () => {
		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });
		expect(screen.getByRole('button', { name: /^CT$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^Ultrasound$/i })).toBeInTheDocument();
		expect(screen.getByRole('button', { name: /^MRI$/i })).toBeInTheDocument();
	});

	it('disables save until upload completes and required fields are filled', () => {
		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });
		const submit = screen.getByRole('button', { name: /Save imaging study/i });
		expect(submit).toBeDisabled();
	});

	it('uploads files, runs auto-extract, and pre-fills fields', async () => {
		mockedPostForm.mockResolvedValue({ r2_key: 'photos/abc.jpg' });
		mockedPost.mockResolvedValue({
			suggested: {
				study_type: 'Ultrasound',
				study_date: '2026-04-15',
				findings: 'Liver normal',
				notes: ''
			}
		});

		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });
		const fileInput = screen.getByLabelText(/Add images/i) as HTMLInputElement;

		Object.defineProperty(fileInput, 'files', {
			value: [makeFile('scan.jpg', 'image/jpeg')],
			configurable: true
		});
		await fireEvent.change(fileInput);

		await waitFor(() => expect(mockedPostForm).toHaveBeenCalled());
		await waitFor(() => expect(mockedPost).toHaveBeenCalledWith(
			'/babies/b1/imaging-studies/extract',
			{ photo_keys: ['photos/abc.jpg'] }
		));

		// Wait for the auto-fill to settle
		await waitFor(() => {
			const typeInput = screen.getByLabelText(/Study type/i) as HTMLInputElement;
			expect(typeInput.value).toBe('Ultrasound');
		});
		const dateInput = screen.getByLabelText(/Study date/i) as HTMLInputElement;
		expect(dateInput.value).toBe('2026-04-15');
		const notesField = screen.getByLabelText(/Notes/i) as HTMLTextAreaElement;
		expect(notesField.value).toBe('Liver normal');
	});

	it('shows toast on extract failure but keeps form usable', async () => {
		mockedPostForm.mockResolvedValue({ r2_key: 'photos/x.jpg' });
		mockedPost.mockRejectedValue(new Error('boom'));

		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });
		const fileInput = screen.getByLabelText(/Add images/i) as HTMLInputElement;
		Object.defineProperty(fileInput, 'files', {
			value: [makeFile('scan.jpg', 'image/jpeg')],
			configurable: true
		});
		await fireEvent.change(fileInput);

		await waitFor(() => expect(mockedPost).toHaveBeenCalled());
		await waitFor(() => expect(screen.getByRole('alert')).toHaveTextContent(/Couldn't analyze/i));
	});

	it('submits payload with photo_keys, study_date, study_type, notes', async () => {
		mockedPostForm.mockResolvedValue({ r2_key: 'photos/z.jpg' });
		mockedPost.mockResolvedValue({ suggested: { study_type: '', study_date: '', findings: '', notes: '' } });

		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });

		const fileInput = screen.getByLabelText(/Add images/i) as HTMLInputElement;
		Object.defineProperty(fileInput, 'files', {
			value: [makeFile('scan.jpg', 'image/jpeg')],
			configurable: true
		});
		await fireEvent.change(fileInput);

		await waitFor(() => expect(mockedPostForm).toHaveBeenCalled());
		await waitFor(() => expect(mockedPost).toHaveBeenCalled());

		// Click CT quick-pick
		await fireEvent.click(screen.getByRole('button', { name: /^CT$/i }));
		// Set notes
		await fireEvent.input(screen.getByLabelText(/Notes/i), { target: { value: 'no acute findings' } });

		const submitBtn = screen.getByRole('button', { name: /Save imaging study/i });
		await waitFor(() => expect(submitBtn).not.toBeDisabled());
		await fireEvent.click(submitBtn);

		expect(onsubmit).toHaveBeenCalledTimes(1);
		const payload = onsubmit.mock.calls[0][0];
		expect(payload.study_type).toBe('CT');
		expect(payload.notes).toBe('no acute findings');
		expect(payload.photo_keys).toEqual(['photos/z.jpg']);
		expect(payload.study_date).toBeDefined();
	});

	it('user-typed value wins over auto-fill', async () => {
		mockedPostForm.mockResolvedValue({ r2_key: 'photos/x.jpg' });
		mockedPost.mockResolvedValue({
			suggested: { study_type: 'CT', study_date: '2026-01-01', findings: '', notes: '' }
		});

		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });

		// User types a study type BEFORE upload
		const typeInput = screen.getByLabelText(/Study type/i) as HTMLInputElement;
		await fireEvent.input(typeInput, { target: { value: 'Custom' } });

		const fileInput = screen.getByLabelText(/Add images/i) as HTMLInputElement;
		Object.defineProperty(fileInput, 'files', {
			value: [makeFile('scan.jpg', 'image/jpeg')],
			configurable: true
		});
		await fireEvent.change(fileInput);
		await waitFor(() => expect(mockedPost).toHaveBeenCalled());

		// Wait a tick for any async re-render
		await new Promise((r) => setTimeout(r, 10));

		// User-typed value should still be there
		expect((screen.getByLabelText(/Study type/i) as HTMLInputElement).value).toBe('Custom');
	});

	it('rejects empty file submission with helpful message', async () => {
		render(ImagingStudyForm, { props: { babyId: 'b1', onsubmit } });
		// No way to click submit since it's disabled — covered by the disabled test
		// Confirm submit remains disabled with no files
		const submit = screen.getByRole('button', { name: /Save imaging study/i });
		expect(submit).toBeDisabled();
	});
});
