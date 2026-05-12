<script lang="ts">
	import { onMount, onDestroy } from 'svelte';

	interface Props {
		message?: string;
		slowMessage?: string;
		slowAfterMs?: number;
		fullscreen?: boolean;
	}

	let {
		message = 'Loading…',
		slowMessage = '',
		slowAfterMs = 0,
		fullscreen = false
	}: Props = $props();

	let slow = $state(false);
	let timer: ReturnType<typeof setTimeout> | undefined;

	onMount(() => {
		if (slowAfterMs > 0 && slowMessage) {
			timer = setTimeout(() => {
				slow = true;
			}, slowAfterMs);
		}
	});

	onDestroy(() => {
		if (timer) clearTimeout(timer);
	});
</script>

<div
	class="spinner-wrap"
	class:fullscreen
	role="status"
	aria-live="polite"
>
	<div class="spinner" aria-hidden="true"></div>
	<p class="msg">{message}</p>
	{#if slow}
		<p class="slow-msg">{slowMessage}</p>
	{/if}
</div>

<style>
	.spinner-wrap {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		gap: var(--space-3);
		padding: var(--space-8) var(--space-4);
		color: var(--color-text-muted);
	}

	.spinner-wrap.fullscreen {
		min-height: 60vh;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 3px solid var(--color-primary-light);
		border-top-color: var(--color-primary);
		border-radius: 50%;
		animation: spin 0.9s linear infinite;
	}

	.msg {
		margin: 0;
		font-size: var(--font-size-sm);
	}

	.slow-msg {
		margin: 0;
		font-size: var(--font-size-xs);
		color: var(--color-text-muted);
		max-width: 28ch;
		text-align: center;
		animation: fade-in 0.3s ease-out;
	}

	@keyframes spin {
		to { transform: rotate(360deg); }
	}

	@keyframes fade-in {
		from { opacity: 0; }
		to { opacity: 1; }
	}

	@media (prefers-reduced-motion: reduce) {
		.spinner {
			animation-duration: 2s;
		}
	}
</style>
