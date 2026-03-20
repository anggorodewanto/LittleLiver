<script lang="ts">
	import { onMount } from 'svelte';
	import { currentUser, fetchCurrentUser } from '$lib/stores/user';
	import {
		activeBaby,
		hasBabies,
		fetchBabies,
		createBaby,
		joinBaby
	} from '$lib/stores/baby';
	import type { CreateBabyInput } from '$lib/stores/baby';
	import FirstLogin from '$lib/components/FirstLogin.svelte';
	import TodayDashboard from '$lib/components/TodayDashboard.svelte';

	let loading = $state(true);
	let createSubmitting = $state(false);
	let joinSubmitting = $state(false);
	let createError = $state('');
	let joinError = $state('');

	onMount(async () => {
		try {
			await fetchCurrentUser();
			if ($currentUser) {
				await fetchBabies();
			}
		} catch (err) {
			console.error('Failed to initialize:', err);
		} finally {
			loading = false;
		}
	});

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

{#if loading}
	<p>Loading...</p>
{:else if !$currentUser}
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
