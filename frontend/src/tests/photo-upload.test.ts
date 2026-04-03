import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi, beforeEach } from 'vitest';
import PhotoUpload from '$lib/components/PhotoUpload.svelte';

describe('PhotoUpload', () => {
	let onupload: ReturnType<typeof vi.fn>;

	beforeEach(() => {
		onupload = vi.fn();
	});

	it('renders a file input for photo selection', () => {
		render(PhotoUpload, { props: { onupload } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input).toBeInTheDocument();
		expect(input.type).toBe('file');
		expect(input.accept).toContain('image/');
	});

	it('accepts all image files', () => {
		render(PhotoUpload, { props: { onupload } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input.accept).toBe('image/*');
	});

	it('calls onupload with selected file', async () => {
		render(PhotoUpload, { props: { onupload } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		const file = new File(['test'], 'photo.jpg', { type: 'image/jpeg' });
		await fireEvent.change(input, { target: { files: [file] } });

		expect(onupload).toHaveBeenCalledTimes(1);
		expect(onupload).toHaveBeenCalledWith(file);
	});

	it('does not call onupload when no file selected', async () => {
		render(PhotoUpload, { props: { onupload } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		await fireEvent.change(input, { target: { files: [] } });

		expect(onupload).not.toHaveBeenCalled();
	});

	it('shows uploading state when uploading prop is true', () => {
		render(PhotoUpload, { props: { onupload, uploading: true } });

		expect(screen.getByText(/uploading/i)).toBeInTheDocument();
	});

	it('shows uploaded photo key when photoKey prop is set', () => {
		render(PhotoUpload, { props: { onupload, photoKey: 'photos/abc123.jpg' } });

		expect(screen.getByText(/photo attached/i)).toBeInTheDocument();
	});

	it('supports multiple file selection when multiple prop is true', () => {
		render(PhotoUpload, { props: { onupload, multiple: true } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input.multiple).toBe(true);
	});

	it('calls onupload for each file when multiple files are selected', async () => {
		render(PhotoUpload, { props: { onupload, multiple: true } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		const file1 = new File(['a'], 'photo1.jpg', { type: 'image/jpeg' });
		const file2 = new File(['b'], 'photo2.jpg', { type: 'image/jpeg' });
		const file3 = new File(['c'], 'photo3.png', { type: 'image/png' });
		await fireEvent.change(input, { target: { files: [file1, file2, file3] } });

		expect(onupload).toHaveBeenCalledTimes(3);
		expect(onupload).toHaveBeenCalledWith(file1);
		expect(onupload).toHaveBeenCalledWith(file2);
		expect(onupload).toHaveBeenCalledWith(file3);
	});

	it('renders hint text when hint prop is provided', () => {
		render(PhotoUpload, { props: { onupload, hint: 'Consistent lighting recommended' } });

		expect(screen.getByText('Consistent lighting recommended')).toBeInTheDocument();
	});

	it('disables input when currentCount reaches maxPhotos', () => {
		render(PhotoUpload, { props: { onupload, currentCount: 4, maxPhotos: 4 } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		expect(input.disabled).toBe(true);
	});

	it('shows photo count when currentCount > 0', () => {
		render(PhotoUpload, { props: { onupload, currentCount: 2, maxPhotos: 4 } });

		expect(screen.getByText('2/4 photos')).toBeInTheDocument();
	});

	it('limits onupload calls when files exceed remaining slots', async () => {
		render(PhotoUpload, { props: { onupload, multiple: true, currentCount: 3, maxPhotos: 4 } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		const file1 = new File(['a'], 'photo1.jpg', { type: 'image/jpeg' });
		const file2 = new File(['b'], 'photo2.jpg', { type: 'image/jpeg' });
		const file3 = new File(['c'], 'photo3.jpg', { type: 'image/jpeg' });
		await fireEvent.change(input, { target: { files: [file1, file2, file3] } });

		expect(onupload).toHaveBeenCalledTimes(1);
		expect(onupload).toHaveBeenCalledWith(file1);
	});

	it('shows limit warning when files are truncated', async () => {
		render(PhotoUpload, { props: { onupload, multiple: true, currentCount: 3, maxPhotos: 4 } });

		const input = screen.getByLabelText(/photo/i) as HTMLInputElement;
		const file1 = new File(['a'], 'photo1.jpg', { type: 'image/jpeg' });
		const file2 = new File(['b'], 'photo2.jpg', { type: 'image/jpeg' });
		const file3 = new File(['c'], 'photo3.jpg', { type: 'image/jpeg' });
		await fireEvent.change(input, { target: { files: [file1, file2, file3] } });

		expect(screen.getByText(/only 1 .* uploaded/i)).toBeInTheDocument();
	});

	it('shows X photos attached when currentCount > 0 and no photoKey', () => {
		render(PhotoUpload, { props: { onupload, currentCount: 3 } });

		expect(screen.getByText('3 photos attached')).toBeInTheDocument();
	});
});
