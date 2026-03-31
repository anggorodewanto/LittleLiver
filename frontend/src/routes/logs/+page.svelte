<script lang="ts">
	import { activeBaby } from '$lib/stores/baby';
	import { LOG_TYPES, type LogTypeConfig } from '$lib/types/logs';
	import RawLogList from '$lib/components/RawLogList.svelte';

	let baby = $derived($activeBaby);
	let selectedTypeKey = $state(LOG_TYPES[0].key);
	let selectedType = $derived(LOG_TYPES.find((t) => t.key === selectedTypeKey)!);
</script>

{#if !baby}
	<p>No baby selected</p>
{:else}
	<h1>Logs</h1>
	<div class="type-selector">
		<label for="log-type">Log Type</label>
		<select id="log-type" bind:value={selectedTypeKey}>
			{#each LOG_TYPES as lt (lt.key)}
				<option value={lt.key}>{lt.label}</option>
			{/each}
		</select>
	</div>
	{#key selectedTypeKey}
		<RawLogList babyId={baby.id} logType={selectedType} />
	{/key}
{/if}

<style>
	.type-selector {
		margin-bottom: var(--space-4);
	}
</style>
