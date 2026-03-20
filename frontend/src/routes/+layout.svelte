<script lang="ts">
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';
	import { registerServiceWorker, setupInstallPrompt } from '$lib/pwa';
	import NavHeader from '$lib/components/NavHeader.svelte';

	let { children } = $props();

	onMount(() => {
		registerServiceWorker();
		setupInstallPrompt();
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="manifest" href="/manifest.json" />
	<meta name="theme-color" content="#4a9c5e" />
</svelte:head>

<NavHeader />
{@render children()}
