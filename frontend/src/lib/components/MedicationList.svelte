<script lang="ts">
	import { untrack } from 'svelte';
	import { apiClient } from '$lib/api';
	import type { Medication, MedicationsResponse } from '$lib/types/medication';
	import { formatFrequency } from '$lib/medication-utils';

	interface Props {
		babyId: string;
		oncreate?: () => void;
		onedit?: (medicationId: string) => void;
		onviewlogs?: (medicationId: string) => void;
	}

	let { babyId, oncreate, onedit, onviewlogs }: Props = $props();

	let loading = $state(true);
	let error = $state<string | null>(null);
	let medications = $state<Medication[]>([]);

	async function fetchMedications(): Promise<void> {
		loading = true;
		error = null;
		try {
			const data = await apiClient.get<MedicationsResponse | Medication[]>(`/babies/${babyId}/medications`);
			medications = Array.isArray(data) ? data : (data.medications ?? []);
		} catch {
			error = 'Failed to load medications';
		} finally {
			loading = false;
		}
	}

	async function toggleActive(med: Medication): Promise<void> {
		try {
			await apiClient.put(`/babies/${babyId}/medications/${med.id}`, {
				name: med.name,
				dose: med.dose,
				frequency: med.frequency,
				schedule_times: med.schedule_times ?? [],
				active: !med.active
			});
			await fetchMedications();
		} catch {
			error = 'Failed to update medication';
		}
	}

	$effect(() => {
		void babyId;
		untrack(() => { void fetchMedications(); });
	});
</script>

{#if loading}
	<div class="loading">Loading...</div>
{:else if error}
	<div class="error">{error}</div>
{:else if medications.length === 0}
	<div class="empty">No medications found.</div>
{:else}
	<div class="medication-list">
		{#each medications as med (med.id)}
			<div
				class="medication-item {med.active ? '' : 'inactive'}"
				data-testid="medication-item"
			>
				<div class="med-info">
					<span class="med-name">{med.name}</span>
					<span class="med-dose">{med.dose}</span>
					<span class="med-frequency">{formatFrequency(med.frequency)}</span>
				</div>
				<div class="med-actions">
					{#if med.active}
						<button type="button" onclick={() => toggleActive(med)}>Deactivate</button>
					{:else}
						<button type="button" onclick={() => toggleActive(med)}>Reactivate</button>
					{/if}
					{#if onedit}
						<button type="button" onclick={() => onedit(med.id)}>Edit</button>
					{/if}
					{#if onviewlogs}
						<button type="button" onclick={() => onviewlogs(med.id)}>View Logs</button>
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}

{#if oncreate}
	<button type="button" class="add-medication" onclick={oncreate}>Add Medication</button>
{/if}
