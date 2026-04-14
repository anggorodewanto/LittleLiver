<script lang="ts">
	import {
		fetchLabSuggestions,
		mergeWithQuickPicks,
		findSuggestionMatch,
		type LabTestSuggestion
	} from '$lib/labSuggestions';

	export interface ExtractedItem {
		test_name: string;
		value: string;
		unit: string;
		normal_range: string;
		confidence: string;
		existing_match?: {
			id: string;
			timestamp: string;
			value: string;
			unit: string;
		};
	}

	export interface ReviewedLabPayload {
		test_name: string;
		value: string;
		unit?: string;
		normal_range?: string;
	}

	interface Props {
		extracted: ExtractedItem[];
		notes: string;
		onconfirm: (items: ReviewedLabPayload[]) => void;
		oncancel: () => void;
		babyId?: string;
	}

	let { extracted, notes, onconfirm, oncancel, babyId }: Props = $props();

	let dbSuggestions = $state<LabTestSuggestion[]>([]);
	let suggestionsReady = $state(false);
	let allSuggestions = $derived(mergeWithQuickPicks(dbSuggestions));

	$effect(() => {
		if (!babyId) {
			suggestionsReady = true;
			return;
		}
		fetchLabSuggestions(babyId).then((data) => {
			dbSuggestions = data;
			suggestionsReady = true;
		});
	});

	interface EditableRow {
		test_name: string;
		value: string;
		unit: string;
		normal_range: string;
		confidence: string;
		checked: boolean;
		existing_match?: ExtractedItem['existing_match'];
		enriched?: boolean;
	}

	// eslint-disable-next-line svelte/prefer-writable-derived -- rows are user-editable (checked, removed, field edits)
	let rows = $state<EditableRow[]>([]);

	$effect(() => {
		rows = extracted.map((item) => ({
			test_name: item.test_name,
			value: item.value,
			unit: item.unit,
			normal_range: item.normal_range,
			confidence: item.confidence,
			checked: !item.existing_match,
			existing_match: item.existing_match
		}));
	});

	function lookupMatch(name: string): LabTestSuggestion | undefined {
		// Prefer DB (canonical names) over QUICK_PICKS so AST → SGOT/AST works.
		return findSuggestionMatch(name, dbSuggestions) ?? findSuggestionMatch(name, allSuggestions);
	}

	$effect(() => {
		if (!suggestionsReady) return;
		for (const row of rows) {
			if (row.enriched) continue;
			row.enriched = true;
			const match = lookupMatch(row.test_name);
			if (!match) continue;
			if (row.test_name !== match.test_name) row.test_name = match.test_name;
			if (!row.unit && match.unit) row.unit = match.unit;
			if (!row.normal_range && match.normal_range) row.normal_range = match.normal_range;
		}
	});

	function handleTestNameInput(index: number) {
		const row = rows[index];
		if (!row) return;
		const match = lookupMatch(row.test_name);
		if (!match) return;
		if (!row.unit && match.unit) row.unit = match.unit;
		if (!row.normal_range && match.normal_range) row.normal_range = match.normal_range;
	}

	function removeRow(index: number) {
		rows = rows.filter((_, i) => i !== index);
	}

	function addRow() {
		rows = [...rows, {
			test_name: '',
			value: '',
			unit: '',
			normal_range: '',
			confidence: 'manual',
			checked: true
		}];
	}

	function handleConfirm() {
		const payload: ReviewedLabPayload[] = rows
			.filter((r) => r.checked)
			.map((r) => {
				const item: ReviewedLabPayload = {
					test_name: r.test_name,
					value: r.value
				};
				if (r.unit) item.unit = r.unit;
				if (r.normal_range) item.normal_range = r.normal_range;
				return item;
			});
		onconfirm(payload);
	}
</script>

