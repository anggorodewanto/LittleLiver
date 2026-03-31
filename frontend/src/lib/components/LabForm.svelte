<script lang="ts">
	import { defaultTimestamp, toISO8601, fromISO8601 } from '$lib/datetime';

	export interface LabPayload {
		timestamp: string;
		test_name: string;
		value: string;
		unit?: string;
		normal_range?: string;
		notes?: string;
	}

	export interface LabInitialData {
		timestamp: string;
		test_name: string;
		value: string;
		unit?: string;
		normal_range?: string;
		notes?: string;
	}

	interface SavedEntry {
		test_name: string;
		value: string;
		unit: string;
		normal_range: string;
	}

	interface Props {
		onsubmit: (data: LabPayload | LabPayload[]) => void;
		initialData?: LabInitialData;
		submitting?: boolean;
		error?: string;
	}

	let { onsubmit, initialData, submitting = false, error = '' }: Props = $props();

	const QUICK_PICKS = [
		{ label: 'Total Bilirubin', testName: 'total_bilirubin', unit: 'mg/dL' },
		{ label: 'Direct Bilirubin', testName: 'direct_bilirubin', unit: 'mg/dL' },
		{ label: 'ALT', testName: 'ALT', unit: 'U/L' },
		{ label: 'AST', testName: 'AST', unit: 'U/L' },
		{ label: 'GGT', testName: 'GGT', unit: 'U/L' },
		{ label: 'Albumin', testName: 'albumin', unit: 'g/dL' },
		{ label: 'INR', testName: 'INR', unit: '' },
		{ label: 'Platelets', testName: 'platelets', unit: '\u00d710\u00b3/\u00b5L' }
	] as const;

	let timestamp = $state(defaultTimestamp());
	let testName = $state('');
	let value = $state('');
	let unit = $state('');
	let normalRange = $state('');
	let notes = $state('');
	let validationError = $state('');
	let savedEntries = $state<SavedEntry[]>([]);

	let isEditMode = $derived(!!initialData);
	let hasSavedEntries = $derived(savedEntries.length > 0);

	$effect(() => {
		timestamp = initialData ? fromISO8601(initialData.timestamp) : defaultTimestamp();
		testName = initialData?.test_name ?? '';
		value = initialData?.value ?? '';
		unit = initialData?.unit ?? '';
		normalRange = initialData?.normal_range ?? '';
		notes = initialData?.notes ?? '';
		validationError = '';
		savedEntries = [];
	});

	function selectQuickPick(pick: typeof QUICK_PICKS[number]) {
		testName = pick.testName;
		unit = pick.unit;
	}

	function validateCurrentEntry(): boolean {
		if (!testName.trim()) {
			validationError = 'Test name is required';
			return false;
		}
		if (!value.trim()) {
			validationError = 'Value is required';
			return false;
		}
		validationError = '';
		return true;
	}

	function buildPayload(entry: SavedEntry): LabPayload {
		const payload: LabPayload = {
			timestamp: toISO8601(timestamp),
			test_name: entry.test_name,
			value: entry.value
		};
		if (entry.unit) payload.unit = entry.unit;
		if (entry.normal_range) payload.normal_range = entry.normal_range;
		return payload;
	}

	function handleAddMore() {
		if (!validateCurrentEntry()) return;

		savedEntries = [...savedEntries, {
			test_name: testName.trim(),
			value: value.trim(),
			unit: unit.trim(),
			normal_range: normalRange.trim()
		}];

		testName = '';
		value = '';
		unit = '';
		normalRange = '';
	}

	function removeEntry(index: number) {
		savedEntries = savedEntries.filter((_, i) => i !== index);
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (hasSavedEntries) {
			const entries: LabPayload[] = savedEntries.map(buildPayload);

			const hasCurrentEntry = testName.trim() && value.trim();
			if (hasCurrentEntry) {
				entries.push(buildPayload({
					test_name: testName.trim(),
					value: value.trim(),
					unit: unit.trim(),
					normal_range: normalRange.trim()
				}));
			}

			if (notes.trim()) {
				for (const entry of entries) {
					entry.notes = notes.trim();
				}
			}

			onsubmit(entries);
			return;
		}

		if (!validateCurrentEntry()) return;

		const payload: LabPayload = {
			timestamp: toISO8601(timestamp),
			test_name: testName.trim(),
			value: value.trim()
		};

		if (unit.trim()) payload.unit = unit.trim();
		if (normalRange.trim()) payload.normal_range = normalRange.trim();
		if (notes.trim()) payload.notes = notes.trim();

		onsubmit(payload);
	}

	function getSubmitLabel(): string {
		if (submitting) return 'Logging...';
		if (isEditMode) return 'Update Lab';
		if (hasSavedEntries) return 'Log Labs';
		return 'Log Lab';
	}

	function getTestLabel(testName: string): string {
		const pick = QUICK_PICKS.find(p => p.testName === testName);
		return pick ? pick.label : testName;
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="lab-timestamp">Timestamp</label>
		<input id="lab-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

	{#if hasSavedEntries}
		<div class="saved-entries">
			<h3>Added tests</h3>
			{#each savedEntries as entry, i (i)}
				<div class="saved-entry">
					<span class="entry-summary">
						<strong>{getTestLabel(entry.test_name)}</strong>: {entry.value}{entry.unit ? ` ${entry.unit}` : ''}
					</span>
					<button type="button" onclick={() => removeEntry(i)} class="remove-btn" aria-label="Remove {entry.test_name}">
						Remove
					</button>
				</div>
			{/each}
		</div>
	{/if}

	<fieldset>
		<legend>Quick Pick</legend>
		<div style="display: flex; gap: 8px; flex-wrap: wrap;">
			{#each QUICK_PICKS as pick (pick.testName)}
				<button
					type="button"
					aria-pressed={testName === pick.testName ? 'true' : 'false'}
					onclick={() => selectQuickPick(pick)}
				>
					{pick.label}
				</button>
			{/each}
		</div>
	</fieldset>

	<div>
		<label for="lab-test-name">Test name</label>
		<input id="lab-test-name" type="text" bind:value={testName} />
	</div>

	<div>
		<label for="lab-value">Value</label>
		<input id="lab-value" type="text" bind:value={value} />
	</div>

	<div>
		<label for="lab-unit">Unit</label>
		<input id="lab-unit" type="text" bind:value={unit} />
	</div>

	<div>
		<label for="lab-normal-range">Normal range</label>
		<input id="lab-normal-range" type="text" bind:value={normalRange} placeholder="e.g., 0.1-1.2" />
	</div>

	{#if !isEditMode}
		<button type="button" onclick={handleAddMore} class="add-more-btn">
			Add More
		</button>
	{/if}

	<div>
		<label for="lab-notes">Notes</label>
		<textarea id="lab-notes" bind:value={notes}></textarea>
	</div>

	{#if validationError}
		<p role="alert">{validationError}</p>
	{/if}

	{#if error}
		<p role="alert">{error}</p>
	{/if}

	<button type="submit" disabled={submitting}>
		{getSubmitLabel()}
	</button>
</form>

<style>
	.saved-entries {
		margin: var(--space-3, 1rem) 0;
		padding: var(--space-2, 0.5rem);
		background: var(--color-surface, #f8f9fa);
		border-radius: var(--radius, 8px);
	}

	.saved-entries h3 {
		margin: 0 0 var(--space-2, 0.5rem);
		font-size: var(--font-size-sm, 0.875rem);
		color: var(--color-text-muted);
	}

	.saved-entry {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: var(--space-1, 0.25rem) 0;
		gap: var(--space-2, 0.5rem);
	}

	.saved-entry + .saved-entry {
		border-top: 1px solid var(--color-border, #e0e0e0);
	}

	.entry-summary {
		font-size: var(--font-size-sm, 0.875rem);
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

	.add-more-btn {
		width: 100%;
		margin: var(--space-2, 0.5rem) 0;
	}
</style>
