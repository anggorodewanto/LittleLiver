<script lang="ts">
	import { activeBaby } from '$lib/stores/baby';
	import LabResultsView from '$lib/components/LabResultsView.svelte';

	let baby = $derived($activeBaby);
</script>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<LabResultsView babyId={baby.id} />
{/if}
