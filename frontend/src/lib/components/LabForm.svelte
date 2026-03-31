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

	interface Props {
		onsubmit: (data: LabPayload) => void;
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

	$effect(() => {
		timestamp = initialData ? fromISO8601(initialData.timestamp) : defaultTimestamp();
		testName = initialData?.test_name ?? '';
		value = initialData?.value ?? '';
		unit = initialData?.unit ?? '';
		normalRange = initialData?.normal_range ?? '';
		notes = initialData?.notes ?? '';
		validationError = '';
	});

	function selectQuickPick(pick: typeof QUICK_PICKS[number]) {
		testName = pick.testName;
		unit = pick.unit;
	}

	function handleSubmit(event: SubmitEvent) {
		event.preventDefault();

		if (!testName.trim()) {
			validationError = 'Test name is required';
			return;
		}

		if (!value.trim()) {
			validationError = 'Value is required';
			return;
		}

		validationError = '';
		const payload: LabPayload = {
			timestamp: toISO8601(timestamp),
			test_name: testName.trim(),
			value: value.trim()
		};

		if (unit.trim()) {
			payload.unit = unit.trim();
		}
		if (normalRange.trim()) {
			payload.normal_range = normalRange.trim();
		}
		if (notes.trim()) {
			payload.notes = notes.trim();
		}

		onsubmit(payload);
	}
</script>

<form onsubmit={handleSubmit}>
	<div>
		<label for="lab-timestamp">Timestamp</label>
		<input id="lab-timestamp" type="datetime-local" bind:value={timestamp} />
	</div>

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
		{submitting ? 'Logging...' : initialData ? 'Update Lab' : 'Log Lab'}
	</button>
</form>
