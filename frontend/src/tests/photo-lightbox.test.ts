import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import PhotoLightbox from '$lib/components/PhotoLightbox.svelte';

const PHOTOS = [
	{ key: 'photos/a.jpg', url: 'https://example.com/a.jpg', thumbnail_url: 'https://example.com/ta.jpg' },
	{ key: 'photos/b.jpg', url: 'https://example.com/b.jpg', thumbnail_url: 'https://example.com/tb.jpg' },
	{ key: 'photos/c.jpg', url: 'https://example.com/c.jpg', thumbnail_url: 'https://example.com/tc.jpg' }
];

describe('PhotoLightbox', () => {
	it('renders the first photo by default', () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, onclose: vi.fn() } });

		const img = container.querySelector('img');
		expect(img).toHaveAttribute('src', PHOTOS[0].url);
	});

	it('renders the photo at startIndex', () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, startIndex: 2, onclose: vi.fn() } });

		const img = container.querySelector('img');
		expect(img).toHaveAttribute('src', PHOTOS[2].url);
	});

	it('calls onclose when close button is clicked', async () => {
		const onclose = vi.fn();
		render(PhotoLightbox, { props: { photos: PHOTOS, onclose } });

		const closeBtn = screen.getByLabelText(/close/i);
		await fireEvent.click(closeBtn);
		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose when backdrop is clicked', async () => {
		const onclose = vi.fn();
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, onclose } });

		const backdrop = container.querySelector('.lightbox-backdrop')!;
		await fireEvent.click(backdrop);
		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose on Escape key', async () => {
		const onclose = vi.fn();
		render(PhotoLightbox, { props: { photos: PHOTOS, onclose } });

		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onclose).toHaveBeenCalled();
	});

	it('hides prev/next controls when there is only one photo', () => {
		render(PhotoLightbox, { props: { photos: [PHOTOS[0]], onclose: vi.fn() } });

		expect(screen.queryByLabelText(/previous photo/i)).not.toBeInTheDocument();
		expect(screen.queryByLabelText(/next photo/i)).not.toBeInTheDocument();
	});

	it('shows prev/next controls when there are multiple photos', () => {
		render(PhotoLightbox, { props: { photos: PHOTOS, onclose: vi.fn() } });

		expect(screen.getByLabelText(/previous photo/i)).toBeInTheDocument();
		expect(screen.getByLabelText(/next photo/i)).toBeInTheDocument();
	});

	it('Next button advances to the next photo', async () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, onclose: vi.fn() } });

		await fireEvent.click(screen.getByLabelText(/next photo/i));
		expect(container.querySelector('img')).toHaveAttribute('src', PHOTOS[1].url);
	});

	it('Prev button goes back to the previous photo', async () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, startIndex: 2, onclose: vi.fn() } });

		await fireEvent.click(screen.getByLabelText(/previous photo/i));
		expect(container.querySelector('img')).toHaveAttribute('src', PHOTOS[1].url);
	});

	it('Next wraps from the last photo to the first', async () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, startIndex: 2, onclose: vi.fn() } });

		await fireEvent.click(screen.getByLabelText(/next photo/i));
		expect(container.querySelector('img')).toHaveAttribute('src', PHOTOS[0].url);
	});

	it('Prev wraps from the first photo to the last', async () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, startIndex: 0, onclose: vi.fn() } });

		await fireEvent.click(screen.getByLabelText(/previous photo/i));
		expect(container.querySelector('img')).toHaveAttribute('src', PHOTOS[2].url);
	});

	it('ArrowRight navigates to the next photo', async () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, onclose: vi.fn() } });

		await fireEvent.keyDown(window, { key: 'ArrowRight' });
		expect(container.querySelector('img')).toHaveAttribute('src', PHOTOS[1].url);
	});

	it('ArrowLeft navigates to the previous photo', async () => {
		const { container } = render(PhotoLightbox, { props: { photos: PHOTOS, startIndex: 1, onclose: vi.fn() } });

		await fireEvent.keyDown(window, { key: 'ArrowLeft' });
		expect(container.querySelector('img')).toHaveAttribute('src', PHOTOS[0].url);
	});
});
