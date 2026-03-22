<script lang="ts">
	import { onMount } from 'svelte';
	import favicon from '$lib/assets/favicon.svg';
	import { registerServiceWorker, setupInstallPrompt, initPushNotifications } from '$lib/pwa';
	import { currentUser } from '$lib/stores/user';
	import NavHeader from '$lib/components/NavHeader.svelte';

	let { children } = $props();

	onMount(() => {
		registerServiceWorker();
		setupInstallPrompt();
	});

	// Initialize push notifications when user is authenticated
	$effect(() => {
		if ($currentUser) {
			initPushNotifications();
		}
	});
</script>

<svelte:head>
	<link rel="icon" href={favicon} />
	<link rel="manifest" href="/manifest.json" />
	<meta name="theme-color" content="#4a9c5e" />
</svelte:head>

<NavHeader />
{@render children()}
