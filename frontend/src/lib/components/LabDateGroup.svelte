<script lang="ts">
	import { goto } from '$app/navigation';
	import type { LabResult } from '$lib/types/lab';
	import { labTestLabel } from '$lib/types/lab';

	interface Props {
		date: string;
		results: LabResult[];
	}

	let { date, results }: Props = $props();

	function handleEdit(id: string): void {
		goto(`/log/lab?edit=${id}`);
	}

	let formattedDate = $derived(
		new Date(date + 'T00:00:00').toLocaleDateString('en-US', {
			month: 'short',
			day: 'numeric',
			year: 'numeric'
		})
	);

	let hasNormalRange = $derived(results.some((r) => r.normal_range));
	let hasNotes = $derived(results.some((r) => r.notes));
</script>

<section class="lab-date-group">
	<h3>{formattedDate}</h3>
	<table>
		<thead>
			<tr>
				<th>Test</th>
				<th>Value</th>
				<th>Unit</th>
				{#if hasNormalRange}
					<th>Range</th>
				{/if}
				{#if hasNotes}
					<th>Notes</th>
				{/if}
				<th></th>
			</tr>
		</thead>
		<tbody>
			{#each results as result (result.id)}
				<tr>
					<td class="test-name">{labTestLabel(result.test_name)}</td>
					<td class="value">{result.value}</td>
					<td class="unit">{result.unit ?? ''}</td>
					{#if hasNormalRange}
						<td class="range">{result.normal_range ?? ''}</td>
					{/if}
					{#if hasNotes}
						<td class="notes">{result.notes ?? ''}</td>
					{/if}
					<td class="actions">
						<button class="btn-edit" onclick={() => handleEdit(result.id)} aria-label="Edit">✏️</button>
					</td>
				</tr>
			{/each}
		</tbody>
	</table>
</section>

<style>
	.lab-date-group {
		background: var(--color-surface);
		border: 1px solid var(--color-border);
		border-radius: var(--radius-md);
		padding: var(--space-4);
		box-shadow: var(--shadow-sm);
	}

	h3 {
		margin: 0 0 var(--space-3) 0;
		font-size: var(--font-size-md);
		color: var(--color-text);
	}

	table {
		width: 100%;
		border-collapse: collapse;
		font-size: var(--font-size-sm);
	}

	th {
		text-align: left;
		padding: var(--space-1) var(--space-2);
		border-bottom: 1.5px solid var(--color-border);
		color: var(--color-text-muted);
		font-weight: 600;
		font-size: var(--font-size-xs);
		text-transform: uppercase;
	}

	td {
		padding: var(--space-1) var(--space-2);
	}

	.test-name {
		font-weight: 500;
	}

	.value {
		font-variant-numeric: tabular-nums;
	}

	.unit, .range {
		color: var(--color-text-muted);
	}

	.notes {
		font-style: italic;
		color: var(--color-text-muted);
	}

	.actions {
		text-align: right;
		width: 1%;
		white-space: nowrap;
	}

	.btn-edit {
		min-height: var(--touch-target);
		padding: var(--space-1) var(--space-2);
		border: 1px solid var(--color-primary);
		border-radius: var(--radius-md);
		font-size: var(--font-size-xs);
		cursor: pointer;
		background: var(--color-surface);
		color: var(--color-primary);
	}
</style>
