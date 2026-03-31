<script lang="ts">
	import { activeBaby } from '$lib/stores/baby';
	import RawLogList from '$lib/components/RawLogList.svelte';

	let baby = $derived($activeBaby);
</script>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>Logs</h1>
	<RawLogList babyId={baby.id} />
{/if}
