<script lang="ts">
	import { goto } from '$app/navigation';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';
	import CarePlanForm from '$lib/components/CarePlanForm.svelte';
	import type { CarePlan, CarePlanRequest } from '$lib/types/carePlan';
	import type { CarePlanFormPayload } from '$lib/components/CarePlanForm.svelte';

	let baby = $derived($activeBaby);
	let submitting = $state(false);
	let error = $state('');

	async function handleSubmit(data: CarePlanFormPayload): Promise<void> {
		if (!baby) return;
		submitting = true;
		error = '';
		try {
			const payload: CarePlanRequest = {
				name: data.name,
				notes: data.notes,
				phases: data.phases
			};
			const plan = await apiClient.post<CarePlan>(`/babies/${baby.id}/care-plans`, payload);
			goto(`/care-plans/${plan.id}`);
		} catch (e) {
			error = (e as Error).message;
			submitting = false;
		}
	}
</script>

<a href="/care-plans" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>New Care Plan</h1>
	<CarePlanForm onsubmit={handleSubmit} {submitting} {error} />
{/if}

<style>
	.back-link { display: inline-block; margin-bottom: var(--space-3, 1rem); }
</style>
