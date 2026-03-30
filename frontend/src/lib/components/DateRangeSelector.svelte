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

<style>
	.date-range-selector {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.date-range-selector button {
		min-height: 36px;
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
		font-size: var(--font-size-sm);
		background: var(--color-surface);
		border: 1.5px solid var(--color-border);
		color: var(--color-text);
	}

	.date-range-selector button:hover {
		border-color: var(--color-primary);
		background: var(--color-primary-light);
	}

	.date-range-selector button.active {
		background: var(--color-primary);
		color: var(--color-text-inverse);
		border-color: var(--color-primary);
	}

	.custom-range {
		display: flex;
		gap: var(--space-2);
		width: 100%;
		align-items: flex-end;
		flex-wrap: wrap;
		margin-top: var(--space-2);
	}

	.custom-range label {
		flex: 1;
		min-width: 120px;
	}

	.custom-range button {
		min-height: var(--touch-target);
	}
</style>
