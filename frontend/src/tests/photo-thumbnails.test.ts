import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import PhotoThumbnails from '$lib/components/PhotoThumbnails.svelte';

const mockPhotos = [
	{ key: 'photos/abc.jpg', url: 'https://example.com/abc.jpg', thumbnail_url: 'https://example.com/thumb_abc.jpg' },
	{ key: 'photos/def.jpg', url: 'https://example.com/def.jpg', thumbnail_url: 'https://example.com/thumb_def.jpg' }
];

describe('PhotoThumbnails', () => {
	it('renders thumbnail images for each photo', () => {
		const { container } = render(PhotoThumbnails, { props: { photos: mockPhotos } });

		const images = container.querySelectorAll('img');
		expect(images).toHaveLength(2);
		expect(images[0]).toHaveAttribute('src', 'https://example.com/thumb_abc.jpg');
		expect(images[1]).toHaveAttribute('src', 'https://example.com/thumb_def.jpg');
	});

	it('falls back to full URL when thumbnail_url is empty', () => {
		const photos = [
			{ key: 'photos/abc.jpg', url: 'https://example.com/abc.jpg', thumbnail_url: '' }
		];
		const { container } = render(PhotoThumbnails, { props: { photos } });

		const img = container.querySelector('img');
		expect(img).toHaveAttribute('src', 'https://example.com/abc.jpg');
	});

	it('renders nothing when photos is empty', () => {
		const { container } = render(PhotoThumbnails, { props: { photos: [] } });
		expect(container.querySelector('.photo-thumbnails')).not.toBeInTheDocument();
	});

	it('calls onphotoclick when thumbnail is clicked', async () => {
		const onphotoclick = vi.fn();
		render(PhotoThumbnails, { props: { photos: mockPhotos, onphotoclick } });

		const buttons = screen.getAllByRole('button');
		await fireEvent.click(buttons[0]);
		expect(onphotoclick).toHaveBeenCalledWith('https://example.com/abc.jpg');
	});

	it('shows remove buttons when removable is true', () => {
		const onremove = vi.fn();
		render(PhotoThumbnails, { props: { photos: mockPhotos, removable: true, onremove } });

		const removeButtons = screen.getAllByLabelText(/remove photo/i);
		expect(removeButtons).toHaveLength(2);
	});

	it('does not show remove buttons when removable is false', () => {
		render(PhotoThumbnails, { props: { photos: mockPhotos } });
		expect(screen.queryByLabelText(/remove photo/i)).not.toBeInTheDocument();
	});

	it('calls onremove with photo key when remove button clicked', async () => {
		const onremove = vi.fn();
		render(PhotoThumbnails, { props: { photos: mockPhotos, removable: true, onremove } });

		const removeButtons = screen.getAllByLabelText(/remove photo/i);
		await fireEvent.click(removeButtons[1]);
		expect(onremove).toHaveBeenCalledWith('photos/def.jpg');
	});
});
