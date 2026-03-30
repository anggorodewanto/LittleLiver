<script lang="ts">
	import { currentUser } from '$lib/stores/user';
	import {
		activeBaby,
		hasBabies,
		createBaby,
		joinBaby
	} from '$lib/stores/baby';
	import type { CreateBabyInput } from '$lib/stores/baby';
	import FirstLogin from '$lib/components/FirstLogin.svelte';
	import TodayDashboard from '$lib/components/TodayDashboard.svelte';

	let createSubmitting = $state(false);
	let joinSubmitting = $state(false);
	let createError = $state('');
	let joinError = $state('');

	async function handleCreate(data: CreateBabyInput): Promise<void> {
		createSubmitting = true;
		createError = '';
		try {
			await createBaby(data);
		} catch {
			createError = 'Failed to create baby';
		} finally {
			createSubmitting = false;
		}
	}

	async function handleJoin(code: string): Promise<void> {
		joinSubmitting = true;
		joinError = '';
		try {
			await joinBaby(code);
		} catch {
			joinError = 'Invalid or expired code';
		} finally {
			joinSubmitting = false;
		}
	}
</script>

<h1>LittleLiver</h1>

{#if !$currentUser}
	<p>Post-Kasai baby health tracking</p>
	<a href="/login">Sign in to get started</a>
{:else if !$hasBabies}
	<FirstLogin
		oncreate={handleCreate}
		onjoin={handleJoin}
		{createSubmitting}
		{joinSubmitting}
		{createError}
		{joinError}
	/>
{:else if $activeBaby}
	<p data-testid="active-baby-name">{$activeBaby.name}</p>
	<TodayDashboard babyId={$activeBaby.id} />
{/if}
