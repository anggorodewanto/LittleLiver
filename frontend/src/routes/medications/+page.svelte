<script lang="ts">
	import { goto } from '$app/navigation';
	import { activeBaby } from '$lib/stores/baby';
	import MedicationList from '$lib/components/MedicationList.svelte';

	let baby = $derived($activeBaby);
</script>

<a href="/" class="back-link">&larr; Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>Medications</h1>
	<MedicationList
		babyId={baby.id}
		oncreate={() => goto('/log/medication')}
		onedit={(medicationId) => goto(`/log/medication?edit=${medicationId}`)}
		onaddlog={(medicationId) => goto(`/log/med?medicationId=${medicationId}`)}
	/>
{/if}

<style>
	.back-link {
		display: inline-flex;
		align-items: center;
		gap: var(--space-1);
		font-size: var(--font-size-sm);
		color: var(--color-text-muted);
		margin-bottom: var(--space-3);
		min-height: var(--touch-target);
		text-decoration: none;
	}

	.back-link:hover {
		color: var(--color-primary);
	}
</style>
