<script lang="ts">
	import { goto } from '$app/navigation';
	import { activeBaby } from '$lib/stores/baby';
	import MedicationList from '$lib/components/MedicationList.svelte';

	let baby = $derived($activeBaby);
</script>

<a href="/">Back</a>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>Medications</h1>
	<MedicationList
		babyId={baby.id}
		oncreate={() => goto('/log/medication')}
		onedit={(medicationId) => goto(`/log/medication?edit=${medicationId}`)}
		onviewlogs={(medicationId) => goto(`/log/med?medication_id=${medicationId}`)}
	/>
{/if}
