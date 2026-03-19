<script lang="ts">
	import { activeBaby } from '$lib/stores/baby';
	import ReportPage from '$lib/components/ReportPage.svelte';
</script>

{#if $activeBaby}
	<ReportPage babyId={$activeBaby.id} babyName={$activeBaby.name} />
{:else}
	<p>No baby selected. Please add or select a baby first.</p>
{/if}
