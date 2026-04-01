import { render, screen, fireEvent } from '@testing-library/svelte';
import { describe, it, expect, vi } from 'vitest';
import PhotoLightbox from '$lib/components/PhotoLightbox.svelte';

describe('PhotoLightbox', () => {
	it('renders full-size image', () => {
		const { container } = render(PhotoLightbox, { props: { url: 'https://example.com/photo.jpg', onclose: vi.fn() } });

		const img = container.querySelector('img');
		expect(img).toHaveAttribute('src', 'https://example.com/photo.jpg');
	});

	it('calls onclose when close button is clicked', async () => {
		const onclose = vi.fn();
		render(PhotoLightbox, { props: { url: 'https://example.com/photo.jpg', onclose } });

		const closeBtn = screen.getByLabelText(/close/i);
		await fireEvent.click(closeBtn);
		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose when backdrop is clicked', async () => {
		const onclose = vi.fn();
		const { container } = render(PhotoLightbox, { props: { url: 'https://example.com/photo.jpg', onclose } });

		const backdrop = container.querySelector('.lightbox-backdrop')!;
		await fireEvent.click(backdrop);
		expect(onclose).toHaveBeenCalled();
	});

	it('calls onclose on Escape key', async () => {
		const onclose = vi.fn();
		render(PhotoLightbox, { props: { url: 'https://example.com/photo.jpg', onclose } });

		await fireEvent.keyDown(window, { key: 'Escape' });
		expect(onclose).toHaveBeenCalled();
	});
});
