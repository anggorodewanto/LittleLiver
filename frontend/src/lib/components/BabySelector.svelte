<script lang="ts">
	import type { Baby } from '$lib/stores/baby';

	interface Props {
		babies: Baby[];
		activeBabyId: string | null;
		onselect: (id: string) => void;
	}

	let { babies, activeBabyId, onselect }: Props = $props();

	function handleChange(event: Event) {
		const target = event.target as HTMLSelectElement;
		onselect(target.value);
	}
</script>

{#if babies.length === 1}
	<span class="baby-name">{babies[0].name}</span>
{:else if babies.length > 1}
	<select class="baby-select" value={activeBabyId} onchange={handleChange}>
		{#each babies as baby}
			<option value={baby.id}>{baby.name}</option>
		{/each}
	</select>
{/if}

<style>
	.baby-name {
		font-weight: 600;
		font-size: var(--font-size-sm);
		color: var(--color-text);
	}

	.baby-select {
		width: auto;
		min-height: 36px;
		font-weight: 600;
		font-size: var(--font-size-sm);
		padding: var(--space-1) var(--space-8) var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
		text-align: center;
	}
</style>
