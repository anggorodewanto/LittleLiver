<script lang="ts">
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { activeBaby } from '$lib/stores/baby';
	import { apiClient } from '$lib/api';
	import type { Medication } from '$lib/types/medication';
	import MedicationContainerList from '$lib/components/MedicationContainerList.svelte';

	let baby = $derived($activeBaby);
	let medId = $derived($page.params.medId);
	let medication = $state<Medication | null>(null);
	let error = $state('');

	async function load() {
		if (!baby || !medId) return;
		try {
			medication = await apiClient.get<Medication>(
				`/babies/${baby.id}/medications/${medId}`
			);
		} catch (e) {
			error = e instanceof Error ? e.message : 'Failed to load medication';
		}
	}

	onMount(() => {
		void load();
	});

	$effect(() => {
		void baby;
		void medId;
		void load();
	});
</script>

<a href="/medications" class="back-link">&larr; Back to medications</a>

{#if !baby}
	<p>No baby selected</p>
{:else if error}
	<p role="alert">{error}</p>
{:else if !medication}
	<p>Loading…</p>
{:else}
	<h1>{medication.name} — Stock</h1>
	{#if !medication.dose_amount || !medication.dose_unit}
		<p class="hint">
			Set <strong>dose amount</strong> and <strong>dose unit</strong> on the medication
			to enable auto-decrement when logging doses. Stock is still tracked manually
			without these fields.
		</p>
	{/if}
	<MedicationContainerList
		babyId={baby.id}
		medicationId={medication.id}
		doseAmount={medication.dose_amount}
		doseUnit={medication.dose_unit}
	/>
{/if}

<style>
	.back-link {
		display: inline-flex;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin-bottom: var(--space-3);
		text-decoration: none;
	}
	.hint {
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
	}
</style>
