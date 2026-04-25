<script lang="ts">
	import { page } from '$app/stores';
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';
	import CarePlanForm from '$lib/components/CarePlanForm.svelte';
	import type { CarePlan, CarePlanRequest } from '$lib/types/carePlan';
	import type { CarePlanFormPayload, CarePlanFormInitial } from '$lib/components/CarePlanForm.svelte';

	let baby = $derived($activeBaby);
	let planId = $derived($page.params.planId);
	let plan = $state<CarePlan | null>(null);
	let loading = $state(true);
	let error = $state('');
	let submitting = $state(false);

	async function load(): Promise<void> {
		if (!baby || !planId) return;
		loading = true;
		try {
			plan = await apiClient.get<CarePlan>(`/babies/${baby.id}/care-plans/${planId}`);
		} catch (e) {
			error = (e as Error).message;
		} finally {
			loading = false;
		}
	}

	onMount(load);
	$effect(() => {
		void planId;
		void baby?.id;
		load();
	});

	let initialData = $derived<CarePlanFormInitial | undefined>(
		plan
			? {
					name: plan.name,
					notes: plan.notes ?? '',
					phases: plan.phases.map((p) => ({
						seq: p.seq,
						label: p.label,
						start_date: p.start_date
					}))
				}
			: undefined
	);

	async function handleSubmit(data: CarePlanFormPayload): Promise<void> {
		if (!baby || !plan) return;
		submitting = true;
		error = '';
		try {
			const payload: CarePlanRequest = {
				name: data.name,
				notes: data.notes,
				phases: data.phases
			};
			plan = await apiClient.put<CarePlan>(`/babies/${baby.id}/care-plans/${plan.id}`, payload);
			goto('/care-plans');
		} catch (e) {
			error = (e as Error).message;
		} finally {
			submitting = false;
		}
	}
</script>

<a href="/care-plans" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else if loading}
	<p>Loading...</p>
{:else if !plan}
	<p role="alert">{error || 'Plan not found'}</p>
{:else}
	<h1>{plan.name}</h1>
	<CarePlanForm onsubmit={handleSubmit} {initialData} {submitting} {error} />
{/if}

<style>
	.back-link { display: inline-block; margin-bottom: var(--space-3, 1rem); }
</style>
