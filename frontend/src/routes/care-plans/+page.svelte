<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';
	import type { CarePlan } from '$lib/types/carePlan';

	let baby = $derived($activeBaby);
	let plans = $state<CarePlan[]>([]);
	let loading = $state(true);
	let error = $state('');

	async function load(babyId: string): Promise<void> {
		loading = true;
		error = '';
		try {
			plans = await apiClient.get<CarePlan[]>(`/babies/${babyId}/care-plans`);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	onMount(() => {
		if (baby) load(baby.id);
	});

	$effect(() => {
		if (baby) load(baby.id);
	});

	async function deletePlan(id: string): Promise<void> {
		if (!baby) return;
		if (!confirm('Delete this care plan? Phases and reminders will be removed.')) return;
		await apiClient.del(`/babies/${baby.id}/care-plans/${id}`);
		await load(baby.id);
	}
</script>

<a href="/" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<header class="page-header">
		<h1>Care Plans</h1>
		<button onclick={() => goto('/care-plans/new')}>+ New plan</button>
	</header>

	{#if loading}
		<p>Loading...</p>
	{:else if error}
		<p role="alert">{error}</p>
	{:else if plans.length === 0}
		<p>No care plans yet. Create one to track a phased schedule like rotating antibiotics.</p>
	{:else}
		<ul class="plan-list">
			{#each plans as plan (plan.id)}
				<li>
					<a href={`/care-plans/${plan.id}`}>
						<strong>{plan.name}</strong>
						<span> · {plan.phases.length} phases · {plan.timezone}</span>
						{#if !plan.active}<span class="inactive">inactive</span>{/if}
					</a>
					<button onclick={() => deletePlan(plan.id)} aria-label="Delete plan {plan.name}">Delete</button>
				</li>
			{/each}
		</ul>
	{/if}
{/if}

<style>
	.back-link { display: inline-block; margin-bottom: var(--space-3, 1rem); }
	.page-header { display: flex; justify-content: space-between; align-items: center; }
	.plan-list { list-style: none; padding: 0; }
	.plan-list li { padding: var(--space-2, 0.5rem) 0; border-bottom: 1px solid var(--color-border, #ddd); display: flex; justify-content: space-between; align-items: center; gap: var(--space-2, 0.5rem); }
	.inactive { font-size: var(--font-size-sm, 0.85rem); color: var(--color-text-muted, #888); margin-left: var(--space-2, 0.5rem); }
</style>
