<script lang="ts">
	import { untrack } from 'svelte';
	import { apiClient } from '$lib/api';
	import type { Medication, MedicationsResponse } from '$lib/types/medication';
	import { formatFrequency } from '$lib/medication-utils';

	interface Props {
		babyId: string;
		oncreate?: () => void;
		onedit?: (medicationId: string) => void;
		onaddlog?: (medicationId: string) => void;
	}

	let { babyId, oncreate, onedit, onaddlog }: Props = $props();

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
					<span class="med-frequency">{formatFrequency(med.frequency, med.interval_days, med.starts_from)}</span>
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
					{#if onaddlog}
						<button type="button" onclick={() => onaddlog(med.id)}>Add Log</button>
					{/if}
				</div>
			</div>
		{/each}
	</div>
{/if}

{#if oncreate}
	<button type="button" class="add-medication" onclick={oncreate}>Add Medication</button>
{/if}

<style>
	.medication-list {
		display: flex;
		flex-direction: column;
		gap: var(--space-2);
	}

	.medication-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: var(--space-3) var(--space-4);
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		box-shadow: var(--shadow-sm);
	}

	.medication-item.inactive {
		opacity: 0.5;
	}

	.med-info {
		display: flex;
		flex-direction: column;
		gap: var(--space-1);
	}

	.med-name {
		font-weight: 600;
	}

	.med-dose,
	.med-frequency {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}

	.med-actions {
		display: flex;
		gap: var(--space-2);
		flex-wrap: wrap;
	}

	.med-actions button {
		font-size: var(--font-size-xs);
		min-height: 36px;
		padding: var(--space-1) var(--space-2);
	}

	.add-medication {
		width: 100%;
		margin-top: var(--space-4);
		background: var(--color-primary);
		color: var(--color-text-inverse);
		min-height: 48px;
		font-weight: 600;
		border-radius: var(--radius-md);
	}

	.add-medication:hover {
		background: var(--color-primary-dark);
	}
</style>
