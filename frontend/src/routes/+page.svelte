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
	<div class="landing">
		<p>Post-Kasai baby health tracking</p>
		<a href="/login" class="cta-link">Sign in to get started</a>
	</div>
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
	<span data-testid="active-baby-name" class="sr-only">{$activeBaby.name}</span>
	<TodayDashboard babyId={$activeBaby.id} baby={$activeBaby} />
{/if}

<style>
	.landing {
		text-align: center;
		padding-top: var(--space-12);
	}

	.landing p {
		color: var(--color-text-muted);
		margin-bottom: var(--space-6);
	}

	.cta-link {
		display: inline-block;
		background: var(--color-primary);
		color: var(--color-text-inverse);
		padding: var(--space-3) var(--space-6);
		border-radius: var(--radius-md);
		font-weight: 600;
		text-decoration: none;
	}

	.cta-link:hover {
		background: var(--color-primary-dark);
		text-decoration: none;
	}

	.sr-only {
		position: absolute;
		width: 1px;
		height: 1px;
		padding: 0;
		margin: -1px;
		overflow: hidden;
		clip: rect(0, 0, 0, 0);
		white-space: nowrap;
		border-width: 0;
	}
</style>
