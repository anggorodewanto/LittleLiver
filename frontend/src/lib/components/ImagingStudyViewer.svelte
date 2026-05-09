<script lang="ts">
	import { onMount } from 'svelte';
	import type { PhotoRef } from '$lib/types/imaging';
	import { isPDFKey } from '$lib/types/imaging';

	interface Props {
		photos: PhotoRef[];
		startIndex?: number;
		onclose: () => void;
	}

	let { photos, startIndex = 0, onclose }: Props = $props();

	let currentIndex = $state(startIndex);
	let pdfPagesByKey = $state<Record<string, string[]>>({}); // r2_key -> array of page data URLs

	let current = $derived(photos[currentIndex]);
	let isCurrentPDF = $derived(current ? isPDFKey(current.key) : false);

	function showPrev() {
		currentIndex = (currentIndex - 1 + photos.length) % photos.length;
	}
	function showNext() {
		currentIndex = (currentIndex + 1) % photos.length;
	}

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') {
			onclose();
			return;
		}
		if (photos.length <= 1) return;
		if (e.key === 'ArrowRight') showNext();
		else if (e.key === 'ArrowLeft') showPrev();
	}

	function handleBackdropClick(e: MouseEvent) {
		if (e.target === e.currentTarget) onclose();
	}

	async function renderPDF(photo: PhotoRef) {
		if (pdfPagesByKey[photo.key]) return;
		try {
			// Lazy-load pdfjs-dist so it only ships when a PDF is opened.
			const pdfjs = await import('pdfjs-dist');
			// Tell pdfjs to use a no-op worker (we render in main thread for simplicity).
			// In a heavier app we'd configure pdfjs.GlobalWorkerOptions.workerSrc.
			pdfjs.GlobalWorkerOptions.workerSrc = '';

			const resp = await fetch(photo.url);
			const buf = await resp.arrayBuffer();
			const doc = await pdfjs.getDocument({ data: buf, useSystemFonts: true }).promise;

			const pageImages: string[] = [];
			for (let p = 1; p <= doc.numPages; p++) {
				const page = await doc.getPage(p);
				const viewport = page.getViewport({ scale: 1.5 });
				const canvas = document.createElement('canvas');
				canvas.width = viewport.width;
				canvas.height = viewport.height;
				const ctx = canvas.getContext('2d');
				if (!ctx) continue;
				await page.render({ canvasContext: ctx, viewport, canvas }).promise;
				pageImages.push(canvas.toDataURL('image/png'));
			}
			pdfPagesByKey = { ...pdfPagesByKey, [photo.key]: pageImages };
		} catch (err) {
			console.error('PDF render failed', err);
			pdfPagesByKey = { ...pdfPagesByKey, [photo.key]: [] };
		}
	}

	$effect(() => {
		if (current && isCurrentPDF) {
			void renderPDF(current);
		}
	});

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="viewer-backdrop" onclick={handleBackdropClick}>
	<div class="viewer-content">
		<button type="button" class="close-btn" aria-label="Close" onclick={onclose}>&times;</button>

		{#if photos.length > 1}
			<button type="button" class="nav-btn nav-prev" aria-label="Previous" onclick={showPrev}>&#8249;</button>
			<button type="button" class="nav-btn nav-next" aria-label="Next" onclick={showNext}>&#8250;</button>
			<span class="counter">{currentIndex + 1} / {photos.length}</span>
		{/if}

		{#if isCurrentPDF}
			{#if pdfPagesByKey[current.key]}
				{#if pdfPagesByKey[current.key].length === 0}
					<div class="pdf-fallback">
						<p>Preview unavailable. <a href={current.url} target="_blank" rel="noopener">Open PDF</a></p>
					</div>
				{:else}
					<div class="pdf-pages">
						{#each pdfPagesByKey[current.key] as pageData, i (i)}
							<img src={pageData} alt="PDF page {i + 1}" class="pdf-page" />
						{/each}
					</div>
				{/if}
			{:else}
				<div class="loading">Loading PDF…</div>
			{/if}
		{:else}
			<img src={current?.url} alt="" class="viewer-img" />
		{/if}
	</div>
</div>

<style>
	.viewer-backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;
		background: rgba(0, 0, 0, 0.92);
		display: flex;
		align-items: center;
		justify-content: center;
	}
	.viewer-content {
		position: relative;
		max-width: 95vw;
		max-height: 95vh;
		overflow: auto;
	}
	.viewer-img {
		display: block;
		max-width: 95vw;
		max-height: 95vh;
		object-fit: contain;
		border-radius: var(--radius-md, 6px);
	}
	.pdf-pages {
		display: flex;
		flex-direction: column;
		gap: 8px;
		max-width: 95vw;
		max-height: 95vh;
		overflow-y: auto;
	}
	.pdf-page {
		display: block;
		max-width: 100%;
		background: white;
		box-shadow: 0 0 4px rgba(0, 0, 0, 0.4);
	}
	.pdf-fallback,
	.loading {
		background: white;
		padding: 2rem;
		border-radius: 8px;
		color: #333;
	}
	.close-btn {
		position: absolute;
		top: -12px;
		right: -12px;
		width: 32px;
		height: 32px;
		min-height: auto;
		border-radius: 50%;
		border: none;
		background: var(--color-surface, white);
		font-size: 20px;
		cursor: pointer;
		z-index: 1;
	}
	.nav-btn {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		width: 44px;
		height: 44px;
		border-radius: 50%;
		border: none;
		background: var(--color-surface, white);
		font-size: 28px;
		line-height: 1;
		cursor: pointer;
		z-index: 1;
	}
	.nav-prev {
		left: -22px;
	}
	.nav-next {
		right: -22px;
	}
	.counter {
		position: absolute;
		bottom: -32px;
		left: 50%;
		transform: translateX(-50%);
		color: white;
		font-size: 0.875rem;
		background: rgba(0, 0, 0, 0.5);
		padding: 2px 8px;
		border-radius: 4px;
	}
</style>
