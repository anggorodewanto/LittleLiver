<script lang="ts">
	import { labTestLabel } from '$lib/types/lab';

	interface Props {
		tests: string[];
		selected: Set<string>;
		onchange: (selected: Set<string>) => void;
		colors?: Map<string, string>;
	}

	let { tests, selected, onchange, colors }: Props = $props();

	function selectAll(): void {
		onchange(new Set());
	}

	function toggleTest(test: string): void {
		const next = new Set(selected);
		if (next.has(test)) {
			next.delete(test);
		} else {
			next.add(test);
		}
		onchange(next);
	}
</script>

<div class="test-filter">
	<button
		type="button"
		class={selected.size === 0 ? 'active' : ''}
		onclick={selectAll}
	>All</button>
	{#each tests as test (test)}
		<button
			type="button"
			class={selected.has(test) ? 'active' : ''}
			onclick={() => toggleTest(test)}
		>{#if colors?.has(test)}<span class="color-dot" style="background-color: {colors.get(test)}"></span>{/if}{labTestLabel(test)}</button>
	{/each}
</div>

<style>
	.test-filter {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2);
		margin-bottom: var(--space-4);
	}

	.test-filter button {
		min-height: 36px;
		padding: var(--space-1) var(--space-3);
		border-radius: var(--radius-full);
		font-size: var(--font-size-sm);
		background: var(--color-surface);
		border: 1.5px solid var(--color-border);
		color: var(--color-text);
	}

	.test-filter button:hover {
		border-color: var(--color-primary);
		background: var(--color-primary-light);
	}

	.color-dot {
		display: inline-block;
		width: 10px;
		height: 10px;
		border-radius: 50%;
		margin-right: var(--space-1, 4px);
		vertical-align: middle;
	}

	.test-filter button.active {
		background: var(--color-primary);
		color: var(--color-text-inverse);
		border-color: var(--color-primary);
	}
</style>
