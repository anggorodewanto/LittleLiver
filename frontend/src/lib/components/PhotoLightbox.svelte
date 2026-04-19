<script lang="ts">
	import { onMount } from 'svelte';

	interface Photo {
		key: string;
		url: string;
		thumbnail_url: string;
	}

	interface Props {
		photos: Photo[];
		startIndex?: number;
		onclose: () => void;
	}

	let { photos, startIndex = 0, onclose }: Props = $props();

	// svelte-ignore state_referenced_locally
	let currentIndex = $state(startIndex);

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

	onMount(() => {
		window.addEventListener('keydown', handleKeydown);
		return () => window.removeEventListener('keydown', handleKeydown);
	});
</script>

<!-- svelte-ignore a11y_click_events_have_key_events -->
<!-- svelte-ignore a11y_no_static_element_interactions -->
<div class="lightbox-backdrop" onclick={handleBackdropClick}>
	<div class="lightbox-content">
		<button type="button" class="close-btn" aria-label="Close" onclick={onclose}>&times;</button>
		{#if photos.length > 1}
			<button type="button" class="nav-btn nav-prev" aria-label="Previous photo" onclick={showPrev}>&#8249;</button>
			<button type="button" class="nav-btn nav-next" aria-label="Next photo" onclick={showNext}>&#8250;</button>
			<span class="counter" aria-live="polite">{currentIndex + 1} / {photos.length}</span>
		{/if}
		<img src={photos[currentIndex].url} alt="" class="lightbox-img" />
	</div>
</div>

<style>
	.lightbox-backdrop {
		position: fixed;
		inset: 0;
		z-index: 1000;
		background: rgba(0, 0, 0, 0.85);
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.lightbox-content {
		position: relative;
		max-width: 90vw;
		max-height: 90vh;
	}

	.lightbox-img {
		display: block;
		max-width: 90vw;
		max-height: 90vh;
		object-fit: contain;
		border-radius: var(--radius-md);
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
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 20px;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		box-shadow: var(--shadow-md);
	}

	.nav-btn {
		position: absolute;
		top: 50%;
		transform: translateY(-50%);
		width: 44px;
		height: 44px;
		min-height: auto;
		border-radius: 50%;
		border: none;
		background: var(--color-surface);
		color: var(--color-text);
		font-size: 28px;
		line-height: 1;
		cursor: pointer;
		display: flex;
		align-items: center;
		justify-content: center;
		padding: 0;
		box-shadow: var(--shadow-md);
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
		color: var(--color-surface);
		font-size: var(--font-size-sm);
		background: rgba(0, 0, 0, 0.5);
		padding: 2px var(--space-2);
		border-radius: var(--radius-md);
	}
</style>
