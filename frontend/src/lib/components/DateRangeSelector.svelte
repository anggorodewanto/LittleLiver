<script lang="ts">
	interface Props {
		selectedRange: string;
		onchange: (range: string, from?: string, to?: string) => void;
	}

	const PRESETS = ['7d', '14d', '30d', '90d'] as const;

	let { selectedRange, onchange }: Props = $props();
	let showCustom = $state(false);
	let customFrom = $state('');
	let customTo = $state('');

	let customRangeValid = $derived(
		customFrom !== '' && customTo !== '' && customFrom <= customTo
	);

	function selectPreset(range: string): void {
		showCustom = false;
		onchange(range);
	}

	function toggleCustom(): void {
		showCustom = true;
	}

	function applyCustom(): void {
		onchange('custom', customFrom, customTo);
	}
</script>

<div class="date-range-selector">
	{#each PRESETS as preset (preset)}
		<button
			type="button"
			class={selectedRange === preset && !showCustom ? 'active' : ''}
			onclick={() => selectPreset(preset)}
		>{preset}</button>
	{/each}
	<button
		type="button"
		class={showCustom ? 'active' : ''}
		onclick={toggleCustom}
	>Custom</button>

	{#if showCustom}
		<div class="custom-range">
			<label>
				From
				<input type="date" bind:value={customFrom} />
			</label>
			<label>
				To
				<input type="date" bind:value={customTo} />
			</label>
			<button type="button" onclick={applyCustom} disabled={!customRangeValid}>Apply</button>
		</div>
	{/if}
</div>
