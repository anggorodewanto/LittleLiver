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
	<span>{babies[0].name}</span>
{:else if babies.length > 1}
	<select value={activeBabyId} onchange={handleChange}>
		{#each babies as baby}
			<option value={baby.id}>{baby.name}</option>
		{/each}
	</select>
{/if}
