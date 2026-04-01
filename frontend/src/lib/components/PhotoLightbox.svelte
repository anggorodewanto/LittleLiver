<script lang="ts">
	import { onMount } from 'svelte';

	interface Props {
		url: string;
		onclose: () => void;
	}

	let { url, onclose }: Props = $props();

	function handleKeydown(e: KeyboardEvent) {
		if (e.key === 'Escape') onclose();
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
		<img src={url} alt="" class="lightbox-img" />
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
</style>
