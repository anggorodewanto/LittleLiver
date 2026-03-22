<script lang="ts">
	import BabySelector from './BabySelector.svelte';
	import { babies, activeBaby, setActiveBaby } from '$lib/stores/baby';
	import { currentUser } from '$lib/stores/user';
	import { apiClient } from '$lib/api';

	let loggingOut = $state(false);

	async function handleLogout(): Promise<void> {
		loggingOut = true;
		try {
			await apiClient.logout();
		} catch {
			// ignore — redirect to login regardless
		}
		window.location.href = '/login';
	}
</script>

{#if $currentUser}
	<header>
		<nav>
			<a href="/">LittleLiver</a>

			{#if $babies.length > 0}
				<BabySelector
					babies={$babies}
					activeBabyId={$activeBaby?.id ?? null}
					onselect={setActiveBaby}
				/>
				<a href="/trends">Trends</a>
				<a href="/report">Report</a>
				<a href="/medications">Medications</a>
			{/if}

			<a href="/settings">Settings</a>

			<button onclick={handleLogout} disabled={loggingOut}>
				{loggingOut ? 'Logging out...' : 'Logout'}
			</button>
		</nav>
	</header>
{/if}