{#if rows.length === 0 && extracted.length === 0}
	<p>No lab results found in this image.</p>
	<button type="button" onclick={oncancel}>Back</button>
{:else}
	<h2>Review Extracted Results</h2>

	{#if notes}
		<p class="extraction-notes">{notes}</p>
	{/if}

	<div class="review-rows">
		{#each rows as row, i (i)}
			<div class="review-row">
				<label class="row-checkbox">
					<input type="checkbox" bind:checked={row.checked} />
				</label>

				<div class="row-fields">
					<div class="field-group">
						<label for={`test-name-${i}`}>Test</label>
						<input
							id={`test-name-${i}`}
							type="text"
							list="lab-test-suggestions"
							bind:value={row.test_name}
							oninput={() => handleTestNameInput(i)}
						/>
					</div>
					<div class="field-group">
						<label for={`value-${i}`}>Value</label>
						<input id={`value-${i}`} type="text" bind:value={row.value} />
					</div>
					<div class="field-group">
						<label for={`unit-${i}`}>Unit</label>
						<input id={`unit-${i}`} type="text" bind:value={row.unit} />
					</div>
					<div class="field-group">
						<label for={`normal-range-${i}`}>Range</label>
						<input id={`normal-range-${i}`} type="text" bind:value={row.normal_range} />
					</div>

					<span
						data-testid={`confidence-${i}`}
						class="confidence-badge confidence-{row.confidence}"
					>
						{row.confidence}
					</span>

					{#if row.existing_match}
						<span class="duplicate-badge">Already logged</span>
					{/if}

					<button
						type="button"
						class="remove-btn"
						aria-label="Remove {row.test_name}"
						onclick={() => removeRow(i)}
					>
						Remove
					</button>
				</div>
			</div>
		{/each}
	</div>

	<datalist id="lab-test-suggestions">
		{#each allSuggestions as suggestion (suggestion.test_name)}
			<option value={suggestion.test_name}></option>
		{/each}
	</datalist>

	<button type="button" class="add-row-btn" onclick={addRow}>Add row</button>

	<div class="actions">
		<button type="button" onclick={oncancel}>Cancel</button>
		<button type="button" onclick={handleConfirm}>Confirm</button>
	</div>
{/if}

<style>
	.review-rows {
		display: flex;
		flex-direction: column;
		gap: var(--space-3, 1rem);
	}

	.review-row {
		display: flex;
		align-items: flex-start;
		gap: var(--space-2, 0.5rem);
		padding: var(--space-2, 0.5rem);
		border: 1px solid var(--color-border, #e0e0e0);
		border-radius: var(--radius, 8px);
	}

	.row-fields {
		display: flex;
		flex-wrap: wrap;
		gap: var(--space-2, 0.5rem);
		align-items: center;
		flex: 1;
	}

	.field-group {
		display: flex;
		flex-direction: column;
		gap: 2px;
	}

	.field-group label {
		font-size: var(--font-size-xs, 0.75rem);
		color: var(--color-text-muted);
	}

	.field-group input {
		width: auto;
		min-width: 60px;
		max-width: 150px;
	}

	.confidence-badge {
		font-size: var(--font-size-xs, 0.75rem);
		padding: 2px 6px;
		border-radius: var(--radius-sm, 4px);
	}

	.confidence-high {
		background: var(--color-success-bg, #d4edda);
		color: var(--color-success, #155724);
	}

	.confidence-medium {
		background: var(--color-warning-bg, #fff3cd);
		color: var(--color-warning, #856404);
	}

	.confidence-low {
		background: var(--color-danger-bg, #f8d7da);
		color: var(--color-danger, #721c24);
	}

	.duplicate-badge {
		font-size: var(--font-size-xs, 0.75rem);
		padding: 2px 6px;
		border-radius: var(--radius-sm, 4px);
		background: var(--color-warning-bg, #fff3cd);
		color: var(--color-warning, #856404);
		font-weight: 600;
	}

	.remove-btn {
		font-size: var(--font-size-xs, 0.75rem);
		padding: var(--space-1, 0.25rem) var(--space-2, 0.5rem);
		color: var(--color-danger, #dc3545);
		background: none;
		border: 1px solid var(--color-danger, #dc3545);
		border-radius: var(--radius-sm, 4px);
		cursor: pointer;
		min-height: auto;
	}

	.actions {
		display: flex;
		gap: var(--space-2, 0.5rem);
		margin-top: var(--space-3, 1rem);
	}

	.extraction-notes {
		font-size: var(--font-size-sm, 0.875rem);
		color: var(--color-text-muted);
		font-style: italic;
		margin-bottom: var(--space-3, 1rem);
	}

	.row-checkbox {
		display: flex;
		align-items: center;
		padding-top: var(--space-3, 1rem);
	}
</style>
